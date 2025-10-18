package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"hypervisor/internal/errmsg"
	"hypervisor/internal/events"
	"hypervisor/internal/models"
	"hypervisor/internal/paths"

	"go.mongodb.org/mongo-driver/mongo"
)

func stageID(releaseID, envTag string) string {
	return fmt.Sprintf("%s-%s", releaseID, envTag)
}

// PrepareStage bootstraps a stage in `pre` status and returns the seeded env template.
func PrepareStage(ctx context.Context, releaseID, envTag string) (*models.Stage, string, error) {
	id := stageID(releaseID, envTag)

	if existing, err := models.GetStageByID(ctx, id); err == nil && existing != nil {
		return nil, "", errmsg.StageAlreadyExists
	} else if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		if events.Em != nil {
			events.Em.StageFailed(releaseID, envTag, err)
		}
		return nil, "", err
	}

	release, err := models.GetReleaseByID(ctx, releaseID)
	if err != nil {
		if events.Em != nil {
			events.Em.StageFailed(releaseID, envTag, err)
		}
		return nil, "", errmsg.StageReleaseNotFound
	}

	repoURL := strings.TrimSpace(os.Getenv("REPO_URL"))
	if repoURL == "" {
		repoURL = "https://github.com/OpenLabsRo/openhack-backend"
	}

	repoPath := paths.OpenHackRepoPath(releaseID)
	if err := cloneAndCheckout(repoURL, repoPath, release.Sha); err != nil {
		if events.Em != nil {
			events.Em.StageFailed(releaseID, envTag, err)
		}
		return nil, "", err
	}

	template, err := loadEnvTemplate()
	if err != nil {
		if events.Em != nil {
			events.Em.StageFailed(releaseID, envTag, err)
		}
		return nil, "", err
	}

	stage := models.Stage{
		ID:        id,
		ReleaseID: releaseID,
		EnvTag:    envTag,
		Status:    models.StageStatusPre,
		EnvText:   template,
		RepoPath:  repoPath,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := models.CreateStage(ctx, stage); err != nil {
		if events.Em != nil {
			events.Em.StageFailed(releaseID, envTag, err)
		}
		return nil, "", err
	}

	if err := writeStageEnvFile(stage.ID, template); err != nil {
		if events.Em != nil {
			events.Em.StageFailed(releaseID, envTag, err)
		}
		return nil, "", err
	}

	if events.Em != nil {
		events.Em.StagePrepared(stage)
	}

	return &stage, template, nil
}

// SubmitStageSession records a new environment snapshot for the stage and promotes status to active.
func SubmitStageSession(ctx context.Context, stageID string, envText string, author string, notes string, source string) (*models.StageSession, *models.Stage, error) {
	stage, err := models.GetStageByID(ctx, stageID)
	if err != nil {
		return nil, nil, errmsg.StageNotFound
	}

	if strings.TrimSpace(envText) == "" {
		return nil, nil, errmsg.StageInvalidRequest
	}

	sessionID := fmt.Sprintf("%s-%d", stageID, time.Now().UnixNano())
	src := strings.TrimSpace(source)
	if src == "" {
		src = "manual"
	}

	session := models.StageSession{
		ID:      sessionID,
		StageID: stageID,
		EnvText: envText,
		Author:  author,
		Notes:   notes,
		Source:  src,
	}

	if err := models.CreateStageSession(ctx, session); err != nil {
		return nil, nil, err
	}

	if err := writeStageEnvFile(stageID, envText); err != nil {
		return nil, nil, err
	}

	stage.EnvText = envText
	stage.LatestSessionID = sessionID
	stage.LastTestResultID = ""
	stage.UpdatedAt = time.Now()
	if stage.Status == models.StageStatusPre {
		stage.Status = models.StageStatusActive
	}

	if err := models.UpdateStage(ctx, *stage); err != nil {
		return nil, nil, err
	}

	if events.Em != nil {
		events.Em.StageSessionCreated(*stage, session)
		events.Em.StageEnvUpdated(*stage, session)
	}

	return &session, stage, nil
}

func writeStageEnvFile(stageID string, envText string) error {
	envDir := paths.OpenHackEnvPath(stageID)
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		return err
	}

	envPath := filepath.Join(envDir, ".env")
	return os.WriteFile(envPath, []byte(envText), 0o640)
}

// StartStageTest bootstraps a manual test run for the provided stage/session.
func StartStageTest(ctx context.Context, stageID, sessionID string) (*models.StageTestResult, error) {
	stage, err := models.GetStageByID(ctx, stageID)
	if err != nil {
		return nil, errmsg.StageNotFound
	}

	session, err := models.GetStageSessionByID(ctx, sessionID)
	if err != nil {
		return nil, errmsg.StageSessionNotFound
	}

	if session.StageID != stage.ID {
		return nil, errmsg.StageInvalidRequest
	}

	resultID := fmt.Sprintf("%s-test-%d", stageID, time.Now().UnixNano())
	wsToken := fmt.Sprintf("%s-token-%d", stageID, time.Now().UnixNano())
	logPath := filepath.Join(paths.OpenHackBaseDir, "runtime", "logs", fmt.Sprintf("%s.log", resultID))

	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return nil, err
	}

	result := models.StageTestResult{
		ID:        resultID,
		StageID:   stage.ID,
		SessionID: session.ID,
		Status:    models.StageTestStatusRunning,
		WsToken:   wsToken,
		LogPath:   logPath,
		StartedAt: time.Now(),
	}

	if err := models.CreateStageTestResult(ctx, result); err != nil {
		return nil, err
	}

	session.TestResultID = result.ID
	if err := models.UpdateStageSession(ctx, *session); err != nil {
		return nil, err
	}

	stage.LastTestResultID = result.ID
	stage.UpdatedAt = time.Now()
	if err := models.UpdateStage(ctx, *stage); err != nil {
		return nil, err
	}

	if events.Em != nil {
		events.Em.StageTestStarted(*stage, *session, result)
	}

	go runStageTest(context.Background(), stage.RepoPath, stage.EnvTag, stage.ID, session.ID, result)

	return &result, nil
}

func runStageTest(ctx context.Context, repoPath, envTag, stageID, sessionID string, result models.StageTestResult) {
	envRoot := paths.OpenHackEnvPath(stageID)
	logFile, err := os.OpenFile(result.LogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		finish := time.Now()
		_ = models.UpdateStageTestStatus(ctx, result.ID, models.StageTestStatusError, &finish, err.Error())
		if events.Em != nil {
			events.Em.StageTestFailed(stageID, sessionID, result.ID, err.Error())
		}
		return
	}
	defer logFile.Close()

	writer := io.MultiWriter(logFile)
	cmdCtx := ctx
	if cmdCtx == nil {
		cmdCtx = context.Background()
	}

	cmd := exec.CommandContext(cmdCtx, "./TEST", "--env-root", envRoot, "--app-version", stageID)
	cmd.Dir = repoPath
	cmd.Stdout = writer
	cmd.Stderr = writer
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("DEPLOYMENT=%s", envTag),
		fmt.Sprintf("APP_VERSION=%s", stageID),
	)

	runErr := cmd.Run()
	finishedAt := time.Now()

	status := models.StageTestStatusPassed
	errMsg := ""

	switch {
	case runErr == nil:
		status = models.StageTestStatusPassed
	case errors.Is(runErr, context.Canceled):
		status = models.StageTestStatusCanceled
	default:
		status = models.StageTestStatusFailed
		errMsg = runErr.Error()
	}

	if err := models.UpdateStageTestStatus(context.Background(), result.ID, status, &finishedAt, errMsg); err != nil {
		return
	}

	if events.Em != nil {
		switch status {
		case models.StageTestStatusPassed:
			events.Em.StageTestPassed(stageID, sessionID, result.ID, finishedAt.Sub(result.StartedAt))
		case models.StageTestStatusCanceled:
			events.Em.StageTestCanceled(stageID, sessionID, result.ID)
		default:
			events.Em.StageTestFailed(stageID, sessionID, result.ID, errMsg)
		}
	}
}

func cloneAndCheckout(repoURL, repoPath, sha string) error {
	if err := os.RemoveAll(repoPath); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(repoPath), 0o755); err != nil {
		return err
	}

	cmd := exec.Command("git", "clone", repoURL, repoPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %w (%s)", err, string(output))
	}

	cmd = exec.Command("git", "checkout", sha)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout failed: %w (%s)", err, string(output))
	}

	return nil
}

func loadEnvTemplate() (string, error) {
	templatePath := paths.OpenHackEnvPath("template/.env")
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// PromoteStage creates a deployment document for the provided stage.
func PromoteStage(ctx context.Context, stageID string) (*models.Deployment, error) {
	stage, err := models.GetStageByID(ctx, stageID)
	if err != nil {
		return nil, errmsg.StageNotFound
	}

	if stage.Status == models.StageStatusPre {
		return nil, errmsg.StageMissingEnv
	}

	deploymentID := stageID
	deployment := models.Deployment{
		ID:         deploymentID,
		Version:    stage.ReleaseID,
		EnvTag:     stage.EnvTag,
		StageID:    stage.ID,
		Port:       nil,
		Status:     "staged",
		CreatedAt:  time.Now(),
		PromotedAt: nil,
	}

	if err := models.CreateDeployment(ctx, deployment); err != nil {
		if events.Em != nil {
			events.Em.DeploymentCreateFailed(deploymentID, err)
		}
		return nil, err
	}

	if events.Em != nil {
		events.Em.DeploymentCreated(deployment)
	}

	stage.Status = models.StageStatusPromoted
	stage.UpdatedAt = time.Now()
	if err := models.UpdateStage(ctx, *stage); err != nil {
		return &deployment, err
	}

	return &deployment, nil
}

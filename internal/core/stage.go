package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"hypervisor/internal/errmsg"
	"hypervisor/internal/events"
	"hypervisor/internal/fsutil"
	"hypervisor/internal/gitops"
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

	repoPath := paths.OpenHackRepoPath(id)
	if err := gitops.CloneAndCheckout(repoURL, repoPath, release.Sha); err != nil {
		if events.Em != nil {
			events.Em.StageFailed(releaseID, envTag, err)
		}
		return nil, "", err
	}

	template, err := ReadEnvTemplate()
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

func writeStageEnvFile(stageID string, envText string) error {
	envDir := paths.OpenHackEnvPath(stageID)
	if err := fsutil.EnsureDir(envDir, 0o755); err != nil {
		return err
	}

	envPath := filepath.Join(envDir, ".env")
	return fsutil.WriteFile(envPath, []byte(envText), 0o640)
}

// ReadStageEnv loads the current .env contents for the provided stage.
func ReadStageEnv(stageID string) (string, error) {
	envPath := filepath.Join(paths.OpenHackEnvPath(stageID), ".env")
	data, err := os.ReadFile(envPath)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// UpdateStageEnv writes the provided environment to disk and updates stage metadata.
func UpdateStageEnv(ctx context.Context, stageID string, envText string) (*models.Stage, error) {
	stage, err := models.GetStageByID(ctx, stageID)
	if err != nil {
		return nil, errmsg.StageNotFound
	}

	if strings.TrimSpace(envText) == "" {
		return nil, errmsg.StageInvalidRequest
	}

	if err := writeStageEnvFile(stageID, envText); err != nil {
		return nil, err
	}

	stage.LastTestID = ""
	stage.UpdatedAt = time.Now()
	if stage.Status == models.StageStatusPre {
		stage.Status = models.StageStatusActive
	}

	if err := models.UpdateStage(ctx, *stage); err != nil {
		return nil, err
	}

	if events.Em != nil {
		events.Em.StageEnvUpdated(*stage)
	}

	return stage, nil
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

// DeleteStage removes a stage and all of its related resources.
func DeleteStage(ctx context.Context, stageID string) error {
	stage, err := models.GetStageByID(ctx, stageID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errmsg.StageNotFound
		}
		return err
	}

	repoPath := paths.OpenHackRepoPath(stage.ID)
	if err := fsutil.RemoveAll(repoPath); err != nil {
		return err
	}

	envDir := paths.OpenHackEnvPath(stage.ID)
	if err := fsutil.RemoveAll(envDir); err != nil {
		return err
	}

	tests, err := models.DeleteTestsByStageID(ctx, stage.ID)
	if err != nil {
		return err
	}

	for _, test := range tests {
		if test.LogPath == "" {
			continue
		}
		if err := fsutil.Remove(test.LogPath); err != nil {
			return err
		}
	}

	if err := models.DeleteStage(ctx, stage.ID); err != nil {
		return err
	}

	if events.Em != nil {
		events.Em.StageDeleted(stage.ID)
	}

	return nil
}

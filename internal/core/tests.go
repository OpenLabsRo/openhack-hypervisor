package core

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"syscall"
	"time"

	"hypervisor/internal/errmsg"
	"hypervisor/internal/events"
	"hypervisor/internal/fs"
	"hypervisor/internal/models"
	"hypervisor/internal/paths"
)

// StartTest bootstraps a manual test run for the provided stage.
func StartTest(ctx context.Context, stageID string) (*models.Test, error) {
	stage, err := models.GetStageByID(ctx, stageID)
	if err != nil {
		return nil, errmsg.StageNotFound
	}

	// If stage is ready, mark as pre until test passes
	if stage.Status == models.StageStatusReady {
		stage.Status = models.StageStatusPre
		stage.UpdatedAt = time.Now()
		if err := models.UpdateStage(ctx, *stage); err != nil {
			return nil, err
		}
	}

	sequence, err := models.NextTestSequence(ctx, stage.ID)
	if err != nil {
		return nil, err
	}

	stage.TestSequence = sequence

	resultID := fmt.Sprintf("%s-test-%d", stageID, sequence)
	wsToken := fmt.Sprintf("%s-token-%d", stageID, time.Now().UnixNano())
	logPath := filepath.Join(paths.OpenHackRuntimeLogsDir, fmt.Sprintf("%s.log", resultID))

	if err := fs.EnsureDir(filepath.Dir(logPath), 0o755); err != nil {
		return nil, err
	}

	test := models.Test{
		ID:        resultID,
		StageID:   stage.ID,
		Status:    models.TestStatusRunning,
		WsToken:   wsToken,
		LogPath:   logPath,
		StartedAt: time.Now(),
	}

	if err := models.CreateTest(ctx, test); err != nil {
		return nil, err
	}

	if events.Em != nil {
		events.Em.TestStarted(*stage, test)
	}

	repoPath := paths.OpenHackRepoPath(stage.ID)
	go runTest(context.Background(), repoPath, stage.ID, test)

	return &test, nil
}

func runTest(ctx context.Context, repoPath, stageID string, test models.Test) {
	envRoot := paths.OpenHackEnvPath(stageID)

	logFile, err := os.OpenFile(test.LogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		finish := time.Now()
		_ = models.UpdateTestStatus(ctx, test.ID, models.TestStatusError, &finish, err.Error())
		if events.Em != nil {
			events.Em.TestFailed(stageID, test.ID, err.Error())
		}
		return
	}
	defer logFile.Close()

	writer := io.MultiWriter(logFile)
	cmdCtx := ctx
	if cmdCtx == nil {
		cmdCtx = context.Background()
	}

	testVersion := fmt.Sprintf("%s_test", stageID)
	cmd := exec.CommandContext(cmdCtx, "./TEST", "--env-root", envRoot, "--app-version", testVersion)
	cmd.Dir = repoPath
	cmd.Stdout = writer
	cmd.Stderr = writer
	runErr := cmd.Run()
	finishedAt := time.Now()

	status := models.TestStatusPassed
	errMsg := ""

	switch {
	case runErr == nil:
		status = models.TestStatusPassed
	case errors.Is(runErr, context.Canceled):
		status = models.TestStatusCanceled
	default:
		status = models.TestStatusFailed
		errMsg = runErr.Error()
	}

	if err := models.UpdateTestStatus(context.Background(), test.ID, status, &finishedAt, errMsg); err != nil {
		return
	}

	if events.Em != nil {
		switch status {
		case models.TestStatusPassed:
			events.Em.TestPassed(stageID, test.ID, finishedAt.Sub(test.StartedAt))
			// Mark stage as ready for deployment
			stage, err := models.GetStageByID(context.Background(), stageID)
			if err == nil && stage.Status == models.StageStatusPre {
				stage.Status = models.StageStatusReady
				stage.UpdatedAt = time.Now()
				models.UpdateStage(context.Background(), *stage)
			}
		case models.TestStatusCanceled:
			events.Em.TestCanceled(stageID, test.ID)
		default:
			events.Em.TestFailed(stageID, test.ID, errMsg)
		}
	}
}

// ListTests returns all tests for a given stage ordered by creation time (most recent first).
func ListTests(ctx context.Context, stageID string) ([]models.Test, error) {
	if _, err := models.GetStageByID(ctx, stageID); err != nil {
		return nil, errmsg.StageNotFound
	}

	tests, err := models.ListTestsByStageID(ctx, stageID)
	if err != nil {
		return nil, err
	}

	// ensure deterministic ordering (most recent first)
	sort.SliceStable(tests, func(i, j int) bool {
		return tests[i].StartedAt.After(tests[j].StartedAt)
	})

	return tests, nil
}

// StatusWriter defines an interface for writing status messages during log streaming.
type StatusWriter interface {
	WriteStatus(level, message string)
}

// StreamLogFile waits for a log file to appear and tails it, sending lines to the writer.
// This function is framework-agnostic and can be tested independently.
func StreamLogFile(ctx context.Context, logPath, testID string, w io.Writer, sw StatusWriter) error {
	sw.WriteStatus("info", "waiting for log file")

	var file *os.File
	var err error

	// Wait for the file to be created.
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()
waitLoop:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			file, err = os.Open(logPath)
			if err == nil {
				break waitLoop
			}
			if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, syscall.ENOENT) {
				return fmt.Errorf("failed to open log file: %w", err)
			}
		}
	}
	defer file.Close()

	sw.WriteStatus("info", "log stream starting")

	// Tail the file.
	return tailFile(ctx, file, testID, w)
}

// tailFile reads from the file and sends new lines to the writer.
func tailFile(ctx context.Context, file *os.File, testID string, w io.Writer) error {
	reader := bufio.NewReader(file)
	pollTicker := time.NewTicker(200 * time.Millisecond)
	defer pollTicker.Stop()

	for {
		// Read all available lines.
		for {
			line, err := reader.ReadBytes('\n')
			if len(line) > 0 {
				// Trim trailing newline characters for clean output.
				line = bytes.TrimRight(line, "\r\n")
				if _, writeErr := w.Write(line); writeErr != nil {
					return writeErr // Likely a closed connection.
				}
			}
			if err == io.EOF {
				break // No more lines right now, wait for more content.
			}
			if err != nil {
				return fmt.Errorf("error reading log file: %w", err)
			}
		}

		// Check if the test run has finished.
		// Use a background context for this check to not fail if the streaming context is canceled.
		latest, err := models.GetTestByID(context.Background(), testID)
		if err == nil && latest.Status != models.TestStatusRunning {
			// The test is done. Do one final read to catch any remaining lines.
			for {
				line, err := reader.ReadBytes('\n')
				if len(line) > 0 {
					line = bytes.TrimRight(line, "\r\n")
					if _, writeErr := w.Write(line); writeErr != nil {
						return writeErr
					}
				}
				if err == io.EOF {
					break
				}
				if err != nil {
					// Don't fail the entire stream for a final read error, just log it.
					// A real error would have been caught in the main loop.
					break
				}
			}
			return nil // End of stream.
		}

		// Wait before polling again.
		select {
		case <-ctx.Done():
			// The client disconnected or the stream was otherwise cancelled.
			// We can do a final read to send any last-minute log lines.
			line, readErr := io.ReadAll(reader)
			if readErr == nil && len(line) > 0 {
				line = bytes.TrimRight(line, "\r\n")
				_, _ = w.Write(line) // Best effort, ignore error
			}
			return ctx.Err()
		case <-pollTicker.C:
			// Continue to the next iteration.
		}
	}
}

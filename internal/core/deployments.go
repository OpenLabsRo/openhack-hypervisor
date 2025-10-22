package core

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"hypervisor/internal/events"
	"hypervisor/internal/fs"
	"hypervisor/internal/models"
	"hypervisor/internal/paths"
	"hypervisor/internal/proxy"
	"hypervisor/internal/systemd"
)

// ProvisionDeployment handles the asynchronous provisioning of a deployment.
// It builds the backend, installs the systemd service, and updates the deployment status.
// All progress is logged to the deployment's log file.
func ProvisionDeployment(dep models.Deployment) {
	ctx := context.Background()

	// Ensure the log directory exists
	if err := os.MkdirAll(filepath.Dir(dep.LogPath), 0o755); err != nil {
		log.Printf("Failed to create log directory for deployment %s: %v", dep.ID, err)
		dep.Status = models.DeploymentStatusProvisionFailed
		models.UpdateDeployment(ctx, dep)
		return
	}

	logFile, err := os.OpenFile(dep.LogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o666)
	if err != nil {
		log.Printf("Failed to open log file for deployment %s: %v", dep.ID, err)
		dep.Status = models.DeploymentStatusProvisionFailed
		models.UpdateDeployment(ctx, dep)
		return
	}
	defer logFile.Close()

	logger := &deploymentLogger{writer: logFile}

	logger.Log("Starting deployment provisioning for %s", dep.ID)

	// Build the backend binary
	logger.Log("Building backend binary...")
	repoPath := paths.OpenHackRepoPath(dep.StageID)
	buildPath := paths.OpenHackBuildsDir
	versionWithoutV := strings.TrimPrefix(dep.Version, "v")
	binaryPath := filepath.Join(buildPath, versionWithoutV)
	if err := buildBackend(repoPath, buildPath, binaryPath, logFile); err != nil {
		logger.Log("Build failed: %v", err)
		dep.Status = models.DeploymentStatusBuildFailed
		models.UpdateDeployment(ctx, dep)
		if events.Em != nil {
			events.Em.DeploymentCreateFailed(dep.ID, err)
		}
		return
	}
	logger.Log("Build completed successfully")

	// Install systemd service
	logger.Log("Installing systemd service...")
	envRoot := paths.OpenHackEnvPath(dep.StageID)
	cfg := systemd.BackendServiceConfig{
		DeploymentID: dep.ID,
		BinaryPath:   binaryPath,
		EnvTag:       dep.EnvTag,
		Port:         *dep.Port,
		EnvRoot:      envRoot,
		Version:      dep.StageID,
	}
	if err := systemd.InstallBackendService(cfg, logFile); err != nil {
		logger.Log("Systemd install failed: %v", err)
		dep.Status = models.DeploymentStatusProvisionFailed
		models.UpdateDeployment(ctx, dep)
		if events.Em != nil {
			events.Em.DeploymentCreateFailed(dep.ID, err)
		}
		return
	}
	logger.Log("Systemd service installed and started")

	// Update deployment to ready. Do NOT auto-promote to main here.
	// Promotion (making this the main deployment) must be an explicit operator action
	// so we only mark the deployment as ready and persist it. The Promote API will
	// handle updating the proxy routing map and marking the stage as promoted.
	dep.Status = models.DeploymentStatusReady
	if err := models.UpdateDeployment(ctx, dep); err != nil {
		logger.Log("Failed to update deployment status: %v", err)
		return
	}

	// Make the deployment immediately routable at /<stageID>/* by updating the route map.
	// Do NOT set PromotedAt here — promotion to main must be explicit.
	proxy.GlobalRouteMap.UpdateDeployment(&dep)

	logger.Log("Deployment is now ready and routable under its stage (not promoted)")

	if events.Em != nil {
		events.Em.DeploymentCreated(dep)
	}
}

// deploymentLogger writes formatted log messages to the writer.
type deploymentLogger struct {
	writer io.Writer
}

func (l *deploymentLogger) Log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.writer, "[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
}

func buildBackend(repoPath, buildPath, binaryPath string, logWriter io.Writer) error {
	if err := fs.EnsureDir(buildPath, 0o755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}

	if _, err := os.Stat(binaryPath); err == nil {
		// Already built
		fmt.Fprintf(logWriter, "[%s] Binary already exists at %s, skipping build\n", time.Now().Format("2006-01-02 15:04:05"), binaryPath)
		return nil
	}

	// Check if Go is available
	checkCmd := exec.Command("sh", "-c", "command -v go >/dev/null 2>&1")
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("Go is not installed or not in PATH")
	}

	fmt.Fprintf(logWriter, "[%s] Running ./BUILD command in %s with output %s\n", time.Now().Format("2006-01-02 15:04:05"), repoPath, buildPath)
	cmd := exec.Command("./BUILD", "--output", buildPath)
	cmd.Dir = repoPath
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build command failed: %w", err)
	}
	fmt.Fprintf(logWriter, "[%s] Build command completed successfully\n", time.Now().Format("2006-01-02 15:04:05"))
	return nil
}

// StreamDeploymentLogFile waits for a deployment log file to appear and tails it, sending lines to the writer.
// Similar to StreamLogFile for tests, but for deployments.
func StreamDeploymentLogFile(ctx context.Context, logPath, deploymentID string, w io.Writer, sw StatusWriter) error {
	sw.WriteStatus("info", "waiting for deployment log file")

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
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to open log file: %w", err)
			}
		}
	}
	defer file.Close()

	sw.WriteStatus("info", "deployment log stream starting")

	// Tail the file.
	return tailDeploymentLogFile(ctx, file, deploymentID, w)
}

// tailDeploymentLogFile reads from the file and sends new lines to the writer.
func tailDeploymentLogFile(ctx context.Context, file *os.File, deploymentID string, w io.Writer) error {
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

		// Check if the deployment has finished provisioning.
		latest, err := models.GetDeploymentByID(context.Background(), deploymentID)
		if err == nil && latest.Status == "ready" {
			// Deployment is ready, stop streaming the file.
			return nil
		}

		// Wait before polling again.
		select {
		case <-ctx.Done():
			// The client disconnected.
			line, readErr := io.ReadAll(reader)
			if readErr == nil && len(line) > 0 {
				line = bytes.TrimRight(line, "\r\n")
				_, _ = w.Write(line) // Best effort
			}
			return ctx.Err()
		case <-pollTicker.C:
			// Continue to the next iteration.
		}
	}
}

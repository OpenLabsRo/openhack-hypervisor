package systemd

import (
	"bytes"
	"errors"
	"fmt"
	"hypervisor/internal/fs"
	"hypervisor/internal/paths"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// BackendServiceConfig carries values rendered into the backend systemd unit template.
type BackendServiceConfig struct {
	DeploymentID string
	BinaryPath   string
	EnvTag       string
	Port         int
	EnvRoot      string
	Version      string
}

// InstallBackendService writes the backend unit file and reloads systemd.
func InstallBackendService(cfg BackendServiceConfig, logWriter io.Writer) error {
	logger := &systemdLogger{writer: logWriter}

	logger.Log("Service will run on port %d", cfg.Port)
	logger.Log("Writing systemd unit file...")
	if err := writeBackendUnit(cfg, logWriter); err != nil {
		logger.Log("Failed to write unit file: %v", err)
		return err
	}
	logger.Log("Unit file written successfully")

	logger.Log("Running systemctl daemon-reload...")
	if err := runSystemctlWithLog(logWriter, "daemon-reload"); err != nil {
		logger.Log("systemctl daemon-reload failed: %v", err)
		return fmt.Errorf("systemctl daemon-reload failed: %w", err)
	}
	logger.Log("systemctl daemon-reload completed")

	logger.Log("Running systemctl enable %s...", serviceName(cfg.DeploymentID))
	if err := runSystemctlWithLog(logWriter, "enable", serviceName(cfg.DeploymentID)); err != nil {
		logger.Log("systemctl enable failed: %v", err)
		return fmt.Errorf("systemctl enable failed: %w", err)
	}
	logger.Log("systemctl enable completed")

	logger.Log("Running systemctl restart %s...", serviceName(cfg.DeploymentID))
	if err := runSystemctlWithLog(logWriter, "restart", serviceName(cfg.DeploymentID)); err != nil {
		logger.Log("systemctl restart failed: %v", err)
		return fmt.Errorf("systemctl restart failed: %w", err)
	}
	logger.Log("systemctl restart completed")

	return nil
}

func writeBackendUnit(cfg BackendServiceConfig, logWriter io.Writer) error {
	logger := &systemdLogger{writer: logWriter}

	logger.Log("Reading embedded unit template...")
	data, err := BackendService.ReadFile(BackendServiceName)
	if err != nil {
		return fmt.Errorf("failed to read embedded unit: %w", err)
	}
	logger.Log("Embedded unit template read successfully")

	logger.Log("Parsing unit template...")
	unitTemplate, err := template.New(BackendServiceName).Parse(string(data))
	if err != nil {
		return fmt.Errorf("failed to parse unit template: %w", err)
	}
	logger.Log("Unit template parsed successfully")

	logger.Log("Rendering unit template with config...")
	var rendered bytes.Buffer
	if err := unitTemplate.Execute(&rendered, cfg); err != nil {
		return fmt.Errorf("failed to render unit template: %w", err)
	}
	logger.Log("Unit template rendered successfully")

	target := filepath.Join(paths.SystemdUnitDir, serviceName(cfg.DeploymentID))
	logger.Log("Removing existing unit file if present...")
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		if !errors.Is(err, os.ErrPermission) && !os.IsPermission(err) {
			return fmt.Errorf("failed to remove existing unit file: %w", err)
		}
	}
	logger.Log("Existing unit file removed (or not present)")

	logger.Log("Writing new unit file to %s...", target)
	if err := fs.WriteFileWithSudo(target, rendered.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to install unit file: %w", err)
	}
	logger.Log("Unit file written successfully")

	return nil
}

// ServiceName returns the systemd service name for a deployment.
func ServiceName(deploymentID string) string {
	return fmt.Sprintf("openhack-backend-%s.service", deploymentID)
}

func serviceName(deploymentID string) string {
	return ServiceName(deploymentID)
}

// StopBackendService stops the backend service if it is installed.
func StopBackendService(deploymentID string) error {
	cmd := exec.Command("sudo", "systemctl", "stop", serviceName(deploymentID))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() == 5 {
				return nil // Service not loaded, ignore
			}
		}
		return fmt.Errorf("systemctl stop failed: %w", err)
	}
	return nil
}

// DisableBackendService disables the backend service.
func DisableBackendService(deploymentID string) error {
	cmd := exec.Command("sudo", "systemctl", "disable", serviceName(deploymentID))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RemoveBackendServiceFile removes the backend systemd service file.
func RemoveBackendServiceFile(deploymentID string) error {
	cmd := exec.Command("sudo", "rm", "-f", filepath.Join(paths.SystemdUnitDir, serviceName(deploymentID)))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// CheckBackendServiceStatus verifies that the backend service is active.
func CheckBackendServiceStatus(deploymentID string) error {
	cmd := exec.Command("systemctl", "is-active", serviceName(deploymentID))
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	status := strings.TrimSpace(string(output))
	if status != "active" {
		return fmt.Errorf("service not active: %s", status)
	}
	return nil
}

// systemdLogger writes formatted log messages to the writer.
type systemdLogger struct {
	writer io.Writer
}

func (l *systemdLogger) Log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.writer, "[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
}

func runSystemctlWithLog(logWriter io.Writer, args ...string) error {
	allArgs := append([]string{"systemctl"}, args...)
	return runCommandWithLog("sudo", allArgs, logWriter)
}

func runCommandWithLog(cmd string, args []string, logWriter io.Writer) error {
	command := exec.Command(cmd, args...)
	output, err := command.CombinedOutput()
	if len(output) > 0 {
		logWriter.Write(output)
	}
	return err
}

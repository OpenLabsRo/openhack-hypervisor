package systemd

import (
	"bytes"
	"errors"
	"fmt"
	"hypervisor/internal/paths"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// ServiceConfig carries values rendered into the systemd unit template.
type ServiceConfig struct {
	BinaryPath string
	Deployment string
	Port       string
	EnvRoot    string
	Version    string
}

// InstallHypervisorService writes the hypervisor unit file and reloads systemd.
func InstallHypervisorService(cfg ServiceConfig, serviceSuffix ...string) error {
	suffix := ""
	if len(serviceSuffix) > 0 {
		suffix = "-" + serviceSuffix[0]
	}

	serviceName := HypervisorServiceName
	if suffix != "" {
		serviceName = strings.TrimSuffix(HypervisorServiceName, ".service") + suffix + ".service"
	}

	if err := writeHypervisorUnit(cfg, serviceName); err != nil {
		return err
	}

	if err := runSystemctl("daemon-reload"); err != nil {
		return fmt.Errorf("systemctl daemon-reload failed: %w", err)
	}

	if err := runSystemctl("enable", serviceName); err != nil {
		return fmt.Errorf("systemctl enable failed: %w", err)
	}

	if err := runSystemctl("restart", serviceName); err != nil {
		return fmt.Errorf("systemctl restart failed: %w", err)
	}

	return nil
}

func writeHypervisorUnit(cfg ServiceConfig, serviceName string) error {
	data, err := HypervisorService.ReadFile(HypervisorServiceName)
	if err != nil {
		return fmt.Errorf("failed to read embedded unit: %w", err)
	}

	unitTemplate, err := template.New(HypervisorServiceName).Parse(string(data))
	if err != nil {
		return fmt.Errorf("failed to parse unit template: %w", err)
	}

	cfg = applyDefaults(cfg)

	var rendered bytes.Buffer
	if err := unitTemplate.Execute(&rendered, cfg); err != nil {
		return fmt.Errorf("failed to render unit template: %w", err)
	}

	target := filepath.Join(paths.SystemdUnitDir, serviceName)
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		if !errors.Is(err, os.ErrPermission) && !os.IsPermission(err) {
			return fmt.Errorf("failed to remove existing unit file: %w", err)
		}
	}

	if err := writeFileWithSudo(target, rendered.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to install unit file: %w", err)
	}

	return nil
}

func applyDefaults(cfg ServiceConfig) ServiceConfig {
	if cfg.BinaryPath == "" {
		cfg.BinaryPath = paths.HypervisorBuildsDir + "/current"
	}
	if cfg.Deployment == "" {
		cfg.Deployment = "prod"
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.EnvRoot == "" {
		cfg.EnvRoot = paths.HypervisorEnvDir
	}
	return cfg
}

func writeFileWithSudo(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create unit directory: %w", err)
	}

	if err := os.WriteFile(path, data, mode); err != nil {
		if !errors.Is(err, os.ErrPermission) && !os.IsPermission(err) {
			return fmt.Errorf("failed to write unit file: %w", err)
		}

		cmd := exec.Command("sudo", "tee", path)
		cmd.Stdin = bytes.NewReader(data)
		cmd.Stdout = io.Discard
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("sudo tee %s failed: %w", path, err)
		}

		if err := runCommand("sudo", "chmod", fmt.Sprintf("%04o", mode), path); err != nil {
			return fmt.Errorf("sudo chmod %s failed: %w", path, err)
		}

		return nil
	}

	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("failed to chmod unit file: %w", err)
	}

	return nil
}

func runSystemctl(args ...string) error {
	return runCommand("sudo", append([]string{"systemctl"}, args...)...)
}

func runCommand(cmd string, args ...string) error {
	command := exec.Command(cmd, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

// StopHypervisorService stops the hypervisor service if it is installed.
func StopHypervisorService() error {
	cmd := exec.Command("sudo", "systemctl", "stop", HypervisorServiceName)
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

// StopService stops a systemd service by name if it is installed.
func StopService(serviceName string) error {
	cmd := exec.Command("sudo", "systemctl", "stop", serviceName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.ExitCode() == 5 {
				return nil // Service not loaded, ignore
			}
		}
		return fmt.Errorf("systemctl stop %s failed: %w", serviceName, err)
	}
	return nil
}

// DisableHypervisorService disables the hypervisor service.
func DisableHypervisorService() error {
	cmd := exec.Command("sudo", "systemctl", "disable", HypervisorServiceName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RemoveServiceFile removes the systemd service file.
func RemoveServiceFile() error {
	cmd := exec.Command("sudo", "rm", "-f", "/lib/systemd/system/"+HypervisorServiceName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ReloadSystemd reloads the systemd daemon.
func ReloadSystemd() error {
	cmd := exec.Command("sudo", "systemctl", "daemon-reload")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// CheckServiceStatus verifies that the hypervisor service is active.
func CheckServiceStatus(serviceName ...string) error {
	name := HypervisorServiceName
	if len(serviceName) > 0 {
		if serviceName[0] != "" {
			name = strings.TrimSuffix(HypervisorServiceName, ".service") + "-" + serviceName[0] + ".service"
		}
	}

	cmd := exec.Command("systemctl", "is-active", name)
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

package fs

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	userutil "hypervisor/internal/hyperctl/user"
	"hypervisor/internal/paths"
)

var hypervisorDirs = []string{
	paths.HypervisorBaseDir,
	paths.HypervisorReposDir,
	paths.HypervisorBuildsDir,
	paths.HypervisorEnvDir,
	paths.HypervisorLogsDir,
}

var openhackDirs = []string{
	paths.OpenHackReposDir,
	paths.OpenHackBuildsDir,
	paths.OpenHackEnvDir,
	paths.OpenHackEnvTemplateDir,
	paths.OpenHackRuntimeDir,
	paths.OpenHackRuntimeLogsDir,
}

// EnsureLayout creates the required directory structure for the hypervisor and backend builds.
func EnsureLayout() error {
	if err := ensureDirs(hypervisorDirs); err != nil {
		return err
	}

	if err := ensureDirs(openhackDirs); err != nil {
		return err
	}

	return nil
}

func ensureDirs(dirs []string) error {
	for _, dir := range dirs {
		info, statErr := os.Stat(dir)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				if err := createDir(dir); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("failed to stat %s: %w", dir, statErr)
			}
		} else if !info.IsDir() {
			return fmt.Errorf("path exists and is not a directory: %s", dir)
		}

		// Set ownership to openhack user
		if err := userutil.ChownToOpenhack(dir); err != nil {
			return fmt.Errorf("failed to chown %s to openhack: %w", dir, err)
		}

		// Set permissions to 0755 (rwxr-xr-x)
		if err := userutil.ChmodPath(dir, "0755"); err != nil {
			return fmt.Errorf("failed to chmod 0755 %s: %w", dir, err)
		}
	}

	return nil
}

func createDir(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		if !os.IsPermission(err) {
			return fmt.Errorf("failed to create %s: %w", path, err)
		}

		// Permission denied, try with sudo
		if err := runSudo("mkdir", "-p", path); err != nil {
			return fmt.Errorf("sudo mkdir -p %s: %w", path, err)
		}
	}

	return nil
}

// RemoveDir removes a directory and all its contents.
func RemoveDir(path string) error {
	if err := os.RemoveAll(path); err != nil {
		if !os.IsPermission(err) {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}

		// Permission denied, try with sudo
		if err := runSudo("rm", "-rf", path); err != nil {
			return fmt.Errorf("sudo rm -rf %s: %w", path, err)
		}
	}

	return nil
}

func runSudo(args ...string) error {
	cmd := exec.Command("sudo", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// WriteFileWithSudo writes a file using sudo if necessary.
func WriteFileWithSudo(path string, data []byte, perm os.FileMode) error {
	// Try writing directly first
	if err := os.WriteFile(path, data, perm); err != nil {
		if !os.IsPermission(err) {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}

		// Permission denied, use sudo tee
		cmd := exec.Command("sudo", "tee", path)
		cmd.Stdin = bytes.NewReader(data)
		cmd.Stdout = io.Discard
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("sudo tee %s failed: %w", path, err)
		}
	}

	// Set correct permissions
	if err := userutil.ChmodPath(path, fmt.Sprintf("%04o", perm)); err != nil {
		return fmt.Errorf("failed to chmod %s: %w", path, err)
	}

	return nil
}

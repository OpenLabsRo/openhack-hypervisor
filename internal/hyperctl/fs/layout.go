package fs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

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
	// Use root ownership for consistency
	// uid := 0
	// gid := 0
	owner := "0:0"

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

		if err := runSudo("chown", "-R", owner, dir); err != nil {
			return fmt.Errorf("sudo chown -R %s %s: %w", owner, dir, err)
		}

		// Ensure 0777 permissions for all users
		if err := runSudo("chmod", "777", dir); err != nil {
			return fmt.Errorf("failed to chmod 777 %s: %w", dir, err)
		}
	}

	return nil
}

func createDir(path string) error {
	if err := os.MkdirAll(path, 0o777); err != nil {
		if !errors.Is(err, os.ErrPermission) && !os.IsPermission(err) {
			return fmt.Errorf("failed to create %s: %w", path, err)
		}

		if err := runSudo("mkdir", "-p", path); err != nil {
			return fmt.Errorf("sudo mkdir -p %s: %w", path, err)
		}
	}

	return nil
}

func RemoveDir(path string) error {
	if err := os.RemoveAll(path); err != nil {
		if !errors.Is(err, os.ErrPermission) && !os.IsPermission(err) {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}

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
	// Always use sudo for systemd files to avoid permission issues
	cmd := exec.Command("sudo", "tee", path)
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo tee %s failed: %w", path, err)
	}

	if err := runSudo("chmod", fmt.Sprintf("%04o", perm), path); err != nil {
		return fmt.Errorf("sudo chmod %s failed: %w", path, err)
	}

	return nil
}

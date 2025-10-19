package fs

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"

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
	current, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to lookup current user: %w", err)
	}

	uid, err := strconv.Atoi(current.Uid)
	if err != nil {
		return fmt.Errorf("invalid uid %s: %w", current.Uid, err)
	}

	gid, err := strconv.Atoi(current.Gid)
	if err != nil {
		return fmt.Errorf("invalid gid %s: %w", current.Gid, err)
	}

	owner := fmt.Sprintf("%s:%s", current.Uid, current.Gid)

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

		if err := os.Chown(dir, uid, gid); err != nil {
			if os.IsPermission(err) {
				if err := runSudo("chown", "-R", owner, dir); err != nil {
					return fmt.Errorf("sudo chown -R %s %s: %w", owner, dir, err)
				}
				continue
			}
			return fmt.Errorf("failed to chown %s: %w", dir, err)
		}
	}

	return nil
}

func createDir(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
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

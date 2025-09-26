package fs

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

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
		if err := os.MkdirAll(dir, 0o755); err != nil {
			if !errors.Is(err, os.ErrPermission) && !os.IsPermission(err) {
				return fmt.Errorf("failed to create %s: %w", dir, err)
			}

			if err := runSudo("mkdir", "-p", dir); err != nil {
				return fmt.Errorf("sudo mkdir -p %s: %w", dir, err)
			}
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

// EnsureEnvFile creates the hypervisor env file with a template if missing.
func EnsureEnvFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat %s: %w", path, err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		if !errors.Is(err, os.ErrPermission) && !os.IsPermission(err) {
			return fmt.Errorf("failed to create env directory: %w", err)
		}
		if err := runSudo("mkdir", "-p", dir); err != nil {
			return fmt.Errorf("sudo mkdir -p %s: %w", dir, err)
		}
	}

	template := `# Hypervisor environment configuration
# Populate required secrets before starting the service.
# Example:
# MONGO_URI=mongodb://...
# JWT_SECRET=...
# GITHUB_WEBHOOK_SECRET=...
`

	if err := os.WriteFile(path, []byte(template), 0o640); err != nil {
		if !os.IsPermission(err) {
			return fmt.Errorf("failed to write env template: %w", err)
		}

		cmd := exec.Command("sudo", "tee", path)
		cmd.Stdin = strings.NewReader(template)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("sudo tee %s failed: %w", path, err)
		}

		if err := runSudo("chmod", "640", path); err != nil {
			return fmt.Errorf("sudo chmod 640 %s failed: %w", path, err)
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

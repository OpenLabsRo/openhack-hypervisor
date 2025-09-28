package fs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"hypervisor/internal/paths"
)

// HypervisorEnvPath returns the absolute path to the hypervisor .env file.
func HypervisorEnvPath() string {
	return filepath.Join(paths.HypervisorEnvDir, ".env")
}

// OpenHackEnvPath returns the absolute path to the OpenHack backend .env file.
func OpenHackEnvPath() string {
	return filepath.Join(paths.OpenHackEnvDir, ".env")
}

// EnsureEnvDirFor creates the parent directory for the provided env file path if missing.
func EnsureEnvDirFor(envPath string) error {
	dir := filepath.Dir(envPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		if os.IsPermission(err) {
			if err := runSudo("mkdir", "-p", dir); err != nil {
				return fmt.Errorf("sudo mkdir -p %s: %w", dir, err)
			}
			return nil
		}
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	return nil
}

// EnvFileExists checks whether the env file is already present (and not a directory).
func EnvFileExists(envPath string) (bool, error) {
	info, err := os.Stat(envPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat %s: %w", envPath, err)
	}
	if info.IsDir() {
		return false, fmt.Errorf("expected %s to be a file, found directory", envPath)
	}
	return true, nil
}

// OpenInEditor launches the requested editor for the env file, allowing the editor to create it.
func OpenInEditor(editor, envPath string) error {
	fields := strings.Fields(editor)
	if len(fields) == 0 {
		return fmt.Errorf("invalid editor command")
	}

	if err := EnsureEnvDirFor(envPath); err != nil {
		return err
	}

	cmd := exec.Command(fields[0], append(fields[1:], envPath)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// EditEnvFileIfMissing launches the editor only when the env file does not yet exist.
func EditEnvFileIfMissing(label, envPath, editor string) error {
	exists, err := EnvFileExists(envPath)
	if err != nil {
		return err
	}

	if exists {
		fmt.Printf("%s environment file already present at %s; skipping editor.\n", label, envPath)
		return nil
	}

	fmt.Printf("Opening %s with %s...\n", envPath, editor)
	if err := OpenInEditor(editor, envPath); err != nil {
		return fmt.Errorf("editor exit error: %w", err)
	}
	fmt.Printf("%s environment file saved.\n", label)

	exists, err = EnvFileExists(envPath)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%s environment file was not created at %s", label, envPath)
	}

	return nil
}

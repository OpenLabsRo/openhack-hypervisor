package fs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// EnsureDir creates a directory path if it does not already exist.
func EnsureDir(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// WriteFile persists contents at the given path with the provided permissions.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// Remove removes a single file or empty directory, ignoring not-exist errors.
func Remove(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// RemoveAll recursively removes the provided path, ignoring not-exist errors.
func RemoveAll(path string) error {
	err := os.RemoveAll(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
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

	if err := runCommand("sudo", "chmod", fmt.Sprintf("%04o", perm), path); err != nil {
		return fmt.Errorf("sudo chmod %s failed: %w", path, err)
	}

	return nil
}

func runCommand(cmd string, args ...string) error {
	command := exec.Command(cmd, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

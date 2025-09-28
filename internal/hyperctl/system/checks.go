package system

import (
	"errors"
	"fmt"
	"os/exec"
)

// ErrMissingDependency indicates that a required binary could not be found.
var ErrMissingDependency = errors.New("missing dependency")

// CheckBinary ensures the given command is available on PATH.
func CheckBinary(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%w: %s", ErrMissingDependency, name)
	}
	return nil
}

// EnsureSystemctlAccessible verifies that systemctl is available and usable.
func EnsureSystemctlAccessible() error {
	if err := CheckBinary("systemctl"); err != nil {
		return err
	}

	cmd := exec.Command("systemctl", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("systemctl not accessible: %w", err)
	}

	return nil
}

// ResolveEditor determines which editor should be used for interactive edits.
func ResolveEditor() string {
	return "vim"
}

// EnsureRedisAvailable validates that redis-server binary is installed.
func EnsureRedisAvailable() error {
	if err := CheckBinary("redis-server"); err != nil {
		return err
	}
	return nil
}

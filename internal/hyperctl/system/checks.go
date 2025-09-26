package system

import (
	"errors"
	"fmt"
	"os"
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
	editor := lookupEnv("VISUAL")
	if editor == "" {
		editor = lookupEnv("EDITOR")
	}
	if editor == "" {
		editor = "vim"
	}
	return editor
}

func lookupEnv(key string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return ""
	}
	return val
}

// EnsureRedisAvailable validates that redis-server binary is installed.
func EnsureRedisAvailable() error {
	if err := CheckBinary("redis-server"); err != nil {
		return err
	}
	return nil
}

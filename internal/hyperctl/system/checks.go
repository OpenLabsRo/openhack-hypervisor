package system

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
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

// CheckPrerequisites verifies that all required system components are available.
func CheckPrerequisites() error {
	fmt.Println("Checking systemctl availability...")
	if err := EnsureSystemctlAccessible(); err != nil {
		return fmt.Errorf("systemctl unavailable: %w", err)
	}
	fmt.Println("systemctl is available")

	fmt.Println("Checking redis-server availability...")
	if err := EnsureRedisAvailable(); err != nil {
		return fmt.Errorf("redis missing: %w", err)
	}
	fmt.Println("redis-server is available")

	editor := ResolveEditor()
	editorBinary := editor
	if fields := strings.Fields(editor); len(fields) > 0 {
		editorBinary = fields[0]
	}

	fmt.Printf("Checking editor %q...\n", editorBinary)
	if err := CheckBinary(editorBinary); err != nil {
		return fmt.Errorf("editor %q not found: %w", editorBinary, err)
	}
	fmt.Printf("Editor %q is available\n", editorBinary)

	return nil
}

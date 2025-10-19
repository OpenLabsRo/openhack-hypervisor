package fs

import (
	"errors"
	"os"
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

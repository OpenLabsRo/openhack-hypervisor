package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"hypervisor/internal/paths"
)

// State represents the persisted hyperctl installation metadata.
type State struct {
	Version   string    `json:"version"`
	BuildPath string    `json:"build_path,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

var (
	// ErrStateNotInitialized is returned when the hypervisor has not been installed yet.
	ErrStateNotInitialized = errors.New("hypervisor state not initialized")
)

func stateFilePath() string {
	return filepath.Join(paths.HypervisorBaseDir, "state.json")
}

// Load retrieves the persisted hyperctl state from disk.
func Load() (State, error) {
	path := stateFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{}, ErrStateNotInitialized
		}
		return State{}, fmt.Errorf("failed to read state file: %w", err)
	}

	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return State{}, fmt.Errorf("failed to parse state file: %w", err)
	}

	return st, nil
}

// Save writes the provided state to disk, updating the timestamp automatically.
func Save(st State) error {
	st.UpdatedAt = time.Now().UTC()

	encoded, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode state: %w", err)
	}

	path := stateFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, encoded, 0o640); err != nil {
		return fmt.Errorf("failed to write temp state file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("failed to persist state file: %w", err)
	}

	return nil
}

// CurrentVersion returns the version recorded in the state file.
func CurrentVersion() (string, error) {
	st, err := Load()
	if err != nil {
		return "", err
	}

	if st.Version == "" {
		return "", fmt.Errorf("state file missing version")
	}

	return st.Version, nil
}

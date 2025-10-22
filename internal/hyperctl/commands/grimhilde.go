package commands

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

// RunGrimhilde handles the `hyperctl grimhilde` subcommand.
// It fetches and runs the hyperctl installation script to update itself.
func RunGrimhilde(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("grimhilde: unexpected arguments")
	}

	// URL of the installation script
	scriptURL := "https://dl.openhack.ro/hyperctl_install.sh" // Replace with actual URL

	fmt.Printf("Fetching update script from %s...\n", scriptURL)

	// Fetch the script
	resp, err := http.Get(scriptURL)
	if err != nil {
		return fmt.Errorf("failed to fetch script: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch script: HTTP %d", resp.StatusCode)
	}

	// Read the script content
	scriptContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read script content: %w", err)
	}

	// Create a temporary file for the script
	tmpDir := os.TempDir()
	scriptPath := filepath.Join(tmpDir, "hyperctl_update.sh")

	if err := os.WriteFile(scriptPath, scriptContent, 0755); err != nil {
		return fmt.Errorf("failed to write script to temp file: %w", err)
	}
	defer os.Remove(scriptPath) // Clean up

	fmt.Printf("Running update script...\n")

	// Execute the script
	cmd := exec.Command("bash", scriptPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("update script failed: %w", err)
	}

	fmt.Println("Hyperctl update completed successfully!")
	return nil
}

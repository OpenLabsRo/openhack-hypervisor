package nginx

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed nginx.conf
var nginxConf embed.FS

const (
	// DefaultNginxConfigPath is the path where the nginx config will be written
	// We use conf.d/app.conf which is the common location for drop-in site configs.
	DefaultNginxConfigPath = "/etc/nginx/conf.d/app.conf"
)

// InstallConfig installs the nginx configuration and enables it
func InstallConfig(configPath string) error {
	// Read embedded nginx.conf and write to sites-available using sudo if necessary
	data, err := nginxConf.ReadFile("nginx.conf")
	if err != nil {
		return fmt.Errorf("failed to read embedded nginx.conf: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(DefaultNginxConfigPath), 0o755); err != nil {
		return fmt.Errorf("failed to create nginx config directory: %w", err)
	}

	if err := os.WriteFile(DefaultNginxConfigPath, data, 0o644); err != nil {
		// fallback to sudo tee if permission denied
		if !os.IsPermission(err) {
			return fmt.Errorf("failed to write nginx config: %w", err)
		}
		cmd := exec.Command("sudo", "tee", DefaultNginxConfigPath)
		cmd.Stdin = bytes.NewReader(data)
		cmd.Stdout = io.Discard
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("sudo tee failed: %w", err)
		}
	}

	// For conf.d we don't need a symlink; nginx will pick up files in conf.d

	return nil
}

// EmbeddedConfig returns the raw embedded nginx.conf contents.
func EmbeddedConfig() (string, error) {
	data, err := nginxConf.ReadFile("nginx.conf")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

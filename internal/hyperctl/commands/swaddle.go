package commands

import (
	"fmt"
	"os/exec"

	"hypervisor/internal/hyperctl/health"
	"hypervisor/internal/hyperctl/nginx"
)

// RunSwaddle handles the `hyperctl swaddle` subcommand.
// It installs the nginx configuration for blue-green deployments.
// The config supports drain mode: when blue returns 503, nginx routes to green.
func RunSwaddle(args []string) error {
	fmt.Println("Installing nginx configuration for hypervisor...")

	// Check if hypervisor service is running
	if err := health.CheckHost("localhost:8080"); err != nil {
		return fmt.Errorf("hypervisor blue service is not running - cannot install config without active service")
	}

	if err := health.CheckHost("localhost:8081"); err != nil {
		return fmt.Errorf("hypervisor green service is not running - cannot install config without active service")
	}

	// Install embedded nginx config
	if err := nginx.InstallConfig(""); err != nil {
		return fmt.Errorf("failed to install nginx config: %w", err)
	}
	fmt.Printf("Nginx configuration installed to %s\n", nginx.DefaultNginxConfigPath)

	// Remove default nginx configurations
	defaultConfigs := []string{
		"/etc/nginx/conf.d/default.conf",
		"/etc/nginx/sites-enabled/default",
	}

	for _, config := range defaultConfigs {
		cmd := exec.Command("sudo", "rm", "-f", config)
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: failed to remove %s: %v\n", config, err)
		} else {
			fmt.Printf("Removed default config: %s\n", config)
		}
	}

	fmt.Println("Nginx configuration installed successfully")
	fmt.Println("Note: Run 'sudo nginx -t' to test configuration and 'sudo systemctl reload nginx' to apply changes")

	return nil
}

// previous copyFile removed; installation uses embedded nginx content

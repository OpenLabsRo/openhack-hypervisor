package commands

import (
	"flag"
	"fmt"
	"io"
	"os"

	"hypervisor/internal/hyperctl/health"
)

// RunSwaddle handles the `hyperctl swaddle` subcommand.
// It installs the nginx configuration for blue-green deployments.
// The config supports drain mode: when blue returns 503, nginx routes to green.
func RunSwaddle(args []string) error {
	// Parse command-line flags
	fs := flag.NewFlagSet("swaddle", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	install := fs.Bool("install", false, "Install nginx configuration and remove defaults")

	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Println("Installing nginx configuration for hypervisor...")

	// Check if hypervisor service is running
	if err := health.CheckHost("localhost:8080"); err != nil {
		return fmt.Errorf("hypervisor blue service is not running - cannot install config without active service")
	}

	if err := health.CheckHost("localhost:8081"); err != nil {
		return fmt.Errorf("hypervisor green service is not running - cannot install config without active service")
	}

	if *install {
		// Copy our nginx config to /etc/nginx/conf.d/app.conf
		srcPath := "internal/hyperctl/nginx/nginx.conf"
		dstPath := "/etc/nginx/conf.d/app.conf"

		if err := copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to install nginx config: %w", err)
		}
		fmt.Printf("Nginx configuration installed to %s\n", dstPath)

		// Remove default nginx configurations
		defaultConfigs := []string{
			"/etc/nginx/conf.d/default.conf",
			"/etc/nginx/sites-enabled/default",
		}

		for _, config := range defaultConfigs {
			if err := os.Remove(config); err != nil && !os.IsNotExist(err) {
				fmt.Printf("Warning: failed to remove %s: %v\n", config, err)
			} else if err == nil {
				fmt.Printf("Removed default config: %s\n", config)
			}
		}

		fmt.Println("Nginx configuration installed successfully")
		fmt.Println("Note: Run 'sudo nginx -t' to test configuration and 'sudo systemctl reload nginx' to apply changes")
	} else {
		// Just show the config
		content, err := os.ReadFile("internal/hyperctl/nginx/nginx.conf")
		if err != nil {
			return fmt.Errorf("failed to read nginx config: %w", err)
		}
		fmt.Print(string(content))
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	// Copy permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

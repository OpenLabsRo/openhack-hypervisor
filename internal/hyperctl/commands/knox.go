package commands

import (
	"fmt"
	"os"
	"os/exec"
)

// RunKnox handles the `hyperctl knox` subcommand.
// It secures the nginx server with SSL using certbot for api.openhack.ro.
func RunKnox(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("knox: unexpected arguments")
	}

	fmt.Println("Securing nginx with certbot for api.openhack.ro...")

	// Run certbot with nginx plugin
	cmd := exec.Command("certbot", "--nginx", "-d", "api.openhack.ro")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("certbot failed: %w", err)
	}

	fmt.Println("SSL certificate installed successfully!")
	fmt.Println("Reloading nginx...")

	// Reload nginx to apply changes
	reloadCmd := exec.Command("nginx", "-s", "reload")
	if err := reloadCmd.Run(); err != nil {
		return fmt.Errorf("failed to reload nginx: %w", err)
	}

	fmt.Println("Knox complete - your server is now secured!")
	return nil
}

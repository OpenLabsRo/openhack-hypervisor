package commands

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"hypervisor/internal/hyperctl/systemd"
)

// RunNagasaki handles the `hyperctl nagasaki` subcommand.
// This command gracefully stops the running hypervisor services (blue and green).
func RunNagasaki(args []string) error {
	fs := flag.NewFlagSet("nagasaki", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Println("Stopping hypervisor services (blue and green)...")
	blueService := "openhack-hypervisor-blue.service"
	greenService := "openhack-hypervisor-green.service"

	if err := systemd.StopService(blueService); err != nil {
		return fmt.Errorf("failed to stop blue service: %w", err)
	}

	if err := systemd.StopService(greenService); err != nil {
		return fmt.Errorf("failed to stop green service: %w", err)
	}

	// Stop all backend services
	fmt.Println("Stopping all backend services...")
	stopCmd := exec.Command("bash", "-c", "systemctl list-units --no-legend --state=active,failed | grep openhack-backend | awk '{print $1}' | xargs -r systemctl stop")
	if output, err := stopCmd.CombinedOutput(); err != nil {
		fmt.Printf("Warning: Failed to stop some backend services: %v\n%s\n", err, string(output))
	} else {
		fmt.Println("Backend services stopped.")
	}

	fmt.Println("All services stopped.")
	return nil
}

package commands

import (
	"flag"
	"fmt"
	"os"

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

	fmt.Println("Hypervisor services stopped.")
	return nil
}

package commands

import (
	"flag"
	"fmt"
	"os"

	"hypervisor/internal/hyperctl/systemd"
)

// RunNagasaki handles the `hyperctl nagasaki` subcommand.
// This command gracefully stops the running hypervisor service.
func RunNagasaki(args []string) error {
	fs := flag.NewFlagSet("nagasaki", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Println("Stopping hypervisor service...")
	if err := systemd.StopHypervisorService(); err != nil {
		return fmt.Errorf("failed to stop hypervisor service: %w", err)
	}

	fmt.Println("Hypervisor service stopped.")
	return nil
}

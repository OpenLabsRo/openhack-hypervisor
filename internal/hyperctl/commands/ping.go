package commands

import (
	"fmt"
	"hypervisor/internal/hyperctl/health"
)

// RunPing handles the `hyperctl ping` subcommand.
// It pings the hypervisor health endpoint at localhost:8080/hypervisor/meta/ping.
func RunPing(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("ping: unexpected arguments")
	}

	// Use a single health check without retries.
	if err := health.CheckOnce(); err != nil {
		return fmt.Errorf("hypervisor is not responding: %w", err)
	}

	fmt.Println("PONG")
	return nil
}

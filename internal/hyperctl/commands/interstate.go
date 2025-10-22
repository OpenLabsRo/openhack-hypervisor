package commands

import (
	"fmt"
	"hypervisor/internal/hyperctl/health"
	"io"
	"net/http"
	"os"
)

// RunInterstate handles the `hyperctl interstate` subcommand.
// It shows the current routing map by calling the hypervisor API.
func RunInterstate(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("interstate: unexpected arguments")
	}

	// Check if hypervisor service is running
	if err := health.CheckHost("localhost:8080"); err != nil {
		return fmt.Errorf("hypervisor blue service is not running - please set it up first using 'hyperctl manhattan'")
	}

	if err := health.CheckHost("localhost:8081"); err != nil {
		return fmt.Errorf("hypervisor green service is not running - please set it up first using 'hyperctl manhattan'")
	}

	// Get hypervisor host
	host := os.Getenv("HYPERVISOR_HOST")
	if host == "" {
		host = "localhost:8080"
	}

	// Make API call to get routing map
	url := fmt.Sprintf("http://%s/hypervisor/routing", host)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to hypervisor API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hypervisor API returned status %d", resp.StatusCode)
	}

	// Read and print the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Print(string(body) + "\n")
	return nil
}

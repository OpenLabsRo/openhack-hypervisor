package commands

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// RunVersion handles the `hyperctl version` subcommand.
// It queries the hypervisor API endpoint to get the running version.
func RunVersion(args []string) error {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	host := fs.String("host", "localhost:8080", "Hypervisor host:port to query")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Query the hypervisor API for version information
	url := fmt.Sprintf("http://%s/hypervisor/meta/version", *host)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("No version detected")
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("No version detected")
		return nil
	}

	// Read the plain text response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("No version detected")
		return nil
	}

	version := strings.TrimSpace(string(body))
	if version == "" {
		fmt.Println("No version detected")
		return nil
	}

	fmt.Println(version)
	return nil
}

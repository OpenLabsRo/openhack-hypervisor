package health

import (
	"fmt"
	"net/http"
	"os"
)

// Check performs a health check on the hypervisor service.
func Check() error {
	host := os.Getenv("HYPERVISOR_HOST")
	if host == "" {
		host = "localhost:8080"
	}

	url := fmt.Sprintf("http://%s/hypervisor/meta/ping", host)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}
	return nil
}

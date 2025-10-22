package health

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

// CheckHost performs a health check on the hypervisor service at the specified host with retries.
func CheckHost(host string) error {
	url := fmt.Sprintf("http://%s/hypervisor/meta/ping", host)

	// Retry for up to 30 seconds, checking every 1 second
	for i := 0; i < 30; i++ {
		resp, err := http.Get(url)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("health check failed: service at %s did not respond after 30 seconds", host)
}

// CheckOnce performs a single health check on the hypervisor service without retries.
func CheckOnce() error {
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

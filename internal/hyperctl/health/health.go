package health

import (
	"fmt"
	"net/http"
)

// Check performs a health check on the hypervisor service.
func Check() error {
	url := "http://localhost:8080/hypervisor/ping"
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

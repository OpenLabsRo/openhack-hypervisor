
package staging

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Release struct {
	Tag string `json:"tag"`
}

func StageLatest() error {
	fmt.Println("Getting releases...")
	resp, err := http.Get("http://localhost:8080/hypervisor/releases")
	if err != nil {
		return fmt.Errorf("failed to get releases: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read releases response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get releases: status %s: %s", resp.Status, string(body))
	}

	var releases []Release
	if err := json.Unmarshal(body, &releases); err != nil {
		return fmt.Errorf("failed to parse releases response: %w", err)
	}

	if len(releases) == 0 {
		return fmt.Errorf("no releases found to stage")
	}

	latestRelease := releases[len(releases)-1]
	tag := latestRelease.Tag

	fmt.Printf("Staging latest release: %s\n", tag)

	stageURL := fmt.Sprintf("http://localhost:8080/hypervisor/releases/stage?version=%s", tag)
	resp, err = http.Post(stageURL, "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to stage release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to stage release with status %s: %s", resp.Status, string(body))
	}

	fmt.Println("Staging process started successfully.")
	return nil
}

package releases

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"hypervisor/internal/models"

	"github.com/gofiber/fiber/v3"
)

func performRequest(t *testing.T, method, path string, payload any) (*http.Response, []byte) {
	t.Helper()

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("failed to marshal payload: %v", err)
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, path, body)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	return resp, respBody
}

func TestListReleases(t *testing.T) {
	resp, body := performRequest(t, http.MethodGet, "/hypervisor/releases", nil)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var payload struct {
		Releases []models.Release `json:"releases"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(payload.Releases) == 0 {
		t.Log("no releases found, which might be okay if the database is empty")
	}
}

package staging

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"hypervisor/internal/models"

	"github.com/fasthttp/websocket"
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

func TestCreateAndListStage(t *testing.T) {
	// Create a release to associate with the stage
	// In a real test, you would create a release first
	releaseID := "test-release"

	// Create a stage
	createResp, createBody := performRequest(t, http.MethodPost, "/hypervisor/stages", map[string]string{
		"releaseId": releaseID,
		"envTag":    "test",
	})

	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, createResp.StatusCode)
	}

	var createPayload struct {
		Stage models.Stage `json:"stage"`
	}

	if err := json.Unmarshal(createBody, &createPayload); err != nil {
		t.Fatalf("failed to unmarshal create response: %v", err)
	}

	if createPayload.Stage.ID == "" {
		t.Fatalf("expected stage ID to be populated")
	}

	// List stages
	listResp, listBody := performRequest(t, http.MethodGet, "/hypervisor/stages", nil)

	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, listResp.StatusCode)
	}

	var listPayload struct {
		Stages []models.Stage `json:"stages"`
	}

	if err := json.Unmarshal(listBody, &listPayload); err != nil {
		t.Fatalf("failed to unmarshal list response: %v", err)
	}

	if len(listPayload.Stages) == 0 {
		t.Fatalf("expected at least one stage to be listed")
	}

	// Get the stage
	getResp, getBody := performRequest(t, http.MethodGet, "/hypervisor/stages/"+createPayload.Stage.ID, nil)

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, getResp.StatusCode)
	}

	var getPayload struct {
		Stage models.Stage `json:"stage"`
	}

	if err := json.Unmarshal(getBody, &getPayload); err != nil {
		t.Fatalf("failed to unmarshal get response: %v", err)
	}

	if getPayload.Stage.ID != createPayload.Stage.ID {
		t.Fatalf("expected stage ID %q, got %q", createPayload.Stage.ID, getPayload.Stage.ID)
	}

	// Start a test run
	startTestResp, startTestBody := performRequest(t, http.MethodPost, "/hypervisor/stages/"+createPayload.Stage.ID+"/tests", nil)

	if startTestResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, startTestResp.StatusCode)
	}

	var startTestPayload models.Test
	if err := json.Unmarshal(startTestBody, &startTestPayload); err != nil {
		t.Fatalf("failed to unmarshal start test response: %v", err)
	}

	// Test the websocket connection
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial("ws://localhost:8080/.ws/stages/"+createPayload.Stage.ID+"/tests/"+startTestPayload.ID, nil)
	if err != nil {
		t.Fatalf("failed to connect to websocket: %v", err)
	}
	defer conn.Close()

	// Read at least one message
	_, _, err = conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read message from websocket: %v", err)
	}
}

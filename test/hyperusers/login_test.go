package hyperusers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"hypervisor/internal/env"
	"hypervisor/internal/errmsg"

	"github.com/gofiber/fiber/v3"
)

const (
	testHyperUserUsername = "testhyperuser"
	testHyperUserPassword = "testhyperuser"
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

func assertErrorMessage(t *testing.T, body []byte, se errmsg.StatusError) {
	t.Helper()

	var payload map[string]string
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	msg, ok := payload["message"]
	if !ok {
		t.Fatalf("expected error message in response")
	}

	if msg != se.Message {
		t.Fatalf("expected error message %q, got %q", se.Message, msg)
	}
}

func TestHyperUsersPing(t *testing.T) {
	resp, body := performRequest(t, http.MethodGet, "/hypervisor/hyperusers/ping", nil)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if string(body) != "PONG" {
		t.Fatalf("expected body to be PONG, got %s", string(body))
	}
}

func TestHyperUsersLoginSuccess(t *testing.T) {
	resp, body := performRequest(t, http.MethodPost, "/hypervisor/hyperusers/login", map[string]string{
		"username": testHyperUserUsername,
		"password": testHyperUserPassword,
	})

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var payload struct {
		Token     string `json:"token"`
		HyperUser struct {
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"hyperuser"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if payload.Token == "" {
		t.Fatalf("expected token to be populated")
	}

	if payload.HyperUser.Username != testHyperUserUsername {
		t.Fatalf("expected username %q, got %q", testHyperUserUsername, payload.HyperUser.Username)
	}

	if payload.HyperUser.Password != "" {
		t.Fatalf("expected password to be sanitized")
	}

	if len(env.JWT_SECRET) == 0 {
		t.Fatalf("expected env to be initialized")
	}
}

func TestHyperUsersLoginWrongPassword(t *testing.T) {
	resp, body := performRequest(t, http.MethodPost, "/hypervisor/hyperusers/login", map[string]string{
		"username": testHyperUserUsername,
		"password": "wrong-password",
	})

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
	}

	assertErrorMessage(t, body, errmsg.HyperUserWrongPassword)
}

func TestHyperUsersLoginUserNotFound(t *testing.T) {
	resp, body := performRequest(t, http.MethodPost, "/hypervisor/hyperusers/login", map[string]string{
		"username": "missing-user",
		"password": "whatever",
	})

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}

	assertErrorMessage(t, body, errmsg.HyperUserNotExists)
}

func TestHyperUsersLoginInvalidPayload(t *testing.T) {
	resp, body := performRequest(t, http.MethodPost, "/hypervisor/hyperusers/login", map[string]string{
		"username": "",
		"password": "",
	})

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}

	assertErrorMessage(t, body, errmsg.HyperUserInvalidPayload)
}

func TestHyperUsersLoginInvalidJSON(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/hypervisor/hyperusers/login", bytes.NewBufferString("not-json"))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}

	assertErrorMessage(t, body, errmsg.HyperUserInvalidPayload)
}

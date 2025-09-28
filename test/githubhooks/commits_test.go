package githubhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"hypervisor/internal/db"
	"hypervisor/internal/models"

	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/bson"
)

// Secrets and static fixture values for the webhook happy-path test.
const (
	testSecret       = "test-secret"
	testDeliveryID   = "delivery-123"
	testRef          = "refs/heads/main"
	testCommitSHA    = "abcdef1234567890"
	testCommitMsg    = "feat: add webhook receiver"
	testCommitAuthor = "Coder"
	testAuthorEmail  = "coder@example.com"
)

// TestGitHubPushEventPersistsCommits asserts commits land in the test collections
// and an audit event is emitted after ingestion.
func TestGitHubPushEventPersistsCommits(t *testing.T) {
	// Ensure the test collections are clean before starting.
	_, _ = db.GitCommits.DeleteMany(db.Ctx, bson.M{"delivery_id": testDeliveryID, "sha": testCommitSHA})
	_, _ = db.Events.DeleteMany(db.Ctx, bson.M{"targetID": testCommitSHA, "action": "github.commit.received"})

	payload := map[string]any{
		"ref": testRef,
		"commits": []map[string]any{
			{
				"id":        testCommitSHA,
				"message":   testCommitMsg,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"author": map[string]string{
					"name":  testCommitAuthor,
					"email": testAuthorEmail,
				},
			},
		},
	}

	body := mustEncodeJSON(t, payload)
	req := newWebhookRequest(t, http.MethodPost, "/hypervisor/github/commits", body)
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-GitHub-Delivery", testDeliveryID)
	req.Header.Set("X-Hub-Signature-256", fmt.Sprintf("sha256=%x", signPayload([]byte(testSecret), body)))

	resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d (body=%s)", http.StatusAccepted, resp.StatusCode, string(b))
	}

	var stored struct {
		DeliveryID string `bson:"delivery_id"`
		Ref        string `bson:"ref"`
		SHA        string `bson:"sha"`
		Message    string `bson:"message"`
		Author     struct {
			Name  string `bson:"name"`
			Email string `bson:"email"`
		} `bson:"author"`
	}

	if err := db.GitCommits.FindOne(db.Ctx, bson.M{"delivery_id": testDeliveryID, "sha": testCommitSHA}).Decode(&stored); err != nil {
		t.Fatalf("failed to find stored commit: %v", err)
	}

	if stored.Ref != testRef {
		t.Fatalf("expected ref %q, got %q", testRef, stored.Ref)
	}

	if stored.Message != testCommitMsg {
		t.Fatalf("expected message %q, got %q", testCommitMsg, stored.Message)
	}

	if stored.Author.Name != testCommitAuthor || stored.Author.Email != testAuthorEmail {
		t.Fatalf("expected author %s <%s>, got %s <%s>", testCommitAuthor, testAuthorEmail, stored.Author.Name, stored.Author.Email)
	}

	var event struct {
		Action string `bson:"action"`
		Target struct {
			ID string `bson:"targetID"`
		} `bson:""`
	}

	if err := db.Events.FindOne(db.Ctx, bson.M{"targetID": testCommitSHA, "action": "github.commit.received"}).Decode(&event); err != nil {
		t.Fatalf("expected event for commit: %v", err)
	}
}

// newWebhookRequest builds a Fiber-compatible HTTP request for the test payload.
func newWebhookRequest(t *testing.T, method, path string, body []byte) *http.Request {
	t.Helper()

	req, err := http.NewRequest(method, path, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	return req
}

// mustEncodeJSON encodes the payload and fails the test on serialization errors.
func mustEncodeJSON(t *testing.T, payload any) []byte {
	t.Helper()

	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	return encoded
}

// signPayload applies the same sha256 HMAC format GitHub sends in headers.
func signPayload(secret, payload []byte) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write(payload)
	return h.Sum(nil)
}

func TestGitHubTagEventCreatesRelease(t *testing.T) {
	// Ensure the test collections are clean before starting.
	_, _ = db.GitCommits.DeleteMany(db.Ctx, bson.M{"delivery_id": testDeliveryID, "sha": testCommitSHA})
	_, _ = db.Releases.DeleteMany(db.Ctx, bson.M{"tag": "v1.2.3"})

	payload := map[string]any{
		"ref": "refs/tags/v1.2.3",
		"commits": []map[string]any{
			{
				"id":        testCommitSHA,
				"message":   "chore: release v1.2.3",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"author": map[string]string{
					"name":  testCommitAuthor,
					"email": testAuthorEmail,
				},
			},
		},
	}

	body := mustEncodeJSON(t, payload)
	req := newWebhookRequest(t, http.MethodPost, "/hypervisor/github/commits", body)
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-GitHub-Delivery", testDeliveryID)
	req.Header.Set("X-Hub-Signature-256", fmt.Sprintf("sha256=%x", signPayload([]byte(testSecret), body)))

	resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d (body=%s)", http.StatusAccepted, resp.StatusCode, string(b))
	}

	// Check if commit is stored
	var storedCommit models.GitCommit
	if err := db.GitCommits.FindOne(db.Ctx, bson.M{"delivery_id": testDeliveryID, "sha": testCommitSHA}).Decode(&storedCommit); err != nil {
		t.Fatalf("failed to find stored commit: %v", err)
	}

	// Check if release is created
	var storedRelease models.Release
	if err := db.Releases.FindOne(db.Ctx, bson.M{"tag": "v1.2.3"}).Decode(&storedRelease); err != nil {
		t.Fatalf("failed to find stored release: %v", err)
	}

	if storedRelease.Status != "new" {
		t.Fatalf("expected release status 'new', got '%s'", storedRelease.Status)
	}
}

func TestGitHubTagEventWithoutCommitsStillCreatesRelease(t *testing.T) {
	const tagRef = "refs/tags/v9.9.9"
	_, _ = db.GitCommits.DeleteMany(db.Ctx, bson.M{"ref": tagRef})
	_, _ = db.Releases.DeleteMany(db.Ctx, bson.M{"tag": "v9.9.9"})

	payload := map[string]any{
		"ref":     tagRef,
		"commits": []map[string]any{},
	}

	body := mustEncodeJSON(t, payload)
	req := newWebhookRequest(t, http.MethodPost, "/hypervisor/github/commits", body)
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-GitHub-Delivery", testDeliveryID)
	req.Header.Set("X-Hub-Signature-256", fmt.Sprintf("sha256=%x", signPayload([]byte(testSecret), body)))

	resp, err := app.Test(req, fiber.TestConfig{Timeout: 0, FailOnTimeout: false})
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d (body=%s)", http.StatusAccepted, resp.StatusCode, string(b))
	}

	var storedRelease models.Release
	if err := db.Releases.FindOne(db.Ctx, bson.M{"tag": "v9.9.9"}).Decode(&storedRelease); err != nil {
		t.Fatalf("failed to find stored release: %v", err)
	}

	if storedRelease.Status != "new" {
		t.Fatalf("expected release status 'new', got '%s'", storedRelease.Status)
	}
}

package githubhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"hypervisor/internal/db"
	"hypervisor/internal/env"
	"hypervisor/internal/errmsg"
	"hypervisor/internal/events"
	"hypervisor/internal/models"
	releases_db "hypervisor/internal/releases/db"
	"hypervisor/internal/transformer"
	"hypervisor/internal/utils"

	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GitHub header keys and values that drive webhook validation.
const (
	signatureHeader = "X-Hub-Signature-256"
	eventHeader     = "X-GitHub-Event"
	deliveryHeader  = "X-GitHub-Delivery"
	pushEvent       = "push"
	signaturePrefix = "sha256="
)

// pushEventPayload models just the fields we rely on from a GitHub push hook.
type pushEventPayload struct {
	Ref     string            `json:"ref"`
	Commits []pushEventCommit `json:"commits"`
}

// pushEventCommit mirrors the subset of commit data we persist.
type pushEventCommit struct {
	ID        string                `json:"id"`
	Message   string                `json:"message"`
	Timestamp string                `json:"timestamp"`
	Author    pushEventCommitAuthor `json:"author"`
}

// pushEventCommitAuthor represents the commit author metadata we store.
type pushEventCommitAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// commitsHandler validates and ingests GitHub push events into MongoDB.
func commitsHandler(c fiber.Ctx) error {
	secret := strings.TrimSpace(env.GITHUB_WEBHOOK_SECRET)
	if secret == "" {
		return utils.StatusError(c, errmsg.GitHubSecretNotConfigured)
	}

	payload := c.Body()

	signature := strings.TrimSpace(c.Get(signatureHeader))
	if signature == "" {
		return utils.StatusError(c, errmsg.GitHubSignatureMissing)
	}

	// Reject requests whose HMAC cannot be verified with our shared secret.
	if !verifySignature(secret, signature, payload) {
		return utils.StatusError(c, errmsg.GitHubSignatureInvalid)
	}

	deliveryID := strings.TrimSpace(c.Get(deliveryHeader))
	if deliveryID == "" {
		return utils.StatusError(c, errmsg.GitHubDeliveryMissing)
	}

	eventType := strings.TrimSpace(c.Get(eventHeader))
	if eventType == "" {
		return utils.StatusError(c, errmsg.GitHubEventMissing)
	}

	// Ignore non-push events silently; GitHub retries other hooks separately.
	if eventType != pushEvent {
		return c.SendStatus(fiber.StatusNoContent)
	}

	var pushPayload pushEventPayload
	if err := json.Unmarshal(payload, &pushPayload); err != nil {
		return utils.StatusError(c, errmsg.GitHubInvalidPayload)
	}

	pushPayload.Ref = strings.TrimSpace(pushPayload.Ref)
	if pushPayload.Ref == "" {
		return utils.StatusError(c, errmsg.GitHubInvalidPayload)
	}

	if len(pushPayload.Commits) == 0 {
		if strings.HasPrefix(pushPayload.Ref, "refs/tags/") {
			tag := strings.TrimPrefix(pushPayload.Ref, "refs/tags/")
			if strings.HasPrefix(tag, "v") {
				release := models.Release{
					Tag:       tag,
					Status:    "new",
					CreatedAt: time.Now(),
				}
				if err := releases_db.Create(release); err != nil {
					return utils.StatusError(c, errmsg.InternalServerError(err))
				}
			}

			if events.Em != nil {
				events.Em.GitHubCommitReceived(
					deliveryID,
					tag,
					pushPayload.Ref,
					"tag: "+tag,
				)
			}
		}

		return c.SendStatus(fiber.StatusAccepted)
	}

	for _, commit := range pushPayload.Commits {
		gitCommit, err := convertCommit(deliveryID, pushPayload.Ref, commit)
		if err != nil {
			return utils.StatusError(c, errmsg.GitHubInvalidPayload)
		}

		// Store and announce each commit individually to stay idempotent per SHA.
		if err := upsertCommit(gitCommit); err != nil {
			return utils.StatusError(c, errmsg.InternalServerError(err))
		}

		// Transform commit into a release if applicable
		if err := transformer.Transform(gitCommit); err != nil {
			// For now, we can log this error and continue.
			// Depending on the requirements, we might want to handle this differently.
			log.Printf("failed to transform commit %s into a release: %v", gitCommit.SHA, err)
		}

		if events.Em != nil {
			events.Em.GitHubCommitReceived(
				deliveryID,
				gitCommit.SHA,
				gitCommit.Ref,
				gitCommit.Message,
			)
		}
	}

	return c.SendStatus(fiber.StatusAccepted)
}

// verifySignature compares a payload MAC against the expected secret-derived value.
func verifySignature(secret, signature string, payload []byte) bool {
	normalized := strings.ToLower(signature)
	if !strings.HasPrefix(normalized, signaturePrefix) {
		return false
	}

	expected := computeSignature(secret, payload)
	return hmac.Equal([]byte(expected), []byte(normalized))
}

// computeSignature renders the GitHub sha256= prefixed HMAC in hex form.
func computeSignature(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)

	return signaturePrefix + hex.EncodeToString(mac.Sum(nil))
}

// convertCommit normalises webhook commit data into the GitCommit model.
func convertCommit(deliveryID, ref string, commit pushEventCommit) (models.GitCommit, error) {
	sha := strings.TrimSpace(commit.ID)
	if sha == "" {
		return models.GitCommit{}, errors.New("missing commit sha")
	}

	timestamp := strings.TrimSpace(commit.Timestamp)
	if timestamp == "" {
		return models.GitCommit{}, errors.New("missing commit timestamp")
	}

	parsedTs, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return models.GitCommit{}, err
	}

	return models.GitCommit{
		DeliveryID: deliveryID,
		Ref:        ref,
		SHA:        sha,
		Message:    strings.TrimSpace(commit.Message),
		Author: models.GitCommitAuthor{
			Name:  strings.TrimSpace(commit.Author.Name),
			Email: strings.TrimSpace(commit.Author.Email),
		},
		Timestamp: parsedTs,
	}, nil
}

// upsertCommit persists a commit keyed by (delivery_id, sha) for idempotency.
func upsertCommit(commit models.GitCommit) error {
	filter := bson.M{
		"delivery_id": commit.DeliveryID,
		"sha":         commit.SHA,
	}

	update := bson.M{
		"$set": bson.M{
			"delivery_id": commit.DeliveryID,
			"ref":         commit.Ref,
			"sha":         commit.SHA,
			"message":     commit.Message,
			"author": bson.M{
				"name":  commit.Author.Name,
				"email": commit.Author.Email,
			},
			"timestamp": commit.Timestamp,
		},
	}

	opts := options.Update().SetUpsert(true)

	_, err := db.GitCommits.UpdateOne(db.Ctx, filter, update, opts)

	return err
}

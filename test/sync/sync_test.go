package sync

import (
	"context"
	"os"
	"testing"

	"hypervisor/internal/core"
	"hypervisor/internal/db"
	"hypervisor/internal/models"

	"go.mongodb.org/mongo-driver/bson"
)

func TestSyncReleases(t *testing.T) {
	ctx := context.Background()

	repoURL := os.Getenv("REPO_URL")
	if repoURL == "" {
		repoURL = "https://github.com/OpenLabsRo/openhack-backend"
	}

	err := core.SyncReleases(ctx, repoURL)
	if err != nil {
		t.Fatalf("SyncReleases failed: %v", err)
	}

	// Check that releases were synced
	cursor, err := db.Releases.Find(ctx, bson.M{})
	if err != nil {
		t.Fatalf("Failed to query releases: %v", err)
	}
	var releases []models.Release
	err = cursor.All(ctx, &releases)
	if err != nil {
		t.Fatalf("Failed to decode releases: %v", err)
	}
	if len(releases) == 0 {
		t.Error("Expected at least one release to be synced")
	}
}

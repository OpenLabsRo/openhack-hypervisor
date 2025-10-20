package core

import (
	"context"
	"os/exec"
	"strings"
	"time"

	"hypervisor/internal/events"
	"hypervisor/internal/models"

	"go.mongodb.org/mongo-driver/mongo"
)

func SyncReleases(ctx context.Context, repoURL string) error {
	if events.Em != nil {
		events.Em.SyncStarted(repoURL)
	}

	// Get tags from git ls-remote
	cmd := exec.Command("git", "ls-remote", "--tags", repoURL)
	output, err := cmd.Output()
	if err != nil {
		if events.Em != nil {
			events.Em.SyncFailed(repoURL, err)
		}
		return err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.Contains(line, "refs/tags/") && !strings.HasSuffix(line, "^{}") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				sha := parts[0]
				ref := parts[1]
				tag := strings.TrimPrefix(ref, "refs/tags/")

				// Check if release exists
				_, err := models.GetReleaseByID(ctx, tag)
				if err == mongo.ErrNoDocuments {
					// Create release
					release := models.Release{
						ID:        tag,
						Sha:       sha,
						CreatedAt: time.Now(),
					}
					err = models.CreateRelease(ctx, release)
					if err != nil {
						if events.Em != nil {
							events.Em.SyncFailed(repoURL, err)
						}
						return err
					}

					if events.Em != nil {
						events.Em.SyncReleaseCreated(tag, sha)
					}
				}
			}
		}
	}

	if events.Em != nil {
		events.Em.SyncFinished(repoURL)
	}

	return nil
}

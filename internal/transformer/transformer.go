package transformer

import (
	"hypervisor/internal/models"
	releases_db "hypervisor/internal/releases/db"
	"strings"
	"time"
)

func Transform(commit models.GitCommit) error {
	if strings.HasPrefix(commit.Ref, "refs/tags/v") {
		release := models.Release{
			Tag:       strings.TrimPrefix(commit.Ref, "refs/tags/"),
			Status:    "new",
			CreatedAt: time.Now(),
		}
		return releases_db.Create(release)
	}
	return nil
}

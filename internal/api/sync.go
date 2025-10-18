package api

import (
	"context"
	"os"

	"hypervisor/internal/core"
	"hypervisor/internal/errmsg"
	"hypervisor/internal/utils"

	"github.com/gofiber/fiber/v3"
)

// StatusResponse represents a generic success payload.
type StatusResponse struct {
	Status string `json:"status"`
}

// SyncHandler refreshes release data from the backing Git repository.
// @Summary Sync release metadata
// @Description Triggers a Git pull of the releases repository to refresh available releases for staging.
// @Tags Hypervisor Sync
// @Produce json
// @Success 200 {object} StatusResponse
// @Failure 500 {object} errmsg._InternalServerError
// @Router /hypervisor/sync [post]
func SyncHandler(c fiber.Ctx) error {
	// Assume repoURL from env or config
	repoURL := os.Getenv("REPO_URL")
	if repoURL == "" {
		repoURL = "https://github.com/OpenLabsRo/openhack-backend" // example
	}

	err := core.SyncReleases(context.Background(), repoURL)
	if err != nil {
		return utils.StatusError(c, errmsg.InternalServerError(err))
	}

	return c.JSON(StatusResponse{Status: "sync completed"})
}

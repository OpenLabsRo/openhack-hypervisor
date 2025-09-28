
package gitcommits

import (
	"hypervisor/internal/git"
	"hypervisor/internal/models"
	"hypervisor/internal/paths"
	"hypervisor/internal/transformer"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
)

const (
	repoURL = "https://github.com/openlabs-org/openhack-backend.git"
)

var (
	repoPath = paths.OpenHackRepoPath("backend")
)

func Routes(app fiber.Router) {
	gitcommits := app.Group("/gitcommits")
	gitcommits.Get("/", getAllCommitsHandler)
	gitcommits.Post("/sync", syncCommitsHandler)
}

func getAllCommitsHandler(c fiber.Ctx) error {
	commits, err := GetAll()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(commits)
}

func syncCommitsHandler(c fiber.Ctx) error {
	if err := git.CloneOrPull(repoURL, repoPath); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	tags, err := git.GetTags(repoPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	for _, tag := range tags {
		commit := models.GitCommit{
			Ref:       "refs/tags/" + tag,
			Message:   "tag: " + tag,
			Timestamp: time.Now(),
		}
		if err := Create(commit); err != nil {
			if !strings.Contains(err.Error(), "duplicate key error") {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
		}
        // Transform
        if err := transformer.Transform(commit); err != nil {
            return c.Status(500).JSON(fiber.Map{"error": err.Error()})
        }
	}

	return c.SendString("OK")
}

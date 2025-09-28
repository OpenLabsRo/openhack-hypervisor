
package api

import (
	"hypervisor/internal/releases/db"
	"hypervisor/internal/staging"
	"github.com/gofiber/fiber/v3"
)

func Routes(app fiber.Router) {
	releases := app.Group("/releases")
	releases.Get("/", getAllReleasesHandler)
	releases.Post("/:tag/stage", stageReleaseHandler)
}

func getAllReleasesHandler(c fiber.Ctx) error {
	releases, err := db.GetAll()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(releases)
}

func stageReleaseHandler(c fiber.Ctx) error {
    tag := c.Params("tag")
    if err := staging.StageRelease(tag); err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }
    return c.SendString("Staging process started for release " + tag)
}

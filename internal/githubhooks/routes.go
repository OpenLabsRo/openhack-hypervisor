// Package githubhooks exposes handlers for GitHub webhook callbacks.
package githubhooks

import "github.com/gofiber/fiber/v3"

// Routes wires the GitHub webhook endpoints under /hypervisor/github.
func Routes(app fiber.Router) {
	group := app.Group("/github")

	// POST /hypervisor/github/commits ingests push notifications from GitHub.
	group.Post("/commits", commitsHandler)
}

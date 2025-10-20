package helpers

import (
	"testing"

	"github.com/gofiber/fiber/v3"
)

func API_ListReleases(
	t *testing.T,
	app *fiber.App,
	token string,
) (bodyBytes []byte, statusCode int) {
	return RequestRunner(t, app,
		"GET",
		"/hypervisor/releases",
		nil,
		&token,
	)
}

func API_SyncReleases(
	t *testing.T,
	app *fiber.App,
	token string,
) (bodyBytes []byte, statusCode int) {
	return RequestRunner(t, app,
		"POST",
		"/hypervisor/releases/sync",
		nil,
		&token,
	)
}

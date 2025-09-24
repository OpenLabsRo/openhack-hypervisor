package githubhooks

import (
	"flag"
	"os"
	"testing"

	"hypervisor/internal"

	"github.com/gofiber/fiber/v3"
)

var app *fiber.App

// TestMain spins up the full Fiber app with the test deployment profile.
func TestMain(m *testing.M) {
	envRoot := flag.String("env-root", "", "directory containing environment files")
	appVersion := flag.String("app-version", "", "application version override")

	flag.Parse()

	// Ensure the webhook handler sees the same secret used by test payloads.
	os.Setenv("GITHUB_WEBHOOK_SECRET", testSecret)

	app = internal.SetupApp("test", *envRoot, *appVersion)

	os.Exit(m.Run())
}

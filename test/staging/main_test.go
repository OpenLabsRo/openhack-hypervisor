package staging

import (
	"flag"
	"os"
	"testing"

	"hypervisor/internal"

	"github.com/gofiber/fiber/v3"
)

var app *fiber.App

func TestMain(m *testing.M) {
	envRoot := flag.String("env-root", "", "directory containing environment files")
	appVersion := "v25.19.0.1"

	flag.Parse()

	app = internal.SetupApp("test", *envRoot, appVersion)

	os.Exit(m.Run())
}

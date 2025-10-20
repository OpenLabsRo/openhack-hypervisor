package releases

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
	appVersion := flag.String("app-version", "", "application version override")

	flag.Parse()

	app = internal.SetupApp("hypervisor_test", *envRoot, *appVersion)

	os.Exit(m.Run())
}

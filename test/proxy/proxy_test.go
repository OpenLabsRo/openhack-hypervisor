package proxy

import (
	"flag"
	"net/http"
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

	app = internal.SetupApp("proxy_test", *envRoot, *appVersion)

	os.Exit(m.Run())
}

func TestProxyAPIRoutesNotIntercepted(t *testing.T) {
	// Test that API routes are not intercepted by the proxy
	req, err := http.NewRequest("GET", "/hypervisor/meta/ping", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for API route, got %d", resp.StatusCode)
	}
}

// @title OpenHack Hypervisor API
// @version 25.10.17.4
// @description Hypervisor orchestration API for managing staged releases, deployments, and hyperuser access tooling.
// @BasePath /hypervisor
// @securityDefinitions.apikey HyperUserAuth
// @in header
// @name Authorization
// @description Provide the hyperuser bearer token as `Bearer <token>`.

// @Tag.name Hypervisor Meta
// @Tag.description Operational probes and metadata about the hypervisor service.

// @Tag.name Hypervisor Releases
// @Tag.description Repository and release synchronization workflows.

// @Tag.name Hypervisor Env
// @Tag.description Environment management for the service templates.

// @Tag.name Hypervisor Stages
// @Tag.description Manage stage lifecycle, sessions, and manual tests.

// @Tag.name Hypervisor Deployments
// @Tag.description Track staged deployment records ready for promotion.

// @Tag.name Hyperusers Meta
// @Tag.description Lightweight availability checks for hyperuser endpoints.

// @Tag.name Hyperusers Auth
// @Tag.description Authentication flows for hyperuser operators.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"hypervisor/internal"
	"hypervisor/internal/env"
	"hypervisor/internal/swagger"

	"github.com/gofiber/fiber/v3"
)

func main() {
	deployment := flag.String("deployment", "", "deployment profile (dev|test|prod)")
	portFlag := flag.String("port", "", "port to listen on")
	envRoot := flag.String("env-root", "", "directory containing environment files")
	appVersion := flag.String("app-version", "", "application version override")

	flag.Parse()

	ensureHome()

	deploy := strings.TrimSpace(*deployment)
	if deploy == "" {
		args := flag.Args()
		if len(args) == 0 {
			fmt.Println("Usage: server --deployment <type> --port <port> [--env-root <dir>] [--app-version <version>]")
			os.Exit(1)
		}
		deploy = strings.TrimSpace(args[0])
	}

	if deploy == "" {
		log.Fatal("deployment is required")
	}

	port := strings.TrimSpace(*portFlag)
	if port == "" {
		log.Fatal("port is required")
	}

	app := internal.SetupApp(deploy, *envRoot, *appVersion)
	swagger.Register(app)

	fmt.Println("HYPERVISOR VERSION:", env.VERSION)

	// Channel to listen for interrupt or terminate signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		fmt.Printf("Starting server on port %s...\n", port)
		if err := app.Listen(fmt.Sprintf(":%s", port), fiber.ListenConfig{
			EnablePrefork: env.PREFORK,
		}); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-c
	fmt.Println("\nReceived shutdown signal, gracefully shutting down...")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Gracefully shutdown the server
	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
		os.Exit(1)
	}

	fmt.Println("Server shutdown complete")
	os.Exit(0)
}

func ensureHome() {
	home := strings.TrimSpace(os.Getenv("HOME"))
	if home == "" {
		home = "/var/openhack"
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		log.Printf("warning: unable to create HOME directory %s: %v", home, err)
		return
	}
	_ = os.Setenv("HOME", home)
}

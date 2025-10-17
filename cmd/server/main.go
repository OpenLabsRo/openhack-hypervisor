package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"hypervisor/internal"
	"hypervisor/internal/env"

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

	fmt.Println("APP VERSION:", env.VERSION)

	if err := app.Listen(fmt.Sprintf(":%s", port), fiber.ListenConfig{
		EnablePrefork: env.PREFORK,
	}); err != nil {
		log.Fatalf("Error listening on port %s: %v", port, err)
	}
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

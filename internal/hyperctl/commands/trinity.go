package commands

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"hypervisor/internal/hyperctl/build"
	fsops "hypervisor/internal/hyperctl/fs"
	"hypervisor/internal/hyperctl/git"
	"hypervisor/internal/hyperctl/health"
	"hypervisor/internal/hyperctl/system"
	"hypervisor/internal/hyperctl/testing"
	"hypervisor/internal/paths"
)

// RunTrinity handles the `hyperctl trinity` subcommand.
// It performs a blue-green deployment: updates blue first, then green.
func RunTrinity(args []string) error {
	// Parse command-line flags
	fs := flag.NewFlagSet("trinity", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	version := fs.String("version", "", "Specific version to build (default: latest)")
	dev := fs.Bool("dev", false, "Development mode: use current directory instead of cloning")

	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Println("Starting blue-green deployment (Trinity)...")

	// Verify system requirements
	if err := system.CheckPrerequisites(); err != nil {
		return err
	}

	// Create necessary directories
	fmt.Println("Ensuring directory layout...")
	if err := fsops.EnsureLayout(); err != nil {
		return err
	}
	fmt.Println("Directory layout ready")

	deployment := "prod"
	if *dev {
		deployment = "dev"
	}

	// Build new version first
	fmt.Println("Building new version...")
	if err := buildAndTestNewVersion(*version, *dev); err != nil {
		return fmt.Errorf("failed to build new version: %w", err)
	}

	// Blue-green deployment sequence
	fmt.Println("Starting blue-green deployment sequence...")

	// Update BLUE first (8080)
	fmt.Println("=== UPDATING BLUE (8080) ===")
	if err := updateService("blue", "localhost:8080", deployment); err != nil {
		return fmt.Errorf("failed to update blue service: %w", err)
	}

	// Update GREEN second (8081)
	fmt.Println("=== UPDATING GREEN (8081) ===")
	if err := updateService("green", "localhost:8081", deployment); err != nil {
		return fmt.Errorf("failed to update green service: %w", err)
	}

	fmt.Println("Blue-green deployment completed successfully!")
	fmt.Println("Both services are now running the updated version.")

	// Final health verification
	fmt.Println("Performing final health checks...")
	if err := health.CheckHost("localhost:8080"); err != nil {
		return fmt.Errorf("blue service final health check failed: %w", err)
	}
	if err := health.CheckHost("localhost:8081"); err != nil {
		return fmt.Errorf("green service final health check failed: %w", err)
	}
	fmt.Println("Final health checks passed")

	return nil
}

func buildAndTestNewVersion(version string, dev bool) error {
	var targetVersion string

	if dev {
		// Development mode: use current directory
		fmt.Println("Development mode: using current directory")
		repoDir := "."

		// Run API_SPEC
		cmd := exec.Command("./API_SPEC")
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run API_SPEC: %w", err)
		}
		fmt.Println("API_SPEC executed successfully")

		// Read version from current directory
		versionData, err := os.ReadFile(filepath.Join(repoDir, "VERSION"))
		if err != nil {
			return fmt.Errorf("failed to read VERSION file in dev mode: %w", err)
		}
		targetVersion = strings.TrimSpace(string(versionData))

		// Build from current directory
		fmt.Printf("Building version %s from current directory...\n", targetVersion)
		buildResult, err := build.Run(repoDir, paths.HypervisorBuildsDir)
		if err != nil {
			return err
		}
		fmt.Printf("Build complete: %s\n", buildResult.BinaryPath)
		fmt.Printf("New version ready: %s\n", buildResult.Version)
		return nil
	}

	// Production mode: check main repo for updates
	fmt.Println("Checking main repository for updates...")
	mainRepoDir := filepath.Join(paths.HypervisorReposDir, "main")

	// Clone or update main repo
	if err := git.CloneOrPull(hypervisorRepoURL, mainRepoDir); err != nil {
		return fmt.Errorf("failed to clone/update main repo: %w", err)
	}

	// Read latest version from main repo
	mainVersionData, err := os.ReadFile(filepath.Join(mainRepoDir, "VERSION"))
	if err != nil {
		return fmt.Errorf("failed to read VERSION from main repo: %w", err)
	}
	latestVersion := strings.TrimSpace(string(mainVersionData))

	// Determine target version
	if version != "" {
		targetVersion = version
		fmt.Printf("Using specified version: %s\n", targetVersion)
	} else {
		targetVersion = latestVersion
		fmt.Printf("Using latest version from main repo: %s\n", targetVersion)
	}

	// Clone the specific version to versioned directory
	fmt.Printf("Cloning version %s...\n", targetVersion)
	versionedRepoDir := filepath.Join(paths.HypervisorReposDir, targetVersion)
	if err := git.CloneOrPull(hypervisorRepoURL, versionedRepoDir); err != nil {
		return fmt.Errorf("failed to clone version %s: %w", targetVersion, err)
	}

	// Checkout the specific version/tag
	if err := git.Checkout(versionedRepoDir, targetVersion); err != nil {
		return fmt.Errorf("failed to checkout version %s: %w", targetVersion, err)
	}

	fmt.Printf("Project cloned to %s\n", versionedRepoDir)

	// Run API_SPEC
	cmd := exec.Command("./API_SPEC")
	cmd.Dir = versionedRepoDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run API_SPEC: %w", err)
	}
	fmt.Println("API_SPEC executed successfully")

	// Run test suite
	fmt.Println("Testing the code...")
	if err := testing.RunTests(versionedRepoDir); err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}

	// Build the application
	fmt.Printf("Building the code into %s...\n", paths.HypervisorBuildsDir)
	buildResult, err := build.Run(versionedRepoDir, paths.HypervisorBuildsDir)
	if err != nil {
		return err
	}
	fmt.Printf("Build complete: %s\n", buildResult.BinaryPath)
	fmt.Printf("New version ready: %s\n", buildResult.Version)

	return nil
}

func updateService(serviceName, apiHost, deployment string) error {
	fmt.Printf("Draining %s service...\n", serviceName)

	// Health check before enabling drain mode
	fmt.Printf("Checking %s service health before draining...\n", serviceName)
	if err := health.CheckHost(apiHost); err != nil {
		return fmt.Errorf("service health check failed before draining: %w", err)
	}

	if err := enableDrainMode(apiHost, true); err != nil {
		return fmt.Errorf("failed to enable drain mode: %w", err)
	}

	// Wait a bit for traffic to drain
	fmt.Println("Waiting for traffic to drain (5 seconds)...")
	time.Sleep(5 * time.Second)

	fmt.Printf("Stopping %s service...\n", serviceName)
	if err := stopService(serviceName); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	fmt.Printf("Updating %s service binary...\n", serviceName)
	if err := updateServiceBinary(serviceName, deployment); err != nil {
		return fmt.Errorf("failed to update service binary: %w", err)
	}

	fmt.Printf("Starting %s service...\n", serviceName)
	if err := startService(serviceName); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Wait a bit for blue service to fully start before disabling drain
	if serviceName == "blue" {
		fmt.Println("Waiting 5 seconds for blue service to stabilize...")
		time.Sleep(5 * time.Second)
	}

	fmt.Printf("Disabling drain mode for %s...\n", serviceName)

	// Health check before disabling drain mode
	fmt.Printf("Checking %s service health before disabling drain...\n", serviceName)
	if err := health.CheckHost(apiHost); err != nil {
		return fmt.Errorf("service health check failed before disabling drain: %w", err)
	}

	if err := enableDrainMode(apiHost, false); err != nil {
		return fmt.Errorf("failed to disable drain mode: %w", err)
	}

	fmt.Printf("Verifying %s service health...\n", serviceName)
	if err := verifyServiceHealth(serviceName); err != nil {
		return fmt.Errorf("service health check failed: %w", err)
	}

	fmt.Printf("✅ %s service updated successfully!\n", serviceName)
	return nil
}

func enableDrainMode(apiHost string, enabled bool) error {
	url := fmt.Sprintf("http://%s/hypervisor/meta/drain", apiHost)

	reqBody := map[string]bool{"enabled": enabled}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("drain API returned status %d", resp.StatusCode)
	}

	return nil
}

func stopService(serviceName string) error {
	cmd := exec.Command("sudo", "systemctl", "stop", fmt.Sprintf("openhack-hypervisor-%s.service", serviceName))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func startService(serviceName string) error {
	cmd := exec.Command("sudo", "systemctl", "start", fmt.Sprintf("openhack-hypervisor-%s.service", serviceName))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func updateServiceBinary(serviceName, deployment string) error {
	// Get the latest built binary from the build process
	buildsDir := paths.HypervisorBuildsDir
	entries, err := os.ReadDir(buildsDir)
	if err != nil {
		return fmt.Errorf("failed to read builds directory: %w", err)
	}

	var latestBinary string
	var latestVersion string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// Binary files are named with just the version (e.g., "25.10.20.3")
		version := entry.Name()
		if latestVersion == "" || version > latestVersion {
			latestVersion = version
			latestBinary = filepath.Join(buildsDir, entry.Name())
		}
	}

	if latestBinary == "" {
		return fmt.Errorf("no hypervisor binary found in %s", buildsDir)
	}

	// Update the systemd service file to point to the new binary
	serviceFile := filepath.Join(paths.SystemdUnitDir, fmt.Sprintf("openhack-hypervisor-%s.service", serviceName))

	// Read current service file
	content, err := os.ReadFile(serviceFile)
	if err != nil {
		return fmt.Errorf("failed to read service file: %w", err)
	}

	// Replace the ExecStart line with new binary path
	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "ExecStart=") {
			lines[i] = fmt.Sprintf("ExecStart=%s --deployment %s --port %s --env-root %s --app-version %s",
				latestBinary, deployment, getServicePort(serviceName), paths.HypervisorEnvDir, latestVersion)
			break
		}
	}

	// Write back the updated service file
	updatedContent := strings.Join(lines, "\n")
	if err := writeFileWithSudo(serviceFile, []byte(updatedContent)); err != nil {
		return fmt.Errorf("failed to write updated service file: %w", err)
	}

	// Reload systemd daemon
	cmd := exec.Command("sudo", "systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	fmt.Printf("Updated %s service to use binary: %s (version %s)\n", serviceName, latestBinary, latestVersion)
	return nil
}

func getServicePort(serviceName string) string {
	switch serviceName {
	case "blue":
		return "8080"
	case "green":
		return "8081"
	default:
		return "8080"
	}
}

func verifyServiceHealth(serviceName string) error {
	// Wait a bit for service to fully start
	time.Sleep(3 * time.Second)

	cmd := exec.Command("systemctl", "is-active", fmt.Sprintf("openhack-hypervisor-%s.service", serviceName))
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	status := string(output)
	if status != "active\n" && status != "active" {
		return fmt.Errorf("service not active: %s", status)
	}
	return nil
}

func writeFileWithSudo(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	cmd := exec.Command("sudo", "tee", path)
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo tee %s failed: %w", path, err)
	}

	if err := runCommand("sudo", "chmod", "0644", path); err != nil {
		return fmt.Errorf("sudo chmod %s failed: %w", path, err)
	}

	return nil
}

func runCommand(cmd string, args ...string) error {
	command := exec.Command(cmd, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

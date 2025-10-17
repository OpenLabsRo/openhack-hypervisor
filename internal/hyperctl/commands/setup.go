package commands

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"hypervisor/internal/hyperctl/build"
	fsops "hypervisor/internal/hyperctl/fs"
	"hypervisor/internal/hyperctl/system"
	"hypervisor/internal/hyperctl/systemd"
	"hypervisor/internal/paths"

	backendgit "hypervisor/internal/git"
)

const (
	hypervisorRepoURL = "https://github.com/OpenLabsRo/openhack-hypervisor"
)

// RunSetup handles the `hyperctl setup` subcommand.
func RunSetup(args []string) error {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dev := fs.Bool("dev", false, "Development mode: use current directory instead of cloning")

	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Println("Starting hypervisor setup...")

	if err := checkPrerequisites(); err != nil {
		return err
	}

	fmt.Println("Ensuring directory layout...")
	if err := fsops.EnsureLayout(); err != nil {
		return err
	}
	fmt.Println("Directory layout ready")

	editor := system.ResolveEditor()

	hypervisorEnvPath := fsops.HypervisorEnvPath()
	if err := fsops.EnsureEnvDirFor(hypervisorEnvPath); err != nil {
		return err
	}
	if err := fsops.EditEnvFileIfMissing("Hypervisor", hypervisorEnvPath, editor); err != nil {
		return err
	}

	fmt.Println("Cloning the project...")
	var repoDir string
	if *dev {
		repoDir = "."
		fmt.Println("Development mode: using current directory")
	} else {
		repoDir = filepath.Join(paths.HypervisorReposDir, "main")
		if err := backendgit.CloneOrPull(hypervisorRepoURL, repoDir); err != nil {
			return fmt.Errorf("failed to clone project: %w", err)
		}
		fmt.Printf("Project cloned to %s\n", repoDir)
	}

	if !*dev {
		fmt.Println("Testing the code...")
		fmt.Println("========== RUNNING TESTS ==========")
		if err := runTest(repoDir); err != nil {
			fmt.Println("========== TESTS FAILED ==========")
			return fmt.Errorf("tests failed: %w", err)
		}
		fmt.Println("========== TESTS PASSED ==========")
	}

	fmt.Printf("Building the code into %s...\n", paths.HypervisorBuildsDir)
	buildResult, err := build.Run(repoDir, paths.HypervisorBuildsDir)
	if err != nil {
		return err
	}
	fmt.Printf("Build complete: %s\n", buildResult.BinaryPath)

	fmt.Println("Installing systemd unit for hypervisor...")
	cfg := systemd.ServiceConfig{
		BinaryPath: buildResult.BinaryPath,
		Deployment: "prod",
		Port:       "8080",
		EnvRoot:    paths.HypervisorEnvDir,
		Version:    buildResult.Version,
		GoPath:     runtime.GOROOT(),
	}
	if err := systemd.InstallHypervisorService(cfg); err != nil {
		return err
	}
	fmt.Println("Systemd unit installed")

	fmt.Println("Reloading systemd...")
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}
	fmt.Println("Systemd reloaded")

	fmt.Println("Checking service status...")
	if err := checkServiceStatus(); err != nil {
		return fmt.Errorf("service status check failed: %w", err)
	}
	fmt.Println("Service is active")

	fmt.Println("Performing health check...")
	if err := healthCheck(); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	fmt.Println("Health check passed")

	fmt.Println("Setup completed successfully.")

	return nil
}

func runTest(repoDir string) error {
	cmd := exec.Command("./TEST")
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func checkServiceStatus() error {
	cmd := exec.Command("systemctl", "is-active", "openhack-hypervisor")
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	status := strings.TrimSpace(string(output))
	if status != "active" {
		return fmt.Errorf("service not active: %s", status)
	}
	return nil
}

func healthCheck() error {
	url := "http://localhost:8080/hypervisor/ping"
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}
	return nil
}

func checkPrerequisites() error {
	fmt.Println("Checking systemctl availability...")
	if err := system.EnsureSystemctlAccessible(); err != nil {
		return fmt.Errorf("systemctl unavailable: %w", err)
	}
	fmt.Println("systemctl is available")

	fmt.Println("Checking redis-server availability...")
	if err := system.EnsureRedisAvailable(); err != nil {
		return fmt.Errorf("redis missing: %w", err)
	}
	fmt.Println("redis-server is available")

	editor := system.ResolveEditor()
	editorBinary := editor
	if fields := strings.Fields(editor); len(fields) > 0 {
		editorBinary = fields[0]
	}

	fmt.Printf("Checking editor %q...\n", editorBinary)
	if err := system.CheckBinary(editorBinary); err != nil {
		return fmt.Errorf("editor %q not found: %w", editorBinary, err)
	}
	fmt.Printf("Editor %q is available\n", editorBinary)

	return nil
}

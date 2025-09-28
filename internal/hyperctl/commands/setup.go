package commands

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"hypervisor/internal/hyperctl/build"
	fsops "hypervisor/internal/hyperctl/fs"
	hyperctlgit "hypervisor/internal/hyperctl/git"
	"hypervisor/internal/hyperctl/state"
	"hypervisor/internal/hyperctl/system"
	"hypervisor/internal/hyperctl/systemd"
	"hypervisor/internal/paths"

	backendgit "hypervisor/internal/git"
)

const (
	hypervisorRepoURL      = "https://github.com/OpenLabsRo/openhack-hypervisor"
	openHackBackendRepoURL = "https://github.com/openlabsro/openhack-backend.git"
	openHackBackendRepoDir = "backend"
)

// RunSetup handles the `hyperctl setup` subcommand.
func RunSetup(args []string) error {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

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

	tagInput := ""
	var tagName string
	var commitHash string
	var err error

	if tagInput == "" {
		fmt.Println("Resolving latest release tag...")
		tagName, commitHash, err = hyperctlgit.LatestRemoteTag(hypervisorRepoURL, "v")
		if err != nil {
			return fmt.Errorf("failed to determine latest tag: %w", err)
		}
	} else {
		tagName = normalizeTag(tagInput)
		fmt.Printf("Resolving commit for tag %s...\n", tagName)
		commitHash, err = hyperctlgit.TagCommit(hypervisorRepoURL, tagName)
		if err != nil {
			return fmt.Errorf("failed to resolve commit for %s: %w", tagName, err)
		}
	}

	version := strings.TrimPrefix(tagName, "v")
	fmt.Printf("Selected version %s (tag %s, commit %s)\n", version, tagName, commitHash)

	repoDir := filepath.Join(paths.HypervisorReposDir, version)
	fmt.Printf("Ensuring repository checkout at %s...\n", repoDir)
	checkoutCommit, err := hyperctlgit.EnsureRepoAtTag(hypervisorRepoURL, repoDir, tagName)
	if err != nil {
		return err
	}
	fmt.Printf("Repository ready at commit %s\n", checkoutCommit)
	commitHash = checkoutCommit

	fmt.Printf("Running BUILD script into %s...\n", paths.HypervisorBuildsDir)
	buildResult, err := build.Run(repoDir, paths.HypervisorBuildsDir)
	if err != nil {
		return err
	}
	fmt.Printf("Build complete: %s\n", buildResult.BinaryPath)

	fmt.Println("Stopping existing hypervisor service (if running)...")
	if err := systemd.StopHypervisorService(); err != nil {
		return err
	}
	fmt.Println("Hypervisor service stopped")

	fmt.Printf("Updating current build symlink to %s...\n", buildResult.BinaryPath)
	if err := updateCurrentSymlink(buildResult.BinaryPath); err != nil {
		return err
	}
	fmt.Println("Current build updated")

	fmt.Println("Persisting installation state...")
	if err := state.Save(state.State{
		Version:   buildResult.Version,
		Tag:       tagName,
		Commit:    commitHash,
		BuildPath: buildResult.BinaryPath,
	}); err != nil {
		return err
	}
	fmt.Println("State saved")

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
	fmt.Println("Systemd unit installed and service restarted")

	openHackEnvPath := fsops.OpenHackEnvPath()
	if err := fsops.EnsureEnvDirFor(openHackEnvPath); err != nil {
		return err
	}
	if err := fsops.EditEnvFileIfMissing("OpenHack", openHackEnvPath, editor); err != nil {
		return err
	}

	fmt.Println("Syncing OpenHack backend repository...")
	backendRepoPath := paths.OpenHackRepoPath(openHackBackendRepoDir)
	if err := backendgit.CloneOrPull(openHackBackendRepoURL, backendRepoPath); err != nil {
		return fmt.Errorf("failed to sync backend repository: %w", err)
	}
	fmt.Println("OpenHack backend repository is up to date")

	if err := waitForServerReady(); err != nil {
		return err
	}

	fmt.Printf("Setup completed successfully for version %s (commit %s).\n", buildResult.Version, commitHash)

	return nil
}

func waitForServerReady() error {
	fmt.Println("Waiting for hypervisor service to be ready...")
	for i := 0; i < 10; i++ {
		resp, err := http.Get("http://localhost:8080/hypervisor/ping")
		if err == nil && resp.StatusCode == http.StatusOK {
			fmt.Println("Hypervisor service is ready.")
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("hypervisor service is not ready after 10 seconds")
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

func updateCurrentSymlink(target string) error {
	link := filepath.Join(paths.HypervisorBuildsDir, "current")
	tmp := link + ".tmp"

	if err := os.Remove(tmp); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove temp symlink: %w", err)
	}

	if err := os.Symlink(target, tmp); err != nil {
		return fmt.Errorf("failed to create temp symlink: %w", err)
	}

	if err := os.Remove(link); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(tmp)
		return fmt.Errorf("failed to replace current symlink: %w", err)
	}

	if err := os.Rename(tmp, link); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("failed to finalize current symlink: %w", err)
	}

	return nil
}

func normalizeTag(tag string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return tag
	}
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	return tag
}

package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"hypervisor/internal/hyperctl/build"
	fsops "hypervisor/internal/hyperctl/fs"
	"hypervisor/internal/hyperctl/git"
	"hypervisor/internal/hyperctl/state"
	"hypervisor/internal/hyperctl/system"
	"hypervisor/internal/hyperctl/systemd"
	"hypervisor/internal/paths"
)

// RunSetup handles the `hyperctl setup` subcommand.
func RunSetup(args []string) error {
	skipEdit := true

	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&skipEdit, "no-edit", false, "skip opening the env file in an editor")

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

	envPath := filepath.Join(paths.HypervisorEnvDir, ".env")
	fmt.Printf("Preparing environment file at %s...\n", envPath)
	if err := fsops.EnsureEnvFile(envPath); err != nil {
		return err
	}
	fmt.Println("Environment file ready")

	if !skipEdit {
		editor := system.ResolveEditor()
		fmt.Printf("Opening %s with %s...\n", envPath, editor)
		if err := launchEditor(editor, envPath); err != nil {
			return fmt.Errorf("editor exit error: %w", err)
		}
		fmt.Println("Environment file saved")
	} else {
		fmt.Println("Skipping interactive edit (--no-edit)")
	}

	repoURL := "https://github.com/OpenLabsRo/openhack-hypervisor"

	tagInput := ""
	var tagName string
	var commitHash string
	var err error

	if tagInput == "" {
		fmt.Println("Resolving latest release tag...")
		tagName, commitHash, err = git.LatestRemoteTag(repoURL, "v")
		if err != nil {
			return fmt.Errorf("failed to determine latest tag: %w", err)
		}
	} else {
		tagName = normalizeTag(tagInput)
		fmt.Printf("Resolving commit for tag %s...\n", tagName)
		commitHash, err = git.TagCommit(repoURL, tagName)
		if err != nil {
			return fmt.Errorf("failed to resolve commit for %s: %w", tagName, err)
		}
	}

	version := strings.TrimPrefix(tagName, "v")
	fmt.Printf("Selected version %s (tag %s, commit %s)\n", version, tagName, commitHash)

	repoDir := filepath.Join(paths.HypervisorReposDir, version)
	fmt.Printf("Ensuring repository checkout at %s...\n", repoDir)
	checkoutCommit, err := git.EnsureRepoAtTag(repoURL, repoDir, tagName)
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
	}
	if err := systemd.InstallHypervisorService(cfg); err != nil {
		return err
	}
	fmt.Println("Systemd unit installed and service restarted")

	if err := runReleaseSync(); err != nil {
		return err
	}

	if err := stageLatestRelease(); err != nil {
		return err
	}

	fmt.Printf("Setup completed successfully for version %s (commit %s).\n", buildResult.Version, commitHash)

	return nil
}

func runReleaseSync() error {
	fmt.Println("Triggering initial release sync...")
	resp, err := http.Post("http://localhost:8080/hypervisor/gitcommits/sync", "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to trigger release sync: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("release sync failed with status: %s", resp.Status)
	}

	fmt.Println("Release sync completed successfully.")
	return nil
}

func stageLatestRelease() error {
	fmt.Println("Getting releases...")
	resp, err := http.Get("http://localhost:8080/hypervisor/releases")
	if err != nil {
		return fmt.Errorf("failed to get releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get releases: status %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read releases response: %w", err)
	}

	type Release struct {
		Tag string `json:"tag"`
	}
	var releases []Release
	if err := json.Unmarshal(body, &releases); err != nil {
		return fmt.Errorf("failed to parse releases response: %w", err)
	}

	if len(releases) == 0 {
		return fmt.Errorf("no releases found to stage")
	}

	latestRelease := releases[len(releases)-1]
	tag := latestRelease.Tag

	fmt.Printf("Staging latest release: %s\n", tag)

	stageURL := fmt.Sprintf("http://localhost:8080/hypervisor/releases/%s/stage", tag)
	resp, err = http.Post(stageURL, "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to stage release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to stage release: status %s", resp.Status)
	}

	fmt.Println("Staging process started successfully.")
	return nil
}

func launchEditor(editor, path string) error {
	fields := strings.Fields(editor)
	if len(fields) == 0 {
		return fmt.Errorf("invalid editor command")
	}

	cmd := exec.Command(fields[0], append(fields[1:], path)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
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

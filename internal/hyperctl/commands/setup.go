package commands

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"hypervisor/internal/hyperctl/build"
	fsops "hypervisor/internal/hyperctl/fs"
	"hypervisor/internal/hyperctl/git"
	"hypervisor/internal/hyperctl/state"
	"hypervisor/internal/hyperctl/system"
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

	fmt.Printf("Setup completed successfully for version %s (commit %s).\n", buildResult.Version, commitHash)

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

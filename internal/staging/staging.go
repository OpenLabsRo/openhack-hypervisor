package staging

import (
	"fmt"
	"hypervisor/internal/git"
	"hypervisor/internal/paths"
	"hypervisor/internal/releases/db"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	repoURL = "https://github.com/openlabsro/openhack-backend.git"
)

func StageRelease(tag string) error {
	repoPath := paths.OpenHackRepoPath(tag)
	repoExists := false

	if info, err := os.Stat(repoPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to stat repo %s: %w", repoPath, err)
		}
	} else if !info.IsDir() {
		return fmt.Errorf("repo path %s exists and is not a directory", repoPath)
	} else {
		repoExists = true
	}

	if !repoExists {
		// Fresh clone of the specific tag.
		if err := git.CloneTag(repoURL, tag, repoPath); err != nil {
			return err
		}
	} else {
		// Re-use existing checkout by syncing and forcing it to the desired tag.
		if err := git.EnsureWorkTree(repoPath); err != nil {
			return err
		}
		if err := git.EnsureRemote(repoPath, repoURL); err != nil {
			return err
		}
		if err := git.FetchTags(repoPath); err != nil {
			return err
		}
	}

	if err := git.CheckoutTag(repoPath, tag); err != nil {
		return err
	}

	envRoot := paths.OpenHackEnvDir
	envFile := filepath.Join(envRoot, ".env")
	if _, err := os.Stat(envFile); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("environment file %s not found; populate it before staging", envFile)
		}
		return fmt.Errorf("failed to access environment file %s: %w", envFile, err)
	}

	repoEnvPath := filepath.Join(repoPath, ".env")
	if _, err := os.Stat(repoEnvPath); os.IsNotExist(err) {
		contents, readErr := os.ReadFile(envFile)
		if readErr != nil {
			return fmt.Errorf("failed to read environment file %s: %w", envFile, readErr)
		}
		if writeErr := os.WriteFile(repoEnvPath, contents, 0o600); writeErr != nil {
			return fmt.Errorf("failed to write environment file to %s: %w", repoEnvPath, writeErr)
		}
	}

	// Ensure the builds directory exists so scripts can emit artefacts there if desired.
	if err := os.MkdirAll(paths.OpenHackBuildsDir, 0o755); err != nil {
		return fmt.Errorf("failed to ensure builds directory: %w", err)
	}

	// 2. Build the release
	buildScript := paths.OpenHackRepoPath(tag, "BUILD")
	buildCmd := exec.Command(buildScript)
	buildCmd.Dir = repoPath
	if output, err := buildCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("build failed: %w\n%s", err, output)
	}

	// 3. Test the release
	testScript := paths.OpenHackRepoPath(tag, "TEST")
	testCmd := exec.Command(testScript)
	testCmd.Dir = repoPath
	if output, err := testCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("test failed: %w\n%s", err, output)
	}

	// 4. Update release status in DB
	if err := db.UpdateStatus(tag, "staged"); err != nil {
		return err
	}

	// 5. Deploy to inactive slot (to be implemented)

	return nil
}

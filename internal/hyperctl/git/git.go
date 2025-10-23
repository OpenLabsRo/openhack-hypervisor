package git

import (
	"fmt"
	"os/exec"

	"hypervisor/internal/hyperctl/user"
)

// CloneOrPull clones the repository if it doesn't exist, or pulls if it does.
// If the directory exists but is not a valid git repo, it removes and re-clones.
func CloneOrPull(repoURL, destDir string) error {
	// Check if destDir is a valid git repo
	if _, err := exec.Command("git", "-C", destDir, "rev-parse", "--git-dir").CombinedOutput(); err == nil {
		// Is a valid git repo, pull latest
		cmd := exec.Command("git", "pull")
		cmd.Dir = destDir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git pull failed: %v: %s", err, string(out))
		}

		// Ensure ownership is correct after pull
		if err := user.ChownToOpenhack(destDir); err != nil {
			return fmt.Errorf("failed to chown repo after pull: %w", err)
		}

		return nil
	}

	// Not a valid git repo - either doesn't exist or is corrupted
	// Remove the directory if it exists (to handle partial/corrupted clones)
	if _, err := exec.Command("test", "-d", destDir).CombinedOutput(); err == nil {
		// Directory exists but is not a valid git repo, remove it
		cmd := exec.Command("rm", "-rf", destDir)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to remove corrupted repo directory %s: %w", destDir, err)
		}
	}

	// Clone the repository
	cmd := exec.Command("git", "clone", repoURL, destDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %v: %s", err, string(out))
	}

	// Chown the cloned repository to openhack user
	if err := user.ChownToOpenhack(destDir); err != nil {
		return fmt.Errorf("failed to chown cloned repo: %w", err)
	}

	return nil
}

// Checkout checks out a specific branch, tag, or commit in the repository.
func Checkout(repoDir, ref string) error {
	// Ensure we have all refs and tags
	fetch := exec.Command("git", "fetch", "--all", "--tags")
	fetch.Dir = repoDir
	if out, err := fetch.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch failed: %v: %s", err, string(out))
	}

	// Try checkout the ref directly
	cmd := exec.Command("git", "checkout", ref)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err == nil {
		return nil
	} else {
		// Try fallback: checkout origin/ref (branch)
		cmd2 := exec.Command("git", "checkout", "origin/"+ref)
		cmd2.Dir = repoDir
		if out2, err2 := cmd2.CombinedOutput(); err2 == nil {
			return nil
		} else {
			return fmt.Errorf("git checkout failed: %v: %s; fallback origin/%s failed: %v: %s", err, string(out), ref, err2, string(out2))
		}
	}
}

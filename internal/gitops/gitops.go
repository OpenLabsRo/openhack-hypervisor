package gitops

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// CloneAndCheckout clones the provided repo into repoPath and checks out the given sha.
func CloneAndCheckout(repoURL, repoPath, sha string) error {
	if err := os.RemoveAll(repoPath); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(repoPath), 0o755); err != nil {
		return err
	}

	cmd := exec.Command("git", "clone", repoURL, repoPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %w (%s)", err, string(output))
	}

	cmd = exec.Command("git", "checkout", sha)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout failed: %w (%s)", err, string(output))
	}

	return nil
}

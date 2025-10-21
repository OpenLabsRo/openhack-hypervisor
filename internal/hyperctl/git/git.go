package git

import (
	"os/exec"
)

// CloneOrPull clones the repository if it doesn't exist, or pulls if it does.
func CloneOrPull(repoURL, destDir string) error {
	// Check if destDir is a git repo
	if _, err := exec.Command("git", "-C", destDir, "rev-parse", "--git-dir").CombinedOutput(); err != nil {
		// Not a git repo, clone
		return exec.Command("git", "clone", repoURL, destDir).Run()
	}
	// Is a git repo, pull
	cmd := exec.Command("git", "pull")
	cmd.Dir = destDir
	return cmd.Run()
}

// Checkout checks out a specific branch, tag, or commit in the repository.
func Checkout(repoDir, ref string) error {
	cmd := exec.Command("git", "checkout", ref)
	cmd.Dir = repoDir
	return cmd.Run()
}

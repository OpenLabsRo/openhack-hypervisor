package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CloneOrPull clones a git repository from the given URL to the specified path.
// If the repository already exists at the given path, it will be pulled.
func CloneOrPull(repoURL, path string) error {
	clone := func() error {
		cmd := exec.Command("git", "clone", repoURL, path)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git clone failed: %w\n%s", err, output)
		}
		return ensureSafeDirectory(path)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return clone()
	} else if err != nil {
		return fmt.Errorf("failed to stat %s: %w", path, err)
	}

	if err := EnsureWorkTree(path); err != nil {
		return err
	}
	if err := EnsureRemote(path, repoURL); err != nil {
		return err
	}

	if err := FetchTags(path); err != nil {
		return err
	}
	if err := runGit(path, "pull", "--ff-only", "origin"); err != nil {
		return err
	}

	return ensureSafeDirectory(path)
}

func EnsureWorkTree(path string) error {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err == nil {
		return ensureSafeDirectory(path)
	}

	if err := runGit(path, "init"); err != nil {
		return fmt.Errorf("failed to initialise git repository at %s: %w", path, err)
	}
	return ensureSafeDirectory(path)
}

func EnsureRemote(path, repoURL string) error {
	cmd := exec.Command("git", "-C", path, "remote", "get-url", "origin")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return runGit(path, "remote", "add", "origin", repoURL)
	}

	current := strings.TrimSpace(string(output))
	if current == repoURL {
		return nil
	}

	return runGit(path, "remote", "set-url", "origin", repoURL)
}

func runGit(path string, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", path}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// FetchTags updates tags and remote references for the repository at path.
func FetchTags(path string) error {
	return runGit(path, "fetch", "--prune", "--tags", "origin")
}

// CheckoutTag ensures the worktree at path is positioned at the requested tag.
func CheckoutTag(path, tag string) error {
	if err := runGit(path, "checkout", "--force", tag); err != nil {
		return fmt.Errorf("git checkout %s failed: %w", tag, err)
	}

	if err := runGit(path, "reset", "--hard", tag); err != nil {
		return fmt.Errorf("git reset --hard %s failed: %w", tag, err)
	}

	return nil
}

// GetTags returns a list of all tags in the git repository at the given path.
func GetTags(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "-C", repoPath, "tag")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	trimmedOutput := strings.TrimSpace(string(output))
	if trimmedOutput == "" {
		return []string{}, nil
	}

	tags := strings.Split(trimmedOutput, "\n")
	return tags, nil
}

func CloneTag(repoURL, tag, path string) error {
	cmd := exec.Command("git", "clone", "--branch", tag, repoURL, path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %w\n%s", err, output)
	}
	return ensureSafeDirectory(path)
}

func ensureSafeDirectory(path string) error {
	clean := filepath.Clean(path)
	cmd := exec.Command("git", "config", "--global", "--add", "safe.directory", clean)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to mark %s as a safe git directory: %w", clean, err)
	}
	return nil
}

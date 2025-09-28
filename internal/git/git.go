package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CloneOrPull clones a git repository from the given URL to the specified path.
// If the repository already exists at the given path, it will be pulled.
func CloneOrPull(repoURL, path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cmd := exec.Command("git", "clone", repoURL, path)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git clone failed: %w\n%s", err, output)
		}
		return nil
	}

	// Path exists, check if it's a git repository
	cmd := exec.Command("git", "-C", path, "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("path %s exists but is not a git repository", path)
	}

	// It's a git repository, so pull
	cmd = exec.Command("git", "-C", path, "pull")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %w\n%s", err, output)
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
    return nil
}
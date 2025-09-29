package git

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const defaultGitHome = "/var/openhack"

// LatestRemoteTag returns the newest tag (by semantic version) and its commit hash.
// Tags are expected to follow the format prefixYY.MM.DD.B (e.g., v24.09.24.0).
func LatestRemoteTag(repoURL, prefix string) (tag, commit string, err error) {
	tags, err := listRemoteTags(repoURL, prefix)
	if err != nil {
		return "", "", err
	}

	if len(tags) == 0 {
		return "", "", fmt.Errorf("no tags with prefix %s found", prefix)
	}

	sort.Slice(tags, func(i, j int) bool {
		return compareVersions(tags[i].versionParts, tags[j].versionParts) > 0
	})

	return tags[0].name, tags[0].commit, nil
}

// TagCommit resolves the commit hash for the provided tag.
func TagCommit(repoURL, tag string) (string, error) {
	tags, err := listRemoteTags(repoURL, "")
	if err != nil {
		return "", err
	}

	for _, t := range tags {
		if t.name == tag {
			return t.commit, nil
		}
	}

	return "", fmt.Errorf("tag %s not found", tag)
}

// EnsureRepoAtTag ensures that destDir contains the repository checked out at the given tag.
// It returns the HEAD commit hash after ensuring the checkout.
func EnsureRepoAtTag(repoURL, destDir, tag string) (string, error) {
	if RepoExists(destDir) {
		cmd := exec.Command("git", "-C", destDir, "fetch", "--tags", "origin")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := applyGitEnv(cmd); err != nil {
			return "", err
		}
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("git fetch failed: %w", err)
		}
		cmd = exec.Command("git", "-C", destDir, "checkout", tag)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := applyGitEnv(cmd); err != nil {
			return "", err
		}
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("git checkout %s failed: %w", tag, err)
		}
		cmd = exec.Command("git", "-C", destDir, "reset", "--hard", tag)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := applyGitEnv(cmd); err != nil {
			return "", err
		}
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("git reset --hard %s failed: %w", tag, err)
		}
	} else {
		if err := os.RemoveAll(destDir); err != nil {
			return "", fmt.Errorf("failed to clean repo dir: %w", err)
		}
		cmd := exec.Command("git", "clone", "--quiet", "--branch", tag, "--single-branch", repoURL, destDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := applyGitEnv(cmd); err != nil {
			return "", err
		}
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("git clone failed: %w", err)
		}
	}

	if err := ensureSafeDirectory(destDir); err != nil {
		return "", err
	}

	commit, err := HeadCommit(destDir)
	if err != nil {
		return "", err
	}

	return commit, nil
}

// HeadCommit returns the HEAD commit hash for the repository at path.
func HeadCommit(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD failed: %w", err)
	}

	hash := strings.TrimSpace(string(output))
	if hash == "" {
		return "", errors.New("empty HEAD hash")
	}

	return hash, nil
}

// RepoExists checks whether destDir already contains a git repository.
func RepoExists(destDir string) bool {
	gitDir := filepath.Join(destDir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return true
	}
	return false
}

type tagInfo struct {
	name         string
	commit       string
	versionParts []int
}

func listRemoteTags(repoURL, prefix string) ([]tagInfo, error) {
	cmd := exec.Command("git", "ls-remote", "--tags", repoURL)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-remote failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	tagMap := make(map[string]*tagInfo)

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		hash := fields[0]
		ref := fields[1]

		if !strings.HasPrefix(ref, "refs/tags/") {
			continue
		}

		tagName := strings.TrimPrefix(ref, "refs/tags/")
		isDereferenced := strings.HasSuffix(tagName, "^{}")
		tagName = strings.TrimSuffix(tagName, "^{}")

		if prefix != "" && !strings.HasPrefix(tagName, prefix) {
			continue
		}

		info, ok := tagMap[tagName]
		if !ok {
			versionStr := strings.TrimPrefix(tagName, prefix)
			parts, err := parseVersion(versionStr)
			if err != nil {
				continue
			}
			info = &tagInfo{name: tagName, versionParts: parts}
			tagMap[tagName] = info
		}

		if isDereferenced || info.commit == "" {
			info.commit = hash
		}
	}

	tags := make([]tagInfo, 0, len(tagMap))
	for _, info := range tagMap {
		if info.commit == "" {
			continue
		}
		tags = append(tags, *info)
	}

	return tags, nil
}

func parseVersion(v string) ([]int, error) {
	v = strings.TrimPrefix(v, "v")
	if v == "" {
		return nil, errors.New("empty version")
	}

	parts := strings.Split(v, ".")
	nums := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid version segment %q", p)
		}
		nums[i] = n
	}

	return nums, nil
}

func ensureSafeDirectory(path string) error {
	clean := filepath.Clean(path)
	cmd := exec.Command("git", "config", "--global", "--add", "safe.directory", clean)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add safe.directory for %s: %w", clean, err)
	}
	return nil
}

func compareVersions(a, b []int) int {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}

	for i := 0; i < maxLen; i++ {
		var va, vb int
		if i < len(a) {
			va = a[i]
		}
		if i < len(b) {
			vb = b[i]
		}

		if va > vb {
			return 1
		}
		if va < vb {
			return -1
		}
	}
	return 0
}

func applyGitEnv(cmd *exec.Cmd) error {
	home := strings.TrimSpace(os.Getenv("HOME"))
	if home == "" {
		home = defaultGitHome
	}
	if err := os.MkdirAll(home, 0o755); err != nil && !os.IsPermission(err) {
		return fmt.Errorf("failed to ensure HOME directory %s: %w", home, err)
	}
	cmd.Env = append(os.Environ(), fmt.Sprintf("HOME=%s", home))
	return nil
}

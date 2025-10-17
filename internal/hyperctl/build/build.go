package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Result captures output details from running the BUILD script.
type Result struct {
	Version    string
	BinaryPath string
}

// Run executes the repo's BUILD script targeting the provided output directory.
func Run(repoDir, outputDir string) (Result, error) {
	versionBytes, err := os.ReadFile(filepath.Join(repoDir, "VERSION"))
	if err != nil {
		return Result{}, fmt.Errorf("failed to read VERSION file: %w", err)
	}

	version := trimWhitespace(string(versionBytes))
	if version == "" {
		return Result{}, fmt.Errorf("VERSION file is empty")
	}

	binaryPath := filepath.Join(outputDir, version)

	if err := os.Remove(binaryPath); err != nil && !os.IsNotExist(err) {
		return Result{}, fmt.Errorf("failed to remove existing binary: %w", err)
	}

	buildScript := filepath.Join(repoDir, "BUILD")
	if _, err := os.Stat(buildScript); err != nil {
		return Result{}, fmt.Errorf("BUILD script not found: %w", err)
	}

	cmd := exec.Command("bash", buildScript, "--output", outputDir)
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = withEnvOverrides(os.Environ(), map[string]string{
		"GO111MODULE": "on",
		"GOWORK":      "off",
	})

	if err := cmd.Run(); err != nil {
		return Result{}, fmt.Errorf("BUILD script failed: %w", err)
	}

	if _, err := os.Stat(binaryPath); err != nil {
		return Result{}, fmt.Errorf("expected build artifact not found: %w", err)
	}

	return Result{
		Version:    version,
		BinaryPath: binaryPath,
	}, nil
}

func trimWhitespace(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\r", ""))
}

func withEnvOverrides(base []string, overrides map[string]string) []string {
	filtered := base[:0]
	for _, kv := range base {
		keep := true
		for key := range overrides {
			if strings.HasPrefix(kv, key+"=") {
				keep = false
				break
			}
		}
		if keep {
			filtered = append(filtered, kv)
		}
	}

	result := append([]string{}, filtered...)
	for key, val := range overrides {
		result = append(result, key+"="+val)
	}
	return result
}

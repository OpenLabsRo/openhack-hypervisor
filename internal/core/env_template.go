package core

import (
	"os"
	"path/filepath"

	"hypervisor/internal/paths"
)

func envTemplateDir() string {
	return paths.OpenHackEnvPath("template")
}

func envTemplatePath() string {
	return filepath.Join(envTemplateDir(), ".env")
}

// ReadEnvTemplate returns the contents of the OpenHack backend template .env file.
func ReadEnvTemplate() (string, error) {
	data, err := os.ReadFile(envTemplatePath())
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// WriteEnvTemplate persists the provided contents to the template .env file.
func WriteEnvTemplate(contents string) error {
	if err := os.MkdirAll(envTemplateDir(), 0o755); err != nil {
		return err
	}

	return os.WriteFile(envTemplatePath(), []byte(contents), 0o666)
}

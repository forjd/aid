package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const DefaultRepoConfig = `[output]
default_mode = "brief"

[indexing]
ignore_paths = ["vendor/", "node_modules/", "storage/"]

[agent]
skill_path = "skills/aid/SKILL.md"
`

func EnsureRepoConfig(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("stat repo config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("create repo config directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(DefaultRepoConfig), 0o644); err != nil {
		return false, fmt.Errorf("write repo config: %w", err)
	}

	return true, nil
}

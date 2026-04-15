//go:build unix

package config

import (
	"fmt"
	"os"
	"path/filepath"
)

func repairRepoConfigPermissions(path string) error {
	dir := filepath.Dir(path)
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("tighten repo config directory permissions: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("tighten repo config permissions: %w", err)
	}
	return nil
}

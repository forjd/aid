//go:build unix

package sqlite

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func repairDBPermissions(path string) error {
	dir := filepath.Dir(path)
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("tighten sqlite directory permissions: %w", err)
	}

	for _, candidate := range []string{path, path + "-wal", path + "-shm"} {
		if err := os.Chmod(candidate, 0o600); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("tighten sqlite file permissions: %w", err)
		}
	}

	return nil
}

//go:build unix

package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenAndMigrateTightenDatabasePermissionsOnUnix(t *testing.T) {
	ctx := context.Background()
	dataDir := filepath.Join(t.TempDir(), "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}

	dbPath := filepath.Join(dataDir, "aid.db")
	sqliteStore, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer sqliteStore.Close()

	if err := sqliteStore.Migrate(ctx); err != nil {
		t.Fatalf("migrate store: %v", err)
	}
	if _, err := sqliteStore.UpsertRepo(ctx, "/tmp/project", "project"); err != nil {
		t.Fatalf("upsert repo: %v", err)
	}

	assertPerm(t, dataDir, 0o700)
	assertPerm(t, dbPath, 0o600)
	assertPerm(t, dbPath+"-wal", 0o600)
	assertPerm(t, dbPath+"-shm", 0o600)
}

func assertPerm(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("unexpected permissions for %s: got %03o want %03o", path, got, want)
	}
}

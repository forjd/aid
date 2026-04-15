//go:build unix

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureRepoConfigAppliesPrivatePermissionsOnUnix(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".aid", "config.toml")

	created, err := EnsureRepoConfig(path)
	if err != nil {
		t.Fatalf("ensure repo config: %v", err)
	}
	if !created {
		t.Fatalf("expected config to be created")
	}

	assertPerm(t, filepath.Dir(path), 0o700)
	assertPerm(t, path, 0o600)
}

func TestEnsureRepoConfigRepairsExistingPermissionsOnUnix(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".aid", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(DefaultRepoConfig), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.Chmod(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("chmod config dir: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod config file: %v", err)
	}

	created, err := EnsureRepoConfig(path)
	if err != nil {
		t.Fatalf("ensure existing repo config: %v", err)
	}
	if created {
		t.Fatalf("expected existing config to stay in place")
	}

	assertPerm(t, filepath.Dir(path), 0o700)
	assertPerm(t, path, 0o600)
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

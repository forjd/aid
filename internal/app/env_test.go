package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverUsesRepoContextAndDataDirOverride(t *testing.T) {
	repoDir := initRepo(t)
	writeFile(t, filepath.Join(repoDir, "README.md"), []byte("hello\n"))
	runGit(t, repoDir, "add", "README.md")
	runGitWithIdentity(t, repoDir, "commit", "-m", "feat: initial commit")

	customDataDir := filepath.Join(t.TempDir(), "aid-data")
	t.Setenv("AID_DATA_DIR", customDataDir)

	env, err := Discover(repoDir)
	if err != nil {
		t.Fatalf("discover environment: %v", err)
	}

	resolvedRepoDir, err := filepath.EvalSymlinks(repoDir)
	if err != nil {
		t.Fatalf("resolve temp repo symlink: %v", err)
	}

	if env.WorkingDir != repoDir {
		t.Fatalf("unexpected working dir: %q", env.WorkingDir)
	}
	if env.RepoRoot != resolvedRepoDir {
		t.Fatalf("unexpected repo root: %q", env.RepoRoot)
	}
	if env.RepoName != filepath.Base(resolvedRepoDir) {
		t.Fatalf("unexpected repo name: %q", env.RepoName)
	}
	if strings.TrimSpace(env.Branch) == "" || env.Branch == "detached" {
		t.Fatalf("unexpected branch: %q", env.Branch)
	}
	if env.AppDataDir != customDataDir {
		t.Fatalf("unexpected app data dir: %q", env.AppDataDir)
	}
	if env.DBPath != filepath.Join(customDataDir, "aid.db") {
		t.Fatalf("unexpected db path: %q", env.DBPath)
	}
	if env.RepoConfigDir != filepath.Join(resolvedRepoDir, ".aid") {
		t.Fatalf("unexpected repo config dir: %q", env.RepoConfigDir)
	}
	if env.RepoConfigPath != filepath.Join(resolvedRepoDir, ".aid", "config.toml") {
		t.Fatalf("unexpected repo config path: %q", env.RepoConfigPath)
	}
}

func TestDiscoverFailsOutsideRepository(t *testing.T) {
	_, err := Discover(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "resolve git repository root") {
		t.Fatalf("expected outside-repo error, got %v", err)
	}
}

func initRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	runGit(t, repoDir, "init", "-q")
	return repoDir
}

func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()

	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func runGit(t *testing.T, repoDir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", append([]string{"-C", repoDir}, args...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
}

func runGitWithIdentity(t *testing.T, repoDir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", append([]string{"-C", repoDir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
}

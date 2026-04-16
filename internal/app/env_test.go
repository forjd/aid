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

func TestDiscoverResolvesRelativeAIDDataDir(t *testing.T) {
	repoDir := initRepo(t)

	workDir := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir to work dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})

	t.Setenv("AID_DATA_DIR", "relative-aid-data")

	env, err := Discover(repoDir)
	if err != nil {
		t.Fatalf("discover with relative AID_DATA_DIR: %v", err)
	}

	if !filepath.IsAbs(env.AppDataDir) {
		t.Fatalf("expected absolute app data dir, got %q", env.AppDataDir)
	}
	if !filepath.IsAbs(env.DBPath) {
		t.Fatalf("expected absolute db path, got %q", env.DBPath)
	}
	resolvedWorkDir, err := filepath.EvalSymlinks(workDir)
	if err != nil {
		t.Fatalf("resolve work dir symlink: %v", err)
	}
	resolvedAppDataDir, err := filepath.EvalSymlinks(env.AppDataDir)
	if err != nil {
		// EvalSymlinks fails for missing paths. Fall back to comparing raw.
		resolvedAppDataDir = env.AppDataDir
	}
	if !strings.HasPrefix(resolvedAppDataDir, resolvedWorkDir) {
		t.Fatalf("expected app data dir %q to be inside work dir %q", resolvedAppDataDir, resolvedWorkDir)
	}
}

func TestResolveDataDirRejectsEmpty(t *testing.T) {
	if _, err := resolveDataDir(""); err == nil {
		t.Fatalf("expected empty path rejection")
	}
}

func TestResolveDataDirProducesAbsoluteCleanPath(t *testing.T) {
	tmp := t.TempDir()
	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir tmp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(previousWD) })

	got, err := resolveDataDir("./sub/../data")
	if err != nil {
		t.Fatalf("resolve data dir: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("expected absolute path, got %q", got)
	}
	if filepath.Base(got) != "data" {
		t.Fatalf("expected clean data dir, got %q", got)
	}
}

func TestRawDataDirHonoursOverride(t *testing.T) {
	t.Setenv("AID_DATA_DIR", "/custom/path")
	raw, err := rawDataDir()
	if err != nil {
		t.Fatalf("raw data dir: %v", err)
	}
	if raw != "/custom/path" {
		t.Fatalf("unexpected raw override: %q", raw)
	}
}

func TestRawDataDirReturnsDefaultWhenOverrideMissing(t *testing.T) {
	t.Setenv("AID_DATA_DIR", "")
	raw, err := rawDataDir()
	if err != nil {
		t.Fatalf("raw data dir: %v", err)
	}
	if raw == "" {
		t.Fatalf("expected default data dir, got empty string")
	}
	if !filepath.IsAbs(raw) {
		t.Fatalf("expected absolute default path, got %q", raw)
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

package git

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootBranchAndStatus(t *testing.T) {
	repoDir := initGitRepo(t)
	writeTrackedFile(t, repoDir, "README.md", "hello\n")
	runGit(t, repoDir, "add", "README.md")
	runGitWithIdentity(t, repoDir, "commit", "-m", "feat: initial commit")

	nested := filepath.Join(repoDir, "nested", "dir")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested dir: %v", err)
	}

	root, err := Root(nested)
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("resolve repo root symlink: %v", err)
	}
	resolvedRepoDir, err := filepath.EvalSymlinks(repoDir)
	if err != nil {
		t.Fatalf("resolve temp repo symlink: %v", err)
	}
	if resolvedRoot != resolvedRepoDir {
		t.Fatalf("unexpected repo root: %q", root)
	}

	branch, err := Branch(repoDir)
	if err != nil {
		t.Fatalf("resolve branch: %v", err)
	}
	if strings.TrimSpace(branch) == "" || branch == "detached" {
		t.Fatalf("expected named branch, got %q", branch)
	}

	status, err := Status(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("status on clean repo: %v", err)
	}
	if status.Dirty || status.Changed != 0 || status.Untracked != 0 {
		t.Fatalf("expected clean status, got %#v", status)
	}

	writeTrackedFile(t, repoDir, "README.md", "updated\n")
	writeTrackedFile(t, repoDir, "notes.txt", "untracked\n")

	status, err = Status(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("status on dirty repo: %v", err)
	}
	if !status.Dirty || status.Changed != 1 || status.Untracked != 1 {
		t.Fatalf("unexpected dirty status: %#v", status)
	}

	headSHA := gitOutput(t, repoDir, "rev-parse", "HEAD")
	runGit(t, repoDir, "checkout", "--detach", headSHA)

	branch, err = Branch(repoDir)
	if err != nil {
		t.Fatalf("resolve branch in detached head: %v", err)
	}
	if branch != "detached" {
		t.Fatalf("expected detached branch, got %q", branch)
	}
}

func TestRootErrorsOutsideRepository(t *testing.T) {
	_, err := Root(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "resolve git repository root") {
		t.Fatalf("expected repo root error, got %v", err)
	}
}

func TestCommitsAndCommitQueries(t *testing.T) {
	repoDir := initGitRepo(t)

	writeTrackedFile(t, repoDir, "one.txt", "one\n")
	runGit(t, repoDir, "add", "one.txt")
	runGitWithIdentity(t, repoDir, "commit", "-m", "feat: add one")
	shaOne := gitOutput(t, repoDir, "rev-parse", "HEAD")

	writeTrackedFile(t, repoDir, "two.txt", "two\n")
	runGit(t, repoDir, "add", "two.txt")
	runGitWithIdentity(t, repoDir, "commit", "-m", "fix: add two")
	shaTwo := gitOutput(t, repoDir, "rev-parse", "HEAD")

	commits, err := Commits(context.Background(), repoDir, 2)
	if err != nil {
		t.Fatalf("list commits: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected two commits, got %#v", commits)
	}
	if commits[0].SHA != shaTwo || commits[0].Summary != "fix: add two" {
		t.Fatalf("unexpected newest commit: %#v", commits[0])
	}
	if len(commits[0].ChangedPaths) != 1 || commits[0].ChangedPaths[0] != "two.txt" {
		t.Fatalf("unexpected newest changed paths: %#v", commits[0].ChangedPaths)
	}

	recent, err := RecentCommits(context.Background(), repoDir, 1)
	if err != nil {
		t.Fatalf("recent commits: %v", err)
	}
	if len(recent) != 1 || recent[0].SHA != shaTwo {
		t.Fatalf("unexpected recent commits: %#v", recent)
	}

	shas, err := AllCommitSHAs(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("all commit shas: %v", err)
	}
	if len(shas) != 2 || shas[0] != shaTwo || shas[1] != shaOne {
		t.Fatalf("unexpected shas: %#v", shas)
	}

	bySHA, err := CommitsBySHA(context.Background(), repoDir, []string{shaOne, shaTwo})
	if err != nil {
		t.Fatalf("commits by sha: %v", err)
	}
	if len(bySHA) != 2 {
		t.Fatalf("expected two commits by sha, got %#v", bySHA)
	}
	found := map[string]Commit{}
	for _, commit := range bySHA {
		found[commit.SHA] = commit
	}
	if found[shaOne].Summary != "feat: add one" || found[shaTwo].Summary != "fix: add two" {
		t.Fatalf("unexpected commits by sha: %#v", found)
	}
}

func TestNoCommitRepositoriesReturnEmptyResults(t *testing.T) {
	repoDir := initGitRepo(t)

	commits, err := Commits(context.Background(), repoDir, 10)
	if err != nil {
		t.Fatalf("list commits in empty repo: %v", err)
	}
	if commits != nil {
		t.Fatalf("expected nil commits for empty repo, got %#v", commits)
	}

	shas, err := AllCommitSHAs(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("list shas in empty repo: %v", err)
	}
	if shas != nil {
		t.Fatalf("expected nil shas for empty repo, got %#v", shas)
	}

	bySHA, err := CommitsBySHA(context.Background(), repoDir, nil)
	if err != nil {
		t.Fatalf("commits by empty sha list: %v", err)
	}
	if bySHA != nil {
		t.Fatalf("expected nil commits for empty sha list, got %#v", bySHA)
	}
}

func TestParseCommitsRejectsInvalidInput(t *testing.T) {
	if _, err := parseCommits([]byte("path/without/header.go"), 1); err == nil {
		t.Fatalf("expected path without header error")
	}
	if _, err := parseCommits([]byte("\x00bad"), 1); err == nil {
		t.Fatalf("expected malformed header error")
	}
	if _, err := parseCommits([]byte("\x00abc\x00Dan\x00not-a-time\x00summary\x00body"), 1); err == nil {
		t.Fatalf("expected invalid time error")
	}
}

func TestParseCommitsHandlesNULTokensAndPathsWithSpaces(t *testing.T) {
	output := bytes.Join([][]byte{
		{},
		[]byte("abc123"),
		[]byte("Dan"),
		[]byte("2026-04-15T12:00:00Z"),
		[]byte("fix: retry path"),
		[]byte("body paragraph"),
		[]byte("\ninternal/app/env.go"),
		[]byte("\ntwo words.txt"),
	}, []byte{0})

	commits, err := parseCommits(output, 1)
	if err != nil {
		t.Fatalf("parse commits: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected one commit, got %#v", commits)
	}
	if len(commits[0].ChangedPaths) != 2 || commits[0].ChangedPaths[1] != "two words.txt" {
		t.Fatalf("unexpected changed paths: %#v", commits[0].ChangedPaths)
	}
	if commits[0].Message != "fix: retry path\n\nbody paragraph" {
		t.Fatalf("unexpected combined message: %q", commits[0].Message)
	}
	if commits[0].Summary != "fix: retry path" {
		t.Fatalf("unexpected summary: %q", commits[0].Summary)
	}
}

func TestCommitsCaptureBody(t *testing.T) {
	repoDir := initGitRepo(t)
	writeTrackedFile(t, repoDir, "auth.txt", "refresh\n")
	runGit(t, repoDir, "add", "auth.txt")

	message := "feat: token refresh\n\nAdds session fingerprint so refresh retry can detect replay."
	cmd := exec.Command("git", "-C", repoDir, "-c", "user.name=Test User", "-c", "user.email=test@example.com", "commit", "-m", message)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit with body: %v\n%s", err, out)
	}

	commits, err := Commits(context.Background(), repoDir, 1)
	if err != nil {
		t.Fatalf("list commits: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected one commit, got %#v", commits)
	}
	if commits[0].Summary != "feat: token refresh" {
		t.Fatalf("unexpected summary: %q", commits[0].Summary)
	}
	if !strings.Contains(commits[0].Message, "session fingerprint") {
		t.Fatalf("expected message to include body, got %q", commits[0].Message)
	}
}

func TestRunGitOutputReturnsExitMessage(t *testing.T) {
	_, err := runGitOutput(context.Background(),"-C", t.TempDir(), "not-a-real-git-command")
	if err == nil || !strings.Contains(err.Error(), "not-a-real-git-command") {
		t.Fatalf("expected git error message, got %v", err)
	}
}

func initGitRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	runGit(t, repoDir, "init", "-q")
	return repoDir
}

func writeTrackedFile(t *testing.T, repoDir, name, content string) {
	t.Helper()

	path := filepath.Join(repoDir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func gitOutput(t *testing.T, repoDir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", append([]string{"-C", repoDir}, args...)...)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(output))
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

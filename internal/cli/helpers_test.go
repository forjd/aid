package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forjd/aid/internal/output"
	"github.com/forjd/aid/internal/store"
)

func TestLookupHelpTarget(t *testing.T) {
	root := rootCommand()

	target, err := lookupHelpTarget(root, []string{"task", "done"})
	if err != nil {
		t.Fatalf("lookup help target: %v", err)
	}
	if target.Path != "aid task done" {
		t.Fatalf("unexpected target path: %q", target.Path)
	}

	target, err = lookupHelpTarget(root, []string{"task", "--help"})
	if err != nil {
		t.Fatalf("lookup help target with flag: %v", err)
	}
	if target.Path != "aid task" {
		t.Fatalf("unexpected help target for flag: %q", target.Path)
	}

	if _, err := lookupHelpTarget(root, []string{"missing"}); err == nil {
		t.Fatalf("expected unknown command error")
	}
}

func TestInferCommandPath(t *testing.T) {
	root := rootCommand()

	got := inferCommandPath(root, []string{"help", "task", "done", "task_1"})
	if got != "aid task done" {
		t.Fatalf("unexpected inferred path: %q", got)
	}

	got = inferCommandPath(root, []string{"note", "missing"})
	if got != "aid note" {
		t.Fatalf("unexpected inferred path for partial command: %q", got)
	}
}

func TestRunWritesJSONErrorPayload(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"note", "missing", "--json"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected non-zero exit code, got %d", exitCode)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr for json errors, got %q", stderr.String())
	}

	var payload struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Error   struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal json error payload: %v\n%s", err, stdout.String())
	}
	if payload.OK || payload.Command != "note" || !strings.Contains(payload.Error.Message, `unknown command "missing"`) {
		t.Fatalf("unexpected json error payload: %#v", payload)
	}
}

func TestRunWritesHumanErrorToStderr(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"--brief", "--verbose"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected non-zero exit code, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout for human error, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "cannot combine --verbose with --json or --brief") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunWithoutArgsAndHelpSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if exitCode := Run(nil, &stdout, &stderr); exitCode != 0 {
		t.Fatalf("expected success for empty args, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "aid - local memory for coding agents and repos") {
		t.Fatalf("expected root help output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()

	if exitCode := Run([]string{"help", "task", "done"}, &stdout, &stderr); exitCode != 0 {
		t.Fatalf("expected success for help subcommand, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "aid task done - Mark a task as done") {
		t.Fatalf("expected leaf help output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr for help subcommand, got %q", stderr.String())
	}
}

func TestParseGlobalOptions(t *testing.T) {
	opts, filtered, err := parseGlobalOptions([]string{"--repo", "/tmp/repo", "note", "list"})
	if err != nil {
		t.Fatalf("parse global options: %v", err)
	}
	if opts.RepoPath != "/tmp/repo" {
		t.Fatalf("unexpected repo path: %q", opts.RepoPath)
	}
	if len(filtered) != 2 || filtered[0] != "note" || filtered[1] != "list" {
		t.Fatalf("unexpected filtered args: %#v", filtered)
	}

	opts, filtered, err = parseGlobalOptions([]string{"--repo=/tmp/repo", "--verbose", "status"})
	if err != nil {
		t.Fatalf("parse global options with equals: %v", err)
	}
	if opts.RepoPath != "/tmp/repo" || !opts.IsVerbose() {
		t.Fatalf("unexpected parsed options: %#v", opts)
	}
	if len(filtered) != 1 || filtered[0] != "status" {
		t.Fatalf("unexpected filtered args: %#v", filtered)
	}

	if _, _, err := parseGlobalOptions([]string{"--repo"}); err == nil {
		t.Fatalf("expected missing repo value error")
	}
}

func TestApplyConfiguredDefaultsUsesRepoConfig(t *testing.T) {
	repoDir := t.TempDir()
	runGit(t, repoDir, "init", "-q")
	if err := os.MkdirAll(filepath.Join(repoDir, ".aid"), 0o755); err != nil {
		t.Fatalf("mkdir .aid: %v", err)
	}

	writeFile(t, filepath.Join(repoDir, ".aid", "config.toml"), []byte(`[output]
default_mode = "verbose"

[indexing]
ignore_paths = ["vendor/"]

[agent]
skill_path = ".agents/skills/aid/SKILL.md"
`))

	opts, err := applyConfiguredDefaults(output.Options{Format: output.FormatHuman, RepoPath: repoDir}, []string{"status"})
	if err != nil {
		t.Fatalf("apply configured defaults: %v", err)
	}
	if !opts.IsVerbose() {
		t.Fatalf("expected verbose default, got %#v", opts)
	}
}

func TestApplyConfiguredDefaultsRejectsInvalidMode(t *testing.T) {
	repoDir := t.TempDir()
	runGit(t, repoDir, "init", "-q")
	if err := os.MkdirAll(filepath.Join(repoDir, ".aid"), 0o755); err != nil {
		t.Fatalf("mkdir .aid: %v", err)
	}

	writeFile(t, filepath.Join(repoDir, ".aid", "config.toml"), []byte(`[output]
default_mode = "loud"
`))

	_, err := applyConfiguredDefaults(output.Options{Format: output.FormatHuman, RepoPath: repoDir}, []string{"status"})
	if err == nil || !strings.Contains(err.Error(), `invalid output.default_mode "loud"`) {
		t.Fatalf("expected invalid default mode error, got %v", err)
	}
}

func TestStubCommandWritesScaffoldMessage(t *testing.T) {
	cmd := stubCommand("sync", "aid sync", "sync", "aid sync", "Sync state")

	var out bytes.Buffer
	if err := cmd.Run(nil, Streams{Out: &out}); err != nil {
		t.Fatalf("run stub command: %v", err)
	}
	if out.String() != "aid sync is scaffolded but not implemented yet.\n" {
		t.Fatalf("unexpected stub output: %q", out.String())
	}
}

func TestJoinArgs(t *testing.T) {
	text, err := joinArgs([]string{"refresh", "retry"}, "query")
	if err != nil {
		t.Fatalf("join args: %v", err)
	}
	if text != "refresh retry" {
		t.Fatalf("unexpected joined args: %q", text)
	}
	if _, err := joinArgs([]string{" ", "\t"}, "query"); err == nil {
		t.Fatalf("expected missing query error")
	}
}

func TestRecentContextCommitsFallsBackToLiveGit(t *testing.T) {
	repoDir := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), "aid-data")

	runGit(t, repoDir, "init", "-q")
	writeFile(t, filepath.Join(repoDir, "README.md"), []byte("hello\n"))
	runGit(t, repoDir, "add", "README.md")
	runGitWithIdentity(t, repoDir, "commit", "-m", "feat: live fallback")

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("chdir to repo: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})
	t.Setenv("AID_DATA_DIR", dataDir)

	_ = runCLI(t, "init")
	runtime, err := openInitializedRepo(t.Context(), Streams{})
	if err != nil {
		t.Fatalf("open initialized repo: %v", err)
	}
	defer runtime.close()

	commits, err := recentContextCommits(t.Context(), runtime, 1)
	if err != nil {
		t.Fatalf("recent context commits: %v", err)
	}
	if len(commits) != 1 || commits[0].Summary != "feat: live fallback" {
		t.Fatalf("unexpected fallback commits: %#v", commits)
	}
}

func TestOpenInitializedRepoRequiresInit(t *testing.T) {
	repoDir := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), "aid-data")

	runGit(t, repoDir, "init", "-q")

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("chdir to repo: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})
	t.Setenv("AID_DATA_DIR", dataDir)

	_, err = openInitializedRepo(t.Context(), Streams{})
	if err == nil || !strings.Contains(err.Error(), `repo not initialised; run "aid init" first`) {
		t.Fatalf("expected init error, got %v", err)
	}
}

func TestTaskCommandHelpers(t *testing.T) {
	cases := []struct {
		status store.TaskStatus
		name   string
		verb   string
	}{
		{status: store.TaskInProgress, name: "start", verb: "Started"},
		{status: store.TaskBlocked, name: "block", verb: "Blocked"},
		{status: store.TaskOpen, name: "reopen", verb: "Reopened"},
		{status: store.TaskDone, name: "done", verb: "Completed"},
	}

	for _, tc := range cases {
		if got := taskCommandName(tc.status); got != tc.name {
			t.Fatalf("unexpected task command name for %q: %q", tc.status, got)
		}
		if got := taskStatusVerb(tc.status); got != tc.verb {
			t.Fatalf("unexpected task status verb for %q: %q", tc.status, got)
		}
	}
}

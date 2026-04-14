package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"--help"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected success, got exit code %d", exitCode)
	}

	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	output := stdout.String()
	for _, want := range []string{
		"aid - local memory for coding agents and repos",
		"Usage:",
		"aid <command> [options]",
		"init",
		"note",
		"history",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected help output to contain %q\n\n%s", want, output)
		}
	}
}

func TestSubcommandHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"note", "--help"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected success, got exit code %d", exitCode)
	}

	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	output := stdout.String()
	for _, want := range []string{
		"aid note - add and inspect repo-scoped notes",
		"aid note <command>",
		"add <text>",
		"list",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected subcommand help to contain %q\n\n%s", want, output)
		}
	}
}

func TestLeafCommandAllowsHelpAsArgument(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"recall", "help"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected success, got exit code %d", exitCode)
	}

	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "aid recall is scaffolded but not implemented yet.") {
		t.Fatalf("expected scaffold message, got %q", output)
	}
}

func TestInitAndCrudFlow(t *testing.T) {
	repoDir := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), "aid-data")

	runGit(t, repoDir, "init", "-q")

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("chdir to temp repo: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})
	t.Setenv("AID_DATA_DIR", dataDir)

	initOut := runCLI(t, "init")
	if !strings.Contains(initOut, "Initialised aid for repo") {
		t.Fatalf("expected init output, got %q", initOut)
	}

	noteOut := runCLI(t, "note", "add", "Refresh token bug occurs after 401 retry")
	if !strings.Contains(noteOut, "Added note note_1") {
		t.Fatalf("expected note output, got %q", noteOut)
	}

	noteListOut := runCLI(t, "note", "list")
	if !strings.Contains(noteListOut, "Refresh token bug occurs after 401 retry") {
		t.Fatalf("expected note list output, got %q", noteListOut)
	}

	taskOut := runCLI(t, "task", "add", "Fix VAT rounding on invoice lines")
	if !strings.Contains(taskOut, "Added task task_1") {
		t.Fatalf("expected task output, got %q", taskOut)
	}

	taskDoneOut := runCLI(t, "task", "done", "task_1")
	if !strings.Contains(taskDoneOut, "Completed task task_1") {
		t.Fatalf("expected task done output, got %q", taskDoneOut)
	}

	decisionOut := runCLI(t, "decide", "add", "Store all monetary values as integer pence")
	if !strings.Contains(decisionOut, "Added decision decision_1") {
		t.Fatalf("expected decision output, got %q", decisionOut)
	}

	decisionListOut := runCLI(t, "decide", "list")
	if !strings.Contains(decisionListOut, "Store all monetary values as integer pence") {
		t.Fatalf("expected decision list output, got %q", decisionListOut)
	}

	if _, err := os.Stat(filepath.Join(repoDir, ".aid", "config.toml")); err != nil {
		t.Fatalf("expected repo config to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "aid.db")); err != nil {
		t.Fatalf("expected sqlite database to exist: %v", err)
	}
}

func TestStatusSupportsBriefAndJSON(t *testing.T) {
	repoDir := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), "aid-data")

	runGit(t, repoDir, "init", "-q")

	previousWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("chdir to temp repo: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousWD)
	})
	t.Setenv("AID_DATA_DIR", dataDir)

	beforeInit := runCLI(t, "status", "--brief")
	if !strings.Contains(beforeInit, "Aid: not initialised") {
		t.Fatalf("expected uninitialised brief status, got %q", beforeInit)
	}

	_ = runCLI(t, "init")
	_ = runCLI(t, "note", "add", "Token refresh bug")
	_ = runCLI(t, "task", "add", "Fix token refresh")
	_ = runCLI(t, "decide", "add", "Store money as integer pence")

	briefStatus := runCLI(t, "status", "--brief")
	if !strings.Contains(briefStatus, "Aid: initialised") {
		t.Fatalf("expected initialised brief status, got %q", briefStatus)
	}
	if !strings.Contains(briefStatus, "Notes: 1") {
		t.Fatalf("expected brief note count, got %q", briefStatus)
	}

	jsonStatus := runCLI(t, "status", "--json")
	var payload struct {
		SchemaVersion string `json:"schema_version"`
		OK            bool   `json:"ok"`
		Command       string `json:"command"`
		Data          struct {
			Initialized bool `json:"initialized"`
			Counts      *struct {
				Notes     int `json:"notes"`
				Decisions int `json:"decisions"`
				Tasks     struct {
					Open int `json:"open"`
				} `json:"tasks"`
			} `json:"counts"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(jsonStatus), &payload); err != nil {
		t.Fatalf("unmarshal status json: %v\n%s", err, jsonStatus)
	}
	if !payload.OK || payload.Command != "status" || !payload.Data.Initialized {
		t.Fatalf("unexpected status json payload: %#v", payload)
	}
	if payload.Data.Counts == nil || payload.Data.Counts.Notes != 1 || payload.Data.Counts.Decisions != 1 || payload.Data.Counts.Tasks.Open != 1 {
		t.Fatalf("unexpected status counts: %#v", payload.Data.Counts)
	}
}

func TestGlobalRepoFlagAndJSONListOutput(t *testing.T) {
	repoDir := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), "aid-data")
	workDir := t.TempDir()

	runGit(t, repoDir, "init", "-q")

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
	t.Setenv("AID_DATA_DIR", dataDir)

	_ = runCLI(t, "--repo", repoDir, "init")
	_ = runCLI(t, "note", "add", "Cross-directory note", "--repo", repoDir)

	jsonNotes := runCLI(t, "--repo", repoDir, "note", "list", "--json")
	var payload struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Data    struct {
			Notes []struct {
				ID   string `json:"id"`
				Text string `json:"text"`
			} `json:"notes"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(jsonNotes), &payload); err != nil {
		t.Fatalf("unmarshal notes json: %v\n%s", err, jsonNotes)
	}
	if !payload.OK || payload.Command != "note list" {
		t.Fatalf("unexpected notes payload: %#v", payload)
	}
	if len(payload.Data.Notes) != 1 || payload.Data.Notes[0].Text != "Cross-directory note" {
		t.Fatalf("unexpected notes data: %#v", payload.Data.Notes)
	}
}

func TestResumeOutputsWorkingSummary(t *testing.T) {
	repoDir := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), "aid-data")

	runGit(t, repoDir, "init", "-q")
	writeFile(t, filepath.Join(repoDir, "README.md"), []byte("hello\n"))
	runGit(t, repoDir, "add", "README.md")
	runGitWithIdentity(t, repoDir, "commit", "-m", "feat: initial repo memory support")

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
	_ = runCLI(t, "note", "add", "Token refresh bug occurs after 401 retry")
	_ = runCLI(t, "task", "add", "Fix token refresh retry path")
	_ = runCLI(t, "decide", "add", "Store money as integer pence")

	briefResume := runCLI(t, "resume", "--brief")
	for _, want := range []string{
		"Branch: main",
		"Task: Fix token refresh retry path",
		"Token refresh bug occurs after 401 retry",
		"Store money as integer pence",
		"feat: initial repo memory support",
	} {
		if !strings.Contains(briefResume, want) {
			t.Fatalf("expected resume output to contain %q\n\n%s", want, briefResume)
		}
	}

	jsonResume := runCLI(t, "resume", "--json")
	var payload struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Data    struct {
			ActiveTask struct {
				Text string `json:"text"`
			} `json:"active_task"`
			ActiveTaskInferred bool `json:"active_task_inferred"`
			Notes              []struct {
				Text string `json:"text"`
			} `json:"notes"`
			RecentCommits []struct {
				Summary string `json:"summary"`
			} `json:"recent_commits"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(jsonResume), &payload); err != nil {
		t.Fatalf("unmarshal resume json: %v\n%s", err, jsonResume)
	}
	if !payload.OK || payload.Command != "resume" || !payload.Data.ActiveTaskInferred {
		t.Fatalf("unexpected resume payload: %#v", payload)
	}
	if payload.Data.ActiveTask.Text != "Fix token refresh retry path" {
		t.Fatalf("unexpected active task: %#v", payload.Data.ActiveTask)
	}
	if len(payload.Data.Notes) == 0 || payload.Data.Notes[0].Text != "Token refresh bug occurs after 401 retry" {
		t.Fatalf("unexpected notes payload: %#v", payload.Data.Notes)
	}
	if len(payload.Data.RecentCommits) == 0 || payload.Data.RecentCommits[0].Summary != "feat: initial repo memory support" {
		t.Fatalf("unexpected commits payload: %#v", payload.Data.RecentCommits)
	}
}

func TestHandoffGenerateAndList(t *testing.T) {
	repoDir := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), "aid-data")

	runGit(t, repoDir, "init", "-q")
	writeFile(t, filepath.Join(repoDir, "README.md"), []byte("hello\n"))
	runGit(t, repoDir, "add", "README.md")
	runGitWithIdentity(t, repoDir, "commit", "-m", "feat: initial repo memory support")
	writeFile(t, filepath.Join(repoDir, "NOTES.txt"), []byte("dirty tree\n"))

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
	_ = runCLI(t, "note", "add", "Token refresh bug occurs after 401 retry")
	_ = runCLI(t, "task", "add", "Fix token refresh retry path")
	_ = runCLI(t, "decide", "add", "Store money as integer pence")

	generateOut := runCLI(t, "handoff", "generate", "--brief")
	for _, want := range []string{
		"Branch: main",
		"Worktree: dirty",
		"Open tasks:",
		"Recent notes:",
		"Key decisions:",
		"Recent commits:",
	} {
		if !strings.Contains(generateOut, want) {
			t.Fatalf("expected handoff output to contain %q\n\n%s", want, generateOut)
		}
	}

	jsonList := runCLI(t, "handoff", "list", "--json")
	var payload struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Data    struct {
			Handoffs []struct {
				ID      string `json:"id"`
				Summary string `json:"summary"`
			} `json:"handoffs"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(jsonList), &payload); err != nil {
		t.Fatalf("unmarshal handoff json: %v\n%s", err, jsonList)
	}
	if !payload.OK || payload.Command != "handoff list" {
		t.Fatalf("unexpected handoff list payload: %#v", payload)
	}
	if len(payload.Data.Handoffs) != 1 || !strings.Contains(payload.Data.Handoffs[0].Summary, "Worktree: dirty") {
		t.Fatalf("unexpected handoffs: %#v", payload.Data.Handoffs)
	}
}

func runCLI(t *testing.T, args ...string) string {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run(args, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected success for %v, got exit code %d with stderr %q", args, exitCode, stderr.String())
	}

	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr for %v, got %q", args, stderr.String())
	}

	return stdout.String()
}

func runGit(t *testing.T, cwd string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = cwd

	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}

func runGitWithIdentity(t *testing.T, cwd string, args ...string) {
	t.Helper()

	prefixed := append([]string{"-c", "user.name=Test User", "-c", "user.email=test@example.com"}, args...)
	runGit(t, cwd, prefixed...)
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()

	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

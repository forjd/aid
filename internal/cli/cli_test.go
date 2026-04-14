package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forjd/aid/internal/app"
	"github.com/forjd/aid/internal/store"
	sqlitestore "github.com/forjd/aid/internal/store/sqlite"
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

	_ = runCLI(t, "init")
	output := runCLI(t, "recall", "help")
	if !strings.Contains(output, "No matching context.") {
		t.Fatalf("expected no matching context, got %q", output)
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
	if !strings.Contains(noteOut, "note_1") {
		t.Fatalf("expected note output, got %q", noteOut)
	}

	noteListOut := runCLI(t, "note", "list")
	if !strings.Contains(noteListOut, "Refresh token bug occurs after 401 retry") {
		t.Fatalf("expected note list output, got %q", noteListOut)
	}

	taskOut := runCLI(t, "task", "add", "Fix VAT rounding on invoice lines")
	if !strings.Contains(taskOut, "task_1") {
		t.Fatalf("expected task output, got %q", taskOut)
	}

	taskStartOut := runCLI(t, "task", "start", "task_1")
	if !strings.Contains(taskStartOut, "task_1") || !strings.Contains(taskStartOut, "in_progress") {
		t.Fatalf("expected task start output, got %q", taskStartOut)
	}

	taskBlockOut := runCLI(t, "task", "block", "task_1")
	if !strings.Contains(taskBlockOut, "task_1") || !strings.Contains(taskBlockOut, "blocked") {
		t.Fatalf("expected task block output, got %q", taskBlockOut)
	}

	taskReopenOut := runCLI(t, "task", "reopen", "task_1")
	if !strings.Contains(taskReopenOut, "task_1") || !strings.Contains(taskReopenOut, "open") {
		t.Fatalf("expected task reopen output, got %q", taskReopenOut)
	}

	taskDoneOut := runCLI(t, "task", "done", "task_1")
	if !strings.Contains(taskDoneOut, "task_1") {
		t.Fatalf("expected task done output, got %q", taskDoneOut)
	}

	decisionOut := runCLI(t, "decide", "add", "Store all monetary values as integer pence")
	if !strings.Contains(decisionOut, "decision_1") {
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

func TestVerboseStatusAndListOutputs(t *testing.T) {
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

	initOut := runCLI(t, "init", "--verbose")
	for _, want := range []string{
		"Repo name:",
		"Config created:",
		"Next steps:",
	} {
		if !strings.Contains(initOut, want) {
			t.Fatalf("expected verbose init output to contain %q\n\n%s", want, initOut)
		}
	}

	_ = runCLI(t, "note", "add", "Token refresh bug")
	_ = runCLI(t, "task", "add", "Fix token refresh")
	_ = runCLI(t, "decide", "add", "Store money as integer pence")

	defaultStatus := runCLI(t, "status")
	if strings.Contains(defaultStatus, "Repo name:") {
		t.Fatalf("expected default status output to stay compact, got %q", defaultStatus)
	}

	verboseStatus := runCLI(t, "status", "--verbose")
	for _, want := range []string{
		"Repo name:",
		"Task breakdown:",
		"Next steps:",
	} {
		if !strings.Contains(verboseStatus, want) {
			t.Fatalf("expected verbose status output to contain %q\n\n%s", want, verboseStatus)
		}
	}

	verboseNotes := runCLI(t, "note", "list", "--verbose")
	for _, want := range []string{
		"Note note_1",
		"Scope: branch",
		"Created:",
		"Text: Token refresh bug",
	} {
		if !strings.Contains(verboseNotes, want) {
			t.Fatalf("expected verbose notes output to contain %q\n\n%s", want, verboseNotes)
		}
	}

	verboseTasks := runCLI(t, "task", "list", "--verbose")
	for _, want := range []string{
		"Task task_1",
		"Status: open",
		"Updated:",
		"Text: Fix token refresh",
	} {
		if !strings.Contains(verboseTasks, want) {
			t.Fatalf("expected verbose tasks output to contain %q\n\n%s", want, verboseTasks)
		}
	}

	verboseDecisions := runCLI(t, "decide", "list", "--verbose")
	for _, want := range []string{
		"Decision decision_1",
		"Created:",
		"Text: Store money as integer pence",
	} {
		if !strings.Contains(verboseDecisions, want) {
			t.Fatalf("expected verbose decisions output to contain %q\n\n%s", want, verboseDecisions)
		}
	}
}

func TestRepoConfigDefaultModeAppliesToUnflaggedCommands(t *testing.T) {
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

	_ = runCLI(t, "init")
	writeFile(t, filepath.Join(repoDir, ".aid", "config.toml"), []byte(`[output]
default_mode = "verbose"

[indexing]
ignore_paths = ["vendor/"]

[agent]
skill_path = "skills/aid/SKILL.md"
`))

	defaultStatus := runCLI(t, "status")
	if !strings.Contains(defaultStatus, "Repo name:") {
		t.Fatalf("expected unflagged status to use verbose mode from config, got %q", defaultStatus)
	}

	briefStatus := runCLI(t, "status", "--brief")
	if strings.Contains(briefStatus, "Repo name:") {
		t.Fatalf("expected explicit --brief to override configured default, got %q", briefStatus)
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
	branch := currentGitBranch(t, repoDir)

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
		"Branch: " + branch,
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

func TestVerboseResumeAndRecallOutputs(t *testing.T) {
	repoDir := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), "aid-data")

	runGit(t, repoDir, "init", "-q")
	writeFile(t, filepath.Join(repoDir, "README.md"), []byte("hello\n"))
	runGit(t, repoDir, "add", "README.md")
	runGitWithIdentity(t, repoDir, "commit", "-m", "feat: initial refresh memory support")

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
	_ = runCLI(t, "decide", "add", "Treat refresh tokens as single-use")
	_ = runCLI(t, "handoff", "generate")
	_ = runCLI(t, "history", "index")

	defaultResume := runCLI(t, "resume")
	if strings.Contains(defaultResume, "Active task inferred:") {
		t.Fatalf("expected default resume output to stay compact, got %q", defaultResume)
	}

	verboseResume := runCLI(t, "resume", "--verbose")
	for _, want := range []string{
		"Repo name:",
		"Active task inferred: yes",
		"Task task_1",
		"Created:",
		"Author:",
	} {
		if !strings.Contains(verboseResume, want) {
			t.Fatalf("expected verbose resume output to contain %q\n\n%s", want, verboseResume)
		}
	}

	verboseRecall := runCLI(t, "recall", "refresh", "--verbose")
	for _, want := range []string{
		"Note note_1",
		"Decision decision_1",
		"Handoff handoff_1",
		"Commit ",
		"Created:",
	} {
		if !strings.Contains(verboseRecall, want) {
			t.Fatalf("expected verbose recall output to contain %q\n\n%s", want, verboseRecall)
		}
	}
}

func TestHandoffGenerateAndList(t *testing.T) {
	repoDir := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), "aid-data")

	runGit(t, repoDir, "init", "-q")
	writeFile(t, filepath.Join(repoDir, "README.md"), []byte("hello\n"))
	runGit(t, repoDir, "add", "README.md")
	runGitWithIdentity(t, repoDir, "commit", "-m", "feat: initial repo memory support")
	branch := currentGitBranch(t, repoDir)
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
		"Branch: " + branch,
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

func TestHistoryIndexAndSearch(t *testing.T) {
	repoDir := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), "aid-data")

	runGit(t, repoDir, "init", "-q")
	writeFile(t, filepath.Join(repoDir, "auth.txt"), []byte("refresh\n"))
	runGit(t, repoDir, "add", "auth.txt")
	runGitWithIdentity(t, repoDir, "commit", "-m", "feat: token refresh retry")
	writeFile(t, filepath.Join(repoDir, "invoice.txt"), []byte("vat\n"))
	runGit(t, repoDir, "add", "invoice.txt")
	runGitWithIdentity(t, repoDir, "commit", "-m", "fix: invoice vat reconciliation")

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

	indexOut := runCLI(t, "history", "index")
	if !strings.Contains(indexOut, "Indexed 2 commits.") {
		t.Fatalf("expected index output, got %q", indexOut)
	}

	briefSearch := runCLI(t, "history", "search", "invoice", "--brief")
	if !strings.Contains(briefSearch, "invoice vat reconciliation") {
		t.Fatalf("expected invoice search result, got %q", briefSearch)
	}

	naturalLanguageSearch := runCLI(t, "history", "search", "why was invoice vat reconciliation added", "--brief")
	if !strings.Contains(naturalLanguageSearch, "invoice vat reconciliation") {
		t.Fatalf("expected natural language search result, got %q", naturalLanguageSearch)
	}

	jsonSearch := runCLI(t, "history", "search", "refresh", "--json")
	var payload struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Data    struct {
			Query   string `json:"query"`
			Commits []struct {
				Summary string   `json:"summary"`
				Paths   []string `json:"changed_paths"`
			} `json:"commits"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(jsonSearch), &payload); err != nil {
		t.Fatalf("unmarshal history search json: %v\n%s", err, jsonSearch)
	}
	if !payload.OK || payload.Command != "history search" || payload.Data.Query != "refresh" {
		t.Fatalf("unexpected history search payload: %#v", payload)
	}
	if len(payload.Data.Commits) != 1 || payload.Data.Commits[0].Summary != "feat: token refresh retry" {
		t.Fatalf("unexpected history search commits: %#v", payload.Data.Commits)
	}
}

func TestVerboseHistoryAndHandoffOutputs(t *testing.T) {
	repoDir := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), "aid-data")

	runGit(t, repoDir, "init", "-q")
	writeFile(t, filepath.Join(repoDir, "auth.txt"), []byte("refresh\n"))
	runGit(t, repoDir, "add", "auth.txt")
	runGitWithIdentity(t, repoDir, "commit", "-m", "feat: token refresh retry")
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
	_ = runCLI(t, "task", "add", "Fix token refresh retry path")

	verboseGenerated := runCLI(t, "handoff", "generate", "--verbose")
	for _, want := range []string{
		"Saved handoff handoff_1",
		"Created:",
		"Summary:",
	} {
		if !strings.Contains(verboseGenerated, want) {
			t.Fatalf("expected verbose handoff generate output to contain %q\n\n%s", want, verboseGenerated)
		}
	}

	verboseList := runCLI(t, "handoff", "list", "--verbose")
	for _, want := range []string{
		"Handoff handoff_1",
		"Created:",
		"Summary:",
	} {
		if !strings.Contains(verboseList, want) {
			t.Fatalf("expected verbose handoff list output to contain %q\n\n%s", want, verboseList)
		}
	}

	indexOut := runCLI(t, "history", "index", "--verbose")
	for _, want := range []string{
		"History index complete",
		"Mode: initial sync",
		"Commits indexed: 1",
	} {
		if !strings.Contains(indexOut, want) {
			t.Fatalf("expected verbose history index output to contain %q\n\n%s", want, indexOut)
		}
	}

	verboseSearch := runCLI(t, "history", "search", "refresh", "--verbose")
	for _, want := range []string{
		"Commit ",
		"Author: Test User",
		"Committed:",
		"Message:",
		"Paths: auth.txt",
	} {
		if !strings.Contains(verboseSearch, want) {
			t.Fatalf("expected verbose history search output to contain %q\n\n%s", want, verboseSearch)
		}
	}
}

func TestHistoryIndexUsesIgnorePathsAndIncrementalSync(t *testing.T) {
	repoDir := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), "aid-data")

	runGit(t, repoDir, "init", "-q")
	if err := os.MkdirAll(filepath.Join(repoDir, "vendor"), 0o755); err != nil {
		t.Fatalf("create vendor dir: %v", err)
	}
	writeFile(t, filepath.Join(repoDir, "vendor", "deps.txt"), []byte("vendor refresh\n"))
	runGit(t, repoDir, "add", "vendor/deps.txt")
	runGitWithIdentity(t, repoDir, "commit", "-m", "chore: vendor refresh")
	writeFile(t, filepath.Join(repoDir, "auth.txt"), []byte("refresh\n"))
	runGit(t, repoDir, "add", "auth.txt")
	runGitWithIdentity(t, repoDir, "commit", "-m", "feat: token refresh retry")

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

	firstIndex := runCLI(t, "history", "index", "--verbose")
	for _, want := range []string{
		"Mode: initial sync",
		"Commits indexed: 1",
		"Commits added: 1",
		"Commits removed: 0",
	} {
		if !strings.Contains(firstIndex, want) {
			t.Fatalf("expected first history index output to contain %q\n\n%s", want, firstIndex)
		}
	}

	vendorSearch := runCLI(t, "history", "search", "vendor", "--brief")
	if !strings.Contains(vendorSearch, "No matching commits.") {
		t.Fatalf("expected ignored vendor commit to stay out of the index, got %q", vendorSearch)
	}

	writeFile(t, filepath.Join(repoDir, "auth_worker.txt"), []byte("retry\n"))
	runGit(t, repoDir, "add", "auth_worker.txt")
	runGitWithIdentity(t, repoDir, "commit", "-m", "fix: tighten refresh retry")

	secondIndex := runCLI(t, "history", "index", "--verbose")
	for _, want := range []string{
		"Mode: incremental sync",
		"Commits indexed: 2",
		"Commits added: 1",
		"Commits updated: 0",
		"Commits removed: 0",
	} {
		if !strings.Contains(secondIndex, want) {
			t.Fatalf("expected second history index output to contain %q\n\n%s", want, secondIndex)
		}
	}
}

func TestHistoryIndexPrunesCommitsNoLongerReachable(t *testing.T) {
	repoDir := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), "aid-data")

	runGit(t, repoDir, "init", "-q")
	writeFile(t, filepath.Join(repoDir, "README.md"), []byte("base\n"))
	runGit(t, repoDir, "add", "README.md")
	runGitWithIdentity(t, repoDir, "commit", "-m", "chore: base commit")
	baseBranch := currentGitBranch(t, repoDir)
	runGit(t, repoDir, "checkout", "-q", "-b", "feature/refresh")
	writeFile(t, filepath.Join(repoDir, "branch_only.txt"), []byte("refresh branch\n"))
	runGit(t, repoDir, "add", "branch_only.txt")
	runGitWithIdentity(t, repoDir, "commit", "-m", "feat: branch-only refresh work")
	runGit(t, repoDir, "checkout", "-q", baseBranch)

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

	firstIndex := runCLI(t, "history", "index", "--verbose")
	for _, want := range []string{
		"Commits indexed: 2",
		"Commits added: 2",
		"Commits removed: 0",
	} {
		if !strings.Contains(firstIndex, want) {
			t.Fatalf("expected first history index output to contain %q\n\n%s", want, firstIndex)
		}
	}

	runGit(t, repoDir, "branch", "-D", "feature/refresh")

	secondIndex := runCLI(t, "history", "index", "--verbose")
	for _, want := range []string{
		"Mode: incremental sync",
		"Commits indexed: 1",
		"Commits removed: 1",
	} {
		if !strings.Contains(secondIndex, want) {
			t.Fatalf("expected second history index output to contain %q\n\n%s", want, secondIndex)
		}
	}

	prunedSearch := runCLI(t, "history", "search", "branch-only", "--brief")
	if !strings.Contains(prunedSearch, "No matching commits.") {
		t.Fatalf("expected deleted branch commit to be pruned from history, got %q", prunedSearch)
	}
}

func TestRecallSearchesAcrossStoredContext(t *testing.T) {
	repoDir := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), "aid-data")

	runGit(t, repoDir, "init", "-q")
	writeFile(t, filepath.Join(repoDir, "auth.txt"), []byte("refresh\n"))
	runGit(t, repoDir, "add", "auth.txt")
	runGitWithIdentity(t, repoDir, "commit", "-m", "feat: token refresh retry")

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
	_ = runCLI(t, "note", "add", "Refresh token bug occurs after 401 retry")
	_ = runCLI(t, "decide", "add", "Treat refresh tokens as single-use")
	_ = runCLI(t, "task", "add", "Fix refresh retry path")
	_ = runCLI(t, "handoff", "generate")
	_ = runCLI(t, "history", "index")

	briefRecall := runCLI(t, "recall", "refresh", "--brief")
	for _, want := range []string{
		"Refresh token bug occurs after 401 retry",
		"Treat refresh tokens as single-use",
		"feat: token refresh retry",
	} {
		if !strings.Contains(briefRecall, want) {
			t.Fatalf("expected recall output to contain %q\n\n%s", want, briefRecall)
		}
	}

	jsonRecall := runCLI(t, "recall", "refresh", "--json")
	var payload struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Data    struct {
			Query string `json:"query"`
			Notes []struct {
				Text string `json:"text"`
			} `json:"notes"`
			Decisions []struct {
				Text string `json:"text"`
			} `json:"decisions"`
			Handoffs []struct {
				Summary string `json:"summary"`
			} `json:"handoffs"`
			Commits []struct {
				Summary string `json:"summary"`
			} `json:"commits"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(jsonRecall), &payload); err != nil {
		t.Fatalf("unmarshal recall json: %v\n%s", err, jsonRecall)
	}
	if !payload.OK || payload.Command != "recall" || payload.Data.Query != "refresh" {
		t.Fatalf("unexpected recall payload: %#v", payload)
	}
	if len(payload.Data.Notes) == 0 || len(payload.Data.Decisions) == 0 || len(payload.Data.Handoffs) == 0 || len(payload.Data.Commits) == 0 {
		t.Fatalf("expected recall results across categories: %#v", payload.Data)
	}
}

func TestRecallReusesCommitSearchRanking(t *testing.T) {
	repoDir := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), "aid-data")

	runGit(t, repoDir, "init", "-q")
	writeFile(t, filepath.Join(repoDir, "auth.txt"), []byte("refresh\n"))
	runGit(t, repoDir, "add", "auth.txt")
	runGitWithIdentity(t, repoDir, "commit", "-m", "feat: token refresh retry")
	writeFile(t, filepath.Join(repoDir, "refresh_worker.txt"), []byte("worker\n"))
	runGit(t, repoDir, "add", "refresh_worker.txt")
	runGitWithIdentity(t, repoDir, "commit", "-m", "chore: tidy auth handlers")

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
	_ = runCLI(t, "history", "index")

	jsonRecall := runCLI(t, "recall", "refresh", "--json")
	var payload struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Data    struct {
			Commits []struct {
				Summary string `json:"summary"`
			} `json:"commits"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(jsonRecall), &payload); err != nil {
		t.Fatalf("unmarshal recall json: %v\n%s", err, jsonRecall)
	}
	if !payload.OK || payload.Command != "recall" {
		t.Fatalf("unexpected recall payload: %#v", payload)
	}
	if len(payload.Data.Commits) < 2 {
		t.Fatalf("expected at least two commit matches, got %#v", payload.Data.Commits)
	}
	if payload.Data.Commits[0].Summary != "feat: token refresh retry" {
		t.Fatalf("expected recall to preserve commit relevance order, got %#v", payload.Data.Commits)
	}
}

func TestResumeAndHandoffHighlightBlockedWork(t *testing.T) {
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
	insertTaskWithStatus(t, repoDir, "Confirm refresh rollback strategy", store.TaskBlocked)

	briefResume := runCLI(t, "resume", "--brief")
	for _, want := range []string{
		"Next:",
		"resolve blocker for Confirm refresh rollback strategy",
	} {
		if !strings.Contains(briefResume, want) {
			t.Fatalf("expected resume output to contain %q\n\n%s", want, briefResume)
		}
	}

	handoffOut := runCLI(t, "handoff", "generate", "--brief")
	for _, want := range []string{
		"Open questions:",
		"What is blocking Confirm refresh rollback strategy?",
		"Recommended next action:",
		"resolve blocker for Confirm refresh rollback strategy",
	} {
		if !strings.Contains(handoffOut, want) {
			t.Fatalf("expected handoff output to contain %q\n\n%s", want, handoffOut)
		}
	}
}

func TestResumeAndHandoffCarryForwardPriorHandoffContext(t *testing.T) {
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

	_ = runCLI(t, "init")
	insertHandoff(t, repoDir, strings.Join([]string{
		"Branch: main",
		"Worktree: clean",
		"Open questions:",
		"- Should retry backoff be preserved for 401 recovery?",
		"Recommended next action:",
		"- inspect auth/session.go before changing retry semantics",
	}, "\n"))

	briefResume := runCLI(t, "resume", "--brief")
	for _, want := range []string{
		"Open questions:",
		"Should retry backoff be preserved for 401 recovery?",
		"Next:",
		"inspect auth/session.go before changing retry semantics",
	} {
		if !strings.Contains(briefResume, want) {
			t.Fatalf("expected resume output to contain %q\n\n%s", want, briefResume)
		}
	}

	handoffOut := runCLI(t, "handoff", "generate", "--brief")
	for _, want := range []string{
		"Open questions:",
		"Should retry backoff be preserved for 401 recovery?",
		"Recommended next action:",
		"inspect auth/session.go before changing retry semantics",
	} {
		if !strings.Contains(handoffOut, want) {
			t.Fatalf("expected handoff output to contain %q\n\n%s", want, handoffOut)
		}
	}
}

func insertTaskWithStatus(t *testing.T, repoDir string, statusText string, status store.TaskStatus) {
	t.Helper()

	ctx := context.Background()
	env, err := app.Discover(repoDir)
	if err != nil {
		t.Fatalf("discover environment: %v", err)
	}

	sqliteStore, err := sqlitestore.Open(env.DBPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer sqliteStore.Close()

	if err := sqliteStore.Migrate(ctx); err != nil {
		t.Fatalf("migrate store: %v", err)
	}

	repo, err := sqliteStore.FindRepoByPath(ctx, env.RepoRoot)
	if err != nil {
		t.Fatalf("find repo: %v", err)
	}
	if repo == nil {
		t.Fatal("expected initialised repo to exist")
	}

	if _, err := sqliteStore.AddTask(ctx, store.AddTaskInput{
		RepoID: repo.ID,
		Branch: env.Branch,
		Scope:  store.ScopeBranch,
		Text:   statusText,
		Status: status,
	}); err != nil {
		t.Fatalf("add task with status %q: %v", status, err)
	}
}

func insertHandoff(t *testing.T, repoDir string, summary string) {
	t.Helper()

	ctx := context.Background()
	env, err := app.Discover(repoDir)
	if err != nil {
		t.Fatalf("discover environment: %v", err)
	}

	sqliteStore, err := sqlitestore.Open(env.DBPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer sqliteStore.Close()

	if err := sqliteStore.Migrate(ctx); err != nil {
		t.Fatalf("migrate store: %v", err)
	}

	repo, err := sqliteStore.FindRepoByPath(ctx, env.RepoRoot)
	if err != nil {
		t.Fatalf("find repo: %v", err)
	}
	if repo == nil {
		t.Fatal("expected initialised repo to exist")
	}

	if _, err := sqliteStore.AddHandoff(ctx, store.AddHandoffInput{
		RepoID:  repo.ID,
		Branch:  env.Branch,
		Summary: summary,
	}); err != nil {
		t.Fatalf("add handoff: %v", err)
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

func currentGitBranch(t *testing.T, cwd string) string {
	t.Helper()

	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = cwd

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git branch --show-current failed: %v", err)
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" {
		t.Fatal("expected current git branch to be non-empty")
	}

	return branch
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()

	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

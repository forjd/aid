package cli

import (
	"bytes"
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

package cli

import (
	"bytes"
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

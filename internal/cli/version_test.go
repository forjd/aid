package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestVersionCommandHumanOutputUsesDevWhenUnset(t *testing.T) {
	SetBuildInfo(BuildInfo{})
	defer SetBuildInfo(BuildInfo{})

	var stdout, stderr bytes.Buffer
	if exit := Run([]string{"version"}, &stdout, &stderr); exit != 0 {
		t.Fatalf("expected success, got %d stderr=%q", exit, stderr.String())
	}
	if !strings.Contains(stdout.String(), "aid dev") {
		t.Fatalf("unexpected version output: %q", stdout.String())
	}
}

func TestVersionCommandHumanOutputIncludesCommitAndDate(t *testing.T) {
	SetBuildInfo(BuildInfo{Version: "1.2.3", Commit: "abcdef", Date: "2026-04-16"})
	defer SetBuildInfo(BuildInfo{})

	var stdout, stderr bytes.Buffer
	if exit := Run([]string{"version"}, &stdout, &stderr); exit != 0 {
		t.Fatalf("expected success, got %d stderr=%q", exit, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"aid 1.2.3", "commit: abcdef", "built:  2026-04-16"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in version output, got %q", want, out)
		}
	}
}

func TestVersionCommandJSONOutput(t *testing.T) {
	SetBuildInfo(BuildInfo{Version: "1.2.3", Commit: "abcdef", Date: "2026-04-16"})
	defer SetBuildInfo(BuildInfo{})

	var stdout, stderr bytes.Buffer
	if exit := Run([]string{"--json", "version"}, &stdout, &stderr); exit != 0 {
		t.Fatalf("expected success, got %d stderr=%q", exit, stderr.String())
	}

	var payload struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Data    struct {
			Version string `json:"version"`
			Commit  string `json:"commit"`
			Date    string `json:"date"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal version json: %v\n%s", err, stdout.String())
	}
	if !payload.OK || payload.Command != "version" {
		t.Fatalf("unexpected version payload: %#v", payload)
	}
	if payload.Data.Version != "1.2.3" || payload.Data.Commit != "abcdef" || payload.Data.Date != "2026-04-16" {
		t.Fatalf("unexpected version data: %#v", payload.Data)
	}
}

func TestVersionCommandRejectsArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if exit := Run([]string{"version", "oops"}, &stdout, &stderr); exit != exitUsage {
		t.Fatalf("expected usage exit %d, got %d", exitUsage, exit)
	}
}

package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestEnsureRepoConfigCreatesFileAndIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".aid", "config.toml")

	created, err := EnsureRepoConfig(path)
	if err != nil {
		t.Fatalf("ensure repo config: %v", err)
	}
	if !created {
		t.Fatalf("expected first ensure call to create config")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if string(content) != DefaultRepoConfig {
		t.Fatalf("unexpected config content:\n%s", string(content))
	}

	created, err = EnsureRepoConfig(path)
	if err != nil {
		t.Fatalf("ensure repo config second time: %v", err)
	}
	if created {
		t.Fatalf("expected second ensure call to be a no-op")
	}
}

func TestLoadRepoConfigParsesSectionsCommentsAndQuotedHashes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := `[output]
default_mode = "verbose" # use full output

[indexing]
ignore_paths = ["vendor/", "docs/#drafts/"]

[agent]
skill_path = ".agents/skills/aid/SKILL.md#stable"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadRepoConfig(path)
	if err != nil {
		t.Fatalf("load repo config: %v", err)
	}

	if cfg.Output.DefaultMode != "verbose" {
		t.Fatalf("unexpected default mode: %q", cfg.Output.DefaultMode)
	}
	if !reflect.DeepEqual(cfg.Indexing.IgnorePaths, []string{"vendor/", "docs/#drafts/"}) {
		t.Fatalf("unexpected ignore paths: %#v", cfg.Indexing.IgnorePaths)
	}
	if cfg.Agent.SkillPath != ".agents/skills/aid/SKILL.md#stable" {
		t.Fatalf("unexpected skill path: %q", cfg.Agent.SkillPath)
	}
}

func TestLoadRepoConfigHandlesMissingFile(t *testing.T) {
	cfg, err := LoadRepoConfig(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatalf("load missing config: %v", err)
	}
	if !reflect.DeepEqual(cfg, RepoConfig{}) {
		t.Fatalf("expected empty config for missing file, got %#v", cfg)
	}
}

func TestLoadRepoConfigRejectsInvalidLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[output]\nnot-valid\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := LoadRepoConfig(path)
	if err == nil {
		t.Fatalf("expected invalid config error")
	}
}

func TestParseStringAndStringList(t *testing.T) {
	value, err := parseString(`"hello \"world\""`)
	if err != nil {
		t.Fatalf("parse string: %v", err)
	}
	if value != `hello "world"` {
		t.Fatalf("unexpected parsed string: %q", value)
	}

	list, err := parseStringList(`["vendor/", "node_modules/"]`)
	if err != nil {
		t.Fatalf("parse string list: %v", err)
	}
	if !reflect.DeepEqual(list, []string{"vendor/", "node_modules/"}) {
		t.Fatalf("unexpected parsed list: %#v", list)
	}

	if _, err := parseString("plain-text"); err == nil {
		t.Fatalf("expected quoted string error")
	}
	if _, err := parseStringList(`[plain-text]`); err == nil {
		t.Fatalf("expected string list error")
	}
}

func TestStripInlineCommentPreservesHashesInsideQuotes(t *testing.T) {
	line := `skill_path = ".agents/skills/aid/SKILL.md#stable" # trailing comment`
	got := stripInlineComment(line)
	want := `skill_path = ".agents/skills/aid/SKILL.md#stable" `
	if got != want {
		t.Fatalf("unexpected stripped line: %q", got)
	}
}

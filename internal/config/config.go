package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DefaultRepoConfig = `[output]
default_mode = "brief"

[indexing]
ignore_paths = ["vendor/", "node_modules/", "storage/"]

[agent]
skill_path = "skills/aid/SKILL.md"
`

type RepoConfig struct {
	Output   OutputConfig
	Indexing IndexingConfig
	Agent    AgentConfig
}

type OutputConfig struct {
	DefaultMode string
}

type IndexingConfig struct {
	IgnorePaths []string
}

type AgentConfig struct {
	SkillPath string
}

func EnsureRepoConfig(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("stat repo config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("create repo config directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(DefaultRepoConfig), 0o644); err != nil {
		return false, fmt.Errorf("write repo config: %w", err)
	}

	return true, nil
}

func LoadRepoConfig(path string) (RepoConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RepoConfig{}, nil
		}
		return RepoConfig{}, fmt.Errorf("read repo config: %w", err)
	}

	var cfg RepoConfig
	section := ""
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := strings.TrimSpace(stripInlineComment(scanner.Text()))
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return RepoConfig{}, fmt.Errorf("parse repo config: invalid line %q", line)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch section + "." + key {
		case "output.default_mode":
			parsed, err := parseString(value)
			if err != nil {
				return RepoConfig{}, fmt.Errorf("parse output.default_mode: %w", err)
			}
			cfg.Output.DefaultMode = parsed
		case "indexing.ignore_paths":
			parsed, err := parseStringList(value)
			if err != nil {
				return RepoConfig{}, fmt.Errorf("parse indexing.ignore_paths: %w", err)
			}
			cfg.Indexing.IgnorePaths = parsed
		case "agent.skill_path":
			parsed, err := parseString(value)
			if err != nil {
				return RepoConfig{}, fmt.Errorf("parse agent.skill_path: %w", err)
			}
			cfg.Agent.SkillPath = parsed
		}
	}

	if err := scanner.Err(); err != nil {
		return RepoConfig{}, fmt.Errorf("scan repo config: %w", err)
	}

	return cfg, nil
}

func stripInlineComment(line string) string {
	inQuote := false
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '"':
			inQuote = !inQuote
		case '#':
			if !inQuote {
				return line[:i]
			}
		}
	}

	return line
}

func parseString(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) < 2 || trimmed[0] != '"' || trimmed[len(trimmed)-1] != '"' {
		return "", fmt.Errorf("expected quoted string")
	}

	return strings.ReplaceAll(trimmed[1:len(trimmed)-1], `\"`, `"`), nil
}

func parseStringList(value string) ([]string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "[]" {
		return nil, nil
	}
	if len(trimmed) < 2 || trimmed[0] != '[' || trimmed[len(trimmed)-1] != ']' {
		return nil, fmt.Errorf("expected string list")
	}

	body := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
	if body == "" {
		return nil, nil
	}

	parts := strings.Split(body, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		parsed, err := parseString(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		items = append(items, parsed)
	}

	return items, nil
}

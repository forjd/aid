package git

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Commit struct {
	SHA          string
	Summary      string
	Message      string
	Author       string
	CommittedAt  time.Time
	ChangedPaths []string
}

func RecentCommits(startDir string, limit int) ([]Commit, error) {
	if limit <= 0 {
		return nil, nil
	}

	cmd := exec.Command("git", "-C", startDir, "log", fmt.Sprintf("-%d", limit), "--format=%H%x1f%an%x1f%aI%x1f%s", "--name-only")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		if errors.As(err, new(*exec.ExitError)) {
			message := strings.TrimSpace(stderr.String())
			if strings.Contains(message, "does not have any commits yet") || strings.Contains(message, "your current branch") && strings.Contains(message, "does not have any commits yet") {
				return nil, nil
			}

			return nil, fmt.Errorf("%s", message)
		}

		return nil, err
	}

	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return nil, nil
	}

	commits := make([]Commit, 0, limit)
	var current *Commit

	scanner := bufio.NewScanner(strings.NewReader(trimmed))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.Contains(line, "\x1f") {
			header := strings.Split(line, "\x1f")
			if len(header) != 4 {
				return nil, fmt.Errorf("unexpected git log header %q", line)
			}

			committedAt, err := time.Parse(time.RFC3339, header[2])
			if err != nil {
				return nil, fmt.Errorf("parse commit time: %w", err)
			}

			commits = append(commits, Commit{
				SHA:         header[0],
				Summary:     header[3],
				Message:     header[3],
				Author:      header[1],
				CommittedAt: committedAt.UTC(),
			})
			current = &commits[len(commits)-1]
			continue
		}

		if current == nil {
			return nil, fmt.Errorf("unexpected git log path without header %q", line)
		}

		current.ChangedPaths = append(current.ChangedPaths, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan git log output: %w", err)
	}

	return commits, nil
}

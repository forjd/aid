package git

import (
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

const (
	commitLogFormat = "--format=%x00%H%x00%an%x00%aI%x00%s"
	commitBatchSize = 256
)

func RecentCommits(startDir string, limit int) ([]Commit, error) {
	return Commits(startDir, limit)
}

func Commits(startDir string, limit int) ([]Commit, error) {
	args := []string{"-C", startDir, "log", "--all"}
	if limit > 0 {
		args = append(args, fmt.Sprintf("-%d", limit))
	}
	args = append(args, commitLogFormat, "--name-only", "-z")

	output, err := runGitOutput(args...)
	if err != nil {
		return nil, err
	}

	return parseCommits(output, limit)
}

func AllCommitSHAs(startDir string) ([]string, error) {
	output, err := runGitOutput("-C", startDir, "rev-list", "--all")
	if err != nil {
		return nil, err
	}

	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return nil, nil
	}

	lines := strings.Split(trimmed, "\n")
	shas := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		shas = append(shas, line)
	}

	return shas, nil
}

func CommitsBySHA(startDir string, shas []string) ([]Commit, error) {
	if len(shas) == 0 {
		return nil, nil
	}

	commits := make([]Commit, 0, len(shas))
	for start := 0; start < len(shas); start += commitBatchSize {
		end := start + commitBatchSize
		if end > len(shas) {
			end = len(shas)
		}

		args := []string{"-C", startDir, "log", "--no-walk=sorted", commitLogFormat, "--name-only", "-z"}
		args = append(args, shas[start:end]...)

		output, err := runGitOutput(args...)
		if err != nil {
			return nil, err
		}

		batch, err := parseCommits(output, end-start)
		if err != nil {
			return nil, err
		}
		commits = append(commits, batch...)
	}

	return commits, nil
}

func parseCommits(output []byte, limit int) ([]Commit, error) {
	if len(bytes.TrimSpace(output)) == 0 {
		return nil, nil
	}

	capHint := limit
	if capHint <= 0 {
		capHint = 32
	}
	commits := make([]Commit, 0, capHint)
	tokens := bytes.Split(output, []byte{0})
	for i := 0; i < len(tokens); {
		if len(tokens[i]) == 0 {
			i++
			if i >= len(tokens) {
				break
			}
			if i+3 >= len(tokens) {
				return nil, fmt.Errorf("unexpected git log header %q", string(bytes.Join(tokens[i:], []byte{0})))
			}

			committedAt, err := time.Parse(time.RFC3339, string(tokens[i+2]))
			if err != nil {
				return nil, fmt.Errorf("parse commit time: %w", err)
			}

			commits = append(commits, Commit{
				SHA:         string(tokens[i]),
				Summary:     string(tokens[i+3]),
				Message:     string(tokens[i+3]),
				Author:      string(tokens[i+1]),
				CommittedAt: committedAt.UTC(),
			})
			current := &commits[len(commits)-1]
			i += 4

			for i < len(tokens) && len(tokens[i]) != 0 {
				path := strings.TrimSpace(string(tokens[i]))
				if path != "" {
					current.ChangedPaths = append(current.ChangedPaths, path)
				}
				i++
			}
			continue
		}

		return nil, fmt.Errorf("unexpected git log path without header %q", string(tokens[i]))
	}

	return commits, nil
}

func runGitOutput(args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		if errors.As(err, new(*exec.ExitError)) {
			message := strings.TrimSpace(stderr.String())
			if strings.Contains(message, "does not have any commits yet") || strings.Contains(message, "your current branch") && strings.Contains(message, "does not have any commits yet") {
				return nil, nil
			}
			if message == "" {
				message = err.Error()
			}

			return nil, fmt.Errorf("%s", message)
		}

		return nil, err
	}

	return output, nil
}

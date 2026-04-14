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

const (
	commitLogFormat = "--format=%H%x1f%an%x1f%aI%x1f%s"
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
	args = append(args, commitLogFormat, "--name-only")

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

		args := []string{"-C", startDir, "log", "--no-walk=sorted", commitLogFormat, "--name-only"}
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
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return nil, nil
	}

	capHint := limit
	if capHint <= 0 {
		capHint = 32
	}
	commits := make([]Commit, 0, capHint)
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

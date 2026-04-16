package git

import (
	"bytes"
	"context"
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
	// Each commit is emitted as a null-delimited record: an empty header
	// (from %x00 at the start of the template) followed by SHA, author,
	// committed-at, subject, and body. --name-only -z then appends the
	// changed paths, each separated by \0, and terminates the record with
	// the next empty header.
	commitLogFormat = "--format=%x00%H%x00%an%x00%aI%x00%s%x00%b"
	commitBatchSize = 256
)

func RecentCommits(ctx context.Context, startDir string, limit int) ([]Commit, error) {
	return Commits(ctx, startDir, limit)
}

func Commits(ctx context.Context, startDir string, limit int) ([]Commit, error) {
	args := []string{"-C", startDir, "log", "--all"}
	if limit > 0 {
		args = append(args, fmt.Sprintf("-%d", limit))
	}
	args = append(args, commitLogFormat, "--name-only", "-z")

	output, err := runGitOutput(ctx, args...)
	if err != nil {
		return nil, err
	}

	return parseCommits(output, limit)
}

func AllCommitSHAs(ctx context.Context, startDir string) ([]string, error) {
	output, err := runGitOutput(ctx, "-C", startDir, "rev-list", "--all")
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

func CommitsBySHA(ctx context.Context, startDir string, shas []string) ([]Commit, error) {
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

		output, err := runGitOutput(ctx, args...)
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
			if i+4 >= len(tokens) {
				return nil, fmt.Errorf("unexpected git log header %q", string(bytes.Join(tokens[i:], []byte{0})))
			}

			committedAt, err := time.Parse(time.RFC3339, string(tokens[i+2]))
			if err != nil {
				return nil, fmt.Errorf("parse commit time: %w", err)
			}

			subject := string(tokens[i+3])
			body := string(tokens[i+4])
			commits = append(commits, Commit{
				SHA:         string(tokens[i]),
				Summary:     subject,
				Message:     combineSubjectAndBody(subject, body),
				Author:      string(tokens[i+1]),
				CommittedAt: committedAt.UTC(),
			})
			current := &commits[len(commits)-1]
			i += 5

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

func combineSubjectAndBody(subject, body string) string {
	trimmedBody := strings.TrimSpace(body)
	if trimmedBody == "" {
		return subject
	}
	return subject + "\n\n" + trimmedBody
}

func runGitOutput(ctx context.Context, args ...string) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, "git", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
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

package cli

import (
	"context"
	"strings"

	"github.com/forjd/aid/internal/git"
	"github.com/forjd/aid/internal/store"
)

const defaultListLimit = 20

func joinArgs(args []string, label string) (string, error) {
	text := strings.TrimSpace(strings.Join(args, " "))
	if text == "" {
		return "", newError(ErrCodeUsage, "missing %s", label)
	}

	return text, nil
}

func recentContextCommits(ctx context.Context, runtime *repoRuntime, limit int) ([]store.Commit, error) {
	commits, err := runtime.store.ListCommits(ctx, runtime.repo.ID, limit)
	if err != nil {
		return nil, err
	}
	if len(commits) > 0 {
		return commits, nil
	}

	liveCommits, err := git.RecentCommits(ctx, runtime.env.RepoRoot, limit)
	if err != nil {
		return nil, err
	}

	commits = make([]store.Commit, 0, len(liveCommits))
	for _, commit := range liveCommits {
		commits = append(commits, store.Commit{
			SHA:          commit.SHA,
			Author:       commit.Author,
			CommittedAt:  commit.CommittedAt,
			Message:      commit.Message,
			Summary:      commit.Summary,
			ChangedPaths: append([]string(nil), commit.ChangedPaths...),
		})
	}

	return commits, nil
}

func taskCommandName(status store.TaskStatus) string {
	switch status {
	case store.TaskInProgress:
		return "start"
	case store.TaskBlocked:
		return "block"
	case store.TaskOpen:
		return "reopen"
	default:
		return "done"
	}
}

func taskStatusVerb(status store.TaskStatus) string {
	switch status {
	case store.TaskInProgress:
		return "Started"
	case store.TaskBlocked:
		return "Blocked"
	case store.TaskOpen:
		return "Reopened"
	default:
		return "Completed"
	}
}

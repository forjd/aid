package handoff

import (
	"fmt"
	"strings"

	"github.com/forjd/aid/internal/git"
	resumepkg "github.com/forjd/aid/internal/resume"
	"github.com/forjd/aid/internal/store"
)

type Snapshot struct {
	Branch  string
	Summary string
}

func Build(branch string, worktree git.WorktreeStatus, bundle resumepkg.Bundle, tasks []store.Task) Snapshot {
	var b strings.Builder

	fmt.Fprintf(&b, "Branch: %s\n", branch)
	fmt.Fprintf(&b, "Worktree: %s\n", worktreeLine(worktree))

	if bundle.ActiveTask != nil {
		fmt.Fprintf(&b, "Active task: %s\n", bundle.ActiveTask.Text)
	} else if bundle.ActiveTaskAmbiguous {
		fmt.Fprintln(&b, "Active task: ambiguous")
	}

	scopedTasks := filterScopedTasks(branch, tasks)
	openTasks := limitOpenTasks(scopedTasks, 5)
	if len(openTasks) > 0 {
		fmt.Fprintln(&b, "Open tasks:")
		for _, task := range openTasks {
			fmt.Fprintf(&b, "- %s [%s]\n", task.Text, task.Status)
		}
	}

	if len(bundle.Notes) > 0 {
		fmt.Fprintln(&b, "Recent notes:")
		for _, note := range bundle.Notes {
			fmt.Fprintf(&b, "- %s\n", note.Text)
		}
	}

	if len(bundle.Decisions) > 0 {
		fmt.Fprintln(&b, "Key decisions:")
		for _, decision := range bundle.Decisions {
			fmt.Fprintf(&b, "- %s\n", decision.Text)
		}
	}

	if len(bundle.RecentCommits) > 0 {
		fmt.Fprintln(&b, "Recent commits:")
		for _, commit := range bundle.RecentCommits {
			fmt.Fprintf(&b, "- %s %s\n", shortSHA(commit.SHA), commit.Summary)
		}
	}

	questions := handoffOpenQuestions(bundle.OpenQuestions, worktree)
	if len(questions) > 0 {
		fmt.Fprintln(&b, "Open questions:")
		for _, question := range questions {
			fmt.Fprintf(&b, "- %s\n", question)
		}
	}

	if bundle.NextAction != nil {
		fmt.Fprintln(&b, "Recommended next action:")
		fmt.Fprintf(&b, "- %s\n", *bundle.NextAction)
	}

	return Snapshot{
		Branch:  branch,
		Summary: strings.TrimSpace(b.String()),
	}
}

func limitOpenTasks(tasks []store.Task, limit int) []store.Task {
	items := make([]store.Task, 0, len(tasks))
	for _, task := range tasks {
		if task.Status == store.TaskDone {
			continue
		}
		items = append(items, task)
		if len(items) == limit {
			break
		}
	}

	return items
}

// filterScopedTasks drops branch-scoped tasks that belong to another branch.
// Tasks with empty scope (legacy rows) or repo scope are always kept.
func filterScopedTasks(branch string, tasks []store.Task) []store.Task {
	if branch == "" {
		return tasks
	}
	filtered := make([]store.Task, 0, len(tasks))
	for _, task := range tasks {
		if task.Scope == store.ScopeBranch && task.Branch != "" && task.Branch != branch {
			continue
		}
		filtered = append(filtered, task)
	}
	return filtered
}

func worktreeLine(status git.WorktreeStatus) string {
	if !status.Dirty {
		return "clean"
	}

	if status.Changed == 0 {
		return fmt.Sprintf("dirty (%d untracked)", status.Untracked)
	}
	if status.Untracked == 0 {
		return fmt.Sprintf("dirty (%d changed)", status.Changed)
	}

	return fmt.Sprintf("dirty (%d changed, %d untracked)", status.Changed, status.Untracked)
}

func shortSHA(value string) string {
	if len(value) <= 7 {
		return value
	}

	return value[:7]
}

func handoffOpenQuestions(bundleQuestions []string, worktree git.WorktreeStatus) []string {
	questions := append([]string(nil), bundleQuestions...)
	if worktree.Dirty {
		questions = append(questions, "Should the current uncommitted changes be kept, finished, or discarded?")
	}
	if len(questions) <= 3 {
		return questions
	}
	return questions[:3]
}

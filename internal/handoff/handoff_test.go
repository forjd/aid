package handoff

import (
	"testing"
	"time"

	"github.com/forjd/aid/internal/git"
	resumepkg "github.com/forjd/aid/internal/resume"
	"github.com/forjd/aid/internal/store"
)

func TestBuild(t *testing.T) {
	nextAction := "inspect retry middleware"
	snapshot := Build(
		"feat/auth",
		git.WorktreeStatus{Dirty: true, Changed: 2, Untracked: 1},
		resumepkg.Bundle{
			ActiveTask: taskPointer(handoffTask(1, "feat/auth", store.TaskInProgress, "Fix refresh retry path")),
			Notes: []store.Note{
				handoffNote(1, "Retry still fails after 401"),
			},
			Decisions: []store.Decision{
				handoffDecision(1, "Use a single refresh retry"),
			},
			RecentCommits: []store.Commit{
				handoffCommit("abc123456789", "fix: refresh retry path"),
			},
			OpenQuestions: []string{
				"Which retry path is still failing?",
				"Should the API return a clearer auth error?",
			},
			NextAction: &nextAction,
		},
		[]store.Task{
			handoffTask(1, "feat/auth", store.TaskInProgress, "Fix refresh retry path"),
			handoffTask(2, "feat/auth", store.TaskOpen, "Review retry logging"),
			handoffTask(3, "feat/auth", store.TaskDone, "Ignore completed work"),
		},
	)

	want := "Branch: feat/auth\n" +
		"Worktree: dirty (2 changed, 1 untracked)\n" +
		"Active task: Fix refresh retry path\n" +
		"Open tasks:\n" +
		"- Fix refresh retry path [in_progress]\n" +
		"- Review retry logging [open]\n" +
		"Recent notes:\n" +
		"- Retry still fails after 401\n" +
		"Key decisions:\n" +
		"- Use a single refresh retry\n" +
		"Recent commits:\n" +
		"- abc1234 fix: refresh retry path\n" +
		"Open questions:\n" +
		"- Which retry path is still failing?\n" +
		"- Should the API return a clearer auth error?\n" +
		"- Should the current uncommitted changes be kept, finished, or discarded?\n" +
		"Recommended next action:\n" +
		"- inspect retry middleware"

	if snapshot.Branch != "feat/auth" {
		t.Fatalf("unexpected branch: %#v", snapshot)
	}
	if snapshot.Summary != want {
		t.Fatalf("unexpected handoff summary:\n%s", snapshot.Summary)
	}
}

func TestBuildWithAmbiguousTaskAndCleanWorktree(t *testing.T) {
	snapshot := Build(
		"feat/auth",
		git.WorktreeStatus{},
		resumepkg.Bundle{ActiveTaskAmbiguous: true},
		nil,
	)

	want := "Branch: feat/auth\nWorktree: clean\nActive task: ambiguous"
	if snapshot.Summary != want {
		t.Fatalf("unexpected minimal handoff summary:\n%s", snapshot.Summary)
	}
}

func TestLimitOpenTasks(t *testing.T) {
	tasks := []store.Task{
		handoffTask(1, "", store.TaskOpen, "One"),
		handoffTask(2, "", store.TaskDone, "Done"),
		handoffTask(3, "", store.TaskBlocked, "Two"),
		handoffTask(4, "", store.TaskOpen, "Three"),
		handoffTask(5, "", store.TaskOpen, "Four"),
		handoffTask(6, "", store.TaskOpen, "Five"),
		handoffTask(7, "", store.TaskOpen, "Six"),
	}

	limited := limitOpenTasks(tasks, 5)
	if len(limited) != 5 {
		t.Fatalf("expected 5 open tasks, got %#v", limited)
	}
	for _, task := range limited {
		if task.Status == store.TaskDone {
			t.Fatalf("did not expect done task in open task list: %#v", limited)
		}
	}
	if limited[4].Text != "Five" {
		t.Fatalf("expected limit to preserve task order, got %#v", limited)
	}
}

func TestWorktreeLine(t *testing.T) {
	tests := []struct {
		name   string
		status git.WorktreeStatus
		want   string
	}{
		{name: "clean", status: git.WorktreeStatus{}, want: "clean"},
		{name: "untracked only", status: git.WorktreeStatus{Dirty: true, Untracked: 2}, want: "dirty (2 untracked)"},
		{name: "changed only", status: git.WorktreeStatus{Dirty: true, Changed: 3}, want: "dirty (3 changed)"},
		{name: "changed and untracked", status: git.WorktreeStatus{Dirty: true, Changed: 3, Untracked: 2}, want: "dirty (3 changed, 2 untracked)"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := worktreeLine(test.status); got != test.want {
				t.Fatalf("worktreeLine(%#v) = %q, want %q", test.status, got, test.want)
			}
		})
	}
}

func TestShortSHA(t *testing.T) {
	if got := shortSHA("abc1234"); got != "abc1234" {
		t.Fatalf("expected unchanged short sha, got %q", got)
	}
	if got := shortSHA("abc123456789"); got != "abc1234" {
		t.Fatalf("expected shortened sha, got %q", got)
	}
}

func TestHandoffOpenQuestions(t *testing.T) {
	questions := handoffOpenQuestions([]string{"One", "Two", "Three"}, git.WorktreeStatus{Dirty: true})
	if len(questions) != 3 || questions[2] != "Three" {
		t.Fatalf("expected existing questions to be kept when already at limit, got %#v", questions)
	}

	questions = handoffOpenQuestions([]string{"One", "Two"}, git.WorktreeStatus{Dirty: true})
	want := []string{"One", "Two", "Should the current uncommitted changes be kept, finished, or discarded?"}
	if len(questions) != len(want) {
		t.Fatalf("unexpected handoff questions: %#v", questions)
	}
	for i := range want {
		if questions[i] != want[i] {
			t.Fatalf("unexpected handoff questions: %#v", questions)
		}
	}
	questions = handoffOpenQuestions(nil, git.WorktreeStatus{})
	if len(questions) != 0 {
		t.Fatalf("expected no questions for clean worktree, got %#v", questions)
	}
}

func taskPointer(task store.Task) *store.Task {
	return &task
}

func handoffTask(id int64, branch string, status store.TaskStatus, text string) store.Task {
	return store.Task{
		ID:        id,
		Branch:    branch,
		Text:      text,
		Status:    status,
		CreatedAt: handoffTime(),
		UpdatedAt: handoffTime(),
	}
}

func handoffNote(id int64, text string) store.Note {
	return store.Note{ID: id, Text: text, CreatedAt: handoffTime()}
}

func handoffDecision(id int64, text string) store.Decision {
	return store.Decision{ID: id, Text: text, CreatedAt: handoffTime()}
}

func handoffCommit(sha, summary string) store.Commit {
	return store.Commit{SHA: sha, Summary: summary, CommittedAt: handoffTime()}
}

func handoffTime() time.Time {
	return time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
}

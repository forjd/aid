package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"aid/internal/store"
)

func TestStoreCRUDFlow(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "aid.db")

	sqliteStore, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer sqliteStore.Close()

	if err := sqliteStore.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := sqliteStore.Migrate(ctx); err != nil {
		t.Fatalf("migrate should be idempotent: %v", err)
	}

	repo, err := sqliteStore.UpsertRepo(ctx, "/tmp/project", "project")
	if err != nil {
		t.Fatalf("upsert repo: %v", err)
	}

	foundRepo, err := sqliteStore.FindRepoByPath(ctx, "/tmp/project")
	if err != nil {
		t.Fatalf("find repo: %v", err)
	}
	if foundRepo == nil || foundRepo.ID != repo.ID {
		t.Fatalf("expected to find repo %d, got %#v", repo.ID, foundRepo)
	}

	note, err := sqliteStore.AddNote(ctx, store.AddNoteInput{
		RepoID: repo.ID,
		Branch: "main",
		Scope:  store.ScopeBranch,
		Text:   "Refresh token bug occurs after retry",
	})
	if err != nil {
		t.Fatalf("add note: %v", err)
	}
	if note.ID == 0 {
		t.Fatalf("expected note id to be set")
	}

	notes, err := sqliteStore.ListNotes(ctx, repo.ID, 10)
	if err != nil {
		t.Fatalf("list notes: %v", err)
	}
	if len(notes) != 1 || notes[0].Text != note.Text {
		t.Fatalf("unexpected notes: %#v", notes)
	}

	task, err := sqliteStore.AddTask(ctx, store.AddTaskInput{
		RepoID: repo.ID,
		Branch: "main",
		Scope:  store.ScopeBranch,
		Text:   "Fix VAT rounding",
		Status: store.TaskOpen,
	})
	if err != nil {
		t.Fatalf("add task: %v", err)
	}

	tasks, err := sqliteStore.ListTasks(ctx, repo.ID, 10)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Status != store.TaskOpen {
		t.Fatalf("unexpected tasks: %#v", tasks)
	}

	completedTask, err := sqliteStore.CompleteTask(ctx, repo.ID, task.ID)
	if err != nil {
		t.Fatalf("complete task: %v", err)
	}
	if completedTask.Status != store.TaskDone {
		t.Fatalf("expected task status %q, got %q", store.TaskDone, completedTask.Status)
	}

	decision, err := sqliteStore.AddDecision(ctx, store.AddDecisionInput{
		RepoID: repo.ID,
		Branch: "main",
		Text:   "Store money as integer pence",
	})
	if err != nil {
		t.Fatalf("add decision: %v", err)
	}

	decisions, err := sqliteStore.ListDecisions(ctx, repo.ID, 10)
	if err != nil {
		t.Fatalf("list decisions: %v", err)
	}
	if len(decisions) != 1 || decisions[0].Text != decision.Text {
		t.Fatalf("unexpected decisions: %#v", decisions)
	}

	counts, err := sqliteStore.StatusCounts(ctx, repo.ID)
	if err != nil {
		t.Fatalf("status counts: %v", err)
	}
	if counts.Notes != 1 || counts.Decisions != 1 {
		t.Fatalf("unexpected note/decision counts: %#v", counts)
	}
	if counts.Tasks.Total != 1 || counts.Tasks.Done != 1 {
		t.Fatalf("unexpected task counts: %#v", counts.Tasks)
	}
}

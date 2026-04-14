package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

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

	handoff, err := sqliteStore.AddHandoff(ctx, store.AddHandoffInput{
		RepoID:  repo.ID,
		Branch:  "main",
		Summary: "Branch: main",
	})
	if err != nil {
		t.Fatalf("add handoff: %v", err)
	}

	handoffs, err := sqliteStore.ListHandoffs(ctx, repo.ID, 10)
	if err != nil {
		t.Fatalf("list handoffs: %v", err)
	}
	if len(handoffs) != 1 || handoffs[0].ID != handoff.ID {
		t.Fatalf("unexpected handoffs: %#v", handoffs)
	}

	indexedAt := time.Now().UTC()
	if err := sqliteStore.ReplaceCommits(ctx, store.ReplaceCommitsInput{
		RepoID:    repo.ID,
		IndexedAt: indexedAt,
		Commits: []store.Commit{
			{
				SHA:          "abc123",
				Author:       "Dan",
				CommittedAt:  indexedAt,
				Message:      "feat: token refresh retry",
				Summary:      "feat: token refresh retry",
				ChangedPaths: []string{"auth.go"},
			},
		},
	}); err != nil {
		t.Fatalf("replace commits: %v", err)
	}

	commits, err := sqliteStore.SearchCommits(ctx, repo.ID, "refresh", 10)
	if err != nil {
		t.Fatalf("search commits: %v", err)
	}
	if len(commits) != 1 || commits[0].SHA != "abc123" {
		t.Fatalf("unexpected commits: %#v", commits)
	}
}

func TestCommitSearchUsesFTSRankingAndRepairsMissingIndex(t *testing.T) {
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

	repo, err := sqliteStore.UpsertRepo(ctx, "/tmp/project", "project")
	if err != nil {
		t.Fatalf("upsert repo: %v", err)
	}

	indexedAt := time.Now().UTC().Round(time.Second)
	if err := sqliteStore.ReplaceCommits(ctx, store.ReplaceCommitsInput{
		RepoID:    repo.ID,
		IndexedAt: indexedAt,
		Commits: []store.Commit{
			{
				SHA:          "aaa111",
				Author:       "Dan",
				CommittedAt:  indexedAt.Add(-2 * time.Hour),
				Message:      "feat: token refresh retry",
				Summary:      "feat: token refresh retry",
				ChangedPaths: []string{"auth/refresh.go"},
			},
			{
				SHA:          "bbb222",
				Author:       "Dan",
				CommittedAt:  indexedAt.Add(-1 * time.Hour),
				Message:      "chore: tidy auth handlers",
				Summary:      "chore: tidy auth handlers",
				ChangedPaths: []string{"workers/refresh_worker.go"},
			},
			{
				SHA:          "ccc333",
				Author:       "Dan",
				CommittedAt:  indexedAt,
				Message:      "fix: invoice vat reconciliation",
				Summary:      "fix: invoice vat reconciliation",
				ChangedPaths: []string{"billing/invoice.txt"},
			},
		},
	}); err != nil {
		t.Fatalf("replace commits: %v", err)
	}

	commits, err := sqliteStore.SearchCommits(ctx, repo.ID, "refresh", 10)
	if err != nil {
		t.Fatalf("search commits: %v", err)
	}
	if len(commits) < 2 {
		t.Fatalf("expected at least two refresh matches, got %#v", commits)
	}
	if commits[0].SHA != "aaa111" || commits[1].SHA != "bbb222" {
		t.Fatalf("expected summary match before path-only match, got %#v", commits)
	}

	naturalLanguage, err := sqliteStore.SearchCommits(ctx, repo.ID, "why was invoice vat reconciliation added", 10)
	if err != nil {
		t.Fatalf("search commits with natural language query: %v", err)
	}
	if len(naturalLanguage) == 0 || naturalLanguage[0].SHA != "ccc333" {
		t.Fatalf("expected invoice commit for natural language query, got %#v", naturalLanguage)
	}

	if _, err := sqliteStore.db.ExecContext(ctx, `DELETE FROM commits_fts WHERE repo_id = ?`, repo.ID); err != nil {
		t.Fatalf("clear commit fts index: %v", err)
	}

	rebuilt, err := sqliteStore.SearchCommits(ctx, repo.ID, "refresh retry", 10)
	if err != nil {
		t.Fatalf("search commits after clearing fts index: %v", err)
	}
	if len(rebuilt) != 1 || rebuilt[0].SHA != "aaa111" {
		t.Fatalf("expected search to rebuild missing fts index, got %#v", rebuilt)
	}
}

package sqlite

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/forjd/aid/internal/store"
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

	notes, err := sqliteStore.ListNotes(ctx, repo.ID, "main", 10)
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

	tasks, err := sqliteStore.ListTasks(ctx, repo.ID, "main", 10)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Status != store.TaskOpen {
		t.Fatalf("unexpected tasks: %#v", tasks)
	}

	inProgressTask, err := sqliteStore.UpdateTaskStatus(ctx, repo.ID, task.ID, store.TaskInProgress)
	if err != nil {
		t.Fatalf("mark task in progress: %v", err)
	}
	if inProgressTask.Status != store.TaskInProgress {
		t.Fatalf("expected task status %q, got %q", store.TaskInProgress, inProgressTask.Status)
	}

	blockedTask, err := sqliteStore.UpdateTaskStatus(ctx, repo.ID, task.ID, store.TaskBlocked)
	if err != nil {
		t.Fatalf("mark task blocked: %v", err)
	}
	if blockedTask.Status != store.TaskBlocked {
		t.Fatalf("expected task status %q, got %q", store.TaskBlocked, blockedTask.Status)
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

	decisions, err := sqliteStore.ListDecisions(ctx, repo.ID, "main", 10)
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

	handoffs, err := sqliteStore.ListHandoffs(ctx, repo.ID, "main", 10)
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

	listedCommits, err := sqliteStore.ListCommits(ctx, repo.ID, 10)
	if err != nil {
		t.Fatalf("list commits: %v", err)
	}
	if len(listedCommits) != 1 || listedCommits[0].SHA != "abc123" {
		t.Fatalf("unexpected listed commits: %#v", listedCommits)
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

func TestContextSearchUsesFTSRankingAndRepairsMissingIndexes(t *testing.T) {
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

	if _, err := sqliteStore.AddNote(ctx, store.AddNoteInput{
		RepoID: repo.ID,
		Branch: "feature",
		Scope:  store.ScopeBranch,
		Text:   "Refresh retry path investigation",
	}); err != nil {
		t.Fatalf("add feature note: %v", err)
	}
	if _, err := sqliteStore.AddNote(ctx, store.AddNoteInput{
		RepoID: repo.ID,
		Branch: "main",
		Scope:  store.ScopeBranch,
		Text:   "Refresh retry fails after 401 retry",
	}); err != nil {
		t.Fatalf("add main note: %v", err)
	}
	if _, err := sqliteStore.AddNote(ctx, store.AddNoteInput{
		RepoID: repo.ID,
		Branch: "main",
		Scope:  store.ScopeBranch,
		Text:   "Refresh worker logging needs cleanup",
	}); err != nil {
		t.Fatalf("add weaker note: %v", err)
	}

	rationale := "Retry the refresh flow only once after a 401 response"
	if _, err := sqliteStore.AddDecision(ctx, store.AddDecisionInput{
		RepoID:    repo.ID,
		Branch:    "main",
		Text:      "Treat tokens as single-use",
		Rationale: &rationale,
	}); err != nil {
		t.Fatalf("add decision with rationale: %v", err)
	}
	if _, err := sqliteStore.AddDecision(ctx, store.AddDecisionInput{
		RepoID: repo.ID,
		Branch: "main",
		Text:   "Refresh policy housekeeping",
	}); err != nil {
		t.Fatalf("add weaker decision: %v", err)
	}

	if _, err := sqliteStore.AddHandoff(ctx, store.AddHandoffInput{
		RepoID:  repo.ID,
		Branch:  "main",
		Summary: "Open questions:\n- Why does refresh retry fail after 401?",
	}); err != nil {
		t.Fatalf("add handoff: %v", err)
	}
	if _, err := sqliteStore.AddHandoff(ctx, store.AddHandoffInput{
		RepoID:  repo.ID,
		Branch:  "main",
		Summary: "Billing handoff for VAT work",
	}); err != nil {
		t.Fatalf("add weaker handoff: %v", err)
	}

	if _, err := sqliteStore.db.ExecContext(ctx, `DELETE FROM notes_fts WHERE repo_id = ?`, repo.ID); err != nil {
		t.Fatalf("clear note fts index: %v", err)
	}

	notes, err := sqliteStore.SearchNotes(ctx, repo.ID, "main", "refresh retry", 10)
	if err != nil {
		t.Fatalf("search notes: %v", err)
	}
	if len(notes) == 0 || notes[0].Text != "Refresh retry fails after 401 retry" {
		t.Fatalf("expected note search to rank the main-branch exact match first, got %#v", notes)
	}
	for _, note := range notes {
		if note.Branch == "feature" {
			t.Fatalf("expected feature-branch note to be filtered out when searching from main, got %#v", notes)
		}
	}

	decisions, err := sqliteStore.SearchDecisions(ctx, repo.ID, "main", "401 retry", 10)
	if err != nil {
		t.Fatalf("search decisions: %v", err)
	}
	if len(decisions) == 0 || decisions[0].Text != "Treat tokens as single-use" {
		t.Fatalf("expected decision search to use indexed rationale text, got %#v", decisions)
	}

	handoffs, err := sqliteStore.SearchHandoffs(ctx, repo.ID, "main", "refresh retry", 10)
	if err != nil {
		t.Fatalf("search handoffs: %v", err)
	}
	if len(handoffs) == 0 || handoffs[0].Summary != "Open questions:\n- Why does refresh retry fail after 401?" {
		t.Fatalf("expected handoff search to rank the matching handoff first, got %#v", handoffs)
	}
}

func TestMigrateTracksSchemaVersion(t *testing.T) {
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
		t.Fatalf("second migrate: %v", err)
	}

	var version int
	if err := sqliteStore.db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatalf("read schema version: %v", err)
	}
	if version != schemaVersion {
		t.Fatalf("expected schema version %d, got %d", schemaVersion, version)
	}
}

func TestSeparateStoreInstancesHandleConcurrentRecallLikeReads(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "aid.db")

	seedStore, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open seed store: %v", err)
	}
	defer seedStore.Close()

	if err := seedStore.Migrate(ctx); err != nil {
		t.Fatalf("migrate seed store: %v", err)
	}

	repo, err := seedStore.UpsertRepo(ctx, "/tmp/project", "project")
	if err != nil {
		t.Fatalf("upsert repo: %v", err)
	}

	if _, err := seedStore.AddNote(ctx, store.AddNoteInput{
		RepoID: repo.ID,
		Branch: "main",
		Scope:  store.ScopeBranch,
		Text:   "Refresh retry bug investigation",
	}); err != nil {
		t.Fatalf("add note: %v", err)
	}

	rationale := "Retry token refresh only once after a 401 response"
	if _, err := seedStore.AddDecision(ctx, store.AddDecisionInput{
		RepoID:    repo.ID,
		Branch:    "main",
		Text:      "Use single refresh retry",
		Rationale: &rationale,
	}); err != nil {
		t.Fatalf("add decision: %v", err)
	}

	if _, err := seedStore.AddHandoff(ctx, store.AddHandoffInput{
		RepoID:  repo.ID,
		Branch:  "main",
		Summary: "Recent work focused on refresh retry failures.",
	}); err != nil {
		t.Fatalf("add handoff: %v", err)
	}

	indexedAt := time.Now().UTC().Round(time.Second)
	if err := seedStore.ReplaceCommits(ctx, store.ReplaceCommitsInput{
		RepoID:    repo.ID,
		IndexedAt: indexedAt,
		Commits: []store.Commit{
			{
				SHA:          "aaa111",
				Author:       "Dan",
				CommittedAt:  indexedAt,
				Message:      "fix: token refresh retry",
				Summary:      "fix: token refresh retry",
				ChangedPaths: []string{"auth/refresh.go"},
			},
		},
	}); err != nil {
		t.Fatalf("replace commits: %v", err)
	}

	runRecallLikeRead := func(query string) error {
		sqliteStore, err := Open(dbPath)
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}
		defer sqliteStore.Close()

		if err := sqliteStore.Migrate(ctx); err != nil {
			return fmt.Errorf("migrate store: %w", err)
		}

		foundRepo, err := sqliteStore.FindRepoByPath(ctx, "/tmp/project")
		if err != nil {
			return fmt.Errorf("find repo: %w", err)
		}
		if foundRepo == nil {
			return fmt.Errorf("repo missing after concurrent open")
		}

		if _, err := sqliteStore.SearchNotes(ctx, foundRepo.ID, "main", query, 10); err != nil {
			return fmt.Errorf("search notes: %w", err)
		}
		if _, err := sqliteStore.SearchDecisions(ctx, foundRepo.ID, "main", query, 10); err != nil {
			return fmt.Errorf("search decisions: %w", err)
		}
		if _, err := sqliteStore.SearchHandoffs(ctx, foundRepo.ID, "main", query, 10); err != nil {
			return fmt.Errorf("search handoffs: %w", err)
		}
		if _, err := sqliteStore.SearchCommits(ctx, foundRepo.ID, query, 10); err != nil {
			return fmt.Errorf("search commits: %w", err)
		}

		return nil
	}

	queries := []string{"refresh retry", "token refresh"}
	for i := 0; i < 20; i++ {
		var wg sync.WaitGroup
		errCh := make(chan error, len(queries))

		for _, query := range queries {
			query := query
			wg.Add(1)
			go func() {
				defer wg.Done()
				errCh <- runRecallLikeRead(query)
			}()
		}

		wg.Wait()
		close(errCh)

		for err := range errCh {
			if err != nil {
				t.Fatalf("concurrent recall-like read failed on iteration %d: %v", i+1, err)
			}
		}
	}
}

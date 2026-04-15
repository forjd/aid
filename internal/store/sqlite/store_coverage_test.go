package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/forjd/aid/internal/store"
)

func TestFindRepoByPathReturnsNilWhenMissing(t *testing.T) {
	ctx := context.Background()
	sqliteStore := openTestStore(t)
	defer sqliteStore.Close()

	repo, err := sqliteStore.FindRepoByPath(ctx, "/tmp/missing")
	if err != nil {
		t.Fatalf("find missing repo: %v", err)
	}
	if repo != nil {
		t.Fatalf("expected no repo, got %#v", repo)
	}
}

func TestUpdateTaskStatusRejectsInvalidAndMissingTasks(t *testing.T) {
	ctx := context.Background()
	sqliteStore := openTestStore(t)
	defer sqliteStore.Close()

	repo, err := sqliteStore.UpsertRepo(ctx, "/tmp/project", "project")
	if err != nil {
		t.Fatalf("upsert repo: %v", err)
	}

	if _, err := sqliteStore.UpdateTaskStatus(ctx, repo.ID, 1, store.TaskStatus("bad")); err == nil {
		t.Fatalf("expected invalid task status error")
	}

	if _, err := sqliteStore.UpdateTaskStatus(ctx, repo.ID, 999, store.TaskDone); err == nil {
		t.Fatalf("expected missing task error")
	}
}

func TestStatusCountsIncludesEveryTaskState(t *testing.T) {
	ctx := context.Background()
	sqliteStore := openTestStore(t)
	defer sqliteStore.Close()

	repo, err := sqliteStore.UpsertRepo(ctx, "/tmp/project", "project")
	if err != nil {
		t.Fatalf("upsert repo: %v", err)
	}

	if _, err := sqliteStore.AddNote(ctx, store.AddNoteInput{
		RepoID: repo.ID,
		Scope:  store.ScopeRepo,
		Text:   "refresh retry note",
	}); err != nil {
		t.Fatalf("add note: %v", err)
	}

	if _, err := sqliteStore.AddDecision(ctx, store.AddDecisionInput{
		RepoID: repo.ID,
		Text:   "keep retry logic simple",
	}); err != nil {
		t.Fatalf("add decision: %v", err)
	}

	statuses := []store.TaskStatus{
		store.TaskOpen,
		store.TaskInProgress,
		store.TaskBlocked,
		store.TaskDone,
	}
	for i, status := range statuses {
		if _, err := sqliteStore.AddTask(ctx, store.AddTaskInput{
			RepoID: repo.ID,
			Scope:  store.ScopeRepo,
			Text:   "task",
			Status: status,
		}); err != nil {
			t.Fatalf("add task %d: %v", i+1, err)
		}
	}

	counts, err := sqliteStore.StatusCounts(ctx, repo.ID)
	if err != nil {
		t.Fatalf("status counts: %v", err)
	}

	if counts.Notes != 1 || counts.Decisions != 1 {
		t.Fatalf("unexpected note or decision counts: %#v", counts)
	}
	if counts.Tasks.Total != 4 || counts.Tasks.Open != 1 || counts.Tasks.InProgress != 1 || counts.Tasks.Blocked != 1 || counts.Tasks.Done != 1 {
		t.Fatalf("unexpected task counts: %#v", counts.Tasks)
	}
}

func TestSyncCommitsTracksAddUpdateRemoveAndNoop(t *testing.T) {
	ctx := context.Background()
	sqliteStore := openTestStore(t)
	defer sqliteStore.Close()

	repo, err := sqliteStore.UpsertRepo(ctx, "/tmp/project", "project")
	if err != nil {
		t.Fatalf("upsert repo: %v", err)
	}

	indexedAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	initialCommits := []store.Commit{
		{
			SHA:          "aaa111",
			Author:       "Dan",
			GitOrder:     0,
			CommittedAt:  indexedAt.Add(-2 * time.Hour),
			Message:      "feat: initial refresh retry",
			Summary:      "feat: initial refresh retry",
			ChangedPaths: []string{"auth/refresh.go"},
		},
		{
			SHA:          "bbb222",
			Author:       "Dan",
			GitOrder:     1,
			CommittedAt:  indexedAt.Add(-1 * time.Hour),
			Message:      "chore: add support docs",
			Summary:      "chore: add support docs",
			ChangedPaths: []string{"docs/aid.md"},
		},
	}

	result, err := sqliteStore.SyncCommits(ctx, store.SyncCommitsInput{
		RepoID:    repo.ID,
		Commits:   initialCommits,
		IndexedAt: indexedAt,
	})
	if err != nil {
		t.Fatalf("initial sync commits: %v", err)
	}
	if !result.Initial || result.Added != 2 || result.Updated != 0 || result.Removed != 0 || result.Total != 2 {
		t.Fatalf("unexpected initial sync result: %#v", result)
	}

	updatedCommits := []store.Commit{
		{
			SHA:          "aaa111",
			Author:       "Dan",
			GitOrder:     0,
			CommittedAt:  indexedAt.Add(-2 * time.Hour),
			Message:      "fix: tighten refresh retry",
			Summary:      "fix: tighten refresh retry",
			ChangedPaths: []string{"auth/refresh.go", "auth/session.go"},
		},
		{
			SHA:          "ccc333",
			Author:       "Dan",
			GitOrder:     1,
			CommittedAt:  indexedAt,
			Message:      "feat: add recall ranking",
			Summary:      "feat: add recall ranking",
			ChangedPaths: []string{"internal/search/search.go"},
		},
	}

	result, err = sqliteStore.SyncCommits(ctx, store.SyncCommitsInput{
		RepoID:    repo.ID,
		Commits:   updatedCommits,
		IndexedAt: indexedAt.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("update sync commits: %v", err)
	}
	if result.Initial || result.Added != 1 || result.Updated != 1 || result.Removed != 1 || result.Total != 2 {
		t.Fatalf("unexpected incremental sync result: %#v", result)
	}

	noop, err := sqliteStore.SyncCommits(ctx, store.SyncCommitsInput{
		RepoID:    repo.ID,
		Commits:   updatedCommits,
		IndexedAt: indexedAt.Add(2 * time.Hour),
	})
	if err != nil {
		t.Fatalf("noop sync commits: %v", err)
	}
	if noop.Added != 0 || noop.Updated != 0 || noop.Removed != 0 || noop.Total != 2 {
		t.Fatalf("unexpected noop sync result: %#v", noop)
	}

	listed, err := sqliteStore.ListCommits(ctx, repo.ID, 10)
	if err != nil {
		t.Fatalf("list commits: %v", err)
	}
	if len(listed) != 2 || listed[0].SHA != "ccc333" || listed[1].SHA != "aaa111" {
		t.Fatalf("unexpected listed commits after sync: %#v", listed)
	}

	found, err := sqliteStore.SearchCommits(ctx, repo.ID, "tighten refresh retry", 10)
	if err != nil {
		t.Fatalf("search commits: %v", err)
	}
	if len(found) == 0 || found[0].SHA != "aaa111" {
		t.Fatalf("expected updated commit to be searchable, got %#v", found)
	}
}

func TestSyncCommitsDeduplicatesRepeatedSHAs(t *testing.T) {
	ctx := context.Background()
	sqliteStore := openTestStore(t)
	defer sqliteStore.Close()

	repo, err := sqliteStore.UpsertRepo(ctx, "/tmp/project", "project")
	if err != nil {
		t.Fatalf("upsert repo: %v", err)
	}

	indexedAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	result, err := sqliteStore.SyncCommits(ctx, store.SyncCommitsInput{
		RepoID: repo.ID,
		Commits: []store.Commit{
			{
				SHA:          "aaa111",
				Author:       "Dan",
				GitOrder:     0,
				CommittedAt:  indexedAt,
				Message:      "feat: add retry",
				Summary:      "feat: add retry",
				ChangedPaths: []string{"auth.go"},
			},
			{
				SHA:          "aaa111",
				Author:       "Dan",
				GitOrder:     0,
				CommittedAt:  indexedAt,
				Message:      "feat: add retry",
				Summary:      "feat: add retry",
				ChangedPaths: []string{"auth.go"},
			},
			{
				SHA:          "bbb222",
				Author:       "Dan",
				GitOrder:     1,
				CommittedAt:  indexedAt.Add(time.Minute),
				Message:      "fix: tighten retry",
				Summary:      "fix: tighten retry",
				ChangedPaths: []string{"session.go"},
			},
		},
		IndexedAt: indexedAt,
	})
	if err != nil {
		t.Fatalf("sync duplicate commits: %v", err)
	}
	if result.Added != 2 || result.Total != 2 {
		t.Fatalf("expected duplicate shas to be ignored, got %#v", result)
	}
}

func TestSearchIgnoresPunctuationOnlyQueries(t *testing.T) {
	ctx := context.Background()
	sqliteStore := openTestStore(t)
	defer sqliteStore.Close()

	repo, err := sqliteStore.UpsertRepo(ctx, "/tmp/project", "project")
	if err != nil {
		t.Fatalf("upsert repo: %v", err)
	}

	if _, err := sqliteStore.AddNote(ctx, store.AddNoteInput{
		RepoID: repo.ID,
		Scope:  store.ScopeRepo,
		Text:   "refresh retry note",
	}); err != nil {
		t.Fatalf("add note: %v", err)
	}

	for _, query := range []string{":-)", "!!!"} {
		notes, err := sqliteStore.SearchNotes(ctx, repo.ID, "main", query, 10)
		if err != nil {
			t.Fatalf("search notes for %q: %v", query, err)
		}
		if len(notes) != 0 {
			t.Fatalf("expected no notes for punctuation-only query %q, got %#v", query, notes)
		}

		decisions, err := sqliteStore.SearchDecisions(ctx, repo.ID, "main", query, 10)
		if err != nil {
			t.Fatalf("search decisions for %q: %v", query, err)
		}
		if len(decisions) != 0 {
			t.Fatalf("expected no decisions for punctuation-only query %q, got %#v", query, decisions)
		}

		handoffs, err := sqliteStore.SearchHandoffs(ctx, repo.ID, "main", query, 10)
		if err != nil {
			t.Fatalf("search handoffs for %q: %v", query, err)
		}
		if len(handoffs) != 0 {
			t.Fatalf("expected no handoffs for punctuation-only query %q, got %#v", query, handoffs)
		}

		commits, err := sqliteStore.SearchCommits(ctx, repo.ID, query, 10)
		if err != nil {
			t.Fatalf("search commits for %q: %v", query, err)
		}
		if len(commits) != 0 {
			t.Fatalf("expected no commits for punctuation-only query %q, got %#v", query, commits)
		}
	}
}

func TestSearchHandlesUnicodeQueries(t *testing.T) {
	ctx := context.Background()
	sqliteStore := openTestStore(t)
	defer sqliteStore.Close()

	repo, err := sqliteStore.UpsertRepo(ctx, "/tmp/project", "project")
	if err != nil {
		t.Fatalf("upsert repo: %v", err)
	}

	if _, err := sqliteStore.AddNote(ctx, store.AddNoteInput{
		RepoID: repo.ID,
		Scope:  store.ScopeRepo,
		Text:   "café retry investigation",
	}); err != nil {
		t.Fatalf("add unicode note: %v", err)
	}
	if _, err := sqliteStore.AddDecision(ctx, store.AddDecisionInput{
		RepoID: repo.ID,
		Text:   "keep café token refresh simple",
	}); err != nil {
		t.Fatalf("add unicode decision: %v", err)
	}
	if _, err := sqliteStore.AddHandoff(ctx, store.AddHandoffInput{
		RepoID:  repo.ID,
		Summary: "Follow up on café retry failure",
	}); err != nil {
		t.Fatalf("add unicode handoff: %v", err)
	}
	indexedAt := time.Date(2026, time.April, 15, 12, 0, 0, 0, time.UTC)
	if err := sqliteStore.ReplaceCommits(ctx, store.ReplaceCommitsInput{
		RepoID:    repo.ID,
		IndexedAt: indexedAt,
		Commits: []store.Commit{
			{
				SHA:          "cafe123",
				Author:       "Dan",
				CommittedAt:  indexedAt,
				Message:      "fix: café retry path",
				Summary:      "fix: café retry path",
				ChangedPaths: []string{"auth/cafe.go"},
			},
		},
	}); err != nil {
		t.Fatalf("replace unicode commits: %v", err)
	}

	notes, err := sqliteStore.SearchNotes(ctx, repo.ID, "main", "café", 10)
	if err != nil || len(notes) != 1 {
		t.Fatalf("unexpected unicode note search result: %#v, err=%v", notes, err)
	}
	decisions, err := sqliteStore.SearchDecisions(ctx, repo.ID, "main", "café", 10)
	if err != nil || len(decisions) != 1 {
		t.Fatalf("unexpected unicode decision search result: %#v, err=%v", decisions, err)
	}
	handoffs, err := sqliteStore.SearchHandoffs(ctx, repo.ID, "main", "café", 10)
	if err != nil || len(handoffs) != 1 {
		t.Fatalf("unexpected unicode handoff search result: %#v, err=%v", handoffs, err)
	}
	commits, err := sqliteStore.SearchCommits(ctx, repo.ID, "café", 10)
	if err != nil || len(commits) != 1 {
		t.Fatalf("unexpected unicode commit search result: %#v, err=%v", commits, err)
	}
}

func TestDecisionAndHandoffSearchRepairMissingFTSIndexes(t *testing.T) {
	ctx := context.Background()
	sqliteStore := openTestStore(t)
	defer sqliteStore.Close()

	repo, err := sqliteStore.UpsertRepo(ctx, "/tmp/project", "project")
	if err != nil {
		t.Fatalf("upsert repo: %v", err)
	}

	rationale := "Retry refresh only once after a 401 response"
	if _, err := sqliteStore.AddDecision(ctx, store.AddDecisionInput{
		RepoID:    repo.ID,
		Branch:    "main",
		Text:      "Treat refresh tokens as single use",
		Rationale: &rationale,
	}); err != nil {
		t.Fatalf("add decision: %v", err)
	}

	if _, err := sqliteStore.AddHandoff(ctx, store.AddHandoffInput{
		RepoID:  repo.ID,
		Branch:  "main",
		Summary: "Open questions:\n- Why does the refresh retry still fail?",
	}); err != nil {
		t.Fatalf("add handoff: %v", err)
	}

	if _, err := sqliteStore.db.ExecContext(ctx, `DELETE FROM decisions_fts WHERE repo_id = ?`, repo.ID); err != nil {
		t.Fatalf("clear decision fts: %v", err)
	}
	if _, err := sqliteStore.db.ExecContext(ctx, `DELETE FROM handoffs_fts WHERE repo_id = ?`, repo.ID); err != nil {
		t.Fatalf("clear handoff fts: %v", err)
	}

	decisions, err := sqliteStore.SearchDecisions(ctx, repo.ID, "main", "401 refresh retry", 10)
	if err != nil {
		t.Fatalf("search decisions: %v", err)
	}
	if len(decisions) != 1 || decisions[0].Text != "Treat refresh tokens as single use" {
		t.Fatalf("unexpected decisions after repair: %#v", decisions)
	}

	handoffs, err := sqliteStore.SearchHandoffs(ctx, repo.ID, "main", "refresh retry fail", 10)
	if err != nil {
		t.Fatalf("search handoffs: %v", err)
	}
	if len(handoffs) != 1 || handoffs[0].Summary != "Open questions:\n- Why does the refresh retry still fail?" {
		t.Fatalf("unexpected handoffs after repair: %#v", handoffs)
	}
}

func TestSearchFunctionsReturnNilForStopWordOnlyQueries(t *testing.T) {
	ctx := context.Background()
	sqliteStore := openTestStore(t)
	defer sqliteStore.Close()

	repo, err := sqliteStore.UpsertRepo(ctx, "/tmp/project", "project")
	if err != nil {
		t.Fatalf("upsert repo: %v", err)
	}

	for _, search := range []struct {
		name string
		run  func() error
	}{
		{
			name: "commits",
			run: func() error {
				results, err := sqliteStore.SearchCommits(ctx, repo.ID, "why the and", 10)
				if err != nil {
					return err
				}
				if results != nil {
					t.Fatalf("expected nil commit results, got %#v", results)
				}
				return nil
			},
		},
		{
			name: "notes",
			run: func() error {
				results, err := sqliteStore.SearchNotes(ctx, repo.ID, "main", "why the and", 10)
				if err != nil {
					return err
				}
				if results != nil {
					t.Fatalf("expected nil note results, got %#v", results)
				}
				return nil
			},
		},
		{
			name: "decisions",
			run: func() error {
				results, err := sqliteStore.SearchDecisions(ctx, repo.ID, "main", "why the and", 10)
				if err != nil {
					return err
				}
				if results != nil {
					t.Fatalf("expected nil decision results, got %#v", results)
				}
				return nil
			},
		},
		{
			name: "handoffs",
			run: func() error {
				results, err := sqliteStore.SearchHandoffs(ctx, repo.ID, "main", "why the and", 10)
				if err != nil {
					return err
				}
				if results != nil {
					t.Fatalf("expected nil handoff results, got %#v", results)
				}
				return nil
			},
		},
	} {
		if err := search.run(); err != nil {
			t.Fatalf("search %s: %v", search.name, err)
		}
	}
}

func TestHelperFunctions(t *testing.T) {
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	existing := store.Commit{
		SHA:          "aaa111",
		Author:       "Dan",
		GitOrder:     1,
		Summary:      "feat: add retry",
		Message:      "feat: add retry",
		CommittedAt:  now,
		ChangedPaths: []string{"auth.go"},
	}

	if !sameCommit(existing, existing) {
		t.Fatalf("expected identical commits to match")
	}

	changed := existing
	changed.ChangedPaths = []string{"auth.go", "session.go"}
	if sameCommit(existing, changed) {
		t.Fatalf("expected changed paths to break sameCommit")
	}

	if parsed := parseTime("2026-01-02T03:04:05Z"); !parsed.Equal(now) {
		t.Fatalf("unexpected parsed time: %v", parsed)
	}
	if parsed := parseTime("not-a-time"); !parsed.IsZero() {
		t.Fatalf("expected zero time for invalid input, got %v", parsed)
	}

	if value := nullableString(""); value != nil {
		t.Fatalf("expected nil nullable string, got %#v", value)
	}
	if value := nullableString("main"); value != "main" {
		t.Fatalf("expected string nullable value, got %#v", value)
	}

	if !isValidTaskStatus(store.TaskOpen) || !isValidTaskStatus(store.TaskInProgress) || !isValidTaskStatus(store.TaskBlocked) || !isValidTaskStatus(store.TaskDone) {
		t.Fatalf("expected built-in task states to be valid")
	}
	if isValidTaskStatus(store.TaskStatus("bad")) {
		t.Fatalf("expected arbitrary task state to be invalid")
	}
}

func TestCurrentSchemaVersion(t *testing.T) {
	ctx := context.Background()
	sqliteStore := openTestStore(t)
	defer sqliteStore.Close()

	version, err := currentSchemaVersion(ctx, sqliteStore.db)
	if err != nil {
		t.Fatalf("current schema version: %v", err)
	}
	if version != schemaVersion {
		t.Fatalf("expected schema version %d, got %d", schemaVersion, version)
	}
}

func TestOpenFailsWhenParentPathIsAFile(t *testing.T) {
	parentFile := filepath.Join(t.TempDir(), "parent")
	if err := os.WriteFile(parentFile, []byte("file\n"), 0o644); err != nil {
		t.Fatalf("write parent file: %v", err)
	}

	_, err := Open(filepath.Join(parentFile, "aid.db"))
	if err == nil || !strings.Contains(err.Error(), "create app data directory") {
		t.Fatalf("expected open parent path error, got %v", err)
	}
}

func TestFTSHelpersWrapExecErrors(t *testing.T) {
	ctx := context.Background()
	note := store.Note{ID: 1, RepoID: 2, Branch: "main", Text: "refresh retry", CreatedAt: time.Now().UTC()}
	decision := store.Decision{ID: 1, RepoID: 2, Branch: "main", Text: "single-use token", CreatedAt: time.Now().UTC()}
	handoff := store.Handoff{ID: 1, RepoID: 2, Branch: "main", Summary: "handoff", CreatedAt: time.Now().UTC()}

	cases := []struct {
		name string
		run  func(exec execer) error
		want string
	}{
		{
			name: "insert note",
			run: func(exec execer) error {
				return insertNoteFTS(ctx, exec, note)
			},
			want: "index note for search",
		},
		{
			name: "insert decision",
			run: func(exec execer) error {
				return insertDecisionFTS(ctx, exec, decision)
			},
			want: "index decision for search",
		},
		{
			name: "insert handoff",
			run: func(exec execer) error {
				return insertHandoffFTS(ctx, exec, handoff)
			},
			want: "index handoff for search",
		},
		{
			name: "rebuild commits delete",
			run: func(exec execer) error {
				return rebuildCommitFTS(ctx, exec, 2)
			},
			want: "clear commit fts index",
		},
		{
			name: "rebuild notes delete",
			run: func(exec execer) error {
				return rebuildNoteFTS(ctx, exec, 2)
			},
			want: "clear note fts index",
		},
		{
			name: "rebuild decisions delete",
			run: func(exec execer) error {
				return rebuildDecisionFTS(ctx, exec, 2)
			},
			want: "clear decision fts index",
		},
		{
			name: "rebuild handoffs delete",
			run: func(exec execer) error {
				return rebuildHandoffFTS(ctx, exec, 2)
			},
			want: "clear handoff fts index",
		},
	}

	for _, tc := range cases {
		err := tc.run(&errorExecer{failOnCall: 1})
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: expected %q error, got %v", tc.name, tc.want, err)
		}
	}

	rebuildCases := []struct {
		name string
		run  func(exec execer) error
		want string
	}{
		{
			name: "rebuild commits insert",
			run: func(exec execer) error {
				return rebuildCommitFTS(ctx, exec, 2)
			},
			want: "rebuild commit fts index",
		},
		{
			name: "rebuild notes insert",
			run: func(exec execer) error {
				return rebuildNoteFTS(ctx, exec, 2)
			},
			want: "rebuild note fts index",
		},
		{
			name: "rebuild decisions insert",
			run: func(exec execer) error {
				return rebuildDecisionFTS(ctx, exec, 2)
			},
			want: "rebuild decision fts index",
		},
		{
			name: "rebuild handoffs insert",
			run: func(exec execer) error {
				return rebuildHandoffFTS(ctx, exec, 2)
			},
			want: "rebuild handoff fts index",
		},
	}

	for _, tc := range rebuildCases {
		err := tc.run(&errorExecer{failOnCall: 2})
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: expected %q error, got %v", tc.name, tc.want, err)
		}
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "aid.db")
	sqliteStore, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := sqliteStore.Migrate(ctx); err != nil {
		t.Fatalf("migrate store: %v", err)
	}

	return sqliteStore
}

type errorExecer struct {
	callCount  int
	failOnCall int
}

func (e *errorExecer) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	e.callCount++
	if e.callCount == e.failOnCall {
		return nil, errors.New("boom")
	}

	return nil, nil
}

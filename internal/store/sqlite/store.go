package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/forjd/aid/internal/store"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

const schemaVersion = 3

var migrations = [][]string{
	{
		`CREATE TABLE IF NOT EXISTS repos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS notes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id INTEGER NOT NULL,
			branch TEXT,
			scope TEXT NOT NULL,
			text TEXT NOT NULL,
			tags TEXT,
			created_at TEXT NOT NULL,
			FOREIGN KEY (repo_id) REFERENCES repos(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id INTEGER NOT NULL,
			branch TEXT,
			scope TEXT NOT NULL,
			text TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (repo_id) REFERENCES repos(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS decisions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id INTEGER NOT NULL,
			branch TEXT,
			text TEXT NOT NULL,
			rationale TEXT,
			created_at TEXT NOT NULL,
			FOREIGN KEY (repo_id) REFERENCES repos(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS handoffs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id INTEGER NOT NULL,
			branch TEXT,
			summary TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY (repo_id) REFERENCES repos(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS commits (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id INTEGER NOT NULL,
			sha TEXT NOT NULL,
			author TEXT NOT NULL,
			committed_at TEXT NOT NULL,
			message TEXT NOT NULL,
			changed_paths TEXT NOT NULL,
			summary TEXT NOT NULL,
			indexed_at TEXT NOT NULL,
			FOREIGN KEY (repo_id) REFERENCES repos(id) ON DELETE CASCADE
		)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS commits_fts USING fts5(
			summary,
			message,
			changed_paths,
			sha,
			commit_id UNINDEXED,
			repo_id UNINDEXED,
			tokenize = 'porter unicode61'
		)`,
		`CREATE TABLE IF NOT EXISTS search_chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id INTEGER NOT NULL,
			source_type TEXT NOT NULL,
			source_id TEXT NOT NULL,
			text TEXT NOT NULL,
			embedding BLOB,
			created_at TEXT NOT NULL,
			FOREIGN KEY (repo_id) REFERENCES repos(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notes_repo_created_at ON notes(repo_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_repo_updated_at ON tasks(repo_id, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_decisions_repo_created_at ON decisions(repo_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_handoffs_repo_created_at ON handoffs(repo_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_commits_repo_committed_at ON commits(repo_id, committed_at DESC)`,
	},
	{
		`CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
			text,
			note_id UNINDEXED,
			repo_id UNINDEXED,
			branch UNINDEXED,
			created_at UNINDEXED,
			tokenize = 'porter unicode61'
		)`,
		`INSERT INTO notes_fts (text, note_id, repo_id, branch, created_at)
		 SELECT text, id, repo_id, COALESCE(branch, ''), created_at
		 FROM notes`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS decisions_fts USING fts5(
			text,
			rationale,
			decision_id UNINDEXED,
			repo_id UNINDEXED,
			branch UNINDEXED,
			created_at UNINDEXED,
			tokenize = 'porter unicode61'
		)`,
		`INSERT INTO decisions_fts (text, rationale, decision_id, repo_id, branch, created_at)
		 SELECT text, COALESCE(rationale, ''), id, repo_id, COALESCE(branch, ''), created_at
		 FROM decisions`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS handoffs_fts USING fts5(
			summary,
			handoff_id UNINDEXED,
			repo_id UNINDEXED,
			branch UNINDEXED,
			created_at UNINDEXED,
			tokenize = 'porter unicode61'
		)`,
		`INSERT INTO handoffs_fts (summary, handoff_id, repo_id, branch, created_at)
		 SELECT summary, id, repo_id, COALESCE(branch, ''), created_at
		 FROM handoffs`,
	},
	{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_commits_repo_sha_unique ON commits(repo_id, sha)`,
	},
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create app data directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	store := &Store{db: db}
	if err := store.configure(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) configure() error {
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
	}

	for _, pragma := range pragmas {
		if _, err := s.db.Exec(pragma); err != nil {
			return fmt.Errorf("configure sqlite: %w", err)
		}
	}

	return nil
}

func (s *Store) Migrate(ctx context.Context) error {
	version, err := currentSchemaVersion(ctx, s.db)
	if err != nil {
		return err
	}

	for version < schemaVersion {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration transaction: %w", err)
		}

		migration := migrations[version]
		for _, statement := range migration {
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("run migration %d: %w", version+1, err)
			}
		}

		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`PRAGMA user_version = %d`, version+1)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("set schema version: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", version+1, err)
		}
		version++
	}

	return nil
}

func (s *Store) UpsertRepo(ctx context.Context, path string, name string) (store.Repo, error) {
	now := nowUTC()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO repos (path, name, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			name = excluded.name,
			updated_at = excluded.updated_at
	`, path, name, formatTime(now), formatTime(now))
	if err != nil {
		return store.Repo{}, fmt.Errorf("upsert repo: %w", err)
	}

	repo, err := s.FindRepoByPath(ctx, path)
	if err != nil {
		return store.Repo{}, err
	}
	if repo == nil {
		return store.Repo{}, fmt.Errorf("repo %q missing after upsert", path)
	}

	return *repo, nil
}

func (s *Store) FindRepoByPath(ctx context.Context, path string) (*store.Repo, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, path, name, created_at, updated_at
		FROM repos
		WHERE path = ?
	`, path)

	repo, err := scanRepo(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find repo by path: %w", err)
	}

	return &repo, nil
}

func (s *Store) AddNote(ctx context.Context, input store.AddNoteInput) (store.Note, error) {
	now := nowUTC()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.Note{}, fmt.Errorf("begin note transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		INSERT INTO notes (repo_id, branch, scope, text, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, input.RepoID, nullableString(input.Branch), string(input.Scope), input.Text, formatTime(now))
	if err != nil {
		return store.Note{}, fmt.Errorf("add note: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return store.Note{}, fmt.Errorf("note last insert id: %w", err)
	}

	note, err := noteByID(ctx, tx, id)
	if err != nil {
		return store.Note{}, err
	}
	if err := insertNoteFTS(ctx, tx, note); err != nil {
		return store.Note{}, err
	}
	if err := tx.Commit(); err != nil {
		return store.Note{}, fmt.Errorf("commit note transaction: %w", err)
	}

	return note, nil
}

func (s *Store) ListNotes(ctx context.Context, repoID int64, limit int) ([]store.Note, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, repo_id, branch, scope, text, created_at
		FROM notes
		WHERE repo_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?
	`, repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()

	var notes []store.Note
	for rows.Next() {
		note, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notes: %w", err)
	}

	return notes, nil
}

func (s *Store) AddTask(ctx context.Context, input store.AddTaskInput) (store.Task, error) {
	now := nowUTC()

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO tasks (repo_id, branch, scope, text, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, input.RepoID, nullableString(input.Branch), string(input.Scope), input.Text, string(input.Status), formatTime(now), formatTime(now))
	if err != nil {
		return store.Task{}, fmt.Errorf("add task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return store.Task{}, fmt.Errorf("task last insert id: %w", err)
	}

	task, err := s.taskByID(ctx, id)
	if err != nil {
		return store.Task{}, err
	}

	return task, nil
}

func (s *Store) ListTasks(ctx context.Context, repoID int64, limit int) ([]store.Task, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, repo_id, branch, scope, text, status, created_at, updated_at
		FROM tasks
		WHERE repo_id = ?
		ORDER BY
			CASE status
				WHEN 'in_progress' THEN 0
				WHEN 'open' THEN 1
				WHEN 'blocked' THEN 2
				WHEN 'done' THEN 3
				ELSE 4
			END,
			updated_at DESC,
			id DESC
		LIMIT ?
	`, repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []store.Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}

	return tasks, nil
}

func (s *Store) CompleteTask(ctx context.Context, repoID int64, taskID int64) (store.Task, error) {
	now := nowUTC()

	result, err := s.db.ExecContext(ctx, `
		UPDATE tasks
		SET status = ?, updated_at = ?
		WHERE repo_id = ? AND id = ?
	`, string(store.TaskDone), formatTime(now), repoID, taskID)
	if err != nil {
		return store.Task{}, fmt.Errorf("complete task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return store.Task{}, fmt.Errorf("task rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return store.Task{}, fmt.Errorf("task %s not found", store.TaskRef(taskID))
	}

	task, err := s.taskByID(ctx, taskID)
	if err != nil {
		return store.Task{}, err
	}

	return task, nil
}

func (s *Store) AddDecision(ctx context.Context, input store.AddDecisionInput) (store.Decision, error) {
	now := nowUTC()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.Decision{}, fmt.Errorf("begin decision transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		INSERT INTO decisions (repo_id, branch, text, rationale, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, input.RepoID, nullableString(input.Branch), input.Text, input.Rationale, formatTime(now))
	if err != nil {
		return store.Decision{}, fmt.Errorf("add decision: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return store.Decision{}, fmt.Errorf("decision last insert id: %w", err)
	}

	decision, err := decisionByID(ctx, tx, id)
	if err != nil {
		return store.Decision{}, err
	}
	if err := insertDecisionFTS(ctx, tx, decision); err != nil {
		return store.Decision{}, err
	}
	if err := tx.Commit(); err != nil {
		return store.Decision{}, fmt.Errorf("commit decision transaction: %w", err)
	}

	return decision, nil
}

func (s *Store) ListDecisions(ctx context.Context, repoID int64, limit int) ([]store.Decision, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, repo_id, branch, text, rationale, created_at
		FROM decisions
		WHERE repo_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?
	`, repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("list decisions: %w", err)
	}
	defer rows.Close()

	var decisions []store.Decision
	for rows.Next() {
		decision, err := scanDecision(rows)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, decision)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate decisions: %w", err)
	}

	return decisions, nil
}

func (s *Store) AddHandoff(ctx context.Context, input store.AddHandoffInput) (store.Handoff, error) {
	now := nowUTC()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.Handoff{}, fmt.Errorf("begin handoff transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		INSERT INTO handoffs (repo_id, branch, summary, created_at)
		VALUES (?, ?, ?, ?)
	`, input.RepoID, nullableString(input.Branch), input.Summary, formatTime(now))
	if err != nil {
		return store.Handoff{}, fmt.Errorf("add handoff: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return store.Handoff{}, fmt.Errorf("handoff last insert id: %w", err)
	}

	handoff, err := handoffByID(ctx, tx, id)
	if err != nil {
		return store.Handoff{}, err
	}
	if err := insertHandoffFTS(ctx, tx, handoff); err != nil {
		return store.Handoff{}, err
	}
	if err := tx.Commit(); err != nil {
		return store.Handoff{}, fmt.Errorf("commit handoff transaction: %w", err)
	}

	return handoff, nil
}

func (s *Store) ListHandoffs(ctx context.Context, repoID int64, limit int) ([]store.Handoff, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, repo_id, branch, summary, created_at
		FROM handoffs
		WHERE repo_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?
	`, repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("list handoffs: %w", err)
	}
	defer rows.Close()

	var handoffs []store.Handoff
	for rows.Next() {
		handoff, err := scanHandoff(rows)
		if err != nil {
			return nil, err
		}
		handoffs = append(handoffs, handoff)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate handoffs: %w", err)
	}

	return handoffs, nil
}

func (s *Store) ReplaceCommits(ctx context.Context, input store.ReplaceCommitsInput) error {
	_, err := s.SyncCommits(ctx, store.SyncCommitsInput{
		RepoID:    input.RepoID,
		Commits:   input.Commits,
		IndexedAt: input.IndexedAt,
	})
	return err
}

func (s *Store) SyncCommits(ctx context.Context, input store.SyncCommitsInput) (store.SyncCommitsResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return store.SyncCommitsResult{}, fmt.Errorf("begin commit index transaction: %w", err)
	}
	defer tx.Rollback()

	existing, err := existingCommitsBySHA(ctx, tx, input.RepoID)
	if err != nil {
		return store.SyncCommitsResult{}, err
	}

	insertStatement, err := tx.PrepareContext(ctx, `
		INSERT INTO commits (repo_id, sha, author, committed_at, message, changed_paths, summary, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return store.SyncCommitsResult{}, fmt.Errorf("prepare commit insert: %w", err)
	}
	defer insertStatement.Close()

	updateStatement, err := tx.PrepareContext(ctx, `
		UPDATE commits
		SET author = ?, committed_at = ?, message = ?, changed_paths = ?, summary = ?, indexed_at = ?
		WHERE repo_id = ? AND sha = ?
	`)
	if err != nil {
		return store.SyncCommitsResult{}, fmt.Errorf("prepare commit update: %w", err)
	}
	defer updateStatement.Close()

	deleteStatement, err := tx.PrepareContext(ctx, `
		DELETE FROM commits
		WHERE id = ?
	`)
	if err != nil {
		return store.SyncCommitsResult{}, fmt.Errorf("prepare commit delete: %w", err)
	}
	defer deleteStatement.Close()

	result := store.SyncCommitsResult{
		Initial: len(existing) == 0,
	}
	indexedAt := formatTime(input.IndexedAt)
	seen := make(map[string]struct{}, len(input.Commits))
	for _, commit := range input.Commits {
		if _, ok := seen[commit.SHA]; ok {
			continue
		}
		seen[commit.SHA] = struct{}{}

		existingCommit, ok := existing[commit.SHA]
		if ok {
			if sameCommit(existingCommit, commit) {
				continue
			}

			if _, err := updateStatement.ExecContext(
				ctx,
				commit.Author,
				formatTime(commit.CommittedAt),
				commit.Message,
				strings.Join(commit.ChangedPaths, "\n"),
				commit.Summary,
				indexedAt,
				input.RepoID,
				commit.SHA,
			); err != nil {
				return store.SyncCommitsResult{}, fmt.Errorf("update commit %s: %w", commit.SHA, err)
			}
			result.Updated++
			continue
		}

		if _, err := insertStatement.ExecContext(
			ctx,
			input.RepoID,
			commit.SHA,
			commit.Author,
			formatTime(commit.CommittedAt),
			commit.Message,
			strings.Join(commit.ChangedPaths, "\n"),
			commit.Summary,
			indexedAt,
		); err != nil {
			return store.SyncCommitsResult{}, fmt.Errorf("insert commit %s: %w", commit.SHA, err)
		}
		result.Added++
	}

	for sha, commit := range existing {
		if _, ok := seen[sha]; ok {
			continue
		}
		if _, err := deleteStatement.ExecContext(ctx, commit.ID); err != nil {
			return store.SyncCommitsResult{}, fmt.Errorf("delete commit %s: %w", commit.SHA, err)
		}
		result.Removed++
	}

	result.Total = len(seen)
	if result.Added > 0 || result.Updated > 0 || result.Removed > 0 {
		if err := rebuildCommitFTS(ctx, tx, input.RepoID); err != nil {
			return store.SyncCommitsResult{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return store.SyncCommitsResult{}, fmt.Errorf("commit commit index transaction: %w", err)
	}

	return result, nil
}

func (s *Store) SearchCommits(ctx context.Context, repoID int64, query string, limit int) ([]store.Commit, error) {
	ftsQuery := buildFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

	if err := s.ensureCommitFTS(ctx, repoID); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.repo_id, c.sha, c.author, c.committed_at, c.message, c.changed_paths, c.summary, c.indexed_at
		FROM commits_fts
		JOIN commits c ON c.id = commits_fts.commit_id
		WHERE commits_fts.repo_id = ?
		  AND commits_fts MATCH ?
		ORDER BY bm25(commits_fts, 10.0, 5.0, 1.0, 20.0), c.committed_at DESC, c.id DESC
		LIMIT ?
	`, repoID, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("search commits: %w", err)
	}
	defer rows.Close()

	var commits []store.Commit
	for rows.Next() {
		commit, err := scanCommit(rows)
		if err != nil {
			return nil, err
		}
		commits = append(commits, commit)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate commits: %w", err)
	}

	return commits, nil
}

func (s *Store) SearchNotes(ctx context.Context, repoID int64, branch string, query string, limit int) ([]store.Note, error) {
	ftsQuery := buildFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

	if err := s.ensureNoteFTS(ctx, repoID); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT n.id, n.repo_id, n.branch, n.scope, n.text, n.created_at
		FROM notes_fts
		JOIN notes n ON n.id = notes_fts.note_id
		WHERE notes_fts.repo_id = ?
		  AND notes_fts MATCH ?
		ORDER BY
			CASE
				WHEN notes_fts.branch = ? THEN 0
				WHEN notes_fts.branch = '' THEN 1
				ELSE 2
			END,
			bm25(notes_fts, 5.0),
			n.created_at DESC,
			n.id DESC
		LIMIT ?
	`, repoID, ftsQuery, branch, limit)
	if err != nil {
		return nil, fmt.Errorf("search notes: %w", err)
	}
	defer rows.Close()

	var notes []store.Note
	for rows.Next() {
		note, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notes: %w", err)
	}

	return notes, nil
}

func (s *Store) SearchDecisions(ctx context.Context, repoID int64, branch string, query string, limit int) ([]store.Decision, error) {
	ftsQuery := buildFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

	if err := s.ensureDecisionFTS(ctx, repoID); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT d.id, d.repo_id, d.branch, d.text, d.rationale, d.created_at
		FROM decisions_fts
		JOIN decisions d ON d.id = decisions_fts.decision_id
		WHERE decisions_fts.repo_id = ?
		  AND decisions_fts MATCH ?
		ORDER BY
			CASE
				WHEN decisions_fts.branch = ? THEN 0
				WHEN decisions_fts.branch = '' THEN 1
				ELSE 2
			END,
			bm25(decisions_fts, 5.0, 2.0),
			d.created_at DESC,
			d.id DESC
		LIMIT ?
	`, repoID, ftsQuery, branch, limit)
	if err != nil {
		return nil, fmt.Errorf("search decisions: %w", err)
	}
	defer rows.Close()

	var decisions []store.Decision
	for rows.Next() {
		decision, err := scanDecision(rows)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, decision)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate decisions: %w", err)
	}

	return decisions, nil
}

func (s *Store) SearchHandoffs(ctx context.Context, repoID int64, branch string, query string, limit int) ([]store.Handoff, error) {
	ftsQuery := buildFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

	if err := s.ensureHandoffFTS(ctx, repoID); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT h.id, h.repo_id, h.branch, h.summary, h.created_at
		FROM handoffs_fts
		JOIN handoffs h ON h.id = handoffs_fts.handoff_id
		WHERE handoffs_fts.repo_id = ?
		  AND handoffs_fts MATCH ?
		ORDER BY
			CASE
				WHEN handoffs_fts.branch = ? THEN 0
				WHEN handoffs_fts.branch = '' THEN 1
				ELSE 2
			END,
			bm25(handoffs_fts, 5.0),
			h.created_at DESC,
			h.id DESC
		LIMIT ?
	`, repoID, ftsQuery, branch, limit)
	if err != nil {
		return nil, fmt.Errorf("search handoffs: %w", err)
	}
	defer rows.Close()

	var handoffs []store.Handoff
	for rows.Next() {
		handoff, err := scanHandoff(rows)
		if err != nil {
			return nil, err
		}
		handoffs = append(handoffs, handoff)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate handoffs: %w", err)
	}

	return handoffs, nil
}

func (s *Store) StatusCounts(ctx context.Context, repoID int64) (store.StatusCounts, error) {
	var counts store.StatusCounts

	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM notes
		WHERE repo_id = ?
	`, repoID).Scan(&counts.Notes); err != nil {
		return store.StatusCounts{}, fmt.Errorf("count notes: %w", err)
	}

	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM decisions
		WHERE repo_id = ?
	`, repoID).Scan(&counts.Decisions); err != nil {
		return store.StatusCounts{}, fmt.Errorf("count decisions: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT status, COUNT(*)
		FROM tasks
		WHERE repo_id = ?
		GROUP BY status
	`, repoID)
	if err != nil {
		return store.StatusCounts{}, fmt.Errorf("count tasks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return store.StatusCounts{}, fmt.Errorf("scan task counts: %w", err)
		}

		counts.Tasks.Total += count
		switch store.TaskStatus(status) {
		case store.TaskOpen:
			counts.Tasks.Open = count
		case store.TaskInProgress:
			counts.Tasks.InProgress = count
		case store.TaskDone:
			counts.Tasks.Done = count
		case store.TaskBlocked:
			counts.Tasks.Blocked = count
		}
	}

	if err := rows.Err(); err != nil {
		return store.StatusCounts{}, fmt.Errorf("iterate task counts: %w", err)
	}

	return counts, nil
}

func (s *Store) ensureCommitFTS(ctx context.Context, repoID int64) error {
	var commitCount int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM commits
		WHERE repo_id = ?
	`, repoID).Scan(&commitCount); err != nil {
		return fmt.Errorf("count commits: %w", err)
	}

	var indexedCount int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM commits_fts
		WHERE repo_id = ?
	`, repoID).Scan(&indexedCount); err != nil {
		return fmt.Errorf("count indexed commits: %w", err)
	}

	if indexedCount == commitCount {
		return nil
	}

	return rebuildCommitFTS(ctx, s.db, repoID)
}

func (s *Store) ensureNoteFTS(ctx context.Context, repoID int64) error {
	var noteCount int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM notes
		WHERE repo_id = ?
	`, repoID).Scan(&noteCount); err != nil {
		return fmt.Errorf("count notes: %w", err)
	}

	var indexedCount int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM notes_fts
		WHERE repo_id = ?
	`, repoID).Scan(&indexedCount); err != nil {
		return fmt.Errorf("count indexed notes: %w", err)
	}

	if indexedCount == noteCount {
		return nil
	}

	return rebuildNoteFTS(ctx, s.db, repoID)
}

func (s *Store) ensureDecisionFTS(ctx context.Context, repoID int64) error {
	var decisionCount int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM decisions
		WHERE repo_id = ?
	`, repoID).Scan(&decisionCount); err != nil {
		return fmt.Errorf("count decisions: %w", err)
	}

	var indexedCount int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM decisions_fts
		WHERE repo_id = ?
	`, repoID).Scan(&indexedCount); err != nil {
		return fmt.Errorf("count indexed decisions: %w", err)
	}

	if indexedCount == decisionCount {
		return nil
	}

	return rebuildDecisionFTS(ctx, s.db, repoID)
}

func (s *Store) ensureHandoffFTS(ctx context.Context, repoID int64) error {
	var handoffCount int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM handoffs
		WHERE repo_id = ?
	`, repoID).Scan(&handoffCount); err != nil {
		return fmt.Errorf("count handoffs: %w", err)
	}

	var indexedCount int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM handoffs_fts
		WHERE repo_id = ?
	`, repoID).Scan(&indexedCount); err != nil {
		return fmt.Errorf("count indexed handoffs: %w", err)
	}

	if indexedCount == handoffCount {
		return nil
	}

	return rebuildHandoffFTS(ctx, s.db, repoID)
}

type queryRower interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func noteByID(ctx context.Context, q queryRower, id int64) (store.Note, error) {
	row := q.QueryRowContext(ctx, `
		SELECT id, repo_id, branch, scope, text, created_at
		FROM notes
		WHERE id = ?
	`, id)

	note, err := scanNote(row)
	if err != nil {
		return store.Note{}, fmt.Errorf("load note: %w", err)
	}

	return note, nil
}

func (s *Store) taskByID(ctx context.Context, id int64) (store.Task, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, repo_id, branch, scope, text, status, created_at, updated_at
		FROM tasks
		WHERE id = ?
	`, id)

	task, err := scanTask(row)
	if err != nil {
		return store.Task{}, fmt.Errorf("load task: %w", err)
	}

	return task, nil
}

func decisionByID(ctx context.Context, q queryRower, id int64) (store.Decision, error) {
	row := q.QueryRowContext(ctx, `
		SELECT id, repo_id, branch, text, rationale, created_at
		FROM decisions
		WHERE id = ?
	`, id)

	decision, err := scanDecision(row)
	if err != nil {
		return store.Decision{}, fmt.Errorf("load decision: %w", err)
	}

	return decision, nil
}

func handoffByID(ctx context.Context, q queryRower, id int64) (store.Handoff, error) {
	row := q.QueryRowContext(ctx, `
		SELECT id, repo_id, branch, summary, created_at
		FROM handoffs
		WHERE id = ?
	`, id)

	handoff, err := scanHandoff(row)
	if err != nil {
		return store.Handoff{}, fmt.Errorf("load handoff: %w", err)
	}

	return handoff, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanRepo(row scanner) (store.Repo, error) {
	var repo store.Repo
	var createdAt string
	var updatedAt string

	if err := row.Scan(&repo.ID, &repo.Path, &repo.Name, &createdAt, &updatedAt); err != nil {
		return store.Repo{}, err
	}

	repo.CreatedAt = parseTime(createdAt)
	repo.UpdatedAt = parseTime(updatedAt)
	return repo, nil
}

func scanNote(row scanner) (store.Note, error) {
	var note store.Note
	var branch sql.NullString
	var scope string
	var createdAt string

	if err := row.Scan(&note.ID, &note.RepoID, &branch, &scope, &note.Text, &createdAt); err != nil {
		return store.Note{}, fmt.Errorf("scan note: %w", err)
	}

	note.Branch = branch.String
	note.Scope = store.Scope(scope)
	note.CreatedAt = parseTime(createdAt)
	return note, nil
}

func scanTask(row scanner) (store.Task, error) {
	var task store.Task
	var branch sql.NullString
	var scope string
	var status string
	var createdAt string
	var updatedAt string

	if err := row.Scan(&task.ID, &task.RepoID, &branch, &scope, &task.Text, &status, &createdAt, &updatedAt); err != nil {
		return store.Task{}, fmt.Errorf("scan task: %w", err)
	}

	task.Branch = branch.String
	task.Scope = store.Scope(scope)
	task.Status = store.TaskStatus(status)
	task.CreatedAt = parseTime(createdAt)
	task.UpdatedAt = parseTime(updatedAt)
	return task, nil
}

func scanDecision(row scanner) (store.Decision, error) {
	var decision store.Decision
	var branch sql.NullString
	var rationale sql.NullString
	var createdAt string

	if err := row.Scan(&decision.ID, &decision.RepoID, &branch, &decision.Text, &rationale, &createdAt); err != nil {
		return store.Decision{}, fmt.Errorf("scan decision: %w", err)
	}

	decision.Branch = branch.String
	decision.CreatedAt = parseTime(createdAt)
	if rationale.Valid {
		value := rationale.String
		decision.Rationale = &value
	}
	return decision, nil
}

func scanHandoff(row scanner) (store.Handoff, error) {
	var handoff store.Handoff
	var branch sql.NullString
	var createdAt string

	if err := row.Scan(&handoff.ID, &handoff.RepoID, &branch, &handoff.Summary, &createdAt); err != nil {
		return store.Handoff{}, fmt.Errorf("scan handoff: %w", err)
	}

	handoff.Branch = branch.String
	handoff.CreatedAt = parseTime(createdAt)
	return handoff, nil
}

func scanCommit(row scanner) (store.Commit, error) {
	var commit store.Commit
	var committedAt string
	var changedPaths string
	var indexedAt string

	if err := row.Scan(
		&commit.ID,
		&commit.RepoID,
		&commit.SHA,
		&commit.Author,
		&committedAt,
		&commit.Message,
		&changedPaths,
		&commit.Summary,
		&indexedAt,
	); err != nil {
		return store.Commit{}, fmt.Errorf("scan commit: %w", err)
	}

	commit.CommittedAt = parseTime(committedAt)
	commit.IndexedAt = parseTime(indexedAt)
	if strings.TrimSpace(changedPaths) != "" {
		commit.ChangedPaths = strings.Split(changedPaths, "\n")
	}
	return commit, nil
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func currentSchemaVersion(ctx context.Context, q queryRower) (int, error) {
	var version int
	if err := q.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	return version, nil
}

func existingCommitsBySHA(ctx context.Context, tx *sql.Tx, repoID int64) (map[string]store.Commit, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, repo_id, sha, author, committed_at, message, changed_paths, summary, indexed_at
		FROM commits
		WHERE repo_id = ?
	`, repoID)
	if err != nil {
		return nil, fmt.Errorf("list existing commits: %w", err)
	}
	defer rows.Close()

	commits := make(map[string]store.Commit)
	for rows.Next() {
		commit, err := scanCommit(rows)
		if err != nil {
			return nil, err
		}
		commits[commit.SHA] = commit
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate existing commits: %w", err)
	}

	return commits, nil
}

func sameCommit(existing store.Commit, candidate store.Commit) bool {
	if existing.Author != candidate.Author ||
		existing.Summary != candidate.Summary ||
		existing.Message != candidate.Message ||
		!existing.CommittedAt.Equal(candidate.CommittedAt) ||
		len(existing.ChangedPaths) != len(candidate.ChangedPaths) {
		return false
	}

	for i := range existing.ChangedPaths {
		if existing.ChangedPaths[i] != candidate.ChangedPaths[i] {
			return false
		}
	}

	return true
}

func insertNoteFTS(ctx context.Context, exec execer, note store.Note) error {
	if _, err := exec.ExecContext(ctx, `
		INSERT INTO notes_fts (text, note_id, repo_id, branch, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, note.Text, note.ID, note.RepoID, note.Branch, formatTime(note.CreatedAt)); err != nil {
		return fmt.Errorf("index note for search: %w", err)
	}
	return nil
}

func insertDecisionFTS(ctx context.Context, exec execer, decision store.Decision) error {
	rationale := ""
	if decision.Rationale != nil {
		rationale = *decision.Rationale
	}
	if _, err := exec.ExecContext(ctx, `
		INSERT INTO decisions_fts (text, rationale, decision_id, repo_id, branch, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, decision.Text, rationale, decision.ID, decision.RepoID, decision.Branch, formatTime(decision.CreatedAt)); err != nil {
		return fmt.Errorf("index decision for search: %w", err)
	}
	return nil
}

func insertHandoffFTS(ctx context.Context, exec execer, handoff store.Handoff) error {
	if _, err := exec.ExecContext(ctx, `
		INSERT INTO handoffs_fts (summary, handoff_id, repo_id, branch, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, handoff.Summary, handoff.ID, handoff.RepoID, handoff.Branch, formatTime(handoff.CreatedAt)); err != nil {
		return fmt.Errorf("index handoff for search: %w", err)
	}
	return nil
}

func rebuildCommitFTS(ctx context.Context, execer execer, repoID int64) error {
	if _, err := execer.ExecContext(ctx, `
		DELETE FROM commits_fts
		WHERE repo_id = ?
	`, repoID); err != nil {
		return fmt.Errorf("clear commit fts index: %w", err)
	}

	if _, err := execer.ExecContext(ctx, `
		INSERT INTO commits_fts (summary, message, changed_paths, sha, commit_id, repo_id)
		SELECT summary, message, changed_paths, sha, id, repo_id
		FROM commits
		WHERE repo_id = ?
	`, repoID); err != nil {
		return fmt.Errorf("rebuild commit fts index: %w", err)
	}

	return nil
}

func rebuildNoteFTS(ctx context.Context, exec execer, repoID int64) error {
	if _, err := exec.ExecContext(ctx, `
		DELETE FROM notes_fts
		WHERE repo_id = ?
	`, repoID); err != nil {
		return fmt.Errorf("clear note fts index: %w", err)
	}

	if _, err := exec.ExecContext(ctx, `
		INSERT INTO notes_fts (text, note_id, repo_id, branch, created_at)
		SELECT text, id, repo_id, COALESCE(branch, ''), created_at
		FROM notes
		WHERE repo_id = ?
	`, repoID); err != nil {
		return fmt.Errorf("rebuild note fts index: %w", err)
	}

	return nil
}

func rebuildDecisionFTS(ctx context.Context, exec execer, repoID int64) error {
	if _, err := exec.ExecContext(ctx, `
		DELETE FROM decisions_fts
		WHERE repo_id = ?
	`, repoID); err != nil {
		return fmt.Errorf("clear decision fts index: %w", err)
	}

	if _, err := exec.ExecContext(ctx, `
		INSERT INTO decisions_fts (text, rationale, decision_id, repo_id, branch, created_at)
		SELECT text, COALESCE(rationale, ''), id, repo_id, COALESCE(branch, ''), created_at
		FROM decisions
		WHERE repo_id = ?
	`, repoID); err != nil {
		return fmt.Errorf("rebuild decision fts index: %w", err)
	}

	return nil
}

func rebuildHandoffFTS(ctx context.Context, exec execer, repoID int64) error {
	if _, err := exec.ExecContext(ctx, `
		DELETE FROM handoffs_fts
		WHERE repo_id = ?
	`, repoID); err != nil {
		return fmt.Errorf("clear handoff fts index: %w", err)
	}

	if _, err := exec.ExecContext(ctx, `
		INSERT INTO handoffs_fts (summary, handoff_id, repo_id, branch, created_at)
		SELECT summary, id, repo_id, COALESCE(branch, ''), created_at
		FROM handoffs
		WHERE repo_id = ?
	`, repoID); err != nil {
		return fmt.Errorf("rebuild handoff fts index: %w", err)
	}

	return nil
}

func buildFTSQuery(raw string) string {
	tokens := uniqueSearchTokens(raw)
	if len(tokens) == 0 {
		return ""
	}

	filtered := filterStopWords(tokens)
	if len(filtered) > 0 {
		tokens = filtered
	}

	joiner := " AND "
	if len(tokens) > 3 {
		joiner = " OR "
	}

	terms := make([]string, 0, len(tokens))
	for _, token := range tokens {
		terms = append(terms, token+"*")
	}

	return strings.Join(terms, joiner)
}

func uniqueSearchTokens(raw string) []string {
	parts := strings.FieldsFunc(strings.ToLower(raw), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	seen := make(map[string]struct{}, len(parts))
	tokens := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		tokens = append(tokens, part)
	}

	return tokens
}

func filterStopWords(tokens []string) []string {
	filtered := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if _, ok := commitSearchStopWords[token]; ok {
			continue
		}
		filtered = append(filtered, token)
	}
	return filtered
}

var commitSearchStopWords = map[string]struct{}{
	"a":     {},
	"an":    {},
	"and":   {},
	"are":   {},
	"did":   {},
	"do":    {},
	"does":  {},
	"for":   {},
	"how":   {},
	"in":    {},
	"is":    {},
	"of":    {},
	"on":    {},
	"or":    {},
	"the":   {},
	"to":    {},
	"was":   {},
	"were":  {},
	"what":  {},
	"when":  {},
	"where": {},
	"which": {},
	"why":   {},
	"with":  {},
}

func nowUTC() time.Time {
	return time.Now().UTC().Round(time.Second)
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

func parseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}

	return parsed
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}

	return value
}

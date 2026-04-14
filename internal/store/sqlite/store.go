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

	"aid/internal/store"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
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
	statements := []string{
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
	}

	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("run migration: %w", err)
		}
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

	result, err := s.db.ExecContext(ctx, `
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

	note, err := s.noteByID(ctx, id)
	if err != nil {
		return store.Note{}, err
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

	result, err := s.db.ExecContext(ctx, `
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

	decision, err := s.decisionByID(ctx, id)
	if err != nil {
		return store.Decision{}, err
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

	result, err := s.db.ExecContext(ctx, `
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

	handoff, err := s.handoffByID(ctx, id)
	if err != nil {
		return store.Handoff{}, err
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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin commit index transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM commits WHERE repo_id = ?`, input.RepoID); err != nil {
		return fmt.Errorf("clear existing commits: %w", err)
	}

	statement, err := tx.PrepareContext(ctx, `
		INSERT INTO commits (repo_id, sha, author, committed_at, message, changed_paths, summary, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare commit insert: %w", err)
	}
	defer statement.Close()

	indexedAt := formatTime(input.IndexedAt)
	for _, commit := range input.Commits {
		if _, err := statement.ExecContext(
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
			return fmt.Errorf("insert commit %s: %w", commit.SHA, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit commit index transaction: %w", err)
	}

	return nil
}

func (s *Store) SearchCommits(ctx context.Context, repoID int64, query string, limit int) ([]store.Commit, error) {
	pattern := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, repo_id, sha, author, committed_at, message, changed_paths, summary, indexed_at
		FROM commits
		WHERE repo_id = ?
		  AND LOWER(summary || ' ' || message || ' ' || changed_paths) LIKE ?
		ORDER BY committed_at DESC, id DESC
		LIMIT ?
	`, repoID, pattern, limit)
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

func (s *Store) noteByID(ctx context.Context, id int64) (store.Note, error) {
	row := s.db.QueryRowContext(ctx, `
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

func (s *Store) decisionByID(ctx context.Context, id int64) (store.Decision, error) {
	row := s.db.QueryRowContext(ctx, `
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

func (s *Store) handoffByID(ctx context.Context, id int64) (store.Handoff, error) {
	row := s.db.QueryRowContext(ctx, `
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

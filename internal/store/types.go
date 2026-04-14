package store

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Scope string

const (
	ScopeRepo   Scope = "repo"
	ScopeBranch Scope = "branch"
)

type TaskStatus string

const (
	TaskOpen       TaskStatus = "open"
	TaskInProgress TaskStatus = "in_progress"
	TaskDone       TaskStatus = "done"
	TaskBlocked    TaskStatus = "blocked"
)

type Repo struct {
	ID        int64
	Path      string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Note struct {
	ID        int64
	RepoID    int64
	Branch    string
	Scope     Scope
	Text      string
	CreatedAt time.Time
}

type Task struct {
	ID        int64
	RepoID    int64
	Branch    string
	Scope     Scope
	Text      string
	Status    TaskStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Decision struct {
	ID        int64
	RepoID    int64
	Branch    string
	Text      string
	Rationale *string
	CreatedAt time.Time
}

type Handoff struct {
	ID        int64
	RepoID    int64
	Branch    string
	Summary   string
	CreatedAt time.Time
}

type Commit struct {
	ID           int64
	RepoID       int64
	SHA          string
	Author       string
	CommittedAt  time.Time
	Message      string
	Summary      string
	ChangedPaths []string
	IndexedAt    time.Time
}

type TaskCounts struct {
	Total      int `json:"total"`
	Open       int `json:"open"`
	InProgress int `json:"in_progress"`
	Done       int `json:"done"`
	Blocked    int `json:"blocked"`
}

type StatusCounts struct {
	Notes     int        `json:"notes"`
	Decisions int        `json:"decisions"`
	Tasks     TaskCounts `json:"tasks"`
}

type AddNoteInput struct {
	RepoID int64
	Branch string
	Scope  Scope
	Text   string
}

type AddTaskInput struct {
	RepoID int64
	Branch string
	Scope  Scope
	Text   string
	Status TaskStatus
}

type AddDecisionInput struct {
	RepoID    int64
	Branch    string
	Text      string
	Rationale *string
}

type AddHandoffInput struct {
	RepoID  int64
	Branch  string
	Summary string
}

type ReplaceCommitsInput struct {
	RepoID    int64
	Commits   []Commit
	IndexedAt time.Time
}

type SyncCommitsInput struct {
	RepoID    int64
	Commits   []Commit
	IndexedAt time.Time
}

type SyncCommitsResult struct {
	Added   int
	Updated int
	Removed int
	Total   int
	Initial bool
}

type Store interface {
	Close() error
	Migrate(ctx context.Context) error
	UpsertRepo(ctx context.Context, path string, name string) (Repo, error)
	FindRepoByPath(ctx context.Context, path string) (*Repo, error)
	AddNote(ctx context.Context, input AddNoteInput) (Note, error)
	ListNotes(ctx context.Context, repoID int64, limit int) ([]Note, error)
	AddTask(ctx context.Context, input AddTaskInput) (Task, error)
	ListTasks(ctx context.Context, repoID int64, limit int) ([]Task, error)
	UpdateTaskStatus(ctx context.Context, repoID int64, taskID int64, status TaskStatus) (Task, error)
	CompleteTask(ctx context.Context, repoID int64, taskID int64) (Task, error)
	AddDecision(ctx context.Context, input AddDecisionInput) (Decision, error)
	ListDecisions(ctx context.Context, repoID int64, limit int) ([]Decision, error)
	AddHandoff(ctx context.Context, input AddHandoffInput) (Handoff, error)
	ListHandoffs(ctx context.Context, repoID int64, limit int) ([]Handoff, error)
	ReplaceCommits(ctx context.Context, input ReplaceCommitsInput) error
	SyncCommits(ctx context.Context, input SyncCommitsInput) (SyncCommitsResult, error)
	ListCommits(ctx context.Context, repoID int64, limit int) ([]Commit, error)
	SearchCommits(ctx context.Context, repoID int64, query string, limit int) ([]Commit, error)
	SearchNotes(ctx context.Context, repoID int64, branch string, query string, limit int) ([]Note, error)
	SearchDecisions(ctx context.Context, repoID int64, branch string, query string, limit int) ([]Decision, error)
	SearchHandoffs(ctx context.Context, repoID int64, branch string, query string, limit int) ([]Handoff, error)
	StatusCounts(ctx context.Context, repoID int64) (StatusCounts, error)
}

func NoteRef(id int64) string {
	return fmt.Sprintf("note_%d", id)
}

func TaskRef(id int64) string {
	return fmt.Sprintf("task_%d", id)
}

func DecisionRef(id int64) string {
	return fmt.Sprintf("decision_%d", id)
}

func HandoffRef(id int64) string {
	return fmt.Sprintf("handoff_%d", id)
}

func ParseTaskRef(value string) (int64, error) {
	return parseRef("task", value)
}

func parseRef(prefix, value string) (int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("missing %s id", prefix)
	}

	numeric := trimmed
	if strings.HasPrefix(trimmed, prefix+"_") {
		numeric = strings.TrimPrefix(trimmed, prefix+"_")
	}

	id, err := strconv.ParseInt(numeric, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid %s id %q", prefix, value)
	}

	return id, nil
}

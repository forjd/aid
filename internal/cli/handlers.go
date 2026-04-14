package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/forjd/aid/internal/app"
	"github.com/forjd/aid/internal/config"
	"github.com/forjd/aid/internal/git"
	handoffpkg "github.com/forjd/aid/internal/handoff"
	"github.com/forjd/aid/internal/output"
	resumepkg "github.com/forjd/aid/internal/resume"
	searchpkg "github.com/forjd/aid/internal/search"
	"github.com/forjd/aid/internal/store"
	sqlitestore "github.com/forjd/aid/internal/store/sqlite"
)

const defaultListLimit = 20

type repoRuntime struct {
	env   app.Environment
	cfg   config.RepoConfig
	store *sqlitestore.Store
	repo  *store.Repo
}

func initCommand(args []string, streams Streams) error {
	if len(args) > 0 {
		return fmt.Errorf("init does not accept arguments")
	}

	ctx := context.Background()
	env, sqliteStore, err := openStore(ctx, streams)
	if err != nil {
		return err
	}
	defer sqliteStore.Close()

	repo, err := sqliteStore.UpsertRepo(ctx, env.RepoRoot, env.RepoName)
	if err != nil {
		return err
	}

	configCreated, err := config.EnsureRepoConfig(env.RepoConfigPath)
	if err != nil {
		return err
	}

	return output.RenderInit(streams.Out, streams.Options, output.InitResult{
		RepoName:      repo.Name,
		RepoPath:      repo.Path,
		Branch:        env.Branch,
		DBPath:        env.DBPath,
		ConfigPath:    env.RepoConfigPath,
		ConfigCreated: configCreated,
	})
}

func noteAddCommand(args []string, streams Streams) error {
	text, err := joinArgs(args, "note text")
	if err != nil {
		return err
	}

	runtime, err := openInitializedRepo(context.Background(), streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	note, err := runtime.store.AddNote(context.Background(), store.AddNoteInput{
		RepoID: runtime.repo.ID,
		Branch: runtime.env.Branch,
		Scope:  store.ScopeBranch,
		Text:   text,
	})
	if err != nil {
		return err
	}

	return output.RenderNoteAdded(streams.Out, streams.Options, note)
}

func noteListCommand(args []string, streams Streams) error {
	if len(args) > 0 {
		return fmt.Errorf("note list does not accept arguments")
	}

	runtime, err := openInitializedRepo(context.Background(), streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	notes, err := runtime.store.ListNotes(context.Background(), runtime.repo.ID, defaultListLimit)
	if err != nil {
		return err
	}

	return output.RenderNotes(streams.Out, streams.Options, notes)
}

func taskAddCommand(args []string, streams Streams) error {
	text, err := joinArgs(args, "task text")
	if err != nil {
		return err
	}

	runtime, err := openInitializedRepo(context.Background(), streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	task, err := runtime.store.AddTask(context.Background(), store.AddTaskInput{
		RepoID: runtime.repo.ID,
		Branch: runtime.env.Branch,
		Scope:  store.ScopeBranch,
		Text:   text,
		Status: store.TaskOpen,
	})
	if err != nil {
		return err
	}

	return output.RenderTaskAdded(streams.Out, streams.Options, task)
}

func taskListCommand(args []string, streams Streams) error {
	if len(args) > 0 {
		return fmt.Errorf("task list does not accept arguments")
	}

	runtime, err := openInitializedRepo(context.Background(), streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	tasks, err := runtime.store.ListTasks(context.Background(), runtime.repo.ID, defaultListLimit)
	if err != nil {
		return err
	}

	return output.RenderTasks(streams.Out, streams.Options, tasks)
}

func taskDoneCommand(args []string, streams Streams) error {
	return taskStatusCommand(args, streams, store.TaskDone)
}

func taskStartCommand(args []string, streams Streams) error {
	return taskStatusCommand(args, streams, store.TaskInProgress)
}

func taskBlockCommand(args []string, streams Streams) error {
	return taskStatusCommand(args, streams, store.TaskBlocked)
}

func taskReopenCommand(args []string, streams Streams) error {
	return taskStatusCommand(args, streams, store.TaskOpen)
}

func taskStatusCommand(args []string, streams Streams, status store.TaskStatus) error {
	if len(args) != 1 {
		return fmt.Errorf("task %s expects exactly one task id", taskCommandName(status))
	}

	taskID, err := store.ParseTaskRef(args[0])
	if err != nil {
		return err
	}

	runtime, err := openInitializedRepo(context.Background(), streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	task, err := runtime.store.UpdateTaskStatus(context.Background(), runtime.repo.ID, taskID, status)
	if err != nil {
		return err
	}

	return output.RenderTaskStatusUpdated(streams.Out, streams.Options, taskCommandName(status), taskStatusVerb(status), task)
}

func decisionAddCommand(args []string, streams Streams) error {
	text, err := joinArgs(args, "decision text")
	if err != nil {
		return err
	}

	runtime, err := openInitializedRepo(context.Background(), streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	decision, err := runtime.store.AddDecision(context.Background(), store.AddDecisionInput{
		RepoID: runtime.repo.ID,
		Branch: runtime.env.Branch,
		Text:   text,
	})
	if err != nil {
		return err
	}

	return output.RenderDecisionAdded(streams.Out, streams.Options, decision)
}

func decisionListCommand(args []string, streams Streams) error {
	if len(args) > 0 {
		return fmt.Errorf("decide list does not accept arguments")
	}

	runtime, err := openInitializedRepo(context.Background(), streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	decisions, err := runtime.store.ListDecisions(context.Background(), runtime.repo.ID, defaultListLimit)
	if err != nil {
		return err
	}

	return output.RenderDecisions(streams.Out, streams.Options, decisions)
}

func statusCommand(args []string, streams Streams) error {
	if len(args) > 0 {
		return fmt.Errorf("status does not accept arguments")
	}

	ctx := context.Background()
	env, sqliteStore, err := openStore(ctx, streams)
	if err != nil {
		return err
	}
	defer sqliteStore.Close()

	repo, err := sqliteStore.FindRepoByPath(ctx, env.RepoRoot)
	if err != nil {
		return err
	}

	configExists := true
	if _, err := os.Stat(env.RepoConfigPath); err != nil {
		if os.IsNotExist(err) {
			configExists = false
		} else {
			return fmt.Errorf("stat repo config: %w", err)
		}
	}

	result := output.StatusResult{
		RepoName:     env.RepoName,
		RepoPath:     env.RepoRoot,
		Branch:       env.Branch,
		DBPath:       env.DBPath,
		ConfigPath:   env.RepoConfigPath,
		ConfigExists: configExists,
		Initialized:  repo != nil,
	}

	if repo != nil {
		counts, err := sqliteStore.StatusCounts(ctx, repo.ID)
		if err != nil {
			return err
		}
		result.Counts = counts
	}

	return output.RenderStatus(streams.Out, streams.Options, result)
}

func resumeCommand(args []string, streams Streams) error {
	if len(args) > 0 {
		return fmt.Errorf("resume does not accept arguments")
	}

	ctx := context.Background()
	runtime, err := openInitializedRepo(ctx, streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	notes, err := runtime.store.ListNotes(ctx, runtime.repo.ID, 20)
	if err != nil {
		return err
	}

	tasks, err := runtime.store.ListTasks(ctx, runtime.repo.ID, 50)
	if err != nil {
		return err
	}

	decisions, err := runtime.store.ListDecisions(ctx, runtime.repo.ID, 20)
	if err != nil {
		return err
	}

	commits, err := recentContextCommits(ctx, runtime, 5)
	if err != nil {
		return err
	}

	handoffs, err := runtime.store.ListHandoffs(ctx, runtime.repo.ID, 3)
	if err != nil {
		return err
	}

	bundle := resumepkg.Build(runtime.env.Branch, notes, tasks, decisions, commits, handoffs)
	return output.RenderResume(streams.Out, streams.Options, output.ResumeResult{
		RepoName: runtime.env.RepoName,
		RepoPath: runtime.env.RepoRoot,
		Branch:   runtime.env.Branch,
		Bundle:   bundle,
	})
}

func handoffGenerateCommand(args []string, streams Streams) error {
	if len(args) > 0 {
		return fmt.Errorf("handoff generate does not accept arguments")
	}

	ctx := context.Background()
	runtime, err := openInitializedRepo(ctx, streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	notes, err := runtime.store.ListNotes(ctx, runtime.repo.ID, 20)
	if err != nil {
		return err
	}

	tasks, err := runtime.store.ListTasks(ctx, runtime.repo.ID, 50)
	if err != nil {
		return err
	}

	decisions, err := runtime.store.ListDecisions(ctx, runtime.repo.ID, 20)
	if err != nil {
		return err
	}

	commits, err := recentContextCommits(ctx, runtime, 5)
	if err != nil {
		return err
	}

	handoffs, err := runtime.store.ListHandoffs(ctx, runtime.repo.ID, 3)
	if err != nil {
		return err
	}

	worktree, err := git.Status(runtime.env.RepoRoot)
	if err != nil {
		return err
	}

	bundle := resumepkg.Build(runtime.env.Branch, notes, tasks, decisions, commits, handoffs)
	snapshot := handoffpkg.Build(runtime.env.Branch, worktree, bundle, tasks)

	handoff, err := runtime.store.AddHandoff(ctx, store.AddHandoffInput{
		RepoID:  runtime.repo.ID,
		Branch:  runtime.env.Branch,
		Summary: snapshot.Summary,
	})
	if err != nil {
		return err
	}

	return output.RenderHandoffGenerated(streams.Out, streams.Options, output.HandoffGenerateResult{
		Handoff: handoff,
	})
}

func handoffListCommand(args []string, streams Streams) error {
	if len(args) > 0 {
		return fmt.Errorf("handoff list does not accept arguments")
	}

	runtime, err := openInitializedRepo(context.Background(), streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	handoffs, err := runtime.store.ListHandoffs(context.Background(), runtime.repo.ID, defaultListLimit)
	if err != nil {
		return err
	}

	return output.RenderHandoffs(streams.Out, streams.Options, handoffs)
}

func historyIndexCommand(args []string, streams Streams) error {
	if len(args) > 0 {
		return fmt.Errorf("history index does not accept arguments")
	}

	ctx := context.Background()
	runtime, err := openInitializedRepo(ctx, streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	reachableSHAs, err := git.AllCommitSHAs(runtime.env.RepoRoot)
	if err != nil {
		return err
	}

	existingCommits, err := runtime.store.ListCommits(ctx, runtime.repo.ID, 0)
	if err != nil {
		return err
	}

	existingBySHA := make(map[string]store.Commit, len(existingCommits))
	for _, commit := range existingCommits {
		existingBySHA[commit.SHA] = commit
	}

	newSHAs := make([]string, 0, len(reachableSHAs))
	for _, sha := range reachableSHAs {
		if _, ok := existingBySHA[sha]; ok {
			continue
		}
		newSHAs = append(newSHAs, sha)
	}

	newCommits, err := git.CommitsBySHA(runtime.env.RepoRoot, newSHAs)
	if err != nil {
		return err
	}

	newBySHA := make(map[string]git.Commit, len(newCommits))
	for _, commit := range newCommits {
		newBySHA[commit.SHA] = commit
	}

	storeCommits := make([]store.Commit, 0, len(reachableSHAs))
	totalReachable := len(reachableSHAs)
	for index, sha := range reachableSHAs {
		if existing, ok := existingBySHA[sha]; ok {
			filtered, keep := filteredStoredCommit(existing, runtime.cfg.Indexing.IgnorePaths)
			if keep {
				filtered.GitOrder = totalReachable - index - 1
				storeCommits = append(storeCommits, filtered)
			}
			continue
		}

		commit, ok := newBySHA[sha]
		if !ok {
			continue
		}
		filtered, keep := filteredGitCommit(commit, runtime.cfg.Indexing.IgnorePaths)
		if keep {
			filtered.GitOrder = totalReachable - index - 1
			storeCommits = append(storeCommits, filtered)
		}
	}

	result, err := runtime.store.SyncCommits(ctx, store.SyncCommitsInput{
		RepoID:    runtime.repo.ID,
		Commits:   storeCommits,
		IndexedAt: time.Now().UTC(),
	})
	if err != nil {
		return err
	}

	return output.RenderHistoryIndexed(streams.Out, streams.Options, output.HistoryIndexResult{
		Indexed: len(storeCommits),
		Added:   result.Added,
		Updated: result.Updated,
		Removed: result.Removed,
		Mode:    historyIndexMode(result.Initial),
	})
}

func historySearchCommand(args []string, streams Streams) error {
	query, err := joinArgs(args, "search query")
	if err != nil {
		return err
	}

	runtime, err := openInitializedRepo(context.Background(), streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	commits, err := runtime.store.SearchCommits(context.Background(), runtime.repo.ID, query, defaultListLimit)
	if err != nil {
		return err
	}

	return output.RenderHistorySearch(streams.Out, streams.Options, output.HistorySearchResult{
		Query:   query,
		Commits: commits,
	})
}

func recallCommand(args []string, streams Streams) error {
	query, err := joinArgs(args, "query")
	if err != nil {
		return err
	}

	ctx := context.Background()
	runtime, err := openInitializedRepo(ctx, streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	notes, err := runtime.store.SearchNotes(ctx, runtime.repo.ID, runtime.env.Branch, query, 10)
	if err != nil {
		return err
	}

	decisions, err := runtime.store.SearchDecisions(ctx, runtime.repo.ID, runtime.env.Branch, query, 10)
	if err != nil {
		return err
	}

	handoffs, err := runtime.store.SearchHandoffs(ctx, runtime.repo.ID, runtime.env.Branch, query, 10)
	if err != nil {
		return err
	}

	commits, err := runtime.store.SearchCommits(ctx, runtime.repo.ID, query, 10)
	if err != nil {
		return err
	}

	result := searchpkg.Build(query, runtime.env.Branch, notes, decisions, handoffs, commits)
	return output.RenderRecall(streams.Out, streams.Options, output.RecallResult{
		Result: result,
	})
}

func openStore(ctx context.Context, streams Streams) (app.Environment, *sqlitestore.Store, error) {
	env, err := app.Discover(streams.Options.RepoPath)
	if err != nil {
		return app.Environment{}, nil, err
	}

	sqliteStore, err := sqlitestore.Open(env.DBPath)
	if err != nil {
		return app.Environment{}, nil, err
	}

	if err := sqliteStore.Migrate(ctx); err != nil {
		_ = sqliteStore.Close()
		return app.Environment{}, nil, err
	}

	return env, sqliteStore, nil
}

func openInitializedRepo(ctx context.Context, streams Streams) (*repoRuntime, error) {
	env, sqliteStore, err := openStore(ctx, streams)
	if err != nil {
		return nil, err
	}

	repo, err := sqliteStore.FindRepoByPath(ctx, env.RepoRoot)
	if err != nil {
		_ = sqliteStore.Close()
		return nil, err
	}
	if repo == nil {
		_ = sqliteStore.Close()
		return nil, fmt.Errorf("repo not initialised; run \"aid init\" first")
	}

	cfg, err := config.LoadRepoConfig(env.RepoConfigPath)
	if err != nil {
		_ = sqliteStore.Close()
		return nil, err
	}

	return &repoRuntime{
		env:   env,
		cfg:   cfg,
		store: sqliteStore,
		repo:  repo,
	}, nil
}

func (runtime *repoRuntime) close() {
	_ = runtime.store.Close()
}

func joinArgs(args []string, label string) (string, error) {
	text := strings.TrimSpace(strings.Join(args, " "))
	if text == "" {
		return "", fmt.Errorf("missing %s", label)
	}

	return text, nil
}

func filterIndexedCommits(commits []git.Commit, ignorePaths []string) []git.Commit {
	if len(ignorePaths) == 0 {
		return commits
	}

	filtered := make([]git.Commit, 0, len(commits))
	for _, commit := range commits {
		paths := filterChangedPaths(commit.ChangedPaths, ignorePaths)
		if len(commit.ChangedPaths) > 0 && len(paths) == 0 {
			continue
		}

		commit.ChangedPaths = paths
		filtered = append(filtered, commit)
	}

	return filtered
}

func recentContextCommits(ctx context.Context, runtime *repoRuntime, limit int) ([]store.Commit, error) {
	commits, err := runtime.store.ListCommits(ctx, runtime.repo.ID, limit)
	if err != nil {
		return nil, err
	}
	if len(commits) > 0 {
		return commits, nil
	}

	liveCommits, err := git.RecentCommits(runtime.env.RepoRoot, limit)
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

func filterChangedPaths(paths []string, ignorePaths []string) []string {
	filtered := make([]string, 0, len(paths))
	for _, path := range paths {
		if matchesIgnoredPath(path, ignorePaths) {
			continue
		}
		filtered = append(filtered, path)
	}
	return filtered
}

func filteredGitCommit(commit git.Commit, ignorePaths []string) (store.Commit, bool) {
	paths := filterChangedPaths(commit.ChangedPaths, ignorePaths)
	if len(commit.ChangedPaths) > 0 && len(paths) == 0 {
		return store.Commit{}, false
	}

	return store.Commit{
		SHA:          commit.SHA,
		Author:       commit.Author,
		CommittedAt:  commit.CommittedAt,
		Message:      commit.Message,
		Summary:      commit.Summary,
		ChangedPaths: paths,
	}, true
}

func filteredStoredCommit(commit store.Commit, ignorePaths []string) (store.Commit, bool) {
	paths := filterChangedPaths(commit.ChangedPaths, ignorePaths)
	if len(commit.ChangedPaths) > 0 && len(paths) == 0 {
		return store.Commit{}, false
	}

	commit.ChangedPaths = paths
	return commit, true
}

func matchesIgnoredPath(path string, ignorePaths []string) bool {
	normalizedPath := strings.TrimPrefix(strings.ReplaceAll(path, "\\", "/"), "./")
	for _, prefix := range ignorePaths {
		normalizedPrefix := strings.TrimPrefix(strings.ReplaceAll(prefix, "\\", "/"), "./")
		if normalizedPrefix != "" && strings.HasPrefix(normalizedPath, normalizedPrefix) {
			return true
		}
	}
	return false
}

func historyIndexMode(initial bool) string {
	if initial {
		return "initial sync"
	}
	return "incremental sync"
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

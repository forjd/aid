package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"aid/internal/app"
	"aid/internal/config"
	"aid/internal/git"
	handoffpkg "aid/internal/handoff"
	"aid/internal/output"
	resumepkg "aid/internal/resume"
	"aid/internal/store"
	sqlitestore "aid/internal/store/sqlite"
)

const defaultListLimit = 20

type repoRuntime struct {
	env   app.Environment
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
	if len(args) != 1 {
		return fmt.Errorf("task done expects exactly one task id")
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

	task, err := runtime.store.CompleteTask(context.Background(), runtime.repo.ID, taskID)
	if err != nil {
		return err
	}

	return output.RenderTaskCompleted(streams.Out, streams.Options, task)
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

	commits, err := git.RecentCommits(runtime.env.RepoRoot, 5)
	if err != nil {
		return err
	}

	bundle := resumepkg.Build(runtime.env.Branch, notes, tasks, decisions, commits)
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

	commits, err := git.RecentCommits(runtime.env.RepoRoot, 5)
	if err != nil {
		return err
	}

	worktree, err := git.Status(runtime.env.RepoRoot)
	if err != nil {
		return err
	}

	bundle := resumepkg.Build(runtime.env.Branch, notes, tasks, decisions, commits)
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

	return &repoRuntime{
		env:   env,
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

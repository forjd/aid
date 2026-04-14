package cli

import (
	"context"
	"fmt"
	"strings"

	"aid/internal/app"
	"aid/internal/config"
	"aid/internal/output"
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
	env, sqliteStore, err := openStore(ctx)
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

	output.RenderInit(streams.Out, output.InitResult{
		RepoName:      repo.Name,
		RepoPath:      repo.Path,
		DBPath:        env.DBPath,
		ConfigPath:    env.RepoConfigPath,
		ConfigCreated: configCreated,
	})

	return nil
}

func noteAddCommand(args []string, streams Streams) error {
	text, err := joinArgs(args, "note text")
	if err != nil {
		return err
	}

	runtime, err := openInitializedRepo(context.Background())
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

	output.RenderNoteAdded(streams.Out, note)
	return nil
}

func noteListCommand(args []string, streams Streams) error {
	if len(args) > 0 {
		return fmt.Errorf("note list does not accept arguments")
	}

	runtime, err := openInitializedRepo(context.Background())
	if err != nil {
		return err
	}
	defer runtime.close()

	notes, err := runtime.store.ListNotes(context.Background(), runtime.repo.ID, defaultListLimit)
	if err != nil {
		return err
	}

	output.RenderNotes(streams.Out, notes)
	return nil
}

func taskAddCommand(args []string, streams Streams) error {
	text, err := joinArgs(args, "task text")
	if err != nil {
		return err
	}

	runtime, err := openInitializedRepo(context.Background())
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

	output.RenderTaskAdded(streams.Out, task)
	return nil
}

func taskListCommand(args []string, streams Streams) error {
	if len(args) > 0 {
		return fmt.Errorf("task list does not accept arguments")
	}

	runtime, err := openInitializedRepo(context.Background())
	if err != nil {
		return err
	}
	defer runtime.close()

	tasks, err := runtime.store.ListTasks(context.Background(), runtime.repo.ID, defaultListLimit)
	if err != nil {
		return err
	}

	output.RenderTasks(streams.Out, tasks)
	return nil
}

func taskDoneCommand(args []string, streams Streams) error {
	if len(args) != 1 {
		return fmt.Errorf("task done expects exactly one task id")
	}

	taskID, err := store.ParseTaskRef(args[0])
	if err != nil {
		return err
	}

	runtime, err := openInitializedRepo(context.Background())
	if err != nil {
		return err
	}
	defer runtime.close()

	task, err := runtime.store.CompleteTask(context.Background(), runtime.repo.ID, taskID)
	if err != nil {
		return err
	}

	output.RenderTaskCompleted(streams.Out, task)
	return nil
}

func decisionAddCommand(args []string, streams Streams) error {
	text, err := joinArgs(args, "decision text")
	if err != nil {
		return err
	}

	runtime, err := openInitializedRepo(context.Background())
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

	output.RenderDecisionAdded(streams.Out, decision)
	return nil
}

func decisionListCommand(args []string, streams Streams) error {
	if len(args) > 0 {
		return fmt.Errorf("decide list does not accept arguments")
	}

	runtime, err := openInitializedRepo(context.Background())
	if err != nil {
		return err
	}
	defer runtime.close()

	decisions, err := runtime.store.ListDecisions(context.Background(), runtime.repo.ID, defaultListLimit)
	if err != nil {
		return err
	}

	output.RenderDecisions(streams.Out, decisions)
	return nil
}

func openStore(ctx context.Context) (app.Environment, *sqlitestore.Store, error) {
	env, err := app.Discover("")
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

func openInitializedRepo(ctx context.Context) (*repoRuntime, error) {
	env, sqliteStore, err := openStore(ctx)
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

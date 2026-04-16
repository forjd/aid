package cli

import (
	"github.com/forjd/aid/internal/git"
	handoffpkg "github.com/forjd/aid/internal/handoff"
	"github.com/forjd/aid/internal/output"
	resumepkg "github.com/forjd/aid/internal/resume"
	searchpkg "github.com/forjd/aid/internal/search"
	"github.com/forjd/aid/internal/store"
)

func resumeCommand(args []string, streams Streams) error {
	if len(args) > 0 {
		return newError(ErrCodeUsage, "resume does not accept arguments")
	}

	ctx := streams.context()
	runtime, err := openInitializedRepo(ctx, streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	notes, err := runtime.store.ListNotes(ctx, runtime.repo.ID, runtime.env.Branch, 20)
	if err != nil {
		return err
	}

	tasks, err := runtime.store.ListTasks(ctx, runtime.repo.ID, runtime.env.Branch, 50)
	if err != nil {
		return err
	}

	decisions, err := runtime.store.ListDecisions(ctx, runtime.repo.ID, runtime.env.Branch, 20)
	if err != nil {
		return err
	}

	commits, err := recentContextCommits(ctx, runtime, 5)
	if err != nil {
		return err
	}

	handoffs, err := runtime.store.ListHandoffs(ctx, runtime.repo.ID, runtime.env.Branch, 3)
	if err != nil {
		return err
	}

	bundle := resumepkg.Build(runtime.env.Branch, notes, tasks, decisions, commits, handoffs)
	return output.RenderResume(streams.Out, streams.Options, output.ResumeResult{
		RepoName: runtime.env.RepoName,
		RepoPath: runtime.env.RepoRoot,
		Branch:   runtime.env.Branch,
		Bundle:   presentResumeBundle(bundle),
	})
}

func handoffGenerateCommand(args []string, streams Streams) error {
	if len(args) > 0 {
		return newError(ErrCodeUsage, "handoff generate does not accept arguments")
	}

	ctx := streams.context()
	runtime, err := openInitializedRepo(ctx, streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	notes, err := runtime.store.ListNotes(ctx, runtime.repo.ID, runtime.env.Branch, 20)
	if err != nil {
		return err
	}

	tasks, err := runtime.store.ListTasks(ctx, runtime.repo.ID, runtime.env.Branch, 50)
	if err != nil {
		return err
	}

	decisions, err := runtime.store.ListDecisions(ctx, runtime.repo.ID, runtime.env.Branch, 20)
	if err != nil {
		return err
	}

	commits, err := recentContextCommits(ctx, runtime, 5)
	if err != nil {
		return err
	}

	handoffs, err := runtime.store.ListHandoffs(ctx, runtime.repo.ID, runtime.env.Branch, 3)
	if err != nil {
		return err
	}

	worktree, err := git.Status(ctx, runtime.env.RepoRoot)
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
		return newError(ErrCodeUsage, "handoff list does not accept arguments")
	}

	ctx := streams.context()
	runtime, err := openInitializedRepo(ctx, streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	handoffs, err := runtime.store.ListHandoffs(ctx, runtime.repo.ID, runtime.env.Branch, defaultListLimit)
	if err != nil {
		return err
	}

	return output.RenderHandoffs(streams.Out, streams.Options, handoffs)
}

func recallCommand(args []string, streams Streams) error {
	query, err := joinArgs(args, "query")
	if err != nil {
		return err
	}

	ctx := streams.context()
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

	result := searchpkg.Build(query, notes, decisions, handoffs, commits)
	return output.RenderRecall(streams.Out, streams.Options, output.RecallResult{
		Result: presentRecallData(result),
	})
}

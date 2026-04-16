package cli

import (
	"time"

	historypkg "github.com/forjd/aid/internal/history"
	"github.com/forjd/aid/internal/output"
)

func historyIndexCommand(args []string, streams Streams) error {
	if len(args) > 0 {
		return newError(ErrCodeUsage, "history index does not accept arguments")
	}

	ctx := streams.context()
	runtime, err := openInitializedRepo(ctx, streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	service := historypkg.Service{
		Git:   historypkg.DefaultGitClient{Ctx: ctx},
		Store: runtime.store,
		Now:   time.Now,
	}

	result, err := service.Index(ctx, runtime.env.RepoRoot, runtime.repo.ID, runtime.cfg.Indexing.IgnorePaths)
	if err != nil {
		return err
	}

	return output.RenderHistoryIndexed(streams.Out, streams.Options, output.HistoryIndexResult{
		Indexed: result.Indexed,
		Added:   result.Added,
		Updated: result.Updated,
		Removed: result.Removed,
		Mode:    historypkg.Mode(result.Initial),
	})
}

func historySearchCommand(args []string, streams Streams) error {
	query, err := joinArgs(args, "search query")
	if err != nil {
		return err
	}

	ctx := streams.context()
	runtime, err := openInitializedRepo(ctx, streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	commits, err := runtime.store.SearchCommits(ctx, runtime.repo.ID, query, defaultListLimit)
	if err != nil {
		return err
	}

	return output.RenderHistorySearch(streams.Out, streams.Options, output.HistorySearchResult{
		Query:   query,
		Commits: commits,
	})
}

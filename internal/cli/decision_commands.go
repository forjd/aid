package cli

import (
	"github.com/forjd/aid/internal/output"
	"github.com/forjd/aid/internal/store"
)

func decisionAddCommand(args []string, streams Streams) error {
	text, err := joinArgs(args, "decision text")
	if err != nil {
		return err
	}

	ctx := streams.context()
	runtime, err := openInitializedRepo(ctx, streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	decision, err := runtime.store.AddDecision(ctx, store.AddDecisionInput{
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
		return newError(ErrCodeUsage, "decide list does not accept arguments")
	}

	ctx := streams.context()
	runtime, err := openInitializedRepo(ctx, streams)
	if err != nil {
		return err
	}
	defer runtime.close()

	decisions, err := runtime.store.ListDecisions(ctx, runtime.repo.ID, runtime.env.Branch, defaultListLimit)
	if err != nil {
		return err
	}

	return output.RenderDecisions(streams.Out, streams.Options, decisions)
}

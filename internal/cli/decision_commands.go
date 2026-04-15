package cli

import (
	"context"
	"fmt"

	"github.com/forjd/aid/internal/output"
	"github.com/forjd/aid/internal/store"
)

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

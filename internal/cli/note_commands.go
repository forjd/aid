package cli

import (
	"context"
	"fmt"

	"github.com/forjd/aid/internal/output"
	"github.com/forjd/aid/internal/store"
)

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

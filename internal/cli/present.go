package cli

import (
	"github.com/forjd/aid/internal/output"
	resumepkg "github.com/forjd/aid/internal/resume"
	searchpkg "github.com/forjd/aid/internal/search"
	"github.com/forjd/aid/internal/store"
)

func presentResumeBundle(bundle resumepkg.Bundle) output.ResumeBundle {
	return output.ResumeBundle{
		ActiveTask:          cloneTaskPointer(bundle.ActiveTask),
		ActiveTaskInferred:  bundle.ActiveTaskInferred,
		ActiveTaskAmbiguous: bundle.ActiveTaskAmbiguous,
		Notes:               append([]store.Note(nil), bundle.Notes...),
		Decisions:           append([]store.Decision(nil), bundle.Decisions...),
		RecentCommits:       append([]store.Commit(nil), bundle.RecentCommits...),
		LatestHandoff:       cloneHandoffPointer(bundle.LatestHandoff),
		OpenQuestions:       append([]string(nil), bundle.OpenQuestions...),
		NextAction:          cloneStringPointer(bundle.NextAction),
	}
}

func presentRecallData(result searchpkg.Result) output.RecallData {
	return output.RecallData{
		Query:     result.Query,
		Notes:     append([]store.Note(nil), result.Notes...),
		Decisions: append([]store.Decision(nil), result.Decisions...),
		Handoffs:  append([]store.Handoff(nil), result.Handoffs...),
		Commits:   append([]store.Commit(nil), result.Commits...),
	}
}

func cloneTaskPointer(task *store.Task) *store.Task {
	if task == nil {
		return nil
	}
	cloned := *task
	return &cloned
}

func cloneHandoffPointer(handoff *store.Handoff) *store.Handoff {
	if handoff == nil {
		return nil
	}
	cloned := *handoff
	return &cloned
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

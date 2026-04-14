package output

import (
	"fmt"
	"io"

	"aid/internal/store"
)

type InitResult struct {
	RepoName      string
	RepoPath      string
	DBPath        string
	ConfigPath    string
	ConfigCreated bool
}

func RenderInit(w io.Writer, result InitResult) {
	state := "existing"
	if result.ConfigCreated {
		state = "created"
	}

	fmt.Fprintf(w, "Initialised aid for repo %s\n", result.RepoName)
	fmt.Fprintf(w, "Repo: %s\n", result.RepoPath)
	fmt.Fprintf(w, "DB: %s\n", result.DBPath)
	fmt.Fprintf(w, "Config: %s (%s)\n", result.ConfigPath, state)
}

func RenderNoteAdded(w io.Writer, note store.Note) {
	fmt.Fprintf(w, "Added note %s%s: %s\n", store.NoteRef(note.ID), branchSuffix(note.Branch), note.Text)
}

func RenderNotes(w io.Writer, notes []store.Note) {
	if len(notes) == 0 {
		fmt.Fprintln(w, "No notes.")
		return
	}

	fmt.Fprintln(w, "Notes:")
	for _, note := range notes {
		fmt.Fprintf(w, "- %s%s %s\n", store.NoteRef(note.ID), branchSuffix(note.Branch), note.Text)
	}
}

func RenderTaskAdded(w io.Writer, task store.Task) {
	fmt.Fprintf(w, "Added task %s [%s]%s: %s\n", store.TaskRef(task.ID), task.Status, branchSuffix(task.Branch), task.Text)
}

func RenderTasks(w io.Writer, tasks []store.Task) {
	if len(tasks) == 0 {
		fmt.Fprintln(w, "No tasks.")
		return
	}

	fmt.Fprintln(w, "Tasks:")
	for _, task := range tasks {
		fmt.Fprintf(w, "- %s [%s]%s %s\n", store.TaskRef(task.ID), task.Status, branchSuffix(task.Branch), task.Text)
	}
}

func RenderTaskCompleted(w io.Writer, task store.Task) {
	fmt.Fprintf(w, "Completed task %s%s: %s\n", store.TaskRef(task.ID), branchSuffix(task.Branch), task.Text)
}

func RenderDecisionAdded(w io.Writer, decision store.Decision) {
	fmt.Fprintf(w, "Added decision %s%s: %s\n", store.DecisionRef(decision.ID), branchSuffix(decision.Branch), decision.Text)
}

func RenderDecisions(w io.Writer, decisions []store.Decision) {
	if len(decisions) == 0 {
		fmt.Fprintln(w, "No decisions.")
		return
	}

	fmt.Fprintln(w, "Decisions:")
	for _, decision := range decisions {
		fmt.Fprintf(w, "- %s%s %s\n", store.DecisionRef(decision.ID), branchSuffix(decision.Branch), decision.Text)
	}
}

func branchSuffix(branch string) string {
	if branch == "" {
		return ""
	}

	return fmt.Sprintf(" [%s]", branch)
}

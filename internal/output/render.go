package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"aid/internal/store"
)

type Format string

const (
	FormatHuman   Format = "human"
	FormatBrief   Format = "brief"
	FormatJSON    Format = "json"
	FormatVerbose Format = "verbose"
)

type Options struct {
	Format   Format
	RepoPath string
}

type InitResult struct {
	RepoName      string
	RepoPath      string
	Branch        string
	DBPath        string
	ConfigPath    string
	ConfigCreated bool
}

type StatusResult struct {
	RepoName     string
	RepoPath     string
	Branch       string
	DBPath       string
	ConfigPath   string
	ConfigExists bool
	Initialized  bool
	Counts       store.StatusCounts
}

func (o Options) IsBrief() bool {
	return o.Format == FormatBrief
}

func (o Options) IsJSON() bool {
	return o.Format == FormatJSON
}

func (o Options) IsVerbose() bool {
	return o.Format == FormatVerbose
}

func WriteError(w io.Writer, command string, err error) error {
	return writeJSON(w, envelope{
		SchemaVersion: "1",
		OK:            false,
		Command:       commandName(command),
		Data:          nil,
		Error:         &errorPayload{Message: err.Error()},
	})
}

func RenderInit(w io.Writer, opts Options, result InitResult) error {
	if opts.IsJSON() {
		return writeJSON(w, envelope{
			SchemaVersion: "1",
			OK:            true,
			Command:       "init",
			Data: struct {
				Repo struct {
					Name   string `json:"name"`
					Path   string `json:"path"`
					Branch string `json:"branch"`
				} `json:"repo"`
				DBPath        string `json:"db_path"`
				ConfigPath    string `json:"config_path"`
				ConfigCreated bool   `json:"config_created"`
			}{
				Repo: struct {
					Name   string `json:"name"`
					Path   string `json:"path"`
					Branch string `json:"branch"`
				}{
					Name:   result.RepoName,
					Path:   result.RepoPath,
					Branch: result.Branch,
				},
				DBPath:        result.DBPath,
				ConfigPath:    result.ConfigPath,
				ConfigCreated: result.ConfigCreated,
			},
			Error: nil,
		})
	}

	if opts.IsBrief() {
		fmt.Fprintf(w, "Initialised %s\n", result.RepoPath)
		return nil
	}

	state := "existing"
	if result.ConfigCreated {
		state = "created"
	}

	fmt.Fprintf(w, "Initialised aid for repo %s\n", result.RepoName)
	fmt.Fprintf(w, "Repo: %s\n", result.RepoPath)
	fmt.Fprintf(w, "Branch: %s\n", result.Branch)
	fmt.Fprintf(w, "DB: %s\n", result.DBPath)
	fmt.Fprintf(w, "Config: %s (%s)\n", result.ConfigPath, state)
	return nil
}

func RenderStatus(w io.Writer, opts Options, result StatusResult) error {
	if opts.IsJSON() {
		return writeJSON(w, envelope{
			SchemaVersion: "1",
			OK:            true,
			Command:       "status",
			Data: struct {
				Repo struct {
					Name   string `json:"name"`
					Path   string `json:"path"`
					Branch string `json:"branch"`
				} `json:"repo"`
				Initialized  bool                `json:"initialized"`
				DBPath       string              `json:"db_path"`
				ConfigPath   string              `json:"config_path"`
				ConfigExists bool                `json:"config_exists"`
				Counts       *store.StatusCounts `json:"counts"`
			}{
				Repo: struct {
					Name   string `json:"name"`
					Path   string `json:"path"`
					Branch string `json:"branch"`
				}{
					Name:   result.RepoName,
					Path:   result.RepoPath,
					Branch: result.Branch,
				},
				Initialized:  result.Initialized,
				DBPath:       result.DBPath,
				ConfigPath:   result.ConfigPath,
				ConfigExists: result.ConfigExists,
				Counts:       countsPointer(result.Initialized, result.Counts),
			},
			Error: nil,
		})
	}

	if opts.IsBrief() {
		state := "not initialised"
		if result.Initialized {
			state = "initialised"
		}

		fmt.Fprintf(w, "Branch: %s\n", result.Branch)
		fmt.Fprintf(w, "Aid: %s\n", state)
		if result.Initialized {
			fmt.Fprintf(w, "Notes: %d\n", result.Counts.Notes)
			fmt.Fprintf(w, "Tasks: open %d, in_progress %d, blocked %d, done %d\n",
				result.Counts.Tasks.Open,
				result.Counts.Tasks.InProgress,
				result.Counts.Tasks.Blocked,
				result.Counts.Tasks.Done,
			)
			fmt.Fprintf(w, "Decisions: %d\n", result.Counts.Decisions)
		} else {
			fmt.Fprintln(w, "Next: run aid init")
		}
		return nil
	}

	fmt.Fprintf(w, "Repo: %s\n", result.RepoPath)
	fmt.Fprintf(w, "Branch: %s\n", result.Branch)
	fmt.Fprintf(w, "DB: %s\n", result.DBPath)
	fmt.Fprintf(w, "Config: %s\n", result.ConfigPath)
	if !result.ConfigExists {
		fmt.Fprintln(w, "Config state: missing")
	} else {
		fmt.Fprintln(w, "Config state: present")
	}

	if !result.Initialized {
		fmt.Fprintln(w, "Aid state: not initialised")
		fmt.Fprintln(w, "Next: run aid init")
		return nil
	}

	fmt.Fprintln(w, "Aid state: initialised")
	fmt.Fprintf(w, "Notes: %d\n", result.Counts.Notes)
	fmt.Fprintf(w, "Tasks: %d total\n", result.Counts.Tasks.Total)
	fmt.Fprintf(w, "  open: %d\n", result.Counts.Tasks.Open)
	fmt.Fprintf(w, "  in_progress: %d\n", result.Counts.Tasks.InProgress)
	fmt.Fprintf(w, "  blocked: %d\n", result.Counts.Tasks.Blocked)
	fmt.Fprintf(w, "  done: %d\n", result.Counts.Tasks.Done)
	fmt.Fprintf(w, "Decisions: %d\n", result.Counts.Decisions)
	return nil
}

func RenderNoteAdded(w io.Writer, opts Options, note store.Note) error {
	if opts.IsJSON() {
		return writeJSON(w, envelope{
			SchemaVersion: "1",
			OK:            true,
			Command:       "note add",
			Data: struct {
				Note notePayload `json:"note"`
			}{
				Note: noteModel(note),
			},
			Error: nil,
		})
	}

	if opts.IsBrief() {
		fmt.Fprintf(w, "%s%s %s\n", store.NoteRef(note.ID), branchSuffix(note.Branch), note.Text)
		return nil
	}

	fmt.Fprintf(w, "Added note %s%s: %s\n", store.NoteRef(note.ID), branchSuffix(note.Branch), note.Text)
	return nil
}

func RenderNotes(w io.Writer, opts Options, notes []store.Note) error {
	if opts.IsJSON() {
		items := make([]notePayload, 0, len(notes))
		for _, note := range notes {
			items = append(items, noteModel(note))
		}

		return writeJSON(w, envelope{
			SchemaVersion: "1",
			OK:            true,
			Command:       "note list",
			Data: struct {
				Notes []notePayload `json:"notes"`
			}{
				Notes: items,
			},
			Error: nil,
		})
	}

	if len(notes) == 0 {
		fmt.Fprintln(w, "No notes.")
		return nil
	}

	if opts.IsBrief() {
		for _, note := range notes {
			fmt.Fprintf(w, "%s%s %s\n", store.NoteRef(note.ID), branchSuffix(note.Branch), note.Text)
		}
		return nil
	}

	fmt.Fprintln(w, "Notes:")
	for _, note := range notes {
		fmt.Fprintf(w, "- %s%s %s\n", store.NoteRef(note.ID), branchSuffix(note.Branch), note.Text)
	}
	return nil
}

func RenderTaskAdded(w io.Writer, opts Options, task store.Task) error {
	if opts.IsJSON() {
		return writeJSON(w, envelope{
			SchemaVersion: "1",
			OK:            true,
			Command:       "task add",
			Data: struct {
				Task taskPayload `json:"task"`
			}{
				Task: taskModel(task),
			},
			Error: nil,
		})
	}

	if opts.IsBrief() {
		fmt.Fprintf(w, "%s [%s]%s %s\n", store.TaskRef(task.ID), task.Status, branchSuffix(task.Branch), task.Text)
		return nil
	}

	fmt.Fprintf(w, "Added task %s [%s]%s: %s\n", store.TaskRef(task.ID), task.Status, branchSuffix(task.Branch), task.Text)
	return nil
}

func RenderTasks(w io.Writer, opts Options, tasks []store.Task) error {
	if opts.IsJSON() {
		items := make([]taskPayload, 0, len(tasks))
		for _, task := range tasks {
			items = append(items, taskModel(task))
		}

		return writeJSON(w, envelope{
			SchemaVersion: "1",
			OK:            true,
			Command:       "task list",
			Data: struct {
				Tasks []taskPayload `json:"tasks"`
			}{
				Tasks: items,
			},
			Error: nil,
		})
	}

	if len(tasks) == 0 {
		fmt.Fprintln(w, "No tasks.")
		return nil
	}

	if opts.IsBrief() {
		for _, task := range tasks {
			fmt.Fprintf(w, "%s [%s]%s %s\n", store.TaskRef(task.ID), task.Status, branchSuffix(task.Branch), task.Text)
		}
		return nil
	}

	fmt.Fprintln(w, "Tasks:")
	for _, task := range tasks {
		fmt.Fprintf(w, "- %s [%s]%s %s\n", store.TaskRef(task.ID), task.Status, branchSuffix(task.Branch), task.Text)
	}
	return nil
}

func RenderTaskCompleted(w io.Writer, opts Options, task store.Task) error {
	if opts.IsJSON() {
		return writeJSON(w, envelope{
			SchemaVersion: "1",
			OK:            true,
			Command:       "task done",
			Data: struct {
				Task taskPayload `json:"task"`
			}{
				Task: taskModel(task),
			},
			Error: nil,
		})
	}

	if opts.IsBrief() {
		fmt.Fprintf(w, "%s [%s]%s %s\n", store.TaskRef(task.ID), task.Status, branchSuffix(task.Branch), task.Text)
		return nil
	}

	fmt.Fprintf(w, "Completed task %s%s: %s\n", store.TaskRef(task.ID), branchSuffix(task.Branch), task.Text)
	return nil
}

func RenderDecisionAdded(w io.Writer, opts Options, decision store.Decision) error {
	if opts.IsJSON() {
		return writeJSON(w, envelope{
			SchemaVersion: "1",
			OK:            true,
			Command:       "decide add",
			Data: struct {
				Decision decisionPayload `json:"decision"`
			}{
				Decision: decisionModel(decision),
			},
			Error: nil,
		})
	}

	if opts.IsBrief() {
		fmt.Fprintf(w, "%s%s %s\n", store.DecisionRef(decision.ID), branchSuffix(decision.Branch), decision.Text)
		return nil
	}

	fmt.Fprintf(w, "Added decision %s%s: %s\n", store.DecisionRef(decision.ID), branchSuffix(decision.Branch), decision.Text)
	return nil
}

func RenderDecisions(w io.Writer, opts Options, decisions []store.Decision) error {
	if opts.IsJSON() {
		items := make([]decisionPayload, 0, len(decisions))
		for _, decision := range decisions {
			items = append(items, decisionModel(decision))
		}

		return writeJSON(w, envelope{
			SchemaVersion: "1",
			OK:            true,
			Command:       "decide list",
			Data: struct {
				Decisions []decisionPayload `json:"decisions"`
			}{
				Decisions: items,
			},
			Error: nil,
		})
	}

	if len(decisions) == 0 {
		fmt.Fprintln(w, "No decisions.")
		return nil
	}

	if opts.IsBrief() {
		for _, decision := range decisions {
			fmt.Fprintf(w, "%s%s %s\n", store.DecisionRef(decision.ID), branchSuffix(decision.Branch), decision.Text)
		}
		return nil
	}

	fmt.Fprintln(w, "Decisions:")
	for _, decision := range decisions {
		fmt.Fprintf(w, "- %s%s %s\n", store.DecisionRef(decision.ID), branchSuffix(decision.Branch), decision.Text)
	}
	return nil
}

type envelope struct {
	SchemaVersion string        `json:"schema_version"`
	OK            bool          `json:"ok"`
	Command       string        `json:"command"`
	Data          any           `json:"data"`
	Error         *errorPayload `json:"error"`
}

type errorPayload struct {
	Message string `json:"message"`
}

type notePayload struct {
	ID        string  `json:"id"`
	Text      string  `json:"text"`
	Branch    *string `json:"branch"`
	Scope     string  `json:"scope"`
	CreatedAt string  `json:"created_at"`
}

type taskPayload struct {
	ID        string  `json:"id"`
	Text      string  `json:"text"`
	Status    string  `json:"status"`
	Branch    *string `json:"branch"`
	Scope     string  `json:"scope"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type decisionPayload struct {
	ID        string  `json:"id"`
	Text      string  `json:"text"`
	Rationale *string `json:"rationale"`
	Branch    *string `json:"branch"`
	CreatedAt string  `json:"created_at"`
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func commandName(path string) string {
	name := strings.TrimSpace(strings.TrimPrefix(path, "aid"))
	if name == "" {
		return "aid"
	}

	return name
}

func countsPointer(initialized bool, counts store.StatusCounts) *store.StatusCounts {
	if !initialized {
		return nil
	}

	result := counts
	return &result
}

func noteModel(note store.Note) notePayload {
	return notePayload{
		ID:        store.NoteRef(note.ID),
		Text:      note.Text,
		Branch:    nullableString(note.Branch),
		Scope:     string(note.Scope),
		CreatedAt: note.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func taskModel(task store.Task) taskPayload {
	return taskPayload{
		ID:        store.TaskRef(task.ID),
		Text:      task.Text,
		Status:    string(task.Status),
		Branch:    nullableString(task.Branch),
		Scope:     string(task.Scope),
		CreatedAt: task.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: task.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func decisionModel(decision store.Decision) decisionPayload {
	return decisionPayload{
		ID:        store.DecisionRef(decision.ID),
		Text:      decision.Text,
		Rationale: decision.Rationale,
		Branch:    nullableString(decision.Branch),
		CreatedAt: decision.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func branchSuffix(branch string) string {
	if branch == "" {
		return ""
	}

	return fmt.Sprintf(" [%s]", branch)
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}

	copy := value
	return &copy
}

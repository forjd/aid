package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	resumepkg "github.com/forjd/aid/internal/resume"
	searchpkg "github.com/forjd/aid/internal/search"
	"github.com/forjd/aid/internal/store"
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

type ResumeResult struct {
	RepoName string
	RepoPath string
	Branch   string
	Bundle   resumepkg.Bundle
}

type HandoffGenerateResult struct {
	Handoff store.Handoff
}

type HistoryIndexResult struct {
	Indexed int
	Added   int
	Updated int
	Removed int
	Mode    string
}

type HistorySearchResult struct {
	Query   string
	Commits []store.Commit
}

type RecallResult struct {
	Result searchpkg.Result
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

	if opts.IsVerbose() {
		fmt.Fprintf(w, "Initialised aid for repo %s\n", result.RepoName)
		fmt.Fprintf(w, "Repo name: %s\n", result.RepoName)
		fmt.Fprintf(w, "Repo path: %s\n", result.RepoPath)
		fmt.Fprintf(w, "Branch: %s\n", result.Branch)
		fmt.Fprintf(w, "Database: %s\n", result.DBPath)
		fmt.Fprintf(w, "Config path: %s\n", result.ConfigPath)
		fmt.Fprintf(w, "Config created: %s\n", yesNo(result.ConfigCreated))
		fmt.Fprintln(w, "Next steps:")
		fmt.Fprintln(w, "- aid status")
		fmt.Fprintln(w, "- aid resume")
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

	if opts.IsVerbose() {
		fmt.Fprintf(w, "Repo name: %s\n", result.RepoName)
		fmt.Fprintf(w, "Repo path: %s\n", result.RepoPath)
		fmt.Fprintf(w, "Branch: %s\n", result.Branch)
		fmt.Fprintf(w, "Database: %s\n", result.DBPath)
		fmt.Fprintf(w, "Config path: %s\n", result.ConfigPath)
		fmt.Fprintf(w, "Config state: %s\n", configState(result.ConfigExists))

		if !result.Initialized {
			fmt.Fprintln(w, "Aid state: not initialised")
			fmt.Fprintln(w, "Next steps:")
			fmt.Fprintln(w, "- aid init")
			return nil
		}

		fmt.Fprintln(w, "Aid state: initialised")
		fmt.Fprintf(w, "Notes: %d\n", result.Counts.Notes)
		fmt.Fprintf(w, "Decisions: %d\n", result.Counts.Decisions)
		fmt.Fprintln(w, "Task breakdown:")
		fmt.Fprintf(w, "- total: %d\n", result.Counts.Tasks.Total)
		fmt.Fprintf(w, "- open: %d\n", result.Counts.Tasks.Open)
		fmt.Fprintf(w, "- in_progress: %d\n", result.Counts.Tasks.InProgress)
		fmt.Fprintf(w, "- blocked: %d\n", result.Counts.Tasks.Blocked)
		fmt.Fprintf(w, "- done: %d\n", result.Counts.Tasks.Done)
		fmt.Fprintln(w, "Next steps:")
		fmt.Fprintln(w, "- aid resume")
		fmt.Fprintln(w, "- aid handoff generate")
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

func RenderResume(w io.Writer, opts Options, result ResumeResult) error {
	if opts.IsJSON() {
		var activeTask *taskPayload
		if result.Bundle.ActiveTask != nil {
			task := taskModel(*result.Bundle.ActiveTask)
			activeTask = &task
		}

		var latestHandoff *handoffPayload
		if result.Bundle.LatestHandoff != nil {
			handoff := handoffModel(*result.Bundle.LatestHandoff)
			latestHandoff = &handoff
		}

		notes := make([]notePayload, 0, len(result.Bundle.Notes))
		for _, note := range result.Bundle.Notes {
			notes = append(notes, noteModel(note))
		}

		decisions := make([]decisionPayload, 0, len(result.Bundle.Decisions))
		for _, decision := range result.Bundle.Decisions {
			decisions = append(decisions, decisionModel(decision))
		}

		commits := make([]commitPayload, 0, len(result.Bundle.RecentCommits))
		for _, commit := range result.Bundle.RecentCommits {
			commits = append(commits, storeCommitModel(commit))
		}

		return writeJSON(w, envelope{
			SchemaVersion: "1",
			OK:            true,
			Command:       "resume",
			Data: struct {
				Repo struct {
					Name   string `json:"name"`
					Path   string `json:"path"`
					Branch string `json:"branch"`
				} `json:"repo"`
				ActiveTask          *taskPayload      `json:"active_task"`
				ActiveTaskInferred  bool              `json:"active_task_inferred"`
				ActiveTaskAmbiguous bool              `json:"active_task_ambiguous"`
				Notes               []notePayload     `json:"notes"`
				Decisions           []decisionPayload `json:"decisions"`
				RecentCommits       []commitPayload   `json:"recent_commits"`
				LatestHandoff       *handoffPayload   `json:"latest_handoff"`
				OpenQuestions       []string          `json:"open_questions"`
				NextAction          *string           `json:"next_action"`
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
				ActiveTask:          activeTask,
				ActiveTaskInferred:  result.Bundle.ActiveTaskInferred,
				ActiveTaskAmbiguous: result.Bundle.ActiveTaskAmbiguous,
				Notes:               notes,
				Decisions:           decisions,
				RecentCommits:       commits,
				LatestHandoff:       latestHandoff,
				OpenQuestions:       append([]string(nil), result.Bundle.OpenQuestions...),
				NextAction:          result.Bundle.NextAction,
			},
			Error: nil,
		})
	}

	if opts.IsVerbose() {
		fmt.Fprintf(w, "Repo name: %s\n", result.RepoName)
		fmt.Fprintf(w, "Repo path: %s\n", result.RepoPath)
		fmt.Fprintf(w, "Branch: %s\n", result.Branch)
		fmt.Fprintf(w, "Active task inferred: %s\n", yesNo(result.Bundle.ActiveTaskInferred))
		fmt.Fprintf(w, "Active task ambiguous: %s\n", yesNo(result.Bundle.ActiveTaskAmbiguous))

		if result.Bundle.ActiveTask != nil {
			fmt.Fprintln(w, "Active task:")
			writeVerboseTask(w, taskModel(*result.Bundle.ActiveTask))
		}

		if len(result.Bundle.Notes) > 0 {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "Notes:")
			for i, note := range result.Bundle.Notes {
				if i > 0 {
					fmt.Fprintln(w)
				}
				writeVerboseNote(w, noteModel(note))
			}
		}

		if len(result.Bundle.Decisions) > 0 {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "Decisions:")
			for i, decision := range result.Bundle.Decisions {
				if i > 0 {
					fmt.Fprintln(w)
				}
				writeVerboseDecision(w, decisionModel(decision))
			}
		}

		if len(result.Bundle.RecentCommits) > 0 {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "Recent commits:")
			for i, commit := range result.Bundle.RecentCommits {
				if i > 0 {
					fmt.Fprintln(w)
				}
				writeVerboseCommit(w, storeCommitModel(commit))
			}
		}

		if result.Bundle.LatestHandoff != nil {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "Latest handoff:")
			writeVerboseHandoff(w, handoffModel(*result.Bundle.LatestHandoff))
		}

		if len(result.Bundle.OpenQuestions) > 0 {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "Open questions:")
			for _, question := range result.Bundle.OpenQuestions {
				fmt.Fprintf(w, "- %s\n", question)
			}
		}

		if result.Bundle.NextAction != nil {
			fmt.Fprintln(w)
			fmt.Fprintln(w, "Next action:")
			fmt.Fprintf(w, "- %s\n", *result.Bundle.NextAction)
		}

		return nil
	}

	fmt.Fprintf(w, "Branch: %s\n", result.Branch)
	if result.Bundle.ActiveTask != nil {
		fmt.Fprintf(w, "Task: %s\n", result.Bundle.ActiveTask.Text)
	} else if result.Bundle.ActiveTaskAmbiguous {
		fmt.Fprintln(w, "Task: ambiguous")
	}

	if len(result.Bundle.Notes) > 0 {
		fmt.Fprintln(w, "Notes:")
		for _, note := range result.Bundle.Notes {
			if opts.IsBrief() {
				fmt.Fprintf(w, "- %s\n", note.Text)
			} else {
				fmt.Fprintf(w, "- %s%s\n", note.Text, branchSuffix(note.Branch))
			}
		}
	}

	if len(result.Bundle.Decisions) > 0 {
		fmt.Fprintln(w, "Decisions:")
		for _, decision := range result.Bundle.Decisions {
			if opts.IsBrief() {
				fmt.Fprintf(w, "- %s\n", decision.Text)
			} else {
				fmt.Fprintf(w, "- %s%s\n", decision.Text, branchSuffix(decision.Branch))
			}
		}
	}

	if len(result.Bundle.RecentCommits) > 0 {
		fmt.Fprintln(w, "Recent commits:")
		for _, commit := range result.Bundle.RecentCommits {
			fmt.Fprintf(w, "- %s %s\n", shortSHA(commit.SHA), commit.Summary)
		}
	}

	if result.Bundle.LatestHandoff != nil {
		fmt.Fprintln(w, "Latest handoff:")
		fmt.Fprintf(w, "- %s%s %s\n", store.HandoffRef(result.Bundle.LatestHandoff.ID), branchSuffix(result.Bundle.LatestHandoff.Branch), previewLine(result.Bundle.LatestHandoff.Summary))
	}

	if len(result.Bundle.OpenQuestions) > 0 {
		fmt.Fprintln(w, "Open questions:")
		for _, question := range result.Bundle.OpenQuestions {
			fmt.Fprintf(w, "- %s\n", question)
		}
	}

	if result.Bundle.NextAction != nil {
		fmt.Fprintln(w, "Next:")
		fmt.Fprintf(w, "- %s\n", *result.Bundle.NextAction)
	}

	return nil
}

func RenderHandoffGenerated(w io.Writer, opts Options, result HandoffGenerateResult) error {
	if opts.IsJSON() {
		return writeJSON(w, envelope{
			SchemaVersion: "1",
			OK:            true,
			Command:       "handoff generate",
			Data: struct {
				Handoff handoffPayload `json:"handoff"`
			}{
				Handoff: handoffModel(result.Handoff),
			},
			Error: nil,
		})
	}

	if opts.IsBrief() {
		fmt.Fprintln(w, result.Handoff.Summary)
		return nil
	}

	if opts.IsVerbose() {
		fmt.Fprintf(w, "Saved handoff %s%s\n", store.HandoffRef(result.Handoff.ID), branchSuffix(result.Handoff.Branch))
		fmt.Fprintf(w, "Created: %s\n", formatTimestamp(result.Handoff.CreatedAt))
		fmt.Fprintln(w, "Summary:")
		writeIndentedText(w, result.Handoff.Summary)
		return nil
	}

	fmt.Fprintf(w, "Saved handoff %s%s\n", store.HandoffRef(result.Handoff.ID), branchSuffix(result.Handoff.Branch))
	fmt.Fprintln(w)
	fmt.Fprintln(w, result.Handoff.Summary)
	return nil
}

func RenderHandoffs(w io.Writer, opts Options, handoffs []store.Handoff) error {
	if opts.IsJSON() {
		items := make([]handoffPayload, 0, len(handoffs))
		for _, handoff := range handoffs {
			items = append(items, handoffModel(handoff))
		}

		return writeJSON(w, envelope{
			SchemaVersion: "1",
			OK:            true,
			Command:       "handoff list",
			Data: struct {
				Handoffs []handoffPayload `json:"handoffs"`
			}{
				Handoffs: items,
			},
			Error: nil,
		})
	}

	if len(handoffs) == 0 {
		fmt.Fprintln(w, "No handoffs.")
		return nil
	}

	if opts.IsBrief() {
		for _, handoff := range handoffs {
			fmt.Fprintf(w, "%s%s %s\n", store.HandoffRef(handoff.ID), branchSuffix(handoff.Branch), previewLine(handoff.Summary))
		}
		return nil
	}

	if opts.IsVerbose() {
		for i, handoff := range handoffs {
			if i > 0 {
				fmt.Fprintln(w)
			}
			writeVerboseHandoff(w, handoffModel(handoff))
		}
		return nil
	}

	fmt.Fprintln(w, "Handoffs:")
	for _, handoff := range handoffs {
		fmt.Fprintf(w, "- %s%s %s\n", store.HandoffRef(handoff.ID), branchSuffix(handoff.Branch), previewLine(handoff.Summary))
	}
	return nil
}

func RenderHistoryIndexed(w io.Writer, opts Options, result HistoryIndexResult) error {
	if opts.IsJSON() {
		return writeJSON(w, envelope{
			SchemaVersion: "1",
			OK:            true,
			Command:       "history index",
			Data: struct {
				Indexed int    `json:"indexed"`
				Added   int    `json:"added"`
				Updated int    `json:"updated"`
				Removed int    `json:"removed"`
				Mode    string `json:"mode"`
			}{
				Indexed: result.Indexed,
				Added:   result.Added,
				Updated: result.Updated,
				Removed: result.Removed,
				Mode:    result.Mode,
			},
			Error: nil,
		})
	}

	if opts.IsVerbose() {
		fmt.Fprintln(w, "History index complete")
		fmt.Fprintf(w, "Commits indexed: %d\n", result.Indexed)
		fmt.Fprintf(w, "Commits added: %d\n", result.Added)
		fmt.Fprintf(w, "Commits updated: %d\n", result.Updated)
		fmt.Fprintf(w, "Commits removed: %d\n", result.Removed)
		fmt.Fprintf(w, "Mode: %s\n", result.Mode)
		return nil
	}

	fmt.Fprintf(w, "Indexed %d commits.\n", result.Indexed)
	return nil
}

func RenderHistorySearch(w io.Writer, opts Options, result HistorySearchResult) error {
	items := make([]commitPayload, 0, len(result.Commits))
	for _, commit := range result.Commits {
		items = append(items, storeCommitModel(commit))
	}

	if opts.IsJSON() {
		return writeJSON(w, envelope{
			SchemaVersion: "1",
			OK:            true,
			Command:       "history search",
			Data: struct {
				Query   string          `json:"query"`
				Commits []commitPayload `json:"commits"`
			}{
				Query:   result.Query,
				Commits: items,
			},
			Error: nil,
		})
	}

	if len(items) == 0 {
		fmt.Fprintln(w, "No matching commits.")
		return nil
	}

	if opts.IsBrief() {
		for _, commit := range items {
			fmt.Fprintf(w, "%s %s\n", shortSHA(commit.SHA), commit.Summary)
		}
		return nil
	}

	if opts.IsVerbose() {
		for i, commit := range items {
			if i > 0 {
				fmt.Fprintln(w)
			}
			writeVerboseCommit(w, commit)
		}
		return nil
	}

	fmt.Fprintln(w, "Commits:")
	for _, commit := range items {
		fmt.Fprintf(w, "- %s %s\n", shortSHA(commit.SHA), commit.Summary)
		if len(commit.ChangedPaths) > 0 {
			fmt.Fprintf(w, "  paths: %s\n", strings.Join(commit.ChangedPaths, ", "))
		}
	}
	return nil
}

func RenderRecall(w io.Writer, opts Options, result RecallResult) error {
	notes := make([]notePayload, 0, len(result.Result.Notes))
	for _, note := range result.Result.Notes {
		notes = append(notes, noteModel(note))
	}

	decisions := make([]decisionPayload, 0, len(result.Result.Decisions))
	for _, decision := range result.Result.Decisions {
		decisions = append(decisions, decisionModel(decision))
	}

	handoffs := make([]handoffPayload, 0, len(result.Result.Handoffs))
	for _, handoff := range result.Result.Handoffs {
		handoffs = append(handoffs, handoffModel(handoff))
	}

	commits := make([]commitPayload, 0, len(result.Result.Commits))
	for _, commit := range result.Result.Commits {
		commits = append(commits, storeCommitModel(commit))
	}

	if opts.IsJSON() {
		return writeJSON(w, envelope{
			SchemaVersion: "1",
			OK:            true,
			Command:       "recall",
			Data: struct {
				Query     string            `json:"query"`
				Notes     []notePayload     `json:"notes"`
				Decisions []decisionPayload `json:"decisions"`
				Handoffs  []handoffPayload  `json:"handoffs"`
				Commits   []commitPayload   `json:"commits"`
			}{
				Query:     result.Result.Query,
				Notes:     notes,
				Decisions: decisions,
				Handoffs:  handoffs,
				Commits:   commits,
			},
			Error: nil,
		})
	}

	if len(notes) == 0 && len(decisions) == 0 && len(handoffs) == 0 && len(commits) == 0 {
		fmt.Fprintln(w, "No matching context.")
		return nil
	}

	if opts.IsVerbose() {
		if len(notes) > 0 {
			fmt.Fprintln(w, "Notes:")
			for i, note := range notes {
				if i > 0 {
					fmt.Fprintln(w)
				}
				writeVerboseNote(w, note)
			}
		}

		if len(decisions) > 0 {
			if len(notes) > 0 {
				fmt.Fprintln(w)
				fmt.Fprintln(w)
			}
			fmt.Fprintln(w, "Decisions:")
			for i, decision := range decisions {
				if i > 0 {
					fmt.Fprintln(w)
				}
				writeVerboseDecision(w, decision)
			}
		}

		if len(handoffs) > 0 {
			if len(notes) > 0 || len(decisions) > 0 {
				fmt.Fprintln(w)
				fmt.Fprintln(w)
			}
			fmt.Fprintln(w, "Handoffs:")
			for i, handoff := range handoffs {
				if i > 0 {
					fmt.Fprintln(w)
				}
				writeVerboseHandoff(w, handoff)
			}
		}

		if len(commits) > 0 {
			if len(notes) > 0 || len(decisions) > 0 || len(handoffs) > 0 {
				fmt.Fprintln(w)
				fmt.Fprintln(w)
			}
			fmt.Fprintln(w, "Commits:")
			for i, commit := range commits {
				if i > 0 {
					fmt.Fprintln(w)
				}
				writeVerboseCommit(w, commit)
			}
		}

		return nil
	}

	if len(notes) > 0 {
		fmt.Fprintln(w, "Notes:")
		for _, note := range notes {
			fmt.Fprintf(w, "- %s%s\n", note.Text, branchSuffixValue(note.Branch))
		}
	}

	if len(decisions) > 0 {
		fmt.Fprintln(w, "Decisions:")
		for _, decision := range decisions {
			fmt.Fprintf(w, "- %s%s\n", decision.Text, branchSuffixValue(decision.Branch))
		}
	}

	if len(handoffs) > 0 {
		fmt.Fprintln(w, "Handoffs:")
		for _, handoff := range handoffs {
			fmt.Fprintf(w, "- %s%s %s\n", handoff.ID, branchSuffixValue(handoff.Branch), previewLine(handoff.Summary))
		}
	}

	if len(commits) > 0 {
		fmt.Fprintln(w, "Commits:")
		for _, commit := range commits {
			fmt.Fprintf(w, "- %s %s\n", shortSHA(commit.SHA), commit.Summary)
		}
	}

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

	if opts.IsVerbose() {
		writeVerboseNote(w, noteModel(note))
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

	if opts.IsVerbose() {
		for i, note := range notes {
			if i > 0 {
				fmt.Fprintln(w)
			}
			writeVerboseNote(w, noteModel(note))
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

	if opts.IsVerbose() {
		writeVerboseTask(w, taskModel(task))
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

	if opts.IsVerbose() {
		for i, task := range tasks {
			if i > 0 {
				fmt.Fprintln(w)
			}
			writeVerboseTask(w, taskModel(task))
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
	return RenderTaskStatusUpdated(w, opts, "done", "Completed", task)
}

func RenderTaskStatusUpdated(w io.Writer, opts Options, command string, verb string, task store.Task) error {
	if opts.IsJSON() {
		return writeJSON(w, envelope{
			SchemaVersion: "1",
			OK:            true,
			Command:       "task " + command,
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

	if opts.IsVerbose() {
		writeVerboseTask(w, taskModel(task))
		return nil
	}

	fmt.Fprintf(w, "%s task %s%s: %s\n", verb, store.TaskRef(task.ID), branchSuffix(task.Branch), task.Text)
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

	if opts.IsVerbose() {
		writeVerboseDecision(w, decisionModel(decision))
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

	if opts.IsVerbose() {
		for i, decision := range decisions {
			if i > 0 {
				fmt.Fprintln(w)
			}
			writeVerboseDecision(w, decisionModel(decision))
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

type commitPayload struct {
	SHA          string   `json:"sha"`
	Summary      string   `json:"summary"`
	Message      string   `json:"message"`
	Author       string   `json:"author"`
	CommittedAt  string   `json:"committed_at"`
	ChangedPaths []string `json:"changed_paths"`
}

type handoffPayload struct {
	ID        string  `json:"id"`
	Branch    *string `json:"branch"`
	Summary   string  `json:"summary"`
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

func storeCommitModel(commit store.Commit) commitPayload {
	return commitPayload{
		SHA:          commit.SHA,
		Summary:      commit.Summary,
		Message:      commit.Message,
		Author:       commit.Author,
		CommittedAt:  commit.CommittedAt.Format("2006-01-02T15:04:05Z07:00"),
		ChangedPaths: append([]string(nil), commit.ChangedPaths...),
	}
}

func handoffModel(handoff store.Handoff) handoffPayload {
	return handoffPayload{
		ID:        store.HandoffRef(handoff.ID),
		Branch:    nullableString(handoff.Branch),
		Summary:   handoff.Summary,
		CreatedAt: handoff.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func branchSuffix(branch string) string {
	if branch == "" {
		return ""
	}

	return fmt.Sprintf(" [%s]", branch)
}

func branchSuffixValue(branch *string) string {
	if branch == nil || *branch == "" {
		return ""
	}

	return fmt.Sprintf(" [%s]", *branch)
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}

	copy := value
	return &copy
}

func shortSHA(value string) string {
	if len(value) <= 7 {
		return value
	}

	return value[:7]
}

func previewLine(value string) string {
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}

	return ""
}

func writeVerboseNote(w io.Writer, note notePayload) {
	fmt.Fprintf(w, "Note %s\n", note.ID)
	fmt.Fprintf(w, "Branch: %s\n", verboseBranch(note.Branch))
	fmt.Fprintf(w, "Scope: %s\n", note.Scope)
	fmt.Fprintf(w, "Created: %s\n", note.CreatedAt)
	fmt.Fprintf(w, "Text: %s\n", note.Text)
}

func writeVerboseTask(w io.Writer, task taskPayload) {
	fmt.Fprintf(w, "Task %s\n", task.ID)
	fmt.Fprintf(w, "Status: %s\n", task.Status)
	fmt.Fprintf(w, "Branch: %s\n", verboseBranch(task.Branch))
	fmt.Fprintf(w, "Scope: %s\n", task.Scope)
	fmt.Fprintf(w, "Created: %s\n", task.CreatedAt)
	fmt.Fprintf(w, "Updated: %s\n", task.UpdatedAt)
	fmt.Fprintf(w, "Text: %s\n", task.Text)
}

func writeVerboseDecision(w io.Writer, decision decisionPayload) {
	fmt.Fprintf(w, "Decision %s\n", decision.ID)
	fmt.Fprintf(w, "Branch: %s\n", verboseBranch(decision.Branch))
	fmt.Fprintf(w, "Created: %s\n", decision.CreatedAt)
	fmt.Fprintf(w, "Text: %s\n", decision.Text)
	if decision.Rationale != nil && strings.TrimSpace(*decision.Rationale) != "" {
		fmt.Fprintf(w, "Rationale: %s\n", *decision.Rationale)
	}
}

func writeVerboseHandoff(w io.Writer, handoff handoffPayload) {
	fmt.Fprintf(w, "Handoff %s\n", handoff.ID)
	fmt.Fprintf(w, "Branch: %s\n", verboseBranch(handoff.Branch))
	fmt.Fprintf(w, "Created: %s\n", handoff.CreatedAt)
	fmt.Fprintln(w, "Summary:")
	writeIndentedText(w, handoff.Summary)
}

func writeVerboseCommit(w io.Writer, commit commitPayload) {
	fmt.Fprintf(w, "Commit %s\n", shortSHA(commit.SHA))
	fmt.Fprintf(w, "Summary: %s\n", commit.Summary)
	fmt.Fprintf(w, "Author: %s\n", commit.Author)
	fmt.Fprintf(w, "Committed: %s\n", commit.CommittedAt)
	if len(commit.ChangedPaths) > 0 {
		fmt.Fprintf(w, "Paths: %s\n", strings.Join(commit.ChangedPaths, ", "))
	}
	if strings.TrimSpace(commit.Message) != "" {
		fmt.Fprintln(w, "Message:")
		writeIndentedText(w, commit.Message)
	}
}

func writeIndentedText(w io.Writer, value string) {
	for _, line := range strings.Split(strings.TrimSpace(value), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fmt.Fprintf(w, "  %s\n", line)
	}
}

func verboseBranch(branch *string) string {
	if branch == nil || strings.TrimSpace(*branch) == "" {
		return "repo"
	}

	return *branch
}

func configState(exists bool) string {
	if exists {
		return "present"
	}

	return "missing"
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}

	return "no"
}

func formatTimestamp(value time.Time) string {
	return value.Format("2006-01-02T15:04:05Z07:00")
}

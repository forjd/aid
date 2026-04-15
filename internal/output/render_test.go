package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	resumepkg "github.com/forjd/aid/internal/resume"
	searchpkg "github.com/forjd/aid/internal/search"
	"github.com/forjd/aid/internal/store"
)

func TestCommandName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "aid", want: "aid"},
		{input: " aid ", want: "aid"},
		{input: "aid note add", want: "note add"},
		{input: "aid history search", want: "history search"},
		{input: "note add", want: "note add"},
	}

	for _, test := range tests {
		if got := commandName(test.input); got != test.want {
			t.Fatalf("commandName(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestWriteError(t *testing.T) {
	t.Run("writes json envelope", func(t *testing.T) {
		payload := renderJSONEnvelope(t, func(w io.Writer) error {
			return WriteError(w, "aid note add", errors.New("bad note"))
		})

		if payload.SchemaVersion != "1" {
			t.Fatalf("unexpected schema version: %#v", payload)
		}
		if payload.OK {
			t.Fatalf("expected error payload, got %#v", payload)
		}
		if payload.Command != "note add" {
			t.Fatalf("unexpected command: %#v", payload)
		}
		if payload.Error == nil || payload.Error.Message != "bad note" {
			t.Fatalf("unexpected error payload: %#v", payload)
		}
		if string(payload.Data) != "null" {
			t.Fatalf("expected null data payload, got %s", payload.Data)
		}
	})

	t.Run("returns writer error", func(t *testing.T) {
		err := WriteError(failingWriter{}, "aid note add", errors.New("bad note"))
		if !errors.Is(err, io.ErrClosedPipe) {
			t.Fatalf("expected io.ErrClosedPipe, got %v", err)
		}
	})
}

func TestRenderNoteAdded(t *testing.T) {
	note := sampleNote()

	human := renderString(t, func(w io.Writer) error {
		return RenderNoteAdded(w, Options{}, note)
	})
	if human != "Added note note_7 [main]: Refresh retry investigation\n" {
		t.Fatalf("unexpected human output:\n%s", human)
	}

	brief := renderString(t, func(w io.Writer) error {
		return RenderNoteAdded(w, Options{Format: FormatBrief}, note)
	})
	if brief != "note_7 [main] Refresh retry investigation\n" {
		t.Fatalf("unexpected brief output:\n%s", brief)
	}

	verbose := renderString(t, func(w io.Writer) error {
		return RenderNoteAdded(w, Options{Format: FormatVerbose}, note)
	})
	if verbose != "Note note_7\nBranch: main\nScope: branch\nCreated: 2026-04-15T10:11:12Z\nText: Refresh retry investigation\n" {
		t.Fatalf("unexpected verbose output:\n%s", verbose)
	}

	jsonPayload := renderJSONEnvelope(t, func(w io.Writer) error {
		return RenderNoteAdded(w, Options{Format: FormatJSON}, note)
	})
	if jsonPayload.Command != "note add" {
		t.Fatalf("unexpected command: %#v", jsonPayload)
	}

	var data struct {
		Note struct {
			ID     string  `json:"id"`
			Text   string  `json:"text"`
			Branch *string `json:"branch"`
			Scope  string  `json:"scope"`
		} `json:"note"`
	}
	unmarshalJSON(t, jsonPayload.Data, &data)
	if data.Note.ID != "note_7" || data.Note.Text != note.Text || data.Note.Branch == nil || *data.Note.Branch != "main" || data.Note.Scope != "branch" {
		t.Fatalf("unexpected note payload: %#v", data)
	}
}

func TestRenderInit(t *testing.T) {
	result := InitResult{
		RepoName:      "aid",
		RepoPath:      "/tmp/aid",
		Branch:        "main",
		DBPath:        "/tmp/aid/.aid/aid.db",
		ConfigPath:    "/tmp/aid/.aid/config.toml",
		ConfigCreated: false,
	}

	brief := renderString(t, func(w io.Writer) error {
		return RenderInit(w, Options{Format: FormatBrief}, result)
	})
	if brief != "Initialised /tmp/aid\n" {
		t.Fatalf("unexpected brief init output:\n%s", brief)
	}

	human := renderString(t, func(w io.Writer) error {
		return RenderInit(w, Options{}, result)
	})
	if human != "Initialised aid for repo aid\nRepo: /tmp/aid\nBranch: main\nDB: /tmp/aid/.aid/aid.db\nConfig: /tmp/aid/.aid/config.toml (existing)\n" {
		t.Fatalf("unexpected human init output:\n%s", human)
	}

	result.ConfigCreated = true
	verbose := renderString(t, func(w io.Writer) error {
		return RenderInit(w, Options{Format: FormatVerbose}, result)
	})
	if verbose != "Initialised aid for repo aid\nRepo name: aid\nRepo path: /tmp/aid\nBranch: main\nDatabase: /tmp/aid/.aid/aid.db\nConfig path: /tmp/aid/.aid/config.toml\nConfig created: yes\nNext steps:\n- aid status\n- aid resume\n" {
		t.Fatalf("unexpected verbose init output:\n%s", verbose)
	}

	jsonPayload := renderJSONEnvelope(t, func(w io.Writer) error {
		return RenderInit(w, Options{Format: FormatJSON}, result)
	})
	if jsonPayload.Command != "init" {
		t.Fatalf("unexpected init json command: %#v", jsonPayload)
	}

	var data struct {
		Repo struct {
			Name   string `json:"name"`
			Path   string `json:"path"`
			Branch string `json:"branch"`
		} `json:"repo"`
		ConfigCreated bool `json:"config_created"`
	}
	unmarshalJSON(t, jsonPayload.Data, &data)
	if data.Repo.Name != "aid" || data.Repo.Path != "/tmp/aid" || data.Repo.Branch != "main" || !data.ConfigCreated {
		t.Fatalf("unexpected init payload: %#v", data)
	}
}

func TestRenderStatus(t *testing.T) {
	notInitialised := StatusResult{
		RepoName:     "aid",
		RepoPath:     "/tmp/aid",
		Branch:       "main",
		DBPath:       "/tmp/aid/.aid/aid.db",
		ConfigPath:   "/tmp/aid/.aid/config.toml",
		ConfigExists: false,
		Initialized:  false,
	}

	brief := renderString(t, func(w io.Writer) error {
		return RenderStatus(w, Options{Format: FormatBrief}, notInitialised)
	})
	if brief != "Branch: main\nAid: not initialised\nNext: run aid init\n" {
		t.Fatalf("unexpected brief status output:\n%s", brief)
	}

	verboseNotInitialised := renderString(t, func(w io.Writer) error {
		return RenderStatus(w, Options{Format: FormatVerbose}, notInitialised)
	})
	if verboseNotInitialised != "Repo name: aid\nRepo path: /tmp/aid\nBranch: main\nDatabase: /tmp/aid/.aid/aid.db\nConfig path: /tmp/aid/.aid/config.toml\nConfig state: missing\nAid state: not initialised\nNext steps:\n- aid init\n" {
		t.Fatalf("unexpected verbose uninitialised status output:\n%s", verboseNotInitialised)
	}

	jsonNotInitialised := renderJSONEnvelope(t, func(w io.Writer) error {
		return RenderStatus(w, Options{Format: FormatJSON}, notInitialised)
	})
	var jsonNotInitialisedData struct {
		Initialized bool                `json:"initialized"`
		Counts      *store.StatusCounts `json:"counts"`
	}
	unmarshalJSON(t, jsonNotInitialised.Data, &jsonNotInitialisedData)
	if jsonNotInitialisedData.Initialized || jsonNotInitialisedData.Counts != nil {
		t.Fatalf("unexpected uninitialised status payload: %#v", jsonNotInitialisedData)
	}

	initialised := StatusResult{
		RepoName:     "aid",
		RepoPath:     "/tmp/aid",
		Branch:       "main",
		DBPath:       "/tmp/aid/.aid/aid.db",
		ConfigPath:   "/tmp/aid/.aid/config.toml",
		ConfigExists: true,
		Initialized:  true,
		Counts: store.StatusCounts{
			Notes:     2,
			Decisions: 1,
			Tasks: store.TaskCounts{
				Total:      3,
				Open:       1,
				InProgress: 1,
				Done:       1,
			},
		},
	}

	verbose := renderString(t, func(w io.Writer) error {
		return RenderStatus(w, Options{Format: FormatVerbose}, initialised)
	})
	if verbose != "Repo name: aid\nRepo path: /tmp/aid\nBranch: main\nDatabase: /tmp/aid/.aid/aid.db\nConfig path: /tmp/aid/.aid/config.toml\nConfig state: present\nAid state: initialised\nNotes: 2\nDecisions: 1\nTask breakdown:\n- total: 3\n- open: 1\n- in_progress: 1\n- blocked: 0\n- done: 1\nNext steps:\n- aid resume\n- aid handoff generate\n" {
		t.Fatalf("unexpected verbose status output:\n%s", verbose)
	}

	human := renderString(t, func(w io.Writer) error {
		return RenderStatus(w, Options{}, initialised)
	})
	if human != "Repo: /tmp/aid\nBranch: main\nDB: /tmp/aid/.aid/aid.db\nConfig: /tmp/aid/.aid/config.toml\nConfig state: present\nAid state: initialised\nNotes: 2\nTasks: 3 total\n  open: 1\n  in_progress: 1\n  blocked: 0\n  done: 1\nDecisions: 1\n" {
		t.Fatalf("unexpected human status output:\n%s", human)
	}

	jsonInitialised := renderJSONEnvelope(t, func(w io.Writer) error {
		return RenderStatus(w, Options{Format: FormatJSON}, initialised)
	})
	var jsonInitialisedData struct {
		Initialized bool                `json:"initialized"`
		Counts      *store.StatusCounts `json:"counts"`
	}
	unmarshalJSON(t, jsonInitialised.Data, &jsonInitialisedData)
	if !jsonInitialisedData.Initialized || jsonInitialisedData.Counts == nil || jsonInitialisedData.Counts.Tasks.InProgress != 1 {
		t.Fatalf("unexpected initialised status payload: %#v", jsonInitialisedData)
	}
}

func TestRenderResume(t *testing.T) {
	result := ResumeResult{
		RepoName: "aid",
		RepoPath: "/tmp/aid",
		Branch:   "main",
		Bundle: resumepkg.Bundle{
			ActiveTask: sampleResumeTaskPointer(),
			Notes: []store.Note{
				sampleNote(),
			},
			Decisions: []store.Decision{
				sampleDecision(),
			},
			RecentCommits: []store.Commit{
				sampleCommit(),
			},
			LatestHandoff: sampleHandoffPointer(),
			OpenQuestions: []string{"Why does refresh retry still fail?"},
			NextAction:    stringPointer("continue Fix refresh retry path"),
		},
	}

	brief := renderString(t, func(w io.Writer) error {
		return RenderResume(w, Options{Format: FormatBrief}, result)
	})
	if brief != "Branch: main\nTask: Fix refresh retry path\nNotes:\n- Refresh retry investigation\nDecisions:\n- Use single refresh retry\nRecent commits:\n- abc1234 fix: refresh retry path\nLatest handoff:\n- handoff_3 [main] Investigate refresh retry failure\nOpen questions:\n- Why does refresh retry still fail?\nNext:\n- continue Fix refresh retry path\n" {
		t.Fatalf("unexpected brief resume output:\n%s", brief)
	}

	verbose := renderString(t, func(w io.Writer) error {
		return RenderResume(w, Options{Format: FormatVerbose}, result)
	})
	for _, want := range []string{
		"Repo name: aid",
		"Active task inferred: no",
		"Task task_9",
		"Handoff handoff_3",
		"Commit abc1234",
		"Message:\n  fix: refresh retry path\n  Handle 401 once.",
		"Next action:\n- continue Fix refresh retry path",
	} {
		if !strings.Contains(verbose, want) {
			t.Fatalf("expected verbose resume output to contain %q\n\n%s", want, verbose)
		}
	}

	jsonPayload := renderJSONEnvelope(t, func(w io.Writer) error {
		return RenderResume(w, Options{Format: FormatJSON}, result)
	})
	if jsonPayload.Command != "resume" {
		t.Fatalf("unexpected resume json command: %#v", jsonPayload)
	}

	var data struct {
		Repo struct {
			Branch string `json:"branch"`
		} `json:"repo"`
		ActiveTask *struct {
			ID string `json:"id"`
		} `json:"active_task"`
		RecentCommits []any    `json:"recent_commits"`
		OpenQuestions []string `json:"open_questions"`
		NextAction    *string  `json:"next_action"`
	}
	unmarshalJSON(t, jsonPayload.Data, &data)
	if data.Repo.Branch != "main" || data.ActiveTask == nil || data.ActiveTask.ID != "task_9" || len(data.RecentCommits) != 1 || len(data.OpenQuestions) != 1 || data.NextAction == nil || *data.NextAction != "continue Fix refresh retry path" {
		t.Fatalf("unexpected resume payload: %#v", data)
	}
}

func TestRenderHandoffGeneratedAndList(t *testing.T) {
	handoff := sampleHandoff()

	briefGenerated := renderString(t, func(w io.Writer) error {
		return RenderHandoffGenerated(w, Options{Format: FormatBrief}, HandoffGenerateResult{Handoff: handoff})
	})
	if briefGenerated != "Investigate refresh retry failure\nNext: inspect auth middleware\n" {
		t.Fatalf("unexpected brief handoff output:\n%s", briefGenerated)
	}

	verboseGenerated := renderString(t, func(w io.Writer) error {
		return RenderHandoffGenerated(w, Options{Format: FormatVerbose}, HandoffGenerateResult{Handoff: handoff})
	})
	if verboseGenerated != "Saved handoff handoff_3 [main]\nCreated: 2026-04-15T10:11:12Z\nSummary:\n  Investigate refresh retry failure\n  Next: inspect auth middleware\n" {
		t.Fatalf("unexpected verbose generated handoff output:\n%s", verboseGenerated)
	}

	jsonGenerated := renderJSONEnvelope(t, func(w io.Writer) error {
		return RenderHandoffGenerated(w, Options{Format: FormatJSON}, HandoffGenerateResult{Handoff: handoff})
	})
	if jsonGenerated.Command != "handoff generate" {
		t.Fatalf("unexpected generated handoff json command: %#v", jsonGenerated)
	}

	emptyList := renderString(t, func(w io.Writer) error {
		return RenderHandoffs(w, Options{}, nil)
	})
	if emptyList != "No handoffs.\n" {
		t.Fatalf("unexpected empty handoff list output:\n%s", emptyList)
	}

	handoffs := []store.Handoff{handoff, sampleRepoHandoff()}
	verboseList := renderString(t, func(w io.Writer) error {
		return RenderHandoffs(w, Options{Format: FormatVerbose}, handoffs)
	})
	for _, want := range []string{"Handoff handoff_3", "Branch: main", "Handoff handoff_5", "Branch: repo"} {
		if !strings.Contains(verboseList, want) {
			t.Fatalf("expected verbose handoff list to contain %q\n\n%s", want, verboseList)
		}
	}

	jsonList := renderJSONEnvelope(t, func(w io.Writer) error {
		return RenderHandoffs(w, Options{Format: FormatJSON}, handoffs)
	})
	var listData struct {
		Handoffs []struct {
			ID string `json:"id"`
		} `json:"handoffs"`
	}
	unmarshalJSON(t, jsonList.Data, &listData)
	if len(listData.Handoffs) != 2 || listData.Handoffs[1].ID != "handoff_5" {
		t.Fatalf("unexpected handoff list payload: %#v", listData)
	}
}

func TestRenderHistoryIndexed(t *testing.T) {
	result := HistoryIndexResult{Indexed: 12, Added: 4, Updated: 3, Removed: 1, Mode: "incremental"}

	human := renderString(t, func(w io.Writer) error {
		return RenderHistoryIndexed(w, Options{}, result)
	})
	if human != "Indexed 12 commits.\n" {
		t.Fatalf("unexpected human history indexed output:\n%s", human)
	}

	verbose := renderString(t, func(w io.Writer) error {
		return RenderHistoryIndexed(w, Options{Format: FormatVerbose}, result)
	})
	if verbose != "History index complete\nCommits indexed: 12\nCommits added: 4\nCommits updated: 3\nCommits removed: 1\nMode: incremental\n" {
		t.Fatalf("unexpected verbose history indexed output:\n%s", verbose)
	}

	jsonPayload := renderJSONEnvelope(t, func(w io.Writer) error {
		return RenderHistoryIndexed(w, Options{Format: FormatJSON}, result)
	})
	var data struct {
		Indexed int    `json:"indexed"`
		Mode    string `json:"mode"`
	}
	unmarshalJSON(t, jsonPayload.Data, &data)
	if data.Indexed != 12 || data.Mode != "incremental" {
		t.Fatalf("unexpected history indexed payload: %#v", data)
	}
}

func TestRenderNotesTaskAddedAndDecisions(t *testing.T) {
	emptyNotes := renderString(t, func(w io.Writer) error {
		return RenderNotes(w, Options{}, nil)
	})
	if emptyNotes != "No notes.\n" {
		t.Fatalf("unexpected empty notes output:\n%s", emptyNotes)
	}

	verboseNotes := renderString(t, func(w io.Writer) error {
		return RenderNotes(w, Options{Format: FormatVerbose}, []store.Note{sampleNote(), sampleRepoNote()})
	})
	for _, want := range []string{"Note note_7", "Branch: main", "Note note_8", "Branch: repo"} {
		if !strings.Contains(verboseNotes, want) {
			t.Fatalf("expected verbose notes output to contain %q\n\n%s", want, verboseNotes)
		}
	}

	humanNotes := renderString(t, func(w io.Writer) error {
		return RenderNotes(w, Options{}, []store.Note{sampleNote()})
	})
	if humanNotes != "Notes:\n- note_7 [main] Refresh retry investigation\n" {
		t.Fatalf("unexpected human notes output:\n%s", humanNotes)
	}

	jsonNotes := renderJSONEnvelope(t, func(w io.Writer) error {
		return RenderNotes(w, Options{Format: FormatJSON}, []store.Note{sampleNote()})
	})
	if jsonNotes.Command != "note list" {
		t.Fatalf("unexpected note list json command: %#v", jsonNotes)
	}

	taskAddedHuman := renderString(t, func(w io.Writer) error {
		return RenderTaskAdded(w, Options{}, sampleTask())
	})
	if taskAddedHuman != "Added task task_9 [done] [feat/auth]: Fix refresh retry path\n" {
		t.Fatalf("unexpected task added output:\n%s", taskAddedHuman)
	}

	taskAddedVerbose := renderString(t, func(w io.Writer) error {
		return RenderTaskAdded(w, Options{Format: FormatVerbose}, sampleTask())
	})
	if taskAddedVerbose != "Task task_9\nStatus: done\nBranch: feat/auth\nScope: branch\nCreated: 2026-04-15T10:11:12Z\nUpdated: 2026-04-15T10:13:12Z\nText: Fix refresh retry path\n" {
		t.Fatalf("unexpected verbose task added output:\n%s", taskAddedVerbose)
	}

	briefTasks := renderString(t, func(w io.Writer) error {
		return RenderTasks(w, Options{Format: FormatBrief}, []store.Task{sampleTask()})
	})
	if briefTasks != "task_9 [done] [feat/auth] Fix refresh retry path\n" {
		t.Fatalf("unexpected brief tasks output:\n%s", briefTasks)
	}

	jsonTaskAdded := renderJSONEnvelope(t, func(w io.Writer) error {
		return RenderTaskAdded(w, Options{Format: FormatJSON}, sampleTask())
	})
	if jsonTaskAdded.Command != "task add" {
		t.Fatalf("unexpected task add json command: %#v", jsonTaskAdded)
	}

	emptyDecisions := renderString(t, func(w io.Writer) error {
		return RenderDecisions(w, Options{}, nil)
	})
	if emptyDecisions != "No decisions.\n" {
		t.Fatalf("unexpected empty decisions output:\n%s", emptyDecisions)
	}

	verboseDecisions := renderString(t, func(w io.Writer) error {
		return RenderDecisions(w, Options{Format: FormatVerbose}, []store.Decision{sampleDecision(), sampleRepoDecision()})
	})
	for _, want := range []string{"Decision decision_4", "Rationale: Avoid duplicate token refreshes", "Decision decision_6", "Branch: repo"} {
		if !strings.Contains(verboseDecisions, want) {
			t.Fatalf("expected verbose decisions output to contain %q\n\n%s", want, verboseDecisions)
		}
	}

	humanDecisions := renderString(t, func(w io.Writer) error {
		return RenderDecisions(w, Options{}, []store.Decision{sampleDecision()})
	})
	if humanDecisions != "Decisions:\n- decision_4 [main] Use single refresh retry\n" {
		t.Fatalf("unexpected human decisions output:\n%s", humanDecisions)
	}

	jsonDecisions := renderJSONEnvelope(t, func(w io.Writer) error {
		return RenderDecisions(w, Options{Format: FormatJSON}, []store.Decision{sampleDecision()})
	})
	if jsonDecisions.Command != "decide list" {
		t.Fatalf("unexpected decision list json command: %#v", jsonDecisions)
	}

	briefHandoffs := renderString(t, func(w io.Writer) error {
		return RenderHandoffs(w, Options{Format: FormatBrief}, []store.Handoff{sampleHandoff()})
	})
	if briefHandoffs != "handoff_3 [main] Investigate refresh retry failure\n" {
		t.Fatalf("unexpected brief handoffs output:\n%s", briefHandoffs)
	}
}

func TestRenderTasksAndTaskCompleted(t *testing.T) {
	empty := renderString(t, func(w io.Writer) error {
		return RenderTasks(w, Options{}, nil)
	})
	if empty != "No tasks.\n" {
		t.Fatalf("unexpected empty task output:\n%s", empty)
	}

	tasks := []store.Task{sampleTask(), sampleOpenTask()}
	humanTasks := renderString(t, func(w io.Writer) error {
		return RenderTasks(w, Options{}, tasks)
	})
	if humanTasks != "Tasks:\n- task_9 [done] [feat/auth] Fix refresh retry path\n- task_10 [open] Review retry logging\n" {
		t.Fatalf("unexpected human task list:\n%s", humanTasks)
	}

	jsonTasks := renderJSONEnvelope(t, func(w io.Writer) error {
		return RenderTasks(w, Options{Format: FormatJSON}, tasks)
	})
	if jsonTasks.Command != "task list" {
		t.Fatalf("unexpected command: %#v", jsonTasks)
	}

	var taskList struct {
		Tasks []struct {
			ID     string  `json:"id"`
			Status string  `json:"status"`
			Branch *string `json:"branch"`
		} `json:"tasks"`
	}
	unmarshalJSON(t, jsonTasks.Data, &taskList)
	if len(taskList.Tasks) != 2 || taskList.Tasks[0].ID != "task_9" || taskList.Tasks[0].Status != "done" {
		t.Fatalf("unexpected task list payload: %#v", taskList)
	}

	completedHuman := renderString(t, func(w io.Writer) error {
		return RenderTaskCompleted(w, Options{}, sampleTask())
	})
	if completedHuman != "Completed task task_9 [feat/auth]: Fix refresh retry path\n" {
		t.Fatalf("unexpected completed task output:\n%s", completedHuman)
	}

	completedBrief := renderString(t, func(w io.Writer) error {
		return RenderTaskCompleted(w, Options{Format: FormatBrief}, sampleTask())
	})
	if completedBrief != "task_9 [done] [feat/auth] Fix refresh retry path\n" {
		t.Fatalf("unexpected completed task brief output:\n%s", completedBrief)
	}

	completedJSON := renderJSONEnvelope(t, func(w io.Writer) error {
		return RenderTaskCompleted(w, Options{Format: FormatJSON}, sampleTask())
	})
	if completedJSON.Command != "task done" {
		t.Fatalf("unexpected command: %#v", completedJSON)
	}

	var completed struct {
		Task struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"task"`
	}
	unmarshalJSON(t, completedJSON.Data, &completed)
	if completed.Task.ID != "task_9" || completed.Task.Status != "done" {
		t.Fatalf("unexpected completed task payload: %#v", completed)
	}
}

func TestRenderDecisionAdded(t *testing.T) {
	decision := sampleDecision()

	human := renderString(t, func(w io.Writer) error {
		return RenderDecisionAdded(w, Options{}, decision)
	})
	if human != "Added decision decision_4 [main]: Use single refresh retry\n" {
		t.Fatalf("unexpected human output:\n%s", human)
	}

	verbose := renderString(t, func(w io.Writer) error {
		return RenderDecisionAdded(w, Options{Format: FormatVerbose}, decision)
	})
	if verbose != "Decision decision_4\nBranch: main\nCreated: 2026-04-15T10:11:12Z\nText: Use single refresh retry\nRationale: Avoid duplicate token refreshes\n" {
		t.Fatalf("unexpected verbose output:\n%s", verbose)
	}

	jsonPayload := renderJSONEnvelope(t, func(w io.Writer) error {
		return RenderDecisionAdded(w, Options{Format: FormatJSON}, decision)
	})
	if jsonPayload.Command != "decide add" {
		t.Fatalf("unexpected command: %#v", jsonPayload)
	}

	var data struct {
		Decision struct {
			ID        string  `json:"id"`
			Text      string  `json:"text"`
			Branch    *string `json:"branch"`
			Rationale *string `json:"rationale"`
		} `json:"decision"`
	}
	unmarshalJSON(t, jsonPayload.Data, &data)
	if data.Decision.ID != "decision_4" || data.Decision.Branch == nil || *data.Decision.Branch != "main" || data.Decision.Rationale == nil || *data.Decision.Rationale != "Avoid duplicate token refreshes" {
		t.Fatalf("unexpected decision payload: %#v", data)
	}
}

func TestRenderHistorySearch(t *testing.T) {
	empty := renderString(t, func(w io.Writer) error {
		return RenderHistorySearch(w, Options{}, HistorySearchResult{Query: "refresh"})
	})
	if empty != "No matching commits.\n" {
		t.Fatalf("unexpected empty history output:\n%s", empty)
	}

	result := HistorySearchResult{Query: "refresh", Commits: []store.Commit{sampleCommit()}}
	human := renderString(t, func(w io.Writer) error {
		return RenderHistorySearch(w, Options{}, result)
	})
	if human != "Commits:\n- abc1234 fix: refresh retry path\n  paths: auth/refresh.go, tests/auth_test.go\n" {
		t.Fatalf("unexpected human history output:\n%s", human)
	}

	brief := renderString(t, func(w io.Writer) error {
		return RenderHistorySearch(w, Options{Format: FormatBrief}, result)
	})
	if brief != "abc1234 fix: refresh retry path\n" {
		t.Fatalf("unexpected brief history output:\n%s", brief)
	}

	verbose := renderString(t, func(w io.Writer) error {
		return RenderHistorySearch(w, Options{Format: FormatVerbose}, result)
	})
	for _, want := range []string{"Commit abc1234", "Summary: fix: refresh retry path", "Paths: auth/refresh.go, tests/auth_test.go", "Message:\n  fix: refresh retry path\n  Handle 401 once."} {
		if !strings.Contains(verbose, want) {
			t.Fatalf("expected verbose history output to contain %q\n\n%s", want, verbose)
		}
	}

	jsonPayload := renderJSONEnvelope(t, func(w io.Writer) error {
		return RenderHistorySearch(w, Options{Format: FormatJSON}, result)
	})
	if jsonPayload.Command != "history search" {
		t.Fatalf("unexpected command: %#v", jsonPayload)
	}

	var data struct {
		Query   string `json:"query"`
		Commits []struct {
			SHA     string   `json:"sha"`
			Summary string   `json:"summary"`
			Paths   []string `json:"changed_paths"`
		} `json:"commits"`
	}
	unmarshalJSON(t, jsonPayload.Data, &data)
	if data.Query != "refresh" || len(data.Commits) != 1 || data.Commits[0].SHA != sampleCommit().SHA || len(data.Commits[0].Paths) != 2 {
		t.Fatalf("unexpected history payload: %#v", data)
	}
}

func TestRenderRecall(t *testing.T) {
	empty := renderString(t, func(w io.Writer) error {
		return RenderRecall(w, Options{}, RecallResult{Result: searchpkg.Result{Query: "refresh"}})
	})
	if empty != "No matching context.\n" {
		t.Fatalf("unexpected empty recall output:\n%s", empty)
	}

	result := RecallResult{Result: searchpkg.Result{
		Query:     "refresh",
		Notes:     []store.Note{sampleNote()},
		Decisions: []store.Decision{sampleDecision()},
		Handoffs:  []store.Handoff{sampleHandoff()},
		Commits:   []store.Commit{sampleCommit()},
	}}

	human := renderString(t, func(w io.Writer) error {
		return RenderRecall(w, Options{}, result)
	})
	if human != "Notes:\n- Refresh retry investigation [main]\nDecisions:\n- Use single refresh retry [main]\nHandoffs:\n- handoff_3 [main] Investigate refresh retry failure\nCommits:\n- abc1234 fix: refresh retry path\n" {
		t.Fatalf("unexpected human recall output:\n%s", human)
	}

	verbose := renderString(t, func(w io.Writer) error {
		return RenderRecall(w, Options{Format: FormatVerbose}, result)
	})
	for _, want := range []string{"Notes:\nNote note_7", "Decisions:\nDecision decision_4", "Handoffs:\nHandoff handoff_3", "Commits:\nCommit abc1234"} {
		if !strings.Contains(verbose, want) {
			t.Fatalf("expected verbose recall output to contain %q\n\n%s", want, verbose)
		}
	}

	jsonPayload := renderJSONEnvelope(t, func(w io.Writer) error {
		return RenderRecall(w, Options{Format: FormatJSON}, result)
	})
	if jsonPayload.Command != "recall" {
		t.Fatalf("unexpected command: %#v", jsonPayload)
	}

	var data struct {
		Query     string `json:"query"`
		Notes     []any  `json:"notes"`
		Decisions []any  `json:"decisions"`
		Handoffs  []any  `json:"handoffs"`
		Commits   []any  `json:"commits"`
	}
	unmarshalJSON(t, jsonPayload.Data, &data)
	if data.Query != "refresh" || len(data.Notes) != 1 || len(data.Decisions) != 1 || len(data.Handoffs) != 1 || len(data.Commits) != 1 {
		t.Fatalf("unexpected recall payload: %#v", data)
	}
}

type jsonEnvelope struct {
	SchemaVersion string          `json:"schema_version"`
	OK            bool            `json:"ok"`
	Command       string          `json:"command"`
	Data          json.RawMessage `json:"data"`
	Error         *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type failingWriter struct{}

func (failingWriter) Write(_ []byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func renderString(t *testing.T, render func(io.Writer) error) string {
	t.Helper()

	var buf bytes.Buffer
	if err := render(&buf); err != nil {
		t.Fatalf("render: %v", err)
	}

	return buf.String()
}

func renderJSONEnvelope(t *testing.T, render func(io.Writer) error) jsonEnvelope {
	t.Helper()

	raw := renderString(t, render)

	var payload jsonEnvelope
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal json payload: %v\n%s", err, raw)
	}

	return payload
}

func unmarshalJSON(t *testing.T, raw []byte, target any) {
	t.Helper()

	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatalf("unmarshal nested payload: %v\n%s", err, raw)
	}
}

func sampleNote() store.Note {
	return store.Note{
		ID:        7,
		Branch:    "main",
		Scope:     store.ScopeBranch,
		Text:      "Refresh retry investigation",
		CreatedAt: sampleTime(),
	}
}

func sampleTask() store.Task {
	return store.Task{
		ID:        9,
		Branch:    "feat/auth",
		Scope:     store.ScopeBranch,
		Text:      "Fix refresh retry path",
		Status:    store.TaskDone,
		CreatedAt: sampleTime(),
		UpdatedAt: sampleTime().Add(2 * time.Minute),
	}
}

func sampleOpenTask() store.Task {
	return store.Task{
		ID:        10,
		Scope:     store.ScopeRepo,
		Text:      "Review retry logging",
		Status:    store.TaskOpen,
		CreatedAt: sampleTime(),
		UpdatedAt: sampleTime().Add(3 * time.Minute),
	}
}

func sampleDecision() store.Decision {
	rationale := "Avoid duplicate token refreshes"
	return store.Decision{
		ID:        4,
		Branch:    "main",
		Text:      "Use single refresh retry",
		Rationale: &rationale,
		CreatedAt: sampleTime(),
	}
}

func sampleRepoDecision() store.Decision {
	return store.Decision{
		ID:        6,
		Text:      "Keep shared retry logging at repo scope",
		CreatedAt: sampleTime().Add(5 * time.Minute),
	}
}

func sampleCommit() store.Commit {
	return store.Commit{
		SHA:          "abc123456789",
		Summary:      "fix: refresh retry path",
		Message:      "fix: refresh retry path\n\nHandle 401 once.",
		Author:       "Dan",
		CommittedAt:  sampleTime(),
		ChangedPaths: []string{"auth/refresh.go", "tests/auth_test.go"},
	}
}

func sampleHandoff() store.Handoff {
	return store.Handoff{
		ID:        3,
		Branch:    "main",
		Summary:   "Investigate refresh retry failure\nNext: inspect auth middleware",
		CreatedAt: sampleTime(),
	}
}

func sampleRepoHandoff() store.Handoff {
	return store.Handoff{
		ID:        5,
		Summary:   "Repo-wide handoff summary",
		CreatedAt: sampleTime().Add(4 * time.Minute),
	}
}

func sampleRepoNote() store.Note {
	return store.Note{
		ID:        8,
		Scope:     store.ScopeRepo,
		Text:      "Shared retry logging touches multiple branches",
		CreatedAt: sampleTime().Add(1 * time.Minute),
	}
}

func sampleResumeTaskPointer() *store.Task {
	task := sampleTask()
	return &task
}

func sampleHandoffPointer() *store.Handoff {
	handoff := sampleHandoff()
	return &handoff
}

func stringPointer(value string) *string {
	return &value
}

func sampleTime() time.Time {
	return time.Date(2026, 4, 15, 10, 11, 12, 0, time.UTC)
}

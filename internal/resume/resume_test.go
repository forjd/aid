package resume

import (
	"fmt"
	"testing"
	"time"

	"github.com/forjd/aid/internal/store"
)

func TestInferActiveTask(t *testing.T) {
	tests := []struct {
		name          string
		branch        string
		tasks         []store.Task
		wantID        int64
		wantInferred  bool
		wantAmbiguous bool
	}{
		{
			name:   "single branch in progress wins",
			branch: "feat/auth",
			tasks: []store.Task{
				resumeTask(1, "feat/auth", store.TaskInProgress, "Fix refresh retry path"),
				resumeTask(2, "feat/auth", store.TaskOpen, "Review retry logging"),
			},
			wantID:       1,
			wantInferred: true,
		},
		{
			name:   "multiple branch in progress tasks are ambiguous",
			branch: "feat/auth",
			tasks: []store.Task{
				resumeTask(1, "feat/auth", store.TaskInProgress, "Fix refresh retry path"),
				resumeTask(2, "feat/auth", store.TaskInProgress, "Review retry logging"),
			},
			wantAmbiguous: true,
		},
		{
			name:   "single branch open task is inferred when nothing is in progress",
			branch: "feat/auth",
			tasks: []store.Task{
				resumeTask(3, "feat/auth", store.TaskOpen, "Fix refresh retry path"),
			},
			wantID:       3,
			wantInferred: true,
		},
		{
			name:   "single repo in progress task is used as fallback",
			branch: "feat/auth",
			tasks: []store.Task{
				resumeTask(4, "other", store.TaskInProgress, "Fix refresh retry path"),
			},
			wantID:       4,
			wantInferred: true,
		},
		{
			name:   "no suitable task returns nil",
			branch: "feat/auth",
			tasks: []store.Task{
				resumeTask(5, "other", store.TaskOpen, "Fix refresh retry path"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, inferred, ambiguous := inferActiveTask(test.branch, test.tasks)
			if inferred != test.wantInferred || ambiguous != test.wantAmbiguous {
				t.Fatalf("unexpected flags inferred=%t ambiguous=%t", inferred, ambiguous)
			}
			if test.wantID == 0 {
				if got != nil {
					t.Fatalf("expected nil task, got %#v", got)
				}
				return
			}
			if got == nil || got.ID != test.wantID {
				t.Fatalf("expected task %d, got %#v", test.wantID, got)
			}
		})
	}
}

func TestRankNotesAndDecisionsPreferBranchThenRecency(t *testing.T) {
	notes := []store.Note{
		resumeNote(1, "", 5, "repo note"),
		resumeNote(2, "feat/auth", 1, "branch note older"),
		resumeNote(3, "other", 10, "other branch note"),
		resumeNote(4, "feat/auth", 2, "branch note newer"),
	}
	rankedNotes := rankNotes("feat/auth", notes, 2)
	if len(rankedNotes) != 2 || rankedNotes[0].ID != 4 || rankedNotes[1].ID != 2 {
		t.Fatalf("unexpected ranked notes: %#v", rankedNotes)
	}

	decisions := []store.Decision{
		resumeDecision(1, "", 6, "repo decision"),
		resumeDecision(2, "feat/auth", 1, "branch decision older"),
		resumeDecision(3, "feat/auth", 3, "branch decision newer"),
		resumeDecision(4, "other", 8, "other decision"),
	}
	rankedDecisions := rankDecisions("feat/auth", decisions, 2)
	if len(rankedDecisions) != 2 || rankedDecisions[0].ID != 3 || rankedDecisions[1].ID != 2 {
		t.Fatalf("unexpected ranked decisions: %#v", rankedDecisions)
	}
}

func TestBranchRank(t *testing.T) {
	if got := branchRank("feat/auth", "feat/auth"); got != 0 {
		t.Fatalf("expected branch match rank 0, got %d", got)
	}
	if got := branchRank("feat/auth", ""); got != 1 {
		t.Fatalf("expected repo rank 1, got %d", got)
	}
	if got := branchRank("feat/auth", "other"); got != 2 {
		t.Fatalf("expected other branch rank 2, got %d", got)
	}
}

func TestInferNextAction(t *testing.T) {
	tests := []struct {
		name      string
		branch    string
		active    *store.Task
		ambiguous bool
		tasks     []store.Task
		want      *string
	}{
		{
			name:   "active task continues",
			branch: "feat/auth",
			active: taskPointer(resumeTask(1, "feat/auth", store.TaskInProgress, "Fix refresh retry path")),
			want:   stringPointer("continue Fix refresh retry path"),
		},
		{
			name:   "branch blocked task resolves blocker",
			branch: "feat/auth",
			tasks:  []store.Task{resumeTask(2, "feat/auth", store.TaskBlocked, "Fix refresh retry path")},
			want:   stringPointer("resolve blocker for Fix refresh retry path"),
		},
		{
			name:      "ambiguous tasks ask for a single active task",
			branch:    "feat/auth",
			ambiguous: true,
			want:      stringPointer("choose a single active task"),
		},
		{
			name:   "branch open task is started",
			branch: "feat/auth",
			tasks:  []store.Task{resumeTask(3, "feat/auth", store.TaskOpen, "Fix refresh retry path")},
			want:   stringPointer("start Fix refresh retry path"),
		},
		{
			name:   "repo blocked task is fallback",
			branch: "feat/auth",
			tasks:  []store.Task{resumeTask(4, "", store.TaskBlocked, "Fix refresh retry path")},
			want:   stringPointer("resolve blocker for Fix refresh retry path"),
		},
		{
			name:   "repo open task is last fallback",
			branch: "feat/auth",
			tasks:  []store.Task{resumeTask(5, "", store.TaskOpen, "Fix refresh retry path")},
			want:   stringPointer("start Fix refresh retry path"),
		},
		{
			name:   "no next action returns nil",
			branch: "feat/auth",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := inferNextAction(test.branch, test.active, test.ambiguous, test.tasks)
			assertOptionalString(t, got, test.want)
		})
	}
}

func TestInferOpenQuestions(t *testing.T) {
	questions := inferOpenQuestions("feat/auth", nil, true, []store.Task{
		resumeTask(1, "feat/auth", store.TaskBlocked, "Fix refresh retry path"),
		resumeTask(2, "other", store.TaskBlocked, "Skip other branch blocker"),
		resumeTask(3, "", store.TaskBlocked, "Resolve shared retry logging"),
		resumeTask(4, "feat/auth", store.TaskBlocked, "Check retry middleware"),
	})

	want := []string{
		"Which task should be the single active task on this branch?",
		"What is blocking Fix refresh retry path?",
		"What is blocking Resolve shared retry logging?",
	}
	if len(questions) != len(want) {
		t.Fatalf("unexpected question count: %#v", questions)
	}
	for i := range want {
		if questions[i] != want[i] {
			t.Fatalf("unexpected questions: %#v", questions)
		}
	}
}

func TestRankHandoffsAndCarryForwardState(t *testing.T) {
	handoffs := []store.Handoff{
		resumeHandoff(1, "", 10, "Open questions:\n- What should happen after the retry fix?\n- Should the current uncommitted changes be kept, finished, or discarded?\nRecommended next action:\n- inspect retry middleware"),
		resumeHandoff(2, "feat/auth", 1, "Open questions:\n- Which retry path is still failing?\nRecommended next action:\n- continue retry investigation"),
		resumeHandoff(3, "other", 20, "Open questions:\n- Ignore other branch question\nRecommended next action:\n- ignore other branch action"),
	}

	ranked := rankHandoffs("feat/auth", handoffs, 2)
	if len(ranked) != 2 || ranked[0].ID != 2 || ranked[1].ID != 1 {
		t.Fatalf("unexpected ranked handoffs: %#v", ranked)
	}

	questions := appendCarryForwardQuestions([]string{"What is blocking Fix refresh retry path?"}, ranked)
	wantQuestions := []string{
		"What is blocking Fix refresh retry path?",
		"Which retry path is still failing?",
		"What should happen after the retry fix?",
	}
	if len(questions) != len(wantQuestions) {
		t.Fatalf("unexpected carry-forward questions: %#v", questions)
	}
	for i := range wantQuestions {
		if questions[i] != wantQuestions[i] {
			t.Fatalf("unexpected carry-forward questions: %#v", questions)
		}
	}

	assertOptionalString(t, carryForwardNextAction(nil, ranked), stringPointer("continue retry investigation"))
	assertOptionalString(t, carryForwardNextAction(stringPointer("start existing work"), ranked), stringPointer("start existing work"))
}

func TestParseHandoffHelpers(t *testing.T) {
	summary := "Branch: feat/auth\nOpen questions:\n- Which retry path is still failing?\n- Should the current uncommitted changes be kept, finished, or discarded?\nRecommended next action:\n- inspect retry middleware\nRecent commits:\n- abc1234 fix: retry middleware\n"

	questions := parseHandoffQuestions(summary)
	if len(questions) != 1 || questions[0] != "Which retry path is still failing?" {
		t.Fatalf("unexpected parsed questions: %#v", questions)
	}

	assertOptionalString(t, parseHandoffNextAction(summary), stringPointer("inspect retry middleware"))
	assertOptionalString(t, parseHandoffNextAction("Recommended next action:\nnot a list item\n- ignored"), nil)

	if !isEphemeralQuestion("Should the current uncommitted changes be kept, finished, or discarded?") {
		t.Fatal("expected ephemeral question to be detected")
	}
	if normalizedText("  Retry Path  ") != "retry path" {
		t.Fatalf("unexpected normalized text: %q", normalizedText("  Retry Path  "))
	}
}

func TestBuildRanksAndCarriesForwardContext(t *testing.T) {
	branch := "feat/auth"
	nextFromHandoff := "inspect retry middleware"

	bundle := Build(
		branch,
		[]store.Note{
			resumeNote(1, "", 6, "repo note"),
			resumeNote(2, branch, 1, "branch note older"),
			resumeNote(3, branch, 2, "branch note newer"),
			resumeNote(4, "other", 10, "other note"),
		},
		[]store.Task{},
		[]store.Decision{
			resumeDecision(1, "", 6, "repo decision"),
			resumeDecision(2, branch, 1, "branch decision older"),
			resumeDecision(3, branch, 2, "branch decision newer"),
		},
		[]store.Commit{
			resumeCommit(1),
			resumeCommit(2),
			resumeCommit(3),
			resumeCommit(4),
			resumeCommit(5),
			resumeCommit(6),
		},
		[]store.Handoff{
			resumeHandoff(1, "", 8, "Open questions:\n- What should happen after the retry fix?\nRecommended next action:\n- inspect shared retry logging"),
			resumeHandoff(2, branch, 1, "Open questions:\n- Which retry path is still failing?\nRecommended next action:\n- "+nextFromHandoff),
		},
	)

	if bundle.ActiveTask != nil || bundle.ActiveTaskInferred || bundle.ActiveTaskAmbiguous {
		t.Fatalf("unexpected active task state: %#v", bundle)
	}
	if len(bundle.Notes) != 3 || bundle.Notes[0].ID != 3 || bundle.Notes[1].ID != 2 || bundle.Notes[2].ID != 1 {
		t.Fatalf("unexpected ranked notes in bundle: %#v", bundle.Notes)
	}
	if len(bundle.Decisions) != 3 || bundle.Decisions[0].ID != 3 || bundle.Decisions[1].ID != 2 || bundle.Decisions[2].ID != 1 {
		t.Fatalf("unexpected ranked decisions in bundle: %#v", bundle.Decisions)
	}
	if len(bundle.RecentCommits) != 5 || bundle.RecentCommits[0].SHA != "sha-01" || bundle.RecentCommits[4].SHA != "sha-05" {
		t.Fatalf("unexpected commits in bundle: %#v", bundle.RecentCommits)
	}
	if bundle.LatestHandoff == nil || bundle.LatestHandoff.ID != 2 {
		t.Fatalf("unexpected latest handoff: %#v", bundle.LatestHandoff)
	}
	wantQuestions := []string{"Which retry path is still failing?", "What should happen after the retry fix?"}
	if len(bundle.OpenQuestions) != len(wantQuestions) {
		t.Fatalf("unexpected open questions: %#v", bundle.OpenQuestions)
	}
	for i := range wantQuestions {
		if bundle.OpenQuestions[i] != wantQuestions[i] {
			t.Fatalf("unexpected open questions: %#v", bundle.OpenQuestions)
		}
	}
	assertOptionalString(t, bundle.NextAction, &nextFromHandoff)
}

func assertOptionalString(t *testing.T, got, want *string) {
	t.Helper()

	if want == nil {
		if got != nil {
			t.Fatalf("expected nil string, got %q", *got)
		}
		return
	}
	if got == nil || *got != *want {
		if got == nil {
			t.Fatalf("expected %q, got nil", *want)
		}
		t.Fatalf("expected %q, got %q", *want, *got)
	}
}

func taskPointer(task store.Task) *store.Task {
	return &task
}

func stringPointer(value string) *string {
	return &value
}

func resumeTask(id int64, branch string, status store.TaskStatus, text string) store.Task {
	return store.Task{
		ID:        id,
		Branch:    branch,
		Scope:     store.ScopeBranch,
		Text:      text,
		Status:    status,
		CreatedAt: resumeTime(int(id)),
		UpdatedAt: resumeTime(int(id) + 1),
	}
}

func resumeNote(id int64, branch string, minutes int, text string) store.Note {
	return store.Note{
		ID:        id,
		Branch:    branch,
		Scope:     store.ScopeBranch,
		Text:      text,
		CreatedAt: resumeTime(minutes),
	}
}

func resumeDecision(id int64, branch string, minutes int, text string) store.Decision {
	return store.Decision{
		ID:        id,
		Branch:    branch,
		Text:      text,
		CreatedAt: resumeTime(minutes),
	}
}

func resumeHandoff(id int64, branch string, minutes int, summary string) store.Handoff {
	return store.Handoff{
		ID:        id,
		Branch:    branch,
		Summary:   summary,
		CreatedAt: resumeTime(minutes),
	}
}

func resumeCommit(id int64) store.Commit {
	return store.Commit{
		ID:          id,
		SHA:         fmt.Sprintf("sha-%02d", id),
		Summary:     fmt.Sprintf("commit %d", id),
		Message:     fmt.Sprintf("commit %d", id),
		Author:      "Dan",
		CommittedAt: resumeTime(int(id)),
	}
}

func resumeTime(minutes int) time.Time {
	base := time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
	return base.Add(time.Duration(minutes) * time.Minute)
}

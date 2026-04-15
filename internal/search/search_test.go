package search

import (
	"fmt"
	"testing"
	"time"

	"github.com/forjd/aid/internal/store"
)

func TestBuildAppliesPackageLimits(t *testing.T) {
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)

	notes := make([]store.Note, 0, 6)
	decisions := make([]store.Decision, 0, 6)
	handoffs := make([]store.Handoff, 0, 4)
	commits := make([]store.Commit, 0, 6)
	for i := range 6 {
		notes = append(notes, store.Note{ID: int64(i + 1), Text: fmt.Sprintf("note-%d", i+1)})
		decisions = append(decisions, store.Decision{ID: int64(i + 1), Text: fmt.Sprintf("decision-%d", i+1)})
		commits = append(commits, store.Commit{
			ID:          int64(i + 1),
			SHA:         fmt.Sprintf("sha-%d", i+1),
			CommittedAt: now.Add(time.Duration(i) * time.Minute),
		})
		if i < 4 {
			handoffs = append(handoffs, store.Handoff{ID: int64(i + 1), Summary: fmt.Sprintf("handoff-%d", i+1)})
		}
	}

	result := Build("refresh retry", "main", notes, decisions, handoffs, commits)

	if result.Query != "refresh retry" {
		t.Fatalf("unexpected query: %q", result.Query)
	}
	if len(result.Notes) != 5 || result.Notes[4].Text != "note-5" {
		t.Fatalf("unexpected notes: %#v", result.Notes)
	}
	if len(result.Decisions) != 5 || result.Decisions[4].Text != "decision-5" {
		t.Fatalf("unexpected decisions: %#v", result.Decisions)
	}
	if len(result.Handoffs) != 3 || result.Handoffs[2].Summary != "handoff-3" {
		t.Fatalf("unexpected handoffs: %#v", result.Handoffs)
	}
	if len(result.Commits) != 5 || result.Commits[4].SHA != "sha-5" {
		t.Fatalf("unexpected commits: %#v", result.Commits)
	}
}

func TestLimitHelpersPassThroughShortSlices(t *testing.T) {
	note := []store.Note{{Text: "note"}}
	decision := []store.Decision{{Text: "decision"}}
	handoff := []store.Handoff{{Summary: "handoff"}}
	commit := []store.Commit{{SHA: "abc123"}}

	if got := limitNotes(note, 5); len(got) != 1 || got[0].Text != "note" {
		t.Fatalf("unexpected limited notes: %#v", got)
	}
	if got := limitDecisions(decision, 5); len(got) != 1 || got[0].Text != "decision" {
		t.Fatalf("unexpected limited decisions: %#v", got)
	}
	if got := limitHandoffs(handoff, 3); len(got) != 1 || got[0].Summary != "handoff" {
		t.Fatalf("unexpected limited handoffs: %#v", got)
	}
	if got := limitCommits(commit, 5); len(got) != 1 || got[0].SHA != "abc123" {
		t.Fatalf("unexpected limited commits: %#v", got)
	}
}

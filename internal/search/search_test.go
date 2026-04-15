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

	result := Build("refresh retry", notes, decisions, handoffs, commits)

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

func TestBuildPreservesShortSlicesAndOrder(t *testing.T) {
	notes := []store.Note{{Text: "note-1"}, {Text: "note-2"}}
	decisions := []store.Decision{{Text: "decision-1"}}
	handoffs := []store.Handoff{{Summary: "handoff-1"}}
	commits := []store.Commit{{SHA: "abc123"}, {SHA: "def456"}}

	result := Build("refresh", notes, decisions, handoffs, commits)

	if len(result.Notes) != 2 || result.Notes[1].Text != "note-2" {
		t.Fatalf("unexpected notes: %#v", result.Notes)
	}
	if len(result.Decisions) != 1 || result.Decisions[0].Text != "decision-1" {
		t.Fatalf("unexpected decisions: %#v", result.Decisions)
	}
	if len(result.Handoffs) != 1 || result.Handoffs[0].Summary != "handoff-1" {
		t.Fatalf("unexpected handoffs: %#v", result.Handoffs)
	}
	if len(result.Commits) != 2 || result.Commits[1].SHA != "def456" {
		t.Fatalf("unexpected commits: %#v", result.Commits)
	}
}

func TestBuildHandlesEmptyInputs(t *testing.T) {
	result := Build("refresh", nil, nil, nil, nil)
	if result.Query != "refresh" {
		t.Fatalf("unexpected query: %q", result.Query)
	}
	if len(result.Notes) != 0 || len(result.Decisions) != 0 || len(result.Handoffs) != 0 || len(result.Commits) != 0 {
		t.Fatalf("expected empty sections, got %#v", result)
	}
}

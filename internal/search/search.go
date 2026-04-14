package search

import "github.com/forjd/aid/internal/store"

type Result struct {
	Query     string
	Notes     []store.Note
	Decisions []store.Decision
	Handoffs  []store.Handoff
	Commits   []store.Commit
}

func Build(query string, _ string, notes []store.Note, decisions []store.Decision, handoffs []store.Handoff, commits []store.Commit) Result {
	return Result{
		Query:     query,
		Notes:     limitNotes(notes, 5),
		Decisions: limitDecisions(decisions, 5),
		Handoffs:  limitHandoffs(handoffs, 3),
		Commits:   limitCommits(commits, 5),
	}
}

func limitNotes(notes []store.Note, limit int) []store.Note {
	if len(notes) <= limit {
		return notes
	}
	return notes[:limit]
}

func limitDecisions(decisions []store.Decision, limit int) []store.Decision {
	if len(decisions) <= limit {
		return decisions
	}
	return decisions[:limit]
}

func limitHandoffs(handoffs []store.Handoff, limit int) []store.Handoff {
	if len(handoffs) <= limit {
		return handoffs
	}
	return handoffs[:limit]
}

func limitCommits(commits []store.Commit, limit int) []store.Commit {
	if len(commits) <= limit {
		return commits
	}
	return commits[:limit]
}

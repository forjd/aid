package search

import "github.com/forjd/aid/internal/store"

type Result struct {
	Query     string
	Notes     []store.Note
	Decisions []store.Decision
	Handoffs  []store.Handoff
	Commits   []store.Commit
}

func Build(query string, notes []store.Note, decisions []store.Decision, handoffs []store.Handoff, commits []store.Commit) Result {
	return Result{
		Query:     query,
		Notes:     store.Limit(notes, 5),
		Decisions: store.Limit(decisions, 5),
		Handoffs:  store.Limit(handoffs, 3),
		Commits:   store.Limit(commits, 5),
	}
}

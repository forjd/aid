package search

import (
	"sort"
	"strings"

	"aid/internal/store"
)

type Result struct {
	Query     string
	Notes     []store.Note
	Decisions []store.Decision
	Handoffs  []store.Handoff
	Commits   []store.Commit
}

func Build(query string, branch string, notes []store.Note, decisions []store.Decision, handoffs []store.Handoff, commits []store.Commit) Result {
	normalized := strings.ToLower(strings.TrimSpace(query))

	return Result{
		Query:     query,
		Notes:     rankNotes(branch, filterNotes(normalized, notes), 5),
		Decisions: rankDecisions(branch, filterDecisions(normalized, decisions), 5),
		Handoffs:  rankHandoffs(branch, filterHandoffs(normalized, handoffs), 3),
		Commits:   limitCommits(commits, 5),
	}
}

func filterNotes(query string, notes []store.Note) []store.Note {
	var items []store.Note
	for _, note := range notes {
		if strings.Contains(strings.ToLower(note.Text), query) {
			items = append(items, note)
		}
	}
	return items
}

func filterDecisions(query string, decisions []store.Decision) []store.Decision {
	var items []store.Decision
	for _, decision := range decisions {
		candidate := decision.Text
		if decision.Rationale != nil {
			candidate += "\n" + *decision.Rationale
		}
		if strings.Contains(strings.ToLower(candidate), query) {
			items = append(items, decision)
		}
	}
	return items
}

func filterHandoffs(query string, handoffs []store.Handoff) []store.Handoff {
	var items []store.Handoff
	for _, handoff := range handoffs {
		if strings.Contains(strings.ToLower(handoff.Summary), query) {
			items = append(items, handoff)
		}
	}
	return items
}

func rankNotes(branch string, notes []store.Note, limit int) []store.Note {
	cloned := append([]store.Note(nil), notes...)
	sort.SliceStable(cloned, func(i, j int) bool {
		leftRank := branchRank(branch, cloned[i].Branch)
		rightRank := branchRank(branch, cloned[j].Branch)
		if leftRank != rightRank {
			return leftRank < rightRank
		}

		if !cloned[i].CreatedAt.Equal(cloned[j].CreatedAt) {
			return cloned[i].CreatedAt.After(cloned[j].CreatedAt)
		}

		return cloned[i].ID > cloned[j].ID
	})

	if len(cloned) <= limit {
		return cloned
	}
	return cloned[:limit]
}

func rankDecisions(branch string, decisions []store.Decision, limit int) []store.Decision {
	cloned := append([]store.Decision(nil), decisions...)
	sort.SliceStable(cloned, func(i, j int) bool {
		leftRank := branchRank(branch, cloned[i].Branch)
		rightRank := branchRank(branch, cloned[j].Branch)
		if leftRank != rightRank {
			return leftRank < rightRank
		}

		if !cloned[i].CreatedAt.Equal(cloned[j].CreatedAt) {
			return cloned[i].CreatedAt.After(cloned[j].CreatedAt)
		}

		return cloned[i].ID > cloned[j].ID
	})

	if len(cloned) <= limit {
		return cloned
	}
	return cloned[:limit]
}

func rankHandoffs(branch string, handoffs []store.Handoff, limit int) []store.Handoff {
	cloned := append([]store.Handoff(nil), handoffs...)
	sort.SliceStable(cloned, func(i, j int) bool {
		leftRank := branchRank(branch, cloned[i].Branch)
		rightRank := branchRank(branch, cloned[j].Branch)
		if leftRank != rightRank {
			return leftRank < rightRank
		}

		if !cloned[i].CreatedAt.Equal(cloned[j].CreatedAt) {
			return cloned[i].CreatedAt.After(cloned[j].CreatedAt)
		}

		return cloned[i].ID > cloned[j].ID
	})

	if len(cloned) <= limit {
		return cloned
	}
	return cloned[:limit]
}

func limitCommits(commits []store.Commit, limit int) []store.Commit {
	if len(commits) <= limit {
		return commits
	}
	return commits[:limit]
}

func branchRank(currentBranch string, itemBranch string) int {
	switch {
	case itemBranch == currentBranch:
		return 0
	case itemBranch == "":
		return 1
	default:
		return 2
	}
}

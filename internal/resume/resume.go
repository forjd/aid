package resume

import (
	"sort"

	"aid/internal/git"
	"aid/internal/store"
)

type Bundle struct {
	ActiveTask          *store.Task
	ActiveTaskInferred  bool
	ActiveTaskAmbiguous bool
	Notes               []store.Note
	Decisions           []store.Decision
	RecentCommits       []git.Commit
	NextAction          *string
}

func Build(branch string, notes []store.Note, tasks []store.Task, decisions []store.Decision, commits []git.Commit) Bundle {
	activeTask, inferred, ambiguous := inferActiveTask(branch, tasks)

	return Bundle{
		ActiveTask:          activeTask,
		ActiveTaskInferred:  inferred,
		ActiveTaskAmbiguous: ambiguous,
		Notes:               rankNotes(branch, notes, 3),
		Decisions:           rankDecisions(branch, decisions, 3),
		RecentCommits:       limitCommits(commits, 5),
		NextAction:          inferNextAction(activeTask, ambiguous),
	}
}

func inferActiveTask(branch string, tasks []store.Task) (*store.Task, bool, bool) {
	branchInProgress := filterTasks(tasks, func(task store.Task) bool {
		return task.Branch == branch && task.Status == store.TaskInProgress
	})
	switch len(branchInProgress) {
	case 1:
		task := branchInProgress[0]
		return &task, true, false
	case 0:
	default:
		return nil, false, true
	}

	branchOpen := filterTasks(tasks, func(task store.Task) bool {
		return task.Branch == branch && task.Status == store.TaskOpen
	})
	switch len(branchOpen) {
	case 1:
		task := branchOpen[0]
		return &task, true, false
	case 0:
	default:
		return nil, false, true
	}

	repoInProgress := filterTasks(tasks, func(task store.Task) bool {
		return task.Status == store.TaskInProgress
	})
	switch len(repoInProgress) {
	case 1:
		task := repoInProgress[0]
		return &task, true, false
	case 0:
	default:
		return nil, false, true
	}

	return nil, false, false
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

	return limitNotes(cloned, limit)
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

	return limitDecisions(cloned, limit)
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

func filterTasks(tasks []store.Task, predicate func(store.Task) bool) []store.Task {
	result := make([]store.Task, 0, len(tasks))
	for _, task := range tasks {
		if predicate(task) {
			result = append(result, task)
		}
	}
	return result
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

func limitCommits(commits []git.Commit, limit int) []git.Commit {
	if len(commits) <= limit {
		return commits
	}
	return commits[:limit]
}

func inferNextAction(activeTask *store.Task, ambiguous bool) *string {
	if activeTask != nil {
		next := "continue " + activeTask.Text
		return &next
	}

	if ambiguous {
		next := "choose a single active task"
		return &next
	}

	return nil
}

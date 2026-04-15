package resume

import (
	"slices"
	"sort"
	"strings"

	"github.com/forjd/aid/internal/store"
)

type Bundle struct {
	ActiveTask          *store.Task
	ActiveTaskInferred  bool
	ActiveTaskAmbiguous bool
	Notes               []store.Note
	Decisions           []store.Decision
	RecentCommits       []store.Commit
	LatestHandoff       *store.Handoff
	OpenQuestions       []string
	NextAction          *string
}

func Build(branch string, notes []store.Note, tasks []store.Task, decisions []store.Decision, commits []store.Commit, handoffs []store.Handoff) Bundle {
	activeTask, inferred, ambiguous := inferActiveTask(branch, tasks)
	rankedHandoffs := rankHandoffs(branch, handoffs, 3)

	var latestHandoff *store.Handoff
	if len(rankedHandoffs) > 0 {
		latest := rankedHandoffs[0]
		latestHandoff = &latest
	}

	openQuestions := inferOpenQuestions(branch, activeTask, ambiguous, tasks)
	openQuestions = appendCarryForwardQuestions(openQuestions, rankedHandoffs)

	nextAction := inferNextAction(branch, activeTask, ambiguous, tasks)
	nextAction = carryForwardNextAction(nextAction, rankedHandoffs)

	return Bundle{
		ActiveTask:          activeTask,
		ActiveTaskInferred:  inferred,
		ActiveTaskAmbiguous: ambiguous,
		Notes:               rankNotes(branch, notes, 3),
		Decisions:           rankDecisions(branch, decisions, 3),
		RecentCommits:       store.Limit(commits, 5),
		LatestHandoff:       latestHandoff,
		OpenQuestions:       openQuestions,
		NextAction:          nextAction,
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
	cloned := slices.Clone(notes)
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

	return store.Limit(cloned, limit)
}

func rankDecisions(branch string, decisions []store.Decision, limit int) []store.Decision {
	cloned := slices.Clone(decisions)
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

	return store.Limit(cloned, limit)
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

func inferNextAction(branch string, activeTask *store.Task, ambiguous bool, tasks []store.Task) *string {
	if activeTask != nil {
		next := "continue " + activeTask.Text
		return &next
	}

	if blocked := firstTask(branch, tasks, store.TaskBlocked, true); blocked != nil {
		next := "resolve blocker for " + blocked.Text
		return &next
	}

	if ambiguous {
		next := "choose a single active task"
		return &next
	}

	if open := firstTask(branch, tasks, store.TaskOpen, true); open != nil {
		next := "start " + open.Text
		return &next
	}

	if blocked := firstTask(branch, tasks, store.TaskBlocked, false); blocked != nil {
		next := "resolve blocker for " + blocked.Text
		return &next
	}

	if open := firstTask(branch, tasks, store.TaskOpen, false); open != nil {
		next := "start " + open.Text
		return &next
	}

	return nil
}

func inferOpenQuestions(branch string, activeTask *store.Task, ambiguous bool, tasks []store.Task) []string {
	questions := make([]string, 0, 3)
	if ambiguous {
		questions = append(questions, "Which task should be the single active task on this branch?")
	}

	if activeTask == nil {
		for _, task := range tasks {
			if task.Status != store.TaskBlocked {
				continue
			}
			if task.Branch != "" && task.Branch != branch {
				continue
			}
			questions = append(questions, "What is blocking "+task.Text+"?")
			if len(questions) == 3 {
				break
			}
		}
	}

	return questions
}

func firstTask(branch string, tasks []store.Task, status store.TaskStatus, branchOnly bool) *store.Task {
	for _, task := range tasks {
		if task.Status != status {
			continue
		}
		if branchOnly && task.Branch != branch {
			continue
		}

		candidate := task
		return &candidate
	}

	return nil
}

func rankHandoffs(branch string, handoffs []store.Handoff, limit int) []store.Handoff {
	cloned := slices.Clone(handoffs)
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

	return store.Limit(cloned, limit)
}

func appendCarryForwardQuestions(current []string, handoffs []store.Handoff) []string {
	if len(current) >= 3 {
		return current[:3]
	}

	seen := make(map[string]struct{}, len(current))
	for _, question := range current {
		seen[normalizedText(question)] = struct{}{}
	}

	for _, handoff := range handoffs {
		for _, question := range parseHandoffQuestions(handoff.Summary) {
			key := normalizedText(question)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}

			current = append(current, question)
			seen[key] = struct{}{}
			if len(current) == 3 {
				return current
			}
		}
	}

	return current
}

func carryForwardNextAction(current *string, handoffs []store.Handoff) *string {
	if current != nil {
		return current
	}

	for _, handoff := range handoffs {
		next := parseHandoffNextAction(handoff.Summary)
		if next == nil {
			continue
		}
		return next
	}

	return nil
}

func parseHandoffQuestions(summary string) []string {
	lines := strings.Split(summary, "\n")
	questions := make([]string, 0, 3)
	inSection := false

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		switch line {
		case "Open questions:":
			inSection = true
			continue
		case "Recommended next action:", "Recent commits:", "Key decisions:", "Recent notes:", "Open tasks:", "Latest handoff:", "Branch:", "Worktree:":
			inSection = false
		}

		if !inSection || !strings.HasPrefix(line, "- ") {
			continue
		}

		question := strings.TrimSpace(strings.TrimPrefix(line, "- "))
		if question == "" || isEphemeralQuestion(question) {
			continue
		}
		questions = append(questions, question)
	}

	return questions
}

func parseHandoffNextAction(summary string) *string {
	lines := strings.Split(summary, "\n")
	inSection := false

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "Recommended next action:" {
			inSection = true
			continue
		}
		if !inSection {
			continue
		}
		if !strings.HasPrefix(line, "- ") {
			if line != "" {
				break
			}
			continue
		}

		next := strings.TrimSpace(strings.TrimPrefix(line, "- "))
		if next == "" {
			return nil
		}
		return &next
	}

	return nil
}

func isEphemeralQuestion(question string) bool {
	return normalizedText(question) == normalizedText("Should the current uncommitted changes be kept, finished, or discarded?")
}

func normalizedText(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

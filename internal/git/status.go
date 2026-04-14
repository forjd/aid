package git

import (
	"strings"
)

type WorktreeStatus struct {
	Dirty     bool
	Changed   int
	Untracked int
}

func Status(startDir string) (WorktreeStatus, error) {
	output, err := run(startDir, "status", "--porcelain")
	if err != nil {
		return WorktreeStatus{}, err
	}

	if strings.TrimSpace(output) == "" {
		return WorktreeStatus{}, nil
	}

	var status WorktreeStatus
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		status.Dirty = true
		if strings.HasPrefix(line, "??") {
			status.Untracked++
			continue
		}

		status.Changed++
	}

	return status, nil
}

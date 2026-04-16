package git

import (
	"context"
	"strings"
)

type WorktreeStatus struct {
	Dirty     bool
	Changed   int
	Untracked int
}

func Status(ctx context.Context, startDir string) (WorktreeStatus, error) {
	output, err := run(ctx, startDir, "status", "--porcelain")
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

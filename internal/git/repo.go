package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func Root(startDir string) (string, error) {
	output, err := run(context.Background(), startDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("resolve git repository root: %w", err)
	}

	return output, nil
}

func Branch(startDir string) (string, error) {
	output, err := run(context.Background(), startDir, "branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("resolve git branch: %w", err)
	}

	if output == "" {
		return "detached", nil
	}

	return output, nil
}

func run(ctx context.Context, startDir string, args ...string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", startDir}, args...)...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if errors.As(err, new(*exec.ExitError)) {
			message := strings.TrimSpace(stderr.String())
			if message == "" {
				message = err.Error()
			}
			return "", fmt.Errorf("%s", message)
		}

		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

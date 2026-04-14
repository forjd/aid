package git

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func Root(startDir string) (string, error) {
	output, err := run(startDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("resolve git repository root: %w", err)
	}

	return output, nil
}

func Branch(startDir string) (string, error) {
	output, err := run(startDir, "branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("resolve git branch: %w", err)
	}

	if output == "" {
		return "detached", nil
	}

	return output, nil
}

func run(startDir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", startDir}, args...)...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
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

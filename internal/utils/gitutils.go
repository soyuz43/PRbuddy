package utils

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ExecGit executes a git command with the given arguments and returns the trimmed output.
func ExecGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w (stderr: %q)",
			strings.Join(args, " "),
			err,
			stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GetRepoPath returns the top-level path of the current Git repository.
func GetRepoPath() (string, error) {
	return ExecGit("rev-parse", "--show-toplevel")
}

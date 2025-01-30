// ./coreutils/gitutils.go
package coreutils

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

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

func GetRepoPath() (string, error) {
	return ExecGit("rev-parse", "--show-toplevel")
}

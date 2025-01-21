// internal/utils/utils.go

package utils

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

// ExecuteGitCommand runs a git command and returns its output
func ExecuteGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", errors.Wrapf(err, "git command failed: git %s", strings.Join(args, " "))
	}
	return strings.TrimSpace(out.String()), nil
}

// GetRepoPath retrieves the top-level directory of the current Git repository
func GetRepoPath() (string, error) {
	repoPath, err := ExecuteGitCommand("rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return repoPath, nil
}

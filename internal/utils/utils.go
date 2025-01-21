// internal/utils/utils.go

package utils

import (
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

// ExecuteGitCommand runs a git command and returns its output
func ExecuteGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "git command failed: git %s", strings.Join(args, " "))
	}
	return strings.TrimSpace(string(output)), nil
}

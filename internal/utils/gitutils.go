package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

// ReadGitignore reads the .gitignore file at the given root directory and returns
// a slice of compiled regular expressions representing ignore patterns.
func ReadGitignore(rootDir string) ([]*regexp.Regexp, error) {
	gitignorePath := filepath.Join(rootDir, ".gitignore")

	file, err := os.Open(gitignorePath)
	if err != nil {
		// If .gitignore doesn't exist, return an empty slice.
		if os.IsNotExist(err) {
			return []*regexp.Regexp{}, nil
		}
		return nil, fmt.Errorf("failed to open .gitignore: %w", err)
	}
	defer file.Close()

	var patterns []*regexp.Regexp
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines or comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Convert the gitignore pattern to a regex:
		// 1) Escape special regex characters
		// 2) Convert "*" to ".*"
		// 3) Anchor the pattern to match entire path
		regexPattern := "^" + regexp.QuoteMeta(line) + "$"
		regexPattern = strings.ReplaceAll(regexPattern, "\\*", ".*")
		regex, err := regexp.Compile(regexPattern)
		if err == nil {
			patterns = append(patterns, regex)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading .gitignore: %w", err)
	}

	return patterns, nil
}

// IsIgnored returns true if the file path matches any of the given ignore patterns.
func IsIgnored(path string, patterns []*regexp.Regexp) bool {
	for _, regex := range patterns {
		if regex.MatchString(path) {
			return true
		}
	}
	return false
}

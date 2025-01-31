package utils

import "strings"

// SplitLines splits a string into lines, removing any trailing newline.
func SplitLines(s string) []string {
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// JoinLines joins a slice of strings into a single string separated by newlines.
func JoinLines(lines []string) string {
	return strings.Join(lines, "\n")
}

// SanitizeBranchName replaces "/" with "_" and spaces with "-" in branch names.
func SanitizeBranchName(branch string) string {
	return strings.ReplaceAll(strings.ReplaceAll(branch, "/", "_"), " ", "-")
}

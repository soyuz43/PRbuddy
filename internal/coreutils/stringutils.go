// ./coreutils/stringutils.go
package coreutils

import "strings"

func SplitLines(s string) []string {
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

func JoinLines(lines []string) string {
	return strings.Join(lines, "\n")
}

func SanitizeBranchName(branch string) string {
	return strings.ReplaceAll(strings.ReplaceAll(branch, "/", "_"), " ", "-")
}

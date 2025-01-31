package utils

import "github.com/fatih/color"

var (
	Cyan    = color.New(color.FgCyan).SprintFunc()
	Green   = color.New(color.FgGreen).SprintFunc()
	Yellow  = color.New(color.FgYellow).SprintFunc()
	Red     = color.New(color.FgRed).SprintFunc()
	Magenta = color.New(color.FgMagenta).SprintFunc()
	Bold    = color.New(color.Bold).SprintFunc()
)

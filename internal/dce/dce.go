package dce

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"regexp"
	"strings"

	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
)

// DCE is the interface describing dynamic context methods
type DCE interface {
	Activate(task string) error
	Deactivate(conversationID string) error
	BuildTaskList(input string) ([]contextpkg.Task, error)
	FilterProjectData(tasks []contextpkg.Task) ([]FilteredData, error)
	AugmentContext(context []contextpkg.Message, filteredData []FilteredData) []contextpkg.Message
}

// FilteredData represents any extra project data discovered by the DCE
type FilteredData struct {
	FileHierarchy string
	LinterResults string
}

// DefaultDCE implements the DCE interface
type DefaultDCE struct{}

// NewDCE creates a new DCE instance (returning the DCE interface)
func NewDCE() DCE {
	return &DefaultDCE{}
}

// Activate initializes the DCE with the given task (high-level user goal).
func (d *DefaultDCE) Activate(task string) error {
	fmt.Printf("[DCE] Activated. User says they're working on: %q\n", task)
	return nil
}

// Deactivate cleans up the DCE for the given conversation
func (d *DefaultDCE) Deactivate(conversationID string) error {
	fmt.Printf("[DCE] Deactivated for conversation ID: %s\n", conversationID)
	return nil
}

// BuildTaskList generates a list of tasks based on user input
//  1. Match user keywords to known files (via `git ls-files`)
//  2. Extract function definitions from matched files via regex
func (d *DefaultDCE) BuildTaskList(input string) ([]contextpkg.Task, error) {
	fmt.Printf("[DCE] Building task list from user input: %s\n", input)

	// 1. Grab the list of all tracked files
	trackedFiles, err := d.getGitTrackedFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve git ls-files: %w", err)
	}

	// 2. Determine which files might be relevant
	//    Simple heuristic: split user input on spaces, check if any token is in filename
	matchedFiles := d.matchFilesByKeywords(trackedFiles, input)
	if len(matchedFiles) == 0 {
		// If no direct matches, create a single catch-all "task" with no files.
		t := contextpkg.Task{
			Description:  input,
			Files:        nil,
			Functions:    nil,
			Dependencies: nil,
			Notes:        []string{"No direct file matches found. Developer can add them manually."},
		}
		return []contextpkg.Task{t}, nil
	}

	// 3. Extract function names from each matched file
	allFunctions := make([]string, 0)
	for _, f := range matchedFiles {
		funcs := d.extractFunctionsFromFile(f)
		allFunctions = append(allFunctions, funcs...)
	}

	// 4. Create a single “mega” Task containing these matched files/functions
	taskList := []contextpkg.Task{
		{
			Description:  input,
			Files:        matchedFiles,
			Functions:    allFunctions,
			Dependencies: nil, // We'll populate these in FilterProjectData
			Notes: []string{
				"Matched via user input + simple filename heuristics",
			},
		},
	}
	return taskList, nil
}

// FilterProjectData uses `git diff` to discover changed functions/files,
// then marks them as dependencies or adds notes for the tasks.
func (d *DefaultDCE) FilterProjectData(tasks []contextpkg.Task) ([]FilteredData, error) {
	fmt.Println("[DCE] Filtering project data based on tasks + git diff")

	// 1. Grab the changed lines from Git diff
	diffOutput, err := d.getGitDiff()
	if err != nil {
		return nil, fmt.Errorf("failed to get git diff: %w", err)
	}

	// 2. Search for function-like patterns in the diff (leading plus sign for new lines)
	funcRegex := regexp.MustCompile(`(?m)^\+.*(def|func|function|public|private|static|void)\s+(\w+)\s*\(`)
	matches := funcRegex.FindAllStringSubmatch(diffOutput, -1)

	var changedFuncs []string
	for _, m := range matches {
		changedFuncs = append(changedFuncs, m[2]) // capture function name group
	}

	// 3. For each changed function, update the relevant Task’s Dependencies/Notes
	for i := range tasks {
		for _, cf := range changedFuncs {
			// If tasks[i].Functions already has cf, highlight it
			if stringSliceContains(tasks[i].Functions, cf) {
				tasks[i].Dependencies = append(tasks[i].Dependencies, cf)
				tasks[i].Notes = append(tasks[i].Notes, fmt.Sprintf("Function %s changed in recent diff.", cf))
			}
		}
	}

	// 4. Create a FilteredData result summarizing the detected changes
	fd := []FilteredData{
		{
			FileHierarchy: "N/A (use file path matching if needed)",
			LinterResults: fmt.Sprintf("Detected %d changed function(s) in diff: %v",
				len(changedFuncs), changedFuncs),
		},
	}

	return fd, nil
}

// AugmentContext adds a system-level message summarizing tasks & changes
func (d *DefaultDCE) AugmentContext(context []contextpkg.Message, filteredData []FilteredData) []contextpkg.Message {
	fmt.Println("[DCE] Augmenting conversation context with tasks and changed file data")

	// Summarize the changes in a single system message
	var builder strings.Builder
	builder.WriteString("**Dynamic Context Engine Summary**\n\n")
	for _, fd := range filteredData {
		builder.WriteString(fmt.Sprintf("- File Hierarchy: %s\n", fd.FileHierarchy))
		builder.WriteString(fmt.Sprintf("- Linter/Change Results: %s\n", fd.LinterResults))
	}

	augmented := append(context, contextpkg.Message{
		Role:    "system",
		Content: builder.String(),
	})
	return augmented
}

// ----------------------
//    Helper Methods
// ----------------------

// Retrieves a list of files tracked by Git
func (d *DefaultDCE) getGitTrackedFiles() ([]string, error) {
	cmd := exec.Command("git", "ls-files")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return lines, nil
}

// Given a set of files & the user’s text, match file paths if they share any words
func (d *DefaultDCE) matchFilesByKeywords(allFiles []string, userInput string) []string {
	words := strings.Fields(strings.ToLower(userInput))
	matched := make([]string, 0)

	for _, file := range allFiles {
		lowerFile := strings.ToLower(file)
		for _, w := range words {
			// Heuristic: require w to be >= 3 chars to reduce noise
			if len(w) >= 3 && strings.Contains(lowerFile, w) {
				matched = append(matched, file)
				break
			}
		}
	}
	return matched
}

// Regex-based function extraction from a file
func (d *DefaultDCE) extractFunctionsFromFile(filePath string) []string {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil
	}

	functionRegex := regexp.MustCompile(`(?m)^\s*(def|func|function|public|private|static|void)\s+(\w+)\s*\(`)
	matches := functionRegex.FindAllStringSubmatch(string(data), -1)

	var funcs []string
	for _, match := range matches {
		funcs = append(funcs, match[2]) // e.g. "myFunction"
	}
	return funcs
}

// Runs `git diff` to see changes
func (d *DefaultDCE) getGitDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--unified=0")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Utility: check if a slice contains a string
func stringSliceContains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

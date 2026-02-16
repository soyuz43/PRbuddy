// test/dce/command_menu/add_test.go
package command_menu_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
	"github.com/soyuz43/prbuddy-go/internal/dce"
	"github.com/soyuz43/prbuddy-go/test"
)

func TestAddCommand_AddsNewTask(t *testing.T) {
	// Setup
	_, littleguy := test.SetupDCEForTesting(t, "Initial task")

	// Add a new task with description that will match test files
	dce.HandleDCECommandMenu("/add Implement test helpers", littleguy)

	// Capture verbose output to see files and functions
	mockOutput := &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)
	dce.HandleDCECommandMenu("/tasks -v", littleguy)

	// Verify the new task appears in the verbose task list
	output := mockOutput.String()
	if !strings.Contains(output, "Implement test helpers") {
		t.Error("Added task not found in task list")
	}

	// Verify task details are displayed in verbose mode
	if !strings.Contains(output, "add_test.go") {
		t.Error("Expected file 'add_test.go' not found in verbose task list")
	}
	if !strings.Contains(output, "help_test.go") {
		t.Error("Expected file 'help_test.go' not found in verbose task list")
	}

	// Verify function presence in verbose mode
	if !strings.Contains(output, "TestAddCommand_AddsNewTask") {
		t.Error("Expected function 'TestAddCommand_AddsNewTask' not found in verbose task list")
	}
	if !strings.Contains(output, "TestHelpCommandDisplaysCorrectly") {
		t.Error("Expected function 'TestHelpCommandDisplaysCorrectly' not found in verbose task list")
	}
}

func TestAddCommand_WithVerboseOutput(t *testing.T) {
	// Setup
	_, littleguy := test.SetupDCEForTesting(t, "Initial task")

	// Add a new task with description that will match test files
	dce.HandleDCECommandMenu("/add Implement test helpers", littleguy)

	// Capture verbose task output
	mockOutput := &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)
	dce.HandleDCECommandMenu("/tasks -v", littleguy)

	// Verify verbose details are displayed
	output := mockOutput.String()
	if !strings.Contains(output, "Implement test helpers") {
		t.Error("Added task not found in verbose task list")
	}

	// Verify all details are displayed in verbose mode
	if !strings.Contains(output, "add_test.go") {
		t.Error("Expected file 'add_test.go' not found in verbose task list")
	}
	if !strings.Contains(output, "TestAddCommand_AddsNewTask") {
		t.Error("Expected function 'TestAddCommand_AddsNewTask' not found in verbose task list")
	}
	if !strings.Contains(output, "Matched via input and file heuristics.") {
		t.Error("Expected notes not displayed in verbose task list")
	}
}

func TestAddCommand_WithEmptyDescription(t *testing.T) {
	// Setup
	_, littleguy := test.SetupDCEForTesting(t, "Initial task")

	// Capture output
	mockOutput := &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)

	// Try to add a task with empty description
	dce.HandleDCECommandMenu("/add", littleguy)

	// Verify error message
	output := mockOutput.String()
	if !strings.Contains(output, "Please provide a task description") {
		t.Errorf("Expected error message about empty description, got: %q", output)
	}
}

func TestAddCommand_WhenDCEInactive(t *testing.T) {
	// Setup - create DCE but don't activate it
	conversationID := contextpkg.GenerateConversationID("test")
	littleguy := dce.NewLittleGuy(conversationID, []contextpkg.Task{})

	// Capture output
	mockOutput := &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)

	// Try to add a task with DCE inactive
	dce.HandleDCECommandMenu("/add Test task", littleguy)

	// Verify output
	output := mockOutput.String()
	if !strings.Contains(output, "Successfully added 1 task(s) to the task list") {
		t.Errorf("Expected success message, got: %q", output)
	}

	// Verify task was added
	mockOutput = &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)
	dce.HandleDCECommandMenu("/tasks", littleguy)
	output = mockOutput.String()

	if !strings.Contains(output, "Test task") {
		t.Error("Added task not found in task list")
	}
}

func TestAddCommand_MultipleTasks(t *testing.T) {
	// Setup
	_, littleguy := test.SetupDCEForTesting(t, "Initial task")

	// Add multiple tasks
	tasks := []string{
		"/add Implement core functionality",
		"/add Add error handling",
		"/add Write documentation",
	}

	for _, task := range tasks {
		dce.HandleDCECommandMenu(task, littleguy)
	}

	// Capture output
	mockOutput := &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)
	dce.HandleDCECommandMenu("/tasks", littleguy)

	output := mockOutput.String()

	// Verify all tasks are present
	for _, task := range tasks {
		description := strings.TrimPrefix(task, "/add ")
		if !strings.Contains(output, description) {
			t.Errorf("Task '%s' not found in task list", description)
		}
	}
}

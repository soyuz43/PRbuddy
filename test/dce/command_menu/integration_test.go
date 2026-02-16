// test/dce/command_menu/integration_test.go
package command_menu_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
	"github.com/soyuz43/prbuddy-go/internal/dce"
	"github.com/soyuz43/prbuddy-go/test"
)

func TestAddAndDisplayTasks(t *testing.T) {
	// Setup
	_, littleguy := test.SetupDCEForTesting(t, "Initial task")

	// Add a new task with description that will match test files
	dce.HandleDCECommandMenu("/add Implement test helpers", littleguy)

	// Capture output of /tasks
	mockOutput := &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)
	dce.HandleDCECommandMenu("/tasks", littleguy)

	// Verify the new task appears in the task list
	output := mockOutput.String()
	if !strings.Contains(output, "Implement test helpers") {
		t.Error("Added task not found in task list")
	}

	// Check if we have files or it's a catch-all task
	if strings.Contains(output, "Files:") || strings.Contains(output, "Functions:") {
		// Task matched files - verify details are displayed
	} else {
		// Task is a catch-all task - verify appropriate message
		if !strings.Contains(output, "No direct file matches found") {
			t.Error("Expected 'No direct file matches found' message for catch-all task")
		}
	}
}

func TestAddCommandWithVerboseTasks(t *testing.T) {
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

	// Check if we have files or it's a catch-all task
	if strings.Contains(output, "Files:") || strings.Contains(output, "Functions:") || strings.Contains(output, "Notes:") {
		// Task matched files - verify details are displayed
	} else {
		// Task is a catch-all task - verify appropriate message
		if !strings.Contains(output, "No direct file matches found") {
			t.Error("Expected 'No direct file matches found' message for catch-all task")
		}
	}
}

func TestAddCommandWhenDCEInactive(t *testing.T) {
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
		t.Error("Expected success message not found when DCE is inactive")
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

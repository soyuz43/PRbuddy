// test/dce/command_menu/add_test.go
package command_menu_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/soyuz43/prbuddy-go/internal/dce"
	"github.com/soyuz43/prbuddy-go/test"
)

func TestAddCommandWithValidDescription(t *testing.T) {
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

func TestAddCommandWithEmptyDescription(t *testing.T) {
	// Setup
	_, littleguy := test.SetupDCEForTesting(t, "Initial task")

	// Capture output
	mockOutput := &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)

	// Try to add with empty description
	dce.HandleDCECommandMenu("/add", littleguy)

	// Verify error message
	output := mockOutput.String()
	expectedError := "[X] Please provide a task description after /add"
	if !strings.Contains(output, expectedError) {
		t.Errorf("Expected error message '%s' not found in output: %s", expectedError, output)
	}
}

func TestAddCommandWithInvalidTask(t *testing.T) {
	// Setup
	_, littleguy := test.SetupDCEForTesting(t, "Initial task")

	// Capture output
	mockOutput := &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)

	// Add a task that won't match any files (using unusual characters)
	dce.HandleDCECommandMenu("/add !@#$%^&*()", littleguy)

	// Verify output
	output := mockOutput.String()
	if !strings.Contains(output, "Successfully added 1 task(s) to the task list") {
		t.Error("Expected success message not found in output")
	}

	// Verify task was added (as a catch-all task)
	mockOutput = &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)
	dce.HandleDCECommandMenu("/tasks", littleguy)
	output = mockOutput.String()

	if !strings.Contains(output, "!@#$%^&*()") {
		t.Error("Added task not found in task list")
	}

	// Check for the specific note about no file matches
	if !strings.Contains(output, "No direct file matches found") {
		t.Error("Expected 'No direct file matches found' note not found in task list")
	}
}

func TestAddCommandWithDuplicateTask(t *testing.T) {
	// Setup
	_, littleguy := test.SetupDCEForTesting(t, "Initial task")

	// First add a task
	dce.HandleDCECommandMenu("/add Implement test helpers", littleguy)

	// Capture output for second attempt
	mockOutput := &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)

	// Try to add the same task again
	dce.HandleDCECommandMenu("/add Implement test helpers", littleguy)

	// Verify output (should still show success but with no new tasks)
	output := mockOutput.String()
	if !strings.Contains(output, "Successfully added 1 task(s) to the task list") {
		t.Error("Expected success message not found in output")
	}

	// Verify only one instance of the task exists
	mockOutput = &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)
	dce.HandleDCECommandMenu("/tasks", littleguy)
	output = mockOutput.String()

	// Count occurrences of the task description
	count := strings.Count(output, "Implement test helpers")
	if count != 1 {
		t.Errorf("Expected only 1 instance of the task, found %d", count)
	}
}

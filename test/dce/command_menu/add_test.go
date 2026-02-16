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

	// Capture output
	mockOutput := &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)

	// Add a new task
	dce.HandleDCECommandMenu("/add Implement authentication system", littleguy)

	// Verify output
	output := mockOutput.String()
	if !strings.Contains(output, "Successfully added 1 task(s) to the task list") {
		t.Error("Expected success message not found in output")
	}
	if !strings.Contains(output, "Implement authentication system") {
		t.Error("Expected task description not found in output")
	}
	if !strings.Contains(output, "Files:") || !strings.Contains(output, "Functions:") {
		t.Error("Expected task details not found in output")
	}

	// Verify task was added to the list by checking /tasks output
	mockOutput = &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)
	dce.HandleDCECommandMenu("/tasks", littleguy)
	output = mockOutput.String()

	if !strings.Contains(output, "Implement authentication system") {
		t.Error("Added task not found in task list")
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
	if !strings.Contains(output, "Please provide a task description after /add") {
		t.Error("Expected error message not found for empty description")
	}

	// Verify no tasks were added
	mockOutput = &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)
	dce.HandleDCECommandMenu("/tasks", littleguy)
	output = mockOutput.String()

	// Count tasks in output (should be 1 for the initial task)
	taskCount := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "  ") && strings.Contains(line, ")") {
			taskCount++
		}
	}

	if taskCount != 1 {
		t.Errorf("Expected 1 task (initial), got %d", taskCount)
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
	if !strings.Contains(output, "No direct file matches found") {
		t.Error("Expected 'no file matches' note not found in output")
	}

	// Verify task was added (as a catch-all task)
	mockOutput = &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)
	dce.HandleDCECommandMenu("/tasks", littleguy)
	output = mockOutput.String()

	if !strings.Contains(output, "!@#$%^&*()") {
		t.Error("Added task not found in task list")
	}
	if !strings.Contains(output, "No direct file matches found") {
		t.Error("Expected 'no file matches' note not found in task list")
	}
}

func TestAddCommandWithDuplicateTask(t *testing.T) {
	// Setup
	_, littleguy := test.SetupDCEForTesting(t, "Initial task")

	// First add a task
	dce.HandleDCECommandMenu("/add Implement authentication system", littleguy)

	// Capture output for second attempt
	mockOutput := &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)

	// Try to add the same task again
	dce.HandleDCECommandMenu("/add Implement authentication system", littleguy)

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
	count := strings.Count(output, "Implement authentication system")
	if count != 1 {
		t.Errorf("Expected only 1 instance of the task, found %d", count)
	}
}

// test/dce/command_menu/integration_test.go
package command_menu_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/soyuz43/prbuddy-go/internal/dce"
	"github.com/soyuz43/prbuddy-go/test"
)

func TestIntegration_FullDCEWorkflow(t *testing.T) {
	// Setup
	_, littleguy := test.SetupDCEForTesting(t, "Initial task")

	// Step 1: Add a task
	dce.HandleDCECommandMenu("/add Implement test helpers", littleguy)

	// Step 2: Check tasks
	mockOutput := &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)
	dce.HandleDCECommandMenu("/tasks", littleguy)
	output := mockOutput.String()
	if !strings.Contains(output, "Implement test helpers") {
		t.Error("Added task not found in task list")
	}

	// Step 3: Check status
	mockOutput = &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)
	dce.HandleDCECommandMenu("/status", littleguy)
	output = mockOutput.String()
	if !strings.Contains(output, "Active Tasks:") {
		t.Error("Status output missing")
	}

	// Step 4: Mark task as completed (FIX: Use task 2, not task 1)
	dce.HandleDCECommandMenu("/complete 2", littleguy)

	// Step 5: Verify task was removed
	mockOutput = &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)
	dce.HandleDCECommandMenu("/tasks", littleguy)
	output = mockOutput.String()
	if strings.Contains(output, "Implement test helpers") {
		t.Error("Completed task still appears in task list")
	}
}

func TestIntegration_DCEActivationDeactivation(t *testing.T) {
	// Setup
	_, littleguy := test.SetupDCEForTesting(t, "Initial task")

	// FIX: Deactivate DCE first since SetupDCEForTesting activates it
	dce.HandleDCECommandMenu("/dce off", littleguy)

	// Test DCE activation
	mockOutput := &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)
	dce.HandleDCECommandMenu("/dce on", littleguy)
	output := mockOutput.String()
	if !strings.Contains(output, "Dynamic Context Engine activated") {
		t.Error("DCE activation message not found")
	}

	// Check status after activation
	mockOutput = &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)
	dce.HandleDCECommandMenu("/status", littleguy)
	output = mockOutput.String()
	if !strings.Contains(output, "ACTIVE") {
		t.Error("DCE should show as ACTIVE")
	}

	// Test DCE deactivation
	mockOutput = &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)
	dce.HandleDCECommandMenu("/dce off", littleguy)
	output = mockOutput.String()
	if !strings.Contains(output, "Dynamic Context Engine deactivated") {
		t.Error("DCE deactivation message not found")
	}
}

func TestIntegration_TaskPrioritization(t *testing.T) {
	// Setup
	_, littleguy := test.SetupDCEForTesting(t, "Initial task")

	// Add multiple tasks
	dce.HandleDCECommandMenu("/add Task 1: Critical bug fix", littleguy)
	dce.HandleDCECommandMenu("/add Task 2: Feature implementation", littleguy)
	dce.HandleDCECommandMenu("/add Task 3: Documentation update", littleguy)

	// Set priorities (FIX: Offset by 1 to account for initial task)
	dce.HandleDCECommandMenu("/priority 2 high", littleguy)
	dce.HandleDCECommandMenu("/priority 3 medium", littleguy)
	dce.HandleDCECommandMenu("/priority 4 low", littleguy)

	// Check priorities
	mockOutput := &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)
	dce.HandleDCECommandMenu("/priority", littleguy)
	output := mockOutput.String()

	// Verify priority labels are present
	expectedPriorities := []string{"[High]", "[Medium]", "[Low]"}
	for _, priority := range expectedPriorities {
		if !strings.Contains(output, priority) {
			t.Errorf("Expected priority '%s' not found in output", priority)
		}
	}
}

func TestIntegration_CommandAliases(t *testing.T) {
	// Setup
	_, littleguy := test.SetupDCEForTesting(t, "Initial task")

	// Test different command aliases
	testCases := []struct {
		command     string
		shouldMatch bool
	}{
		{"/task", true},
		{"/tasks", true},
		{"/cmds", true},
		{"/commands", true},
		{"/help", true},
		{"/invalid", false},
	}

	for _, tc := range testCases {
		t.Run("Command_"+tc.command, func(t *testing.T) {
			mockOutput := &MockOutputWriter{Buffer: &bytes.Buffer{}}
			SetOutputForTests(mockOutput)
			result := dce.HandleDCECommandMenu(tc.command, littleguy)
			if tc.shouldMatch && !result {
				t.Errorf("Command '%s' should have been handled but wasn't", tc.command)
			}
			if !tc.shouldMatch && result {
				t.Errorf("Command '%s' should not have been handled but was", tc.command)
			}
		})
	}
}

func TestIntegration_RefreshTaskList(t *testing.T) {
	// Setup
	_, littleguy := test.SetupDCEForTesting(t, "Initial task")

	// Add a task
	dce.HandleDCECommandMenu("/add Implement test helpers", littleguy)

	// Manually trigger refresh
	mockOutput := &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)
	dce.HandleDCECommandMenu("/refresh", littleguy)
	output := mockOutput.String()
	if !strings.Contains(output, "Refreshing task list from git changes") {
		t.Error("Refresh command output not as expected")
	}
}

func TestIntegration_InvalidCommands(t *testing.T) {
	// Setup
	_, littleguy := test.SetupDCEForTesting(t, "Initial task")

	invalidCommands := []string{
		"/add",               // missing description
		"/priority",          // missing args
		"/complete",          // missing task number
		"/complete abc",      // invalid task number
		"/priority abc high", // invalid task number
		"/dce unknown",       // invalid dce subcommand
	}

	for _, cmd := range invalidCommands {
		t.Run("Invalid_"+cmd, func(t *testing.T) {
			mockOutput := &MockOutputWriter{Buffer: &bytes.Buffer{}}
			SetOutputForTests(mockOutput)
			dce.HandleDCECommandMenu(cmd, littleguy)
			output := mockOutput.String()
			// Should see some error message (not empty)
			if output == "" {
				t.Errorf("Expected error output for invalid command '%s', got empty", cmd)
			}
		})
	}
}

// test/dce/command_menu/help_test.go
package command_menu_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/soyuz43/prbuddy-go/internal/dce"
	"github.com/soyuz43/prbuddy-go/test"
)

func TestHelpCommandDisplaysCorrectly(t *testing.T) {
	// Setup
	_, littleguy := test.SetupDCEForTesting(t, "Test task")

	// Capture output
	mockOutput := &MockOutputWriter{Buffer: &bytes.Buffer{}}
	SetOutputForTests(mockOutput)

	// Test /help
	dce.HandleDCECommandMenu("/help", littleguy)
	output := mockOutput.String()

	// Verify command menu is displayed
	if !strings.Contains(output, "Available DCE Commands") {
		t.Error("Help output doesn't contain command menu header")
	}

	// Verify /add is in the command list
	if !strings.Contains(output, "/add <description>") {
		t.Error("Help output doesn't include /add command")
	}

	// Verify all command aliases are mentioned
	if !strings.Contains(output, "/commands, /cmds, /help") {
		t.Error("Help output doesn't properly list all command aliases")
	}
}

func TestAllHelpCommandAliasesWork(t *testing.T) {
	// Setup
	_, littleguy := test.SetupDCEForTesting(t, "Test task")

	// Test all help command variants
	commands := []string{"/help", "/commands", "/cmds"}

	for _, cmd := range commands {
		t.Run(fmt.Sprintf("Command_%s", cmd), func(t *testing.T) {
			mockOutput := &MockOutputWriter{Buffer: &bytes.Buffer{}}
			SetOutputForTests(mockOutput)

			dce.HandleDCECommandMenu(cmd, littleguy)
			output := mockOutput.String()

			if !strings.Contains(output, "Available DCE Commands") {
				t.Errorf("Output for '%s' doesn't contain command menu", cmd)
			}

			if !strings.Contains(output, "/add <description>") {
				t.Errorf("Output for '%s' doesn't include /add command", cmd)
			}
		})
	}
}

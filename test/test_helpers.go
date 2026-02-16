// test/test_helpers.go
package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
	"github.com/soyuz43/prbuddy-go/internal/dce"
	"github.com/soyuz43/prbuddy-go/internal/utils"
)

// SetupTestRepository creates a temporary Git repository for testing
func SetupTestRepository(t *testing.T) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "prbuddy-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Initialize Git repository
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	if _, err := utils.ExecGit("init"); err != nil {
		t.Fatalf("Failed to init Git repo: %v", err)
	}

	// Create test files with realistic content
	files := map[string]string{
		"cmd/context.go": `package cmd

import (
	"fmt"
)

func init() {
	fmt.Println("Initializing command package")
}

func ExampleFunction() {
	// Example implementation
}
`,
		"internal/contextpkg/context.go": `package contextpkg

type Message struct {
	Role    string ` + "`" + `json:"role"` + "`" + `
	Content string ` + "`" + `json:"content"` + "`" + `
}

type Task struct {
	Description string   ` + "`" + `json:"description"` + "`" + `
	Files       []string ` + "`" + `json:"files"` + "`" + `
	Functions   []string ` + "`" + `json:"functions"` + "`" + `
}
`,
		"internal/dce/dce.go": `package dce

type DCE interface {
	Activate(task string) error
	Deactivate(conversationID string) error
	BuildTaskList(input string) ([]contextpkg.Task, []string, error)
}

type DefaultDCE struct{}

func NewDCE() DCE {
	return &DefaultDCE{}
}
`,
		"internal/dce/command_menu.go": `package dce

func HandleDCECommandMenu(input string, littleguy *LittleGuy) bool {
	// Command handling logic
	return true
}
`,
		"README.md": "# Test Repository\nThis is a test repository for PRBuddy-Go",
	}

	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	// Add and commit files
	if _, err := utils.ExecGit("add", "."); err != nil {
		t.Fatalf("Failed to add files: %v", err)
	}
	if _, err := utils.ExecGit("commit", "-m", "Initial commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	return tmpDir
}

// CleanupTestRepository removes the temporary repository
func CleanupTestRepository(t *testing.T, repoPath string) {
	t.Helper()
	if err := os.RemoveAll(repoPath); err != nil {
		t.Logf("Failed to cleanup test repository: %v", err)
	}
}

// SetupDCEForTesting initializes a DCE instance for testing
func SetupDCEForTesting(t *testing.T, initialTask string) (string, *dce.LittleGuy) {
	t.Helper()

	// Create a conversation ID
	conversationID := contextpkg.GenerateConversationID("test")

	// Initialize DCE
	dceInstance := dce.NewDCE()
	if err := dceInstance.Activate(initialTask); err != nil {
		t.Fatalf("Failed to activate DCE: %v", err)
	}

	// Get the LittleGuy instance
	littleguy, exists := dce.GetDCEContextManager().GetContext(conversationID)
	if !exists {
		t.Fatal("Failed to get LittleGuy instance after DCE activation")
	}

	return conversationID, littleguy
}

// AssertTaskContains checks if a task contains specific elements
func AssertTaskContains(t *testing.T, task contextpkg.Task, description string, files []string, functions []string) {
	t.Helper()

	if task.Description != description {
		t.Errorf("Expected task description '%s', got '%s'", description, task.Description)
	}

	// Check files
	if len(files) != len(task.Files) {
		t.Errorf("Expected %d files, got %d", len(files), len(task.Files))
	} else {
		for i, file := range files {
			if task.Files[i] != file {
				t.Errorf("Expected file '%s' at index %d, got '%s'", file, i, task.Files[i])
			}
		}
	}

	// Check functions
	if len(functions) != len(task.Functions) {
		t.Errorf("Expected %d functions, got %d", len(functions), len(task.Functions))
	} else {
		for i, fn := range functions {
			if task.Functions[i] != fn {
				t.Errorf("Expected function '%s' at index %d, got '%s'", fn, i, task.Functions[i])
			}
		}
	}
}

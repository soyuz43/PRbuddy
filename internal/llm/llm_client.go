// internal/llm/llm_client.go

package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
	"github.com/soyuz43/prbuddy-go/internal/dce"
	"github.com/soyuz43/prbuddy-go/internal/utils"
)

// LLMClient defines the interface for interacting with the LLM
type LLMClient interface {
	GetChatResponse(messages []contextpkg.Message) (string, error)
}

// DefaultLLMClient implements the LLMClient interface
type DefaultLLMClient struct{}

// GetChatResponse sends messages to the LLM endpoint and returns the assistant's response
func (c *DefaultLLMClient) GetChatResponse(messages []contextpkg.Message) (string, error) {
	model, endpoint := GetLLMConfig()

	requestBody := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"options": map[string]interface{}{
			"num_ctx": 8192,
		},
		"stream": false,
	}

	jsonBody, err := utils.MarshalJSON(requestBody)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal request body")
	}

	resp, err := http.Post(endpoint+"/api/chat", "application/json", bytes.NewBuffer([]byte(jsonBody)))
	if err != nil {
		return "", errors.Wrap(err, "failed to send POST request to LLM")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM responded with status code %d", resp.StatusCode)
	}

	var llmResp LLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&llmResp); err != nil {
		return "", errors.Wrap(err, "failed to decode LLM response")
	}

	if llmResp.Message.Content == "" {
		return "", fmt.Errorf("empty response from LLM")
	}

	logrus.Info("Received response from LLM successfully.")
	return llmResp.Message.Content, nil
}

// Global LLM client instance
var (
	llmClient LLMClient = &DefaultLLMClient{}
)

// SetLLMClient allows injecting a different LLMClient (useful for testing or future extensions)
func SetLLMClient(client LLMClient) {
	llmClient = client
}

// LLMResponse represents the response structure from the LLM
type LLMResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

// HandleExtensionQuickAssist handles extension requests with conversation context
// If ephemeral == true, the conversation is only kept in-memory (no disk saving).
// conversationID: optional ID. If not found and ephemeral, a new ephemeral conversation is created.
func HandleExtensionQuickAssist(conversationID, input string, ephemeral bool) (string, error) {
	if input == "" {
		return "", fmt.Errorf("no user message provided")
	}

	// Get existing conversation or create a new ephemeral one
	conv, exists := contextpkg.ConversationManagerInstance.GetConversation(conversationID)
	if !exists {
		// Generate a new conversation ID if not provided or not found
		if conversationID == "" {
			conversationID = contextpkg.GenerateConversationID("ephemeral")
		}
		conv = contextpkg.ConversationManagerInstance.StartConversation(conversationID, "", ephemeral)
	}

	// Add user message to the conversation
	conv.AddMessage("user", input)

	// If ephemeral, initialize and use DCE
	if ephemeral {
		// Initialize DCE instance
		dceInstance := dce.NewDCE()
		if err := dceInstance.Activate(input); err != nil {
			return "", fmt.Errorf("DCE activation failed: %w", err)
		}
		defer dceInstance.Deactivate(conversationID)

		// Build task list from user input
		taskList, err := dceInstance.BuildTaskList(input)
		if err != nil {
			return "", fmt.Errorf("failed to build task list: %w", err)
		}

		// Filter project data based on tasks
		filteredData, err := dceInstance.FilterProjectData(taskList)
		if err != nil {
			return "", fmt.Errorf("failed to filter project data: %w", err)
		}

		// Build initial context and augment with filtered data
		augmentedContext := dceInstance.AugmentContext(conv.BuildContext(), filteredData)
		conv.SetMessages(augmentedContext)
	}

	// Build the final context for LLM response generation
	context := conv.BuildContext()

	// Retrieve response from LLM
	response, err := llmClient.GetChatResponse(context)
	if err != nil {
		return "", fmt.Errorf("failed to get response from LLM: %w", err)
	}

	// Add assistant response to the conversation
	conv.AddMessage("assistant", response)

	return response, nil
}

// HandleCLIQuickAssist handles CLI requests (purely stateless for quick usage)
func HandleCLIQuickAssist(input string) (string, error) {
	// Build stateless context
	statelessMessages := []contextpkg.Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: input},
	}

	// Get response from LLM
	response, err := llmClient.GetChatResponse(statelessMessages)
	if err != nil {
		return "", err
	}

	return response, nil
}

// StartPRConversation initiates a new PR conversation with a commit message and diffs
func StartPRConversation(commitMessage, diffs string) (string, string, error) {
	// Generate conversation ID
	conversationID := fmt.Sprintf("pr-%d", time.Now().UnixNano())

	// Create new conversation (persistent - ephemeral=false)
	conv := contextpkg.ConversationManagerInstance.StartConversation(conversationID, diffs, false)

	// Generate initial prompt for the PR
	prompt := fmt.Sprintf(`
You are an assistant designed to generate a detailed pull request (PR) description based on the following commit message and code changes.

**Commit Message:**
%s

**Code Changes:**
%s

!TASK: Provide a comprehensive PR title and description that explain the changes and adhere to documentation and GitHub best practices. Format the pull request in raw markdown with headers. Clearly separate the pull request and other components of the response with three backticks and append the draft PR in code blocks.
`, commitMessage, diffs)

	// Add initial user message
	conv.AddMessage("user", prompt)

	// Get initial response from LLM
	response, err := llmClient.GetChatResponse(conv.BuildContext())
	if err != nil {
		return "", "", err
	}

	// Add assistant response
	conv.AddMessage("assistant", response)

	return conversationID, response, nil
}

// ContinuePRConversation continues an existing PR conversation
func ContinuePRConversation(conversationID, input string) (string, error) {
	// For PR conversations, ephemeral=false, so we skip that param
	return HandleExtensionQuickAssist(conversationID, input, false)
}

// GeneratePreDraftPR fetches the latest commit message and diffs
func GeneratePreDraftPR() (string, string, error) {
	commitMsg, err := utils.ExecuteGitCommand("log", "-1", "--pretty=%B")
	if err != nil {
		return "", "", errors.Wrap(err, "failed to get latest commit message")
	}

	diff, err := utils.GetDiffs(utils.DiffSinceLastCommit)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to get git diff")
	}

	return commitMsg, diff, nil
}

// GenerateDraftPR uses the LLM's chat endpoint to generate a PR draft (stateless)
func GenerateDraftPR(commitMessage, diffs string) (string, error) {
	prompt := fmt.Sprintf(`
You are an assistant designed to generate a detailed pull request (PR) description based on the following commit message and code changes.

**Commit Message:**
%s

**Code Changes:**
%s

!TASK: Provide a comprehensive PR title and description that explain the changes and adhere to documentation and GitHub best practices. Format the pull request in raw markdown with headers. Clearly separate the pull request and other components of the response with three backticks and append the draft PR in code blocks.
`, commitMessage, diffs)

	statelessMessages := []contextpkg.Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: prompt},
	}

	response, err := llmClient.GetChatResponse(statelessMessages)
	if err != nil {
		return "", err
	}

	return response, nil
}

// GenerateWhatSummary generates a summary of git diffs using the LLM
func GenerateWhatSummary() (string, error) {
	diffs, err := utils.GetDiffs(utils.DiffAllLocalChanges)
	if err != nil {
		return "", fmt.Errorf("failed to get diffs: %w", err)
	}

	if diffs == "" {
		return "No changes detected since the last commit.", nil
	}

	prompt := fmt.Sprintf(`
These are the git diffs for the repository:

%s

---
!TASK::
1. Provide a meticulous natural language summary of each of the changes. Do so by file. Describe each change made in full.
2. List and separate changes for each file changed using numbered points and markdown formatting.
3. Only describe the changes explicitly present in the diffs. Do not infer, speculate, or invent additional content.
4. Focus on helping the developer reorient themselves and understand where they left off.
`, diffs)

	statelessMessages := []contextpkg.Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: prompt},
	}

	return llmClient.GetChatResponse(statelessMessages)
}

// GetLLMConfig gets the current model and endpoint from environment variables
func GetLLMConfig() (string, string) {
	endpoint := os.Getenv("PRBUDDY_LLM_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	// Use the active model from contextpkg if set, else fallback
	m := contextpkg.GetActiveModel()
	if m == "" {
		// fallback to environment or default
		m = os.Getenv("PRBUDDY_LLM_MODEL")
		if m == "" {
			m = "hermes3"
		}
	}

	return m, endpoint
}

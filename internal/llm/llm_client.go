// internal/llm/llm_client.go

package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/soyuz43/prbuddy-go/internal/contextpkg"
	"github.com/soyuz43/prbuddy-go/internal/dce"
	"github.com/soyuz43/prbuddy-go/internal/utils"
)

//------------------------------------------------------------------------------
// LLMClient INTERFACE + DEFAULT IMPLEMENTATION
//------------------------------------------------------------------------------

// LLMClient defines the interface for interacting with the LLM (Ollama).
type LLMClient interface {
	// For non-streaming calls
	GetChatResponse(messages []contextpkg.Message) (string, error)
	// For streaming calls (accumulate chunks under the hood)
	StreamChatResponse(messages []contextpkg.Message) (<-chan string, error)
}

// DefaultLLMClient implements the LLMClient interface using Ollama’s /api/chat.
type DefaultLLMClient struct{}

//------------------------------------------------------------------------------
// NON-STREAMING METHOD: GetChatResponse
//------------------------------------------------------------------------------

func (c *DefaultLLMClient) GetChatResponse(messages []contextpkg.Message) (string, error) {
	model, endpoint := GetLLMConfig()

	// Request body: force "stream": false
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

	resp, err := http.Post(endpoint+"/api/chat", "application/json", strings.NewReader(jsonBody))
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

	logrus.Info("Received response from LLM successfully (non-stream).")
	return llmResp.Message.Content, nil
}

//------------------------------------------------------------------------------
// STREAMING METHOD: StreamChatResponse
//------------------------------------------------------------------------------

// StreamChatResponse reads lines from Ollama’s /api/chat as soon as they arrive.
// Each line is expected to be a complete JSON object. When "done" = true, we stop.
func (c *DefaultLLMClient) StreamChatResponse(messages []contextpkg.Message) (<-chan string, error) {
	model, endpoint := GetLLMConfig()

	reqBody := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"stream":   true,
		"options": map[string]interface{}{
			"num_ctx": 8192,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint+"/api/chat", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute HTTP request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}

	outChan := make(chan string)

	go func() {
		defer resp.Body.Close()
		defer close(outChan)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var chunk OllamaStreamChunk
			if err := json.Unmarshal([]byte(line), &chunk); err != nil {
				// Log parse errors but keep going
				logrus.Errorf("Failed to unmarshal streaming chunk: %v", err)
				continue
			}

			// If "done" is true, streaming has ended
			if chunk.Done {
				break
			}

			// Send content if present
			if chunk.Message != nil && chunk.Message.Content != "" {
				outChan <- chunk.Message.Content
			}
		}

		// If there's a scanning error, log it
		if err := scanner.Err(); err != nil {
			logrus.Errorf("Scanner error while reading streaming response: %v", err)
		}
	}()

	return outChan, nil
}

//------------------------------------------------------------------------------
// DATA STRUCTS & GLOBAL
//------------------------------------------------------------------------------

// LLMResponse represents the top-level structure from Ollama (non-streaming).
type LLMResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

// OllamaStreamChunk is used during streaming (partial response).
type OllamaStreamChunk struct {
	Model   string `json:"model,omitempty"`
	Message *struct {
		Role    string   `json:"role,omitempty"`
		Content string   `json:"content,omitempty"`
		Images  []string `json:"images,omitempty"`
	} `json:"message,omitempty"`
	Done bool `json:"done,omitempty"`
}

// llmClient is the global instance implementing LLMClient.
var llmClient LLMClient = &DefaultLLMClient{}

// SetLLMClient allows injecting a different LLMClient (useful for testing or future extensions).
func SetLLMClient(client LLMClient) {
	llmClient = client
}

//------------------------------------------------------------------------------
// PUBLIC HANDLER FUNCTIONS
//------------------------------------------------------------------------------

// HandleQuickAssist returns the final LLM response for a persistent conversation,
// accumulating the streaming output behind-the-scenes into one string.
func HandleQuickAssist(conversationID, input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("no user message provided")
	}

	// Retrieve or create conversation
	conv, exists := contextpkg.ConversationManagerInstance.GetConversation(conversationID)
	if !exists {
		if conversationID == "" {
			conversationID = contextpkg.GenerateConversationID("persistent")
		}
		conv = contextpkg.ConversationManagerInstance.StartConversation(conversationID, "", false)
	}

	// 1) Add user's message
	conv.AddMessage("user", input)

	// 2) Build final context for LLM
	context := conv.BuildContext()

	// 3) Stream from LLM
	streamChan, err := llmClient.StreamChatResponse(context)
	if err != nil {
		return "", fmt.Errorf("failed to stream response: %w", err)
	}

	// 4) Collect the streaming chunks
	var builder strings.Builder
	for chunk := range streamChan {
		builder.WriteString(chunk)
	}
	finalResponse := builder.String()

	// 5) Store assistant's final response in conversation
	conv.AddMessage("assistant", finalResponse)

	return finalResponse, nil
}

// HandleDCERequest handles ephemeral (DCE-driven) requests, returning the final text
// from a fresh ephemeral conversation, after running your DCE logic.
func HandleDCERequest(conversationID, input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("no user message provided")
	}

	// Get or create ephemeral conversation
	conv, exists := contextpkg.ConversationManagerInstance.GetConversation(conversationID)
	if !exists {
		if conversationID == "" {
			conversationID = contextpkg.GenerateConversationID("ephemeral")
		}
		conv = contextpkg.ConversationManagerInstance.StartConversation(conversationID, "", true)
	}

	conv.AddMessage("user", input)

	// Initialize and use DCE
	dceInstance := dce.NewDCE()
	if err := dceInstance.Activate(input); err != nil {
		return "", fmt.Errorf("DCE activation failed: %w", err)
	}
	defer dceInstance.Deactivate(conversationID)

	// Build task list
	taskList, buildLogs, err := dceInstance.BuildTaskList(input)
	if err != nil {
		return "", fmt.Errorf("failed to build task list: %w", err)
	}

	fmt.Println("=== Task List ===")
	for i, task := range taskList {
		fmt.Printf("Task %d:\n", i+1)
		fmt.Printf("  Description: %s\n", task.Description)
		if len(task.Files) > 0 {
			fmt.Printf("  Files: %v\n", task.Files)
		}
		if len(task.Functions) > 0 {
			fmt.Printf("  Functions: %v\n", task.Functions)
		}
		if len(task.Dependencies) > 0 {
			fmt.Printf("  Dependencies: %v\n", task.Dependencies)
		}
		if len(task.Notes) > 0 {
			fmt.Printf("  Notes: %v\n", task.Notes)
		}
	}
	fmt.Println("==================")

	// Add build logs to conversation + console
	for _, logMsg := range buildLogs {
		conv.AddMessage("system", "[DCE] "+logMsg)
		fmt.Println("[DCE]", logMsg)
	}

	// Filter project data
	filteredData, filterLogs, err := dceInstance.FilterProjectData(taskList)
	if err != nil {
		return "", fmt.Errorf("failed to filter project data: %w", err)
	}
	for _, logMsg := range filterLogs {
		conv.AddMessage("system", "[DCE] "+logMsg)
		fmt.Println("[DCE]", logMsg)
	}

	// Augment conversation with filtered data
	augmentedContext := dceInstance.AugmentContext(conv.BuildContext(), filteredData)
	conv.SetMessages(augmentedContext)

	// Save expanded context for debugging
	if err := utils.SaveContextToFile(conv.ID, augmentedContext); err != nil {
		logrus.Errorf("Failed to save context to file: %v", err)
	}
	if err := utils.SaveConcatenatedContextToFile(conv.ID, augmentedContext); err != nil {
		logrus.Errorf("Failed to save concatenated context to file: %v", err)
	}

	// Build final context
	context := conv.BuildContext()

	// Retrieve response (non-streaming) from LLM
	response, err := llmClient.GetChatResponse(context)
	if err != nil {
		return "", fmt.Errorf("failed to get response from LLM: %w", err)
	}

	conv.AddMessage("assistant", response)
	return response, nil
}

// StartPRConversation initiates a new PR conversation with a commit message and diffs.
func StartPRConversation(commitMessage, diffs string) (string, string, error) {
	// Generate a conversation ID
	conversationID := fmt.Sprintf("pr-%d", time.Now().UnixNano())
	conv := contextpkg.ConversationManagerInstance.StartConversation(conversationID, diffs, false)

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

	// Get initial response (non-streaming)
	response, err := llmClient.GetChatResponse(conv.BuildContext())
	if err != nil {
		return "", "", err
	}

	// Add assistant response
	conv.AddMessage("assistant", response)
	return conversationID, response, nil
}

// ContinuePRConversation reuses HandleQuickAssist for continuing a normal (persistent) PR conversation.
func ContinuePRConversation(conversationID, input string) (string, error) {
	return HandleQuickAssist(conversationID, input)
}

// GeneratePreDraftPR obtains the latest commit message and diff, then returns them for usage in PR creation.
func GeneratePreDraftPR() (string, string, error) {
	commitMsg, err := utils.ExecGit("log", "-1", "--pretty=%B")
	if err != nil {
		return "", "", errors.Wrap(err, "failed to get latest commit message")
	}
	diff, err := utils.ExecGit("diff", "HEAD~1", "HEAD")
	if err != nil {
		return "", "", errors.Wrap(err, "failed to get git diff")
	}

	// Intelligent truncation: prioritize added lines and metadata
	truncatedDiff := contextpkg.TruncateDiff(diff, 1000) // Adjust max lines as needed
	return commitMsg, truncatedDiff, nil
}

// GenerateDraftPR uses the LLM's chat endpoint to generate a PR draft (stateless).
func GenerateDraftPR(commitMessage, diffs string) (string, error) {
	prompt := fmt.Sprintf(`
/contextualize: You are a developer, tasked to generate a detailed pull request (PR) description based on the following commit message and code changes.

**Commit Message:**
%s

**Code Changes:**
%s

!TASK: Provide a comprehensive PR title and description that explain the changes and adhere to documentation and GitHub best practices. Format the pull request in raw markdown with headers. Clearly separate the pull request and other components of the response with three backticks and append the draft PR in code blocks. Do not include line-by-line changes, limit any included snippets to 5 or less lines.
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

// GenerateWhatSummary generates a summary of git diffs using the LLM (stateless).
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

//------------------------------------------------------------------------------
// UTILITY FUNCTION: reads model/endpoint from environment
//------------------------------------------------------------------------------

func GetLLMConfig() (string, string) {
	endpoint := os.Getenv("PRBUDDY_LLM_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	m := contextpkg.GetActiveModel()
	if m == "" {
		m = os.Getenv("PRBUDDY_LLM_MODEL")
		if m == "" {
			m = "deepseek-r1:8b"
		}
	}
	return m, endpoint
}

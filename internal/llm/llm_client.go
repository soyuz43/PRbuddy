// internal/llm/llm_client.go

package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/soyuz43/prbuddy-go/internal/utils"
)

// Message represents a chat message for LLM interactions
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMResponse represents the response structure from the LLM
type LLMResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

var (
	quickAssistContext []Message
	contextMutex       sync.Mutex
)

// HandleCLIQuickAssist handles CLI requests (stateless)
func HandleCLIQuickAssist(input string) (string, error) {
	response, err := GetChatResponse([]Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: input},
	})
	if err != nil {
		return "", err
	}
	return response, nil
}

// HandleExtensionQuickAssist handles extension requests (stateful)
func HandleExtensionQuickAssist(input string) (string, error) {
	contextMutex.Lock()
	defer contextMutex.Unlock()

	quickAssistContext = append(quickAssistContext, Message{
		Role:    "user",
		Content: input,
	})

	response, err := GetChatResponse(quickAssistContext)
	if err != nil {
		return "", err
	}

	quickAssistContext = append(quickAssistContext, Message{
		Role:    "assistant",
		Content: response,
	})

	return response, nil
}

// GeneratePreDraftPR generates the pre-draft PR based on the latest commit
func GeneratePreDraftPR() (commitMessage string, diffs string, err error) {
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

// GenerateDraftPR uses the LLM's chat endpoint to generate a draft PR
func GenerateDraftPR(commitMessage, diffs string) (string, error) {
	prompt := fmt.Sprintf(`
You are an assistant designed to generate a detailed pull request (PR) description based on the following commit message and code changes.

**Commit Message:**
%s

**Code Changes:**
%s

Please provide a comprehensive PR title and description that explain the changes and adhere to documentation and GitHub best practices. Format the pull request in raw markdown with headers. Clearly separate the pull request and other components of the response with three backticks and append the draft PR in code blocks.
`, commitMessage, diffs)

	response, err := GetChatResponse([]Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: prompt},
	})
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

	return GetChatResponse([]Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: prompt},
	})
}

// GetChatResponse handles multi-turn conversations with the LLM
func GetChatResponse(messages []Message) (string, error) {
	model, endpoint := GetLLMConfig()

	requestBody := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"options": map[string]interface{}{
			"num_ctx": 8192,
		},
		"stream": false,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal request body")
	}

	resp, err := http.Post(endpoint+"/api/chat", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", errors.Wrap(err, "failed to send POST request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM responded with status code %d", resp.StatusCode)
	}

	var llmResp LLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&llmResp); err != nil {
		return "", errors.Wrap(err, "failed to decode response")
	}

	if llmResp.Message.Content == "" {
		return "", fmt.Errorf("empty response from LLM")
	}

	logrus.Info("Received response from LLM successfully.")
	return llmResp.Message.Content, nil
}

// GetLLMConfig gets current model configuration
func GetLLMConfig() (string, string) {
	model := os.Getenv("PRBUDDY_LLM_MODEL")
	endpoint := os.Getenv("PRBUDDY_LLM_ENDPOINT")

	if model == "" {
		model = "hermes3"
	}
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	return model, endpoint
}

// internal/llm/llm_client.go

package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// LLMResponse represents the response structure from the LLM
type LLMResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

// GeneratePreDraftPR generates the pre-draft PR based on the latest commit
func GeneratePreDraftPR() (commitMessage string, diffs string, err error) {
	// Get the latest commit message
	commitMsg, err := executeGitCommand("git", "log", "-1", "--pretty=%B")
	if err != nil {
		return "", "", errors.Wrap(err, "failed to get latest commit message")
	}

	// Get the diff for the latest commit
	diff, err := executeGitCommand("git", "diff", "HEAD~1", "HEAD")
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

	response, err := GetLLMResponse(prompt, "You are a helpful assistant.")
	if err != nil {
		return "", err
	}

	return response.Message.Content, nil
}

// GenerateSummary generates a summary of git diffs using the LLM
func GenerateSummary(gitDiffs string) (string, error) {
	// Prepare the prompt for the LLM
	prompt := fmt.Sprintf(`
These are the git diffs for the repository:

%s

---
!TASK::
1. Provide a meticulous natural language summary of each of the changes. Do so by file. Describe each change made in full.
2. List and separate changes for each file changed using numbered points and markdown formatting.
3. Only describe the changes explicitly present in the diffs. Do not infer, speculate, or invent additional content.
4. Focus on helping the developer reorient themselves and understand where they left off.
`, gitDiffs)

	// Call the LLM to generate the summary
	summary, err := GetLLMResponse(prompt, "You are a helpful assistant.")
	if err != nil {
		return "", errors.Wrap(err, "failed to generate summary from LLM")
	}

	return summary.Message.Content, nil
}

// GetLLMResponse interacts with Ollama's chat endpoint to get a response from the LLM
func GetLLMResponse(prompt, systemMessage string) (LLMResponse, error) {
	apiURL := "http://localhost:11434/api/chat" // Hardcoded as per your requirements
	modelName := "hermes3"                      // Hardcoded as per your requirements

	requestBody := map[string]interface{}{
		"model": modelName,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": systemMessage,
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"options": map[string]interface{}{
			"num_ctx": 8192,
		},
		"stream": false,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return LLMResponse{}, errors.Wrap(err, "failed to marshal request body")
	}

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return LLMResponse{}, errors.Wrap(err, "failed to send POST request to LLM")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return LLMResponse{}, fmt.Errorf("LLM responded with status code %d", resp.StatusCode)
	}

	var llmResp LLMResponse
	err = json.NewDecoder(resp.Body).Decode(&llmResp)
	if err != nil {
		return LLMResponse{}, errors.Wrap(err, "failed to decode LLM response")
	}

	if llmResp.Message.Content == "" {
		return LLMResponse{}, fmt.Errorf("empty response from LLM")
	}

	logrus.Info("Received response from LLM successfully.")
	return llmResp, nil
}

// executeGitCommand executes a git command and returns its output as a string
func executeGitCommand(args ...string) (string, error) {
	cmd := exec.Command(args[0], args[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", errors.Wrapf(err, "git command failed: %s", strings.Join(args, " "))
	}
	return strings.TrimSpace(out.String()), nil
}

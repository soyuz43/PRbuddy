// internal/database/database_client.go

package database

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	internalgithub "github.com/soyuz43/prbuddy-go/internal/github" // Alias internal github package
	"github.com/soyuz43/prbuddy-go/internal/llm"                   // Ensure llm is imported if used
)

// DatabaseClient encapsulates the SQLite database connection
type DatabaseClient struct {
	DB *sql.DB
}

// PullRequest represents a simplified pull request structure
type PullRequest struct {
	Number int
	Title  string
	Body   string
	State  string
	Merged bool
	// Add other necessary fields
}

// ConvertGitHubPRToDatabasePR converts an internal PullRequest to a database.PullRequest
func ConvertGitHubPRToDatabasePR(pr *internalgithub.PullRequest) PullRequest {
	return PullRequest{
		Number: pr.Number,
		Title:  pr.Title,
		Body:   pr.Body,
		State:  pr.State,
		Merged: pr.Merged,
		// Add other necessary field mappings if needed
	}
}

// NewDatabase initializes and returns a new DatabaseClient
func NewDatabase(dbPath string) (*DatabaseClient, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open database")
	}

	client := &DatabaseClient{DB: db}

	err = client.createTables()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create tables")
	}

	return client, nil
}

// createTables creates the necessary tables if they don't exist
func (c *DatabaseClient) createTables() error {
	createPRTable := `
	CREATE TABLE IF NOT EXISTS pull_requests (
		number INTEGER PRIMARY KEY,
		title TEXT,
		body TEXT,
		state TEXT,
		merged BOOLEAN
	);
	`

	_, err := c.DB.Exec(createPRTable)
	if err != nil {
		return errors.Wrap(err, "failed to create pull_requests table")
	}

	// Create comments table if needed
	createCommentsTable := `
	CREATE TABLE IF NOT EXISTS comments (
		id INTEGER PRIMARY KEY,
		pr_number INTEGER,
		user TEXT,
		body TEXT,
		created_at TEXT,
		updated_at TEXT,
		FOREIGN KEY(pr_number) REFERENCES pull_requests(number)
	);
	`

	_, err = c.DB.Exec(createCommentsTable)
	if err != nil {
		return errors.Wrap(err, "failed to create comments table")
	}

	return nil
}

// InsertPullRequest inserts a pull request into the database
func (c *DatabaseClient) InsertPullRequest(pr PullRequest) error {
	insertQuery := `
	INSERT OR IGNORE INTO pull_requests (number, title, body, state, merged)
	VALUES (?, ?, ?, ?, ?);
	`

	_, err := c.DB.Exec(insertQuery, pr.Number, pr.Title, pr.Body, pr.State, pr.Merged)
	if err != nil {
		return errors.Wrap(err, "failed to insert pull request")
	}

	return nil
}

// FetchPRDetails retrieves pull request details by their numbers
func (c *DatabaseClient) FetchPRDetails(prNumbers []int) ([]PullRequest, error) {
	if len(prNumbers) == 0 {
		return nil, nil
	}

	// Generate the required number of placeholders (?, ?, ...)
	placeholders := strings.TrimRight(strings.Repeat("?,", len(prNumbers)), ",")

	query := fmt.Sprintf(`
	SELECT number, title, body, state, merged
	FROM pull_requests
	WHERE number IN (%s);`, placeholders)

	rows, err := c.DB.Query(query, interfaceSlice(prNumbers)...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query pull_requests")
	}
	defer rows.Close()

	var prs []PullRequest
	for rows.Next() {
		var pr PullRequest
		err := rows.Scan(&pr.Number, &pr.Title, &pr.Body, &pr.State, &pr.Merged)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan pull request row")
		}
		prs = append(prs, pr)
	}

	return prs, nil
}

// GenerateSummary generates a summary of git diffs using the LLM
func (c *DatabaseClient) GenerateSummary(gitDiffs string) (string, error) {
	// Prepare the prompt for the LLM
	prompt := fmt.Sprintf(`
These are the git diffs for the repository:

%s

---
!TASK::
1. Provide a meticulous natural language summary of each of the changes. Do so by file. Describe each change made in full.
2. List and separate changes for each file changed using numbered points, and using markdown standards in formatting.
3. Only describe the changes explicitly present in the diffs. Do not infer, speculate, or invent additional content.
4. Focus on helping the developer reorient themselves and where they left off.
`, gitDiffs)

	// Call the LLM to generate the summary
	summary, err := llm.GenerateSummary(prompt)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate summary from LLM")
	}

	return summary, nil
}

// HasCommits checks if there are any commits in the repository
func HasCommits() (bool, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	err := cmd.Run()
	if err != nil {
		return false, nil // No commits found
	}
	return true, nil
}

// GetGitDiff retrieves the git diff based on provided arguments
func GetGitDiff(args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"diff"}, args...)...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", errors.Wrap(err, "failed to get git diff")
	}
	return out.String(), nil
}

// GetUntrackedFiles retrieves untracked files in the repository
func GetUntrackedFiles() (string, error) {
	cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", errors.Wrap(err, "failed to get untracked files")
	}
	return out.String(), nil
}

// interfaceSlice converts a slice of ints to a slice of interfaces
func interfaceSlice(slice []int) []interface{} {
	s := make([]interface{}, len(slice))
	for i, v := range slice {
		s[i] = v
	}
	return s
}

// DeleteDatabase deletes the SQLite database file
func DeleteDatabase(dbPath string) error {
	err := os.Remove(dbPath)
	if err != nil {
		return fmt.Errorf("failed to delete database: %w", err)
	}
	return nil
}

// Close closes the database connection
func (c *DatabaseClient) Close() error {
	return c.DB.Close()
}

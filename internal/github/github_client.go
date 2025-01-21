// internal/github/github_client.go

package github

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-github/v50/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

// PullRequest represents a simplified pull request structure
type PullRequest struct {
	Number int
	Title  string
	Body   string
	State  string
	Merged bool
	// Add other necessary fields
}

// GetRemoteURL fetches the GitHub remote URL from the git configuration
func GetRemoteURL() (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", errors.Wrap(err, "failed to get remote URL")
	}
	remoteURL := strings.TrimSpace(out.String())
	return remoteURL, nil
}

// FetchPullRequests retrieves pull requests from the specified GitHub repository
func FetchPullRequests(remoteURL string) ([]PullRequest, error) {
	ctx := context.Background()

	// Extract owner and repo from remote URL
	owner, repo, err := ParseRemoteURL(remoteURL)
	if err != nil {
		return nil, err
	}

	// Authenticate using GitHub token from environment variable
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, errors.New("GITHUB_TOKEN not set in environment variables")
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	// List pull requests
	opts := &github.PullRequestListOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}
	var allPRs []PullRequest

	for {
		prs, resp, err := client.PullRequests.List(ctx, owner, repo, opts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list pull requests")
		}

		for _, pr := range prs {
			allPRs = append(allPRs, PullRequest{
				Number: pr.GetNumber(),
				Title:  pr.GetTitle(),
				Body:   pr.GetBody(),
				State:  pr.GetState(),
				Merged: pr.GetMerged(),
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allPRs, nil
}

// ParseRemoteURL parses the GitHub remote URL to extract owner and repository name
func ParseRemoteURL(remoteURL string) (owner string, repo string, err error) {
	// Supports both HTTPS and SSH URLs
	if strings.HasPrefix(remoteURL, "git@") {
		// Example: git@github.com:owner/repo.git
		parts := strings.Split(remoteURL, ":")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid SSH remote URL format")
		}
		path := strings.TrimSuffix(parts[1], ".git")
		segments := strings.Split(path, "/")
		if len(segments) != 2 {
			return "", "", fmt.Errorf("invalid SSH remote URL path")
		}
		return segments[0], segments[1], nil
	} else if strings.HasPrefix(remoteURL, "https://") || strings.HasPrefix(remoteURL, "http://") {
		// Example: https://github.com/owner/repo.git
		urlWithoutProtocol := strings.TrimPrefix(remoteURL, "https://")
		urlWithoutProtocol = strings.TrimPrefix(urlWithoutProtocol, "http://")
		urlWithoutSuffix := strings.TrimSuffix(urlWithoutProtocol, ".git")
		segments := strings.Split(urlWithoutSuffix, "/")
		if len(segments) < 3 {
			return "", "", fmt.Errorf("invalid HTTPS remote URL format")
		}
		return segments[1], segments[2], nil
	} else {
		return "", "", fmt.Errorf("unsupported remote URL format")
	}
}

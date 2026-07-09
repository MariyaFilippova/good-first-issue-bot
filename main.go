package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Issue struct {
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	HTMLURL     string    `json:"html_url"`
	Labels      []Label   `json:"labels"`
	PullRequest *struct{} `json:"pull_request"`
}

type Label struct {
	Name string `json:"name"`
}

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		_, err := fmt.Fprintln(os.Stderr, "please set GITHUB_TOKEN")
		if err != nil {
			return
		}
		os.Exit(1)
	}

	repo := "Chevrotain/chevrotain"
	label := "good first issue"

	// A context carries deadlines/cancellation down the call stack. context.Background()
	// is the empty root; later we'll derive timeouts and shutdown signals from it.
	issues, err := fetchIssues(context.Background(), token, repo, label)
	if err != nil {
		_, err2 := fmt.Fprintln(os.Stderr, "error:", err)
		if err2 != nil {
			return
		}
		os.Exit(1)
	}

	count := 0
	for _, is := range issues {
		if is.PullRequest != nil {
			continue // skip PRs that snuck into the issues list
		}
		count++
		fmt.Printf("#%-6d %s\n         %s\n", is.Number, is.Title, is.HTMLURL)
	}
	fmt.Printf("\n%d open \"%s\" issues in %s\n", count, label, repo)
}

func fetchIssues(ctx context.Context, token, repo, label string) ([]Issue, error) {
	endpoint := "https://api.github.com/repos/" + repo + "/issues"

	q := url.Values{}
	q.Set("labels", label)
	q.Set("state", "open")
	q.Set("sort", "updated")
	q.Set("direction", "desc")
	q.Set("per_page", "100")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
	if err != nil {
		// %w wraps the underlying error so callers can inspect it later.
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github returned %s", resp.Status)
	}

	var issues []Issue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}
	return issues, nil
}

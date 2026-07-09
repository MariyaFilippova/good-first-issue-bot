package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"awesomeProject/internal/store"
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
		fmt.Fprintln(os.Stderr, "please set GITHUB_TOKEN")
		os.Exit(1)
	}

	// A context carries deadlines/cancellation down the call stack. context.Background()
	// is the empty root; later we'll derive timeouts and shutdown signals from it.
	ctx := context.Background()

	// Connect to Postgres up front so we fail immediately if the DB is unreachable.
	pool, err := store.ConnectDB(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "database error:", err)
		os.Exit(1)
	}
	defer pool.Close()
	fmt.Println("connected to Postgres ✓")

	// Wire the pool into our data-access layer. We name the variable `st` (not
	// `store`) so it doesn't shadow the imported package `store`.
	st := store.NewStore(pool)

	if err := st.AddRepo(ctx, "golang", "go"); err != nil {
		fmt.Fprintln(os.Stderr, "AddRepo failed:", err)
	}

	userID, err := st.AddUser(ctx, "maxa.spb6@gmail.com")
	if err != nil {
		fmt.Fprintln(os.Stderr, "AddUser failed:", err)
	}
	fmt.Println("added user:", userID)

	repo := "Chevrotain/chevrotain"
	label := "good first issue"

	issues, err := fetchIssues(ctx, token, repo, label)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
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

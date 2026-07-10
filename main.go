package main

import (
	"context"
	"fmt"
	"os"

	"awesomeProject/internal/github"
	"awesomeProject/internal/store"
)

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

	// Our two dependencies: the data-access layer and the GitHub client.
	st := store.NewStore(pool)
	gh := github.NewClient(token)

	if err := st.AddRepo(ctx, "golang", "go"); err != nil {
		fmt.Fprintln(os.Stderr, "AddRepo failed:", err)
	}
	if _, err := st.AddUser(ctx, "maxa.spb6@gmail.com"); err != nil {
		fmt.Fprintln(os.Stderr, "AddUser failed:", err)
	}

	owner, name, label := "Chevrotain", "chevrotain", "good first issue"
	issues, err := gh.FetchIssues(ctx, owner, name, label)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fetch issues:", err)
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
	fmt.Printf("\n%d open %q issues in %s/%s\n", count, label, owner, name)
}

package main

import (
	"context"
	"log"
	"os"
	"time"

	"awesomeProject/internal/auth"
	"awesomeProject/internal/github"
	"awesomeProject/internal/notify"
	"awesomeProject/internal/store"

	"github.com/joho/godotenv"
)

func main() {
	// Load a .env file into the environment if present. Ignore the error: in
	// production there's no .env — the vars are set by the platform instead.
	_ = godotenv.Load()

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("please set GITHUB_TOKEN")
	}

	ctx := context.Background()
	pool, err := store.ConnectDB(ctx)
	if err != nil {
		log.Fatal("database error: ", err)
	}
	defer pool.Close()

	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		log.Fatal("set GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET")
	}

	st := store.NewStore(pool)
	gh := github.NewClient(token)
	a := auth.New(st, clientID, clientSecret, "http://localhost:8080/auth/callback")
	mailer := notify.NewMailer(os.Getenv("RESEND_API_KEY"), os.Getenv("RESEND_FROM"))

	go pollLoop(ctx, st, gh, mailer)

	log.Println("server listening on :8080")
	if err := notify.Serve(st, a, gh); err != nil {
		log.Fatal("server: ", err)
	}
}

// pollLoop runs one poll pass every 30 seconds, forever.
func pollLoop(ctx context.Context, st *store.Store, gh *github.Client, mailer *notify.Mailer) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		poll(ctx, st, gh, mailer)
		<-ticker.C
	}
}

func poll(ctx context.Context, st *store.Store, gh *github.Client, mailer *notify.Mailer) {
	due, err := st.DueRepos(ctx, 100)
	if err != nil {
		log.Println("DueRepos:", err)
		return
	}

	for _, repo := range due {
		res, err := gh.FetchIssues(ctx, repo.Owner, repo.Name, "good first issue", repo.ETag)
		if err != nil {
			log.Println("FetchIssues:", err)
			continue
		}

		if res.NotModified {
			if err := st.UpdateRepoAfterPoll(ctx, repo.ID, nil); err != nil {
				log.Println("UpdateRepoAfterPoll:", err)
			}
			continue
		}

		if err := mailer.Notify(ctx, st, repo, res.Issues); err != nil {
			log.Println("Notify:", err)
		}

		var newETag *string
		if res.ETag != "" {
			newETag = &res.ETag
		}
		if err := st.UpdateRepoAfterPoll(ctx, repo.ID, newETag); err != nil {
			log.Println("UpdateRepoAfterPoll:", err)
		}
	}
}

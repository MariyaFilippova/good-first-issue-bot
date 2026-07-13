package notify

import (
	"context"
	"html/template"
	"log"
	"net/http"

	"awesomeProject/internal/auth"
	"awesomeProject/internal/github"
	"awesomeProject/internal/store"
)

var homeTmpl = template.Must(template.New("home").Parse(homeHTML))

type pageData struct {
	LoggedIn bool
	Repos    []store.Repo
}

func Serve(s *store.Store, a *auth.Auth, gh *github.Client) error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		var data pageData
		if userID, ok := a.CurrentUser(r); ok {
			data.LoggedIn = true
			repos, err := s.ListSubscribedRepos(r.Context(), userID)
			if err != nil {
				log.Println("ListSubscribedRepos:", err)
				http.Error(w, "server error", http.StatusInternalServerError)
				return
			}
			data.Repos = repos
		}
		if err := homeTmpl.Execute(w, data); err != nil {
			log.Println("template:", err)
		}
	})

	mux.HandleFunc("GET /auth/login", a.HandleLogin)
	mux.HandleFunc("GET /auth/callback", a.HandleCallback)
	mux.HandleFunc("GET /logout", a.HandleLogout)

	mux.HandleFunc("POST /subscribe", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := a.CurrentUser(r)
		if !ok {
			http.Error(w, "login required", http.StatusUnauthorized)
			return
		}
		owner, name := r.FormValue("owner"), r.FormValue("name")
		if owner == "" || name == "" {
			http.Error(w, "owner and repo are required", http.StatusBadRequest)
			return
		}
		if err := s.Subscribe(r.Context(), userID, owner, name); err != nil {
			log.Println("Subscribe:", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		// Mark the repo's CURRENT issues as already-seen for this user, so they
		// only get issues that appear AFTER subscribing (no backlog flood).
		// Best-effort: a failure just means they might get the backlog once.
		if err := seedNotified(r.Context(), gh, s, userID, owner, name); err != nil {
			log.Println("seedNotified:", err)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	mux.HandleFunc("POST /unsubscribe", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := a.CurrentUser(r)
		if !ok {
			http.Error(w, "login required", http.StatusUnauthorized)
			return
		}
		if err := s.Unsubscribe(r.Context(), userID, r.FormValue("owner"), r.FormValue("name")); err != nil {
			log.Println("Unsubscribe:", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	return http.ListenAndServe(":8080", mux)
}

func seedNotified(ctx context.Context, gh *github.Client, s *store.Store, userID int64, owner, name string) error {
	res, err := gh.FetchIssues(ctx, owner, name, "good first issue", nil)
	if err != nil {
		return err
	}
	repoID, err := s.GetRepo(ctx, owner, name)
	if err != nil {
		return err
	}
	for _, issue := range res.Issues {
		if issue.PullRequest != nil {
			continue
		}
		if _, err := s.MarkNotified(ctx, userID, repoID, issue.Id); err != nil {
			return err
		}
	}
	return nil
}

const homeHTML = `<!doctype html>
<html>
<head><meta charset="utf-8"><title>Good First Issue Bot</title></head>
<body style="font-family: system-ui, sans-serif; max-width: 640px; margin: 48px auto; line-height: 1.5;">
  <h1>🔔 Good First Issue Bot</h1>
  {{if .LoggedIn}}
    <p>You're logged in. <a href="/logout">Log out</a></p>

    <h2>Subscribe to a repo</h2>
    <form method="POST" action="/subscribe">
      <input name="owner" placeholder="owner (e.g. golang)" required>
      <input name="name" placeholder="repo (e.g. go)" required>
      <button type="submit">Subscribe</button>
    </form>

    <h2>Your subscriptions</h2>
    <ul>
      {{range .Repos}}
        <li>
          {{.Owner}}/{{.Name}}
          <form method="POST" action="/unsubscribe" style="display:inline">
            <input type="hidden" name="owner" value="{{.Owner}}">
            <input type="hidden" name="name" value="{{.Name}}">
            <button type="submit">unsubscribe</button>
          </form>
        </li>
      {{else}}
        <li>No subscriptions yet.</li>
      {{end}}
    </ul>
  {{else}}
    <p>Get notified about new “good first issue”s in the repos you follow.</p>
    <p><a href="/auth/login"><button>Login with GitHub</button></a></p>
  {{end}}
</body>
</html>`

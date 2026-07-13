package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"awesomeProject/internal/auth"
	"awesomeProject/internal/github"
	"awesomeProject/internal/store"
)

func Serve(s *store.Store, a *auth.Auth, gh *github.Client) error {
	mux := http.NewServeMux()

	// GitHub OAuth flow.
	mux.HandleFunc("GET /auth/login", a.HandleLogin)
	mux.HandleFunc("GET /auth/callback", a.HandleCallback)
	mux.HandleFunc("GET /logout", a.HandleLogout)

	mux.HandleFunc("/api/status", withCORS(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := a.CurrentUser(r)
		if !ok {
			http.Error(w, "login required", http.StatusUnauthorized)
			return
		}
		owner, name := r.URL.Query().Get("owner"), r.URL.Query().Get("name")
		subscribed, err := s.IsSubscribed(r.Context(), userID, owner, name)
		if err != nil {
			log.Println("IsSubscribed:", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]bool{"subscribed": subscribed})
	}))

	mux.HandleFunc("/api/subscribe", withCORS(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := a.CurrentUser(r)
		if !ok {
			http.Error(w, "login required", http.StatusUnauthorized)
			return
		}
		owner, name, err := decodeRepo(r)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := s.Subscribe(r.Context(), userID, owner, name); err != nil {
			log.Println("Subscribe:", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		// Seed so the user only gets FUTURE issues, not the current backlog.
		if err := seedNotified(r.Context(), gh, s, userID, owner, name); err != nil {
			log.Println("seedNotified:", err)
		}
		writeJSON(w, map[string]bool{"subscribed": true})
	}))

	mux.HandleFunc("/api/unsubscribe", withCORS(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := a.CurrentUser(r)
		if !ok {
			http.Error(w, "login required", http.StatusUnauthorized)
			return
		}
		owner, name, err := decodeRepo(r)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := s.Unsubscribe(r.Context(), userID, owner, name); err != nil {
			log.Println("Unsubscribe:", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]bool{"subscribed": false})
	}))

	return http.ListenAndServe(":8080", mux)
}

// decodeRepo reads {"owner":"...","name":"..."} from a JSON request body.
func decodeRepo(r *http.Request) (string, string, error) {
	var body struct {
		Owner string `json:"owner"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return "", "", err
	}
	if body.Owner == "" || body.Name == "" {
		return "", "", fmt.Errorf("owner and name required")
	}
	return body.Owner, body.Name, nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// withCORS reflects the request Origin and allows credentials, so the extension
// can call these endpoints with cookies. It also answers CORS preflight (OPTIONS).
func withCORS(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h(w, r)
	}
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

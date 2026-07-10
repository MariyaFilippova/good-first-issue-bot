package notify

import (
	"awesomeProject/internal/store"
	"context"
	"encoding/json"
	"net/http"
)

func Serve(ct context.Context, s *store.Store) error {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /subscribe", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email string `json:"email"`
			Owner string `json:"owner"`
			Name  string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		err := s.Subscribe(ct, req.Email, req.Owner, req.Name)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
		}
	})

	mux.HandleFunc("POST /repo", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Owner string `json:"owner"`
			Name  string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := s.AddRepo(r.Context(), req.Owner, req.Name); err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("POST /unsubscribe", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email string `json:"email"`
			Owner string `json:"owner"`
			Name  string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := s.Unsubscribe(r.Context(), req.Email, req.Owner, req.Name); err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
	})

	mux.HandleFunc("GET /subscriptions/{email}", func(w http.ResponseWriter, r *http.Request) {
		userID, err := s.GetUser(r.Context(), r.PathValue("email"))
		if err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		subs, err := s.ListSubscriptions(r.Context(), userID)
		if err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(subs)
	})

	return http.ListenAndServe(":8080", mux)
}

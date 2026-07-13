package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"awesomeProject/internal/store"

	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
)

type Auth struct {
	store    *store.Store
	oauth    *oauth2.Config
	sessions *SessionStore
}

func New(st *store.Store, clientID, clientSecret, redirectURL string) *Auth {
	return &Auth{
		store:    st,
		sessions: NewSessionStore(),
		oauth: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"read:user", "user:email"},
			Endpoint:     githuboauth.Endpoint,
		},
	}
}

func (a *Auth) HandleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := randomToken()
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   300,
	})
	http.Redirect(w, r, a.oauth.AuthCodeURL(state), http.StatusTemporaryRedirect)
}

func (a *Auth) HandleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || r.URL.Query().Get("state") != stateCookie.Value {
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}

	token, err := a.oauth.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "token exchange failed", http.StatusBadGateway)
		return
	}

	ghUser, err := fetchGitHubUser(r.Context(), a.oauth.Client(r.Context(), token))
	if err != nil {
		http.Error(w, "failed to fetch github user", http.StatusBadGateway)
		return
	}
	if ghUser.Email == "" {
		http.Error(w, "no verified email on your github account", http.StatusBadRequest)
		return
	}

	userID, err := a.store.UpsertGitHubUser(r.Context(), ghUser.ID, ghUser.Email)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	sid, err := a.sessions.Create(userID)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sid,
		Path:     "/",
		HttpOnly: true,
		// SameSite=None + Secure lets the browser extension send this cookie on
		// its cross-origin API calls. (Chrome allows Secure cookies on localhost.)
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		MaxAge:   86400,
	})
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<p>✅ Logged in. You can close this tab and use the browser extension.</p>`)
}

func (a *Auth) CurrentUser(r *http.Request) (int64, bool) {
	c, err := r.Cookie("session")
	if err != nil {
		return 0, false
	}
	return a.sessions.User(c.Value)
}

func (a *Auth) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1, // delete now
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

type githubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Email string `json:"email"`
}

func fetchGitHubUser(ctx context.Context, client *http.Client) (githubUser, error) {
	var u githubUser
	if err := getJSON(ctx, client, "https://api.github.com/user", &u); err != nil {
		return u, err
	}
	if u.Email == "" {
		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}
		if err := getJSON(ctx, client, "https://api.github.com/user/emails", &emails); err != nil {
			return u, err
		}
		for _, e := range emails {
			if e.Primary && e.Verified {
				u.Email = e.Email
				break
			}
		}
	}
	return u, nil
}

func getJSON(ctx context.Context, client *http.Client, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github %s: %s", url, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

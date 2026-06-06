package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"auth/session"

	"golang.org/x/oauth2"
)

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	state, err := s.oauth.GenerateState()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		Expires:  time.Now().Add(10 * time.Minute),
		HttpOnly: true,
	})

	url := s.oauth.AuthURL(state)
	http.Redirect(w, r, url, http.StatusFound)
}

func (s *Server) handleProtectedRoute(w http.ResponseWriter, r *http.Request) {
	session, ok := r.Context().Value(session.SessionKeyType("session")).(map[string]any)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, ok := session["user_id"].(string)
	if !ok {
		http.Error(w, "invalid session data", http.StatusUnauthorized)
		return
	}

	var (
		accessToken  string
		refreshToken string
		tokenExpiry  time.Time
	)

	row := s.db.QueryRowContext(r.Context(), `
			SELECT access_token, refresh_token, token_expiry FROM user_tokens
			WHERE user_id = $1`, userID)
	err := row.Scan(&accessToken, &refreshToken, &tokenExpiry)
	if err != nil {
		if err != sql.ErrNoRows {
			http.Error(w, "token not found", http.StatusInternalServerError)
			return
		}
		http.Error(w, "db read error", http.StatusInternalServerError)
		return
	}

	if time.Until(tokenExpiry) < 5*time.Minute {
		tokenSource := s.oauthCfg.TokenSource(r.Context(), &oauth2.Token{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			Expiry:       tokenExpiry,
		})

		newToken, err := tokenSource.Token()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		accessToken = newToken.AccessToken
		refreshToken = newToken.RefreshToken
		tokenExpiry = newToken.Expiry

		_, err = s.db.ExecContext(r.Context(), `
			UPDATE user_tokens
			SET
				access_token = $1,
				refresh_token = $2,
				token_expiry = $3,
				updated_at = CURRENT_TIMESTAMP
			WHERE
				user_id = $4;
		`, accessToken, refreshToken, tokenExpiry, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	http.ServeFile(w, r, "profile.html")
}

func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "missing state cookei", http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	})

	if stateCookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	token, err := s.oauth.ExchangeCode(r.Context(), code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := s.oauthCfg.Client(r.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Error(w, err.Error(), resp.StatusCode)
		return
	}
	defer resp.Body.Close()

	data := make(map[string]any)
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Println(data)

	userID := data["id"]
	email := data["email"]
	name := data["name"]

	_, err = s.db.ExecContext(r.Context(), `
			INSERT INTO users(id, email, name, provider)
			VALUES($1, $2, $3, $4)
			ON CONFLICT(id) DO UPDATE SET
				email = excluded.email,
				name = excluded.name,
				updated_at = CURRENT_TIMESTAMP;
		`, userID.(string), email, name, "google")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = s.db.ExecContext(r.Context(), `
			INSERT INTO user_tokens(user_id, access_token, refresh_token, token_expiry)
			VALUES($1, $2, $3, $4)
			ON CONFLICT(user_id) DO UPDATE SET
				access_token = excluded.access_token,
				refresh_token = COALESCE(excluded.refresh_token, user_tokens.refresh_token),
				token_expiry = excluded.token_expiry,
				updated_at = CURRENT_TIMESTAMP;
		`, userID.(string), token.AccessToken, token.RefreshToken, token.Expiry)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = s.sm.CreateSession(w, r, userID.(string))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Println("session created")

	http.Redirect(w, r, "/protected", http.StatusSeeOther)
}

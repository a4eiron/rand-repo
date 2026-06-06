// Package session implements session based auth services
package session

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"auth/session/store"
)

type SessionManager struct {
	config SessionConfig
	store  store.SessionStore
}

type SessionConfig struct {
	Path         string
	CookieName   string
	CookieDomain string
	CookieSecure bool
	MaxAge       int
	Expires      time.Duration
}

func NewSessionManager(store store.SessionStore, config SessionConfig) *SessionManager {
	return &SessionManager{
		store:  store,
		config: config,
	}
}

func (sm *SessionManager) CreateSession(w http.ResponseWriter, r *http.Request, userID string) error {
	sessionID, err := generateSessionID()
	if err != nil {
		return errors.New("failed to generate secure sessionID")
	}
	data := map[string]any{"user_id": userID}

	err = sm.store.Create(r.Context(), sessionID, data, sm.config.Expires)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sm.config.CookieName,
		Value:    sessionID,
		Path:     sm.config.Path,
		Domain:   sm.config.CookieDomain,
		Expires:  time.Now().Add(sm.config.Expires),
		MaxAge:   sm.config.MaxAge,
		Secure:   sm.config.CookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (sm *SessionManager) GetSession(r *http.Request) (map[string]any, error) {
	cookie, err := r.Cookie(sm.config.CookieName)
	if err != nil {
		return nil, err
	}
	return sm.store.Get(r.Context(), cookie.Value)
}

func (sm *SessionManager) DeleteSession(w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie(sm.config.CookieName)
	if err != nil {
		return err
	}

	err = sm.store.Delete(r.Context(), cookie.Value)
	if err != nil {
		return nil
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sm.config.CookieName,
		Value:    "",
		Path:     "/",
		Domain:   sm.config.CookieDomain,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

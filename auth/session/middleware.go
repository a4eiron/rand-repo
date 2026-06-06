package session

import (
	"context"
	"net/http"
)

type SessionKeyType string

func (sm *SessionManager) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := sm.GetSession(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), SessionKeyType("session"), session)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

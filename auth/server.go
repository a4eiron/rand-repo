package main

import (
	"database/sql"
	"net/http"

	"auth/config"
	internalOAuth "auth/oauth2"
	"auth/session"

	"golang.org/x/oauth2"
)

type Server struct {
	// not able to find other place to put it
	// ??? no no no
	oauth    *internalOAuth.OAuth2Service
	oauthCfg *oauth2.Config

	cfg config.Config

	db  *sql.DB
	sm  *session.SessionManager
	mux *http.ServeMux
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) Routes() {
	s.mux.HandleFunc("GET /", s.handleIndex)
	s.mux.HandleFunc("GET /api/auth/login/google", s.loginHandler)
	s.mux.HandleFunc("GET /api/auth/callback/google", s.handleCallback)
	s.mux.HandleFunc("GET /protected", s.sm.AuthMiddleware(s.handleProtectedRoute))
}

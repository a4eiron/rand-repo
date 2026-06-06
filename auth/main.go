package main

import (
	"log"
	"net/http"

	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"auth/config"
	internalOAuth "auth/oauth2"
	"auth/session"
	"auth/session/store"
)

func main() {
	log.SetFlags(log.Lshortfile)

	cfg := config.Load()

	db, err := NewDatabase(cfg.DBPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	if err := CreateTables(db); err != nil {
		log.Fatalln(err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Protocol: 2,
	})

	sessionStore := store.NewRedisStore(redisClient)
	sm := session.NewSessionManager(
		sessionStore, session.SessionConfig{
			Path:         "/",
			CookieName:   cfg.Session.CookieName,
			CookieDomain: cfg.Session.CookieDomain,
			CookieSecure: cfg.Session.CookieSecure,
			MaxAge:       cfg.Session.MaxAge,
			Expires:      cfg.Session.Expires,
		})

	oauthCfg := &oauth2.Config{
		RedirectURL:  cfg.Google.RedirectURL,
		ClientID:     cfg.Google.ClientID,
		ClientSecret: cfg.Google.ClientSecret,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	oauthService := internalOAuth.NewOAuth2Service(oauthCfg)

	svr := &Server{
		cfg:      cfg,
		oauth:    oauthService,
		oauthCfg: oauthCfg,

		db:  db,
		sm:  sm,
		mux: http.NewServeMux(),
	}
	svr.Routes()

	if err := http.ListenAndServe(cfg.Port, svr); err != nil {
		log.Fatalln(err)
	}
}

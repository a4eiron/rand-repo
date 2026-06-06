package config

import (
	"os"
	"time"

	"auth/session"
)

type Config struct {
	Env       string
	Port      string
	DBPath    string
	RedisAddr string
	Session   session.SessionConfig
	Google    GoogleConfig
}

type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

func Load() Config {
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}
	return Config{
		Env:       env,
		Port:      getEnv("PORT", ":4000"),
		DBPath:    getEnv("DATABASE_PATH", "./app.db"),
		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
		Session: session.SessionConfig{
			CookieName:   "app",
			CookieDomain: "localhost",
			MaxAge:       15 * 60,
			Expires:      15 * time.Minute,
			CookieSecure: env == "production",
		},
		Google: GoogleConfig{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:4000/api/auth/callback/google"),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

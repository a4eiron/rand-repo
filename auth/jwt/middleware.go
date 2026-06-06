package jwt

import (
	"context"
	"net/http"
	"strings"
)

type ClaimsKeyType string

func (s *JWTService) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header missing", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := s.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsKeyType("claims"), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

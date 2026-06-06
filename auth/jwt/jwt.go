// Package jwt implements JWT auth service
package jwt

import (
	"errors"
	"maps"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

type JWTConfig struct {
	SecretKey      string
	ExpiryDuration time.Duration
	Issuer         string
}

type JWTService struct {
	config JWTConfig
}

func NewJWTService(cfg JWTConfig) *JWTService {
	return &JWTService{config: cfg}
}

func (s *JWTService) GenerateToken(sub string, claims map[string]any) (string, error) {
	now := time.Now().UTC()

	mc := jwt.MapClaims{}
	maps.Copy(mc, claims)

	mc["sub"] = sub
	mc["iat"] = now.Unix()
	mc["exp"] = now.Add(s.config.ExpiryDuration).Unix()
	mc["iss"] = s.config.Issuer

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, mc)
	return token.SignedString([]byte(s.config.SecretKey))
}

func (s *JWTService) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.config.SecretKey), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

package oauth2

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/oauth2"
)

type OAuth2Service struct {
	config *oauth2.Config
}

func NewOAuth2Service(config *oauth2.Config) *OAuth2Service {
	return &OAuth2Service{config: config}
}

func (s *OAuth2Service) AuthURL(state string) string {
	return s.config.AuthCodeURL(state)
}

func (s *OAuth2Service) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return s.config.Exchange(ctx, code)
}

func (s *OAuth2Service) GenerateState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

package store

import (
	"context"
	"time"
)

type SessionStore interface {
	Create(ctx context.Context, sessionID string, data map[string]any, expiry time.Duration) error
	Get(ctx context.Context, sessionID string) (map[string]any, error)
	Delete(ctx context.Context, sessionID string) error
}

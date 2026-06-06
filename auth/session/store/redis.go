package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

func (s *RedisStore) Create(ctx context.Context, sessionID string, data map[string]any, expiry time.Duration) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, sessionID, jsonData, expiry).Err()
}

func (s *RedisStore) Get(ctx context.Context, sessionID string) (map[string]any, error) {
	val, err := s.client.Get(ctx, sessionID).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errors.New("session not found")
		}
		return nil, err
	}

	data := make(map[string]any)
	err = json.Unmarshal(val, &data)
	return data, err
}

func (s *RedisStore) Delete(ctx context.Context, sessionID string) error {
	return s.client.Del(ctx, sessionID).Err()
}

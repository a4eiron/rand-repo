package main

import (
	"context"

	"github.com/redis/go-redis/v9"
)

func NewRedisStore() (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	_, err := client.Ping(context.Background()).Result()
	return client, err
}

package main

import "time"

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type Post struct {
	ID        int64     `redis:"id" json:"id"`
	UserID    int64     `redis:"user_id" json:"user_id"`
	Content   string    `redis:"content" json:"content"`
	LikeCount int64     `redis:"like_count" json:"like_count"`
	CreatedAt time.Time `redis:"created_at" json:"created_at"`
}

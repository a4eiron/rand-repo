package main

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "nv"
	password = "postgres"
	dbname   = "feed_db"
)

func NewDatabase() (*sql.DB, error) {
	pgInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", pgInfo)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func createTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users(
			id BIGSERIAL PRIMARY KEY,
			username TEXT UNIQUE NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS follows(
			follower_id BIGINT REFERENCES users(id),
			followee_id BIGINT REFERENCES users(id),
			PRIMARY KEY(follower_id, followee_id)
		)`,
		`CREATE TABLE IF NOT EXISTS posts(
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT REFERENCES users(id),
			content TEXT NOT NULL,
			like_count BIGINT DEFAULT 0,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func InsertUser(db *sql.DB, username string) (int64, error) {
	var id int64
	err := db.QueryRow(
		`INSERT INTO users(username) 
		VALUES($1) RETURNING id`, username).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func InsertPost(db *sql.DB, userID int64, content string) (int64, error) {
	var id int64
	err := db.QueryRow(
		`INSERT INTO posts(user_id, content)
		VALUES($1, $2) RETURNING id`, userID, content).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func QueryUserFeed(db *sql.DB, userID int64) ([]Post, error) {
	rows, err := db.Query(
		`SELECT p.id, p.user_id, p.content, p.like_count, p.created_at
		FROM posts p
		JOIN follows f ON f.followee_id = p.user_id
		WHERE f.follower_id = $1
		ORDER BY created_at DESC
		LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}

	posts := make([]Post, 0)

	for rows.Next() {
		var post Post
		rows.Scan(&post.ID, &post.UserID, &post.Content, &post.LikeCount, &post.CreatedAt)
		posts = append(posts, post)
	}

	return posts, rows.Err()
}

func AddFollower(db *sql.DB, followeeID, followerID int64) error {
	_, err := db.Exec(
		`INSERT INTO follows(followee_id, follower_id)
		VALUES($1, $2)`, followeeID, followerID)
	if err != nil {
		return err
	}
	return nil
}

func UpdatePostLikeCount(db *sql.DB, postID, count int64) error {
	_, err := db.Exec(`
		UPDATE posts
		SET like_count = like_count + $1
		WHERE id = $2`, count, postID)
	if err != nil {
		return err
	}
	return nil
}

func GetFollowers(db *sql.DB, userID int64) ([]int64, error) {
	rows, err := db.Query(`SELECT follower_id FROM follows WHERE followee_id = $1`, userID)
	if err != nil {
		return nil, err
	}

	followerIDs := make([]int64, 0)
	for rows.Next() {
		var followerID int64
		rows.Scan(&followerID)
		followerIDs = append(followerIDs, followerID)
	}

	return followerIDs, rows.Err()
}

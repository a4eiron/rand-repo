package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type App struct {
	db  *sql.DB
	rdb *redis.Client
}

func NewApp(db *sql.DB, rdb *redis.Client) *App {
	return &App{
		db:  db,
		rdb: rdb,
	}
}

func (a *App) InsertUser(username string) (int64, error) {
	return InsertUser(a.db, username)
}

func (a *App) InsertPost(userID int64, content string) (int64, error) {
	id, err := InsertPost(a.db, userID, content)
	if err != nil {
		return 0, err
	}

	// invalidate all followers feed
	followerIDs, err := GetFollowers(a.db, userID)
	if err != nil {
		log.Println("failed to get followers:", err)
		return id, err
	}

	for _, followerID := range followerIDs {
		key := fmt.Sprintf("user:feed:%d", followerID)
		_, err := a.rdb.Del(context.Background(), key).Result()
		if err != nil {
			log.Println("failed to invalidate user feed; userID:", followerID, err)
		}
	}

	return id, err
}

func (a *App) QueryUserFeed(userID int64) ([]Post, error) {
	ctx := context.Background()
	userIDStr := fmt.Sprintf("user:feed:%d", userID)

	// check cache first
	postIDs, err := a.rdb.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key: userIDStr, Start: 0, Stop: -1,
	}).Result()
	if err == nil && len(postIDs) > 0 {
		// cache hit
		var cachedPosts []Post

		for _, postIDStr := range postIDs {
			var post Post
			err := a.rdb.HGetAll(ctx, postIDStr).Scan(&post)
			if err != nil {
				log.Println("failed to scan post:", err)
				continue
			}

			cachedPosts = append(cachedPosts, post)
		}

		if len(cachedPosts) > 0 {
			log.Println("CACHE HIT")
			return cachedPosts, nil
		}
	}

	// cache miss??
	// hit the db
	log.Println("CACHE MISS")
	posts, err := QueryUserFeed(a.db, userID)
	if err != nil {
		log.Println(err)
		return nil, errors.New("db error")
	}

	// populate the cache
	for _, post := range posts {
		postIDStr := fmt.Sprintf("post:%d", post.ID)
		a.rdb.ZAdd(ctx, userIDStr, redis.Z{
			Score:  float64(post.CreatedAt.Unix()),
			Member: postIDStr,
		})

		_, err := a.rdb.HSet(ctx, postIDStr, post).Result()
		if err != nil {
			log.Println("faild to cache post hash:", err)
		}
	}

	return posts, nil
}

func (a *App) AddFollower(followeeID, followerID int64) error {
	return AddFollower(a.db, followeeID, followerID)
}

func (a *App) AddLike(postID, userID int64) error {
	ctx := context.Background()
	counterKey := fmt.Sprintf("likes:%d", postID)
	dirtyKey := "likes:dirty"

	pipe := a.rdb.Pipeline()
	pipe.Incr(ctx, counterKey)
	pipe.SAdd(ctx, dirtyKey, postID)
	_, err := pipe.Exec(ctx)
	return err
}

func (a *App) LikeFlusher(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.flushLikes(ctx); err != nil {
				log.Println(err)
			}
		}
	}
}

func (a *App) flushLikes(ctx context.Context) error {
	postIDs, err := a.rdb.SPopN(ctx, "likes:dirty", 100).Result()
	if err != nil || len(postIDs) == 0 {
		return err
	}

	for _, postIDStr := range postIDs {
		counterKey := fmt.Sprintf("likes:%s", postIDStr)

		val, err := a.rdb.GetDel(ctx, counterKey).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}

			a.rdb.SAdd(ctx, "likes:dirty", postIDStr)
			continue
		}

		count, err := strconv.ParseInt(val, 10, 64)
		if err != nil || count == 0 {
			continue
		}

		postID, err := strconv.ParseInt(postIDStr, 10, 64)
		if err != nil {
			continue
		}

		err = UpdatePostLikeCount(a.db, postID, count)
		if err != nil {
			log.Println(err)
			a.rdb.IncrBy(ctx, counterKey, count)
			a.rdb.SAdd(ctx, "likes:dirty", postID)
		}
	}

	return nil
}

func main() {
	log.SetFlags(log.Lshortfile)

	db, err := NewDatabase()
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()
	log.Println("db up")

	rdb, err := NewRedisStore()
	if err != nil {
		log.Fatalln(err)
	}
	defer rdb.Close()
	log.Println("redis up")

	if err := createTables(db); err != nil {
		log.Fatalln("create tables:", err)
	}

	app := NewApp(db, rdb)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go app.LikeFlusher(ctx)

	//////////////////////////////////////////////////////////////////
	// user2ID, err := app.InsertUser("user-2")
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	// for i := range 500000 {
	// 	_, err := app.InsertPost(1, fmt.Sprintf("user-2 post-%d", i))
	// 	if err != nil {
	// 		log.Fatalln("insert post:", err)
	// 	}
	// }
	//
	// user1ID, err := app.InsertUser("user-1")
	// if err != nil {
	// 	log.Fatalln("insrt user:", err)
	// }
	//
	// _, err = app.InsertPost(user1ID, "user-1 post-1")
	// if err != nil {
	// 	log.Fatalln("insrt post:", err)
	// }
	//
	// if err := app.AddFollower(user2ID, user1ID); err != nil {
	// 	log.Fatal("add follower:", err)
	// }
	//

	posts, err := app.QueryUserFeed(2)
	if err != nil {
		log.Fatalln("user feed query:", err)
	}

	for _, post := range posts {
		fmt.Println(post)
	}
}

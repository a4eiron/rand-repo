package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	mathRand "math/rand/v2"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const LOCKEY = "lock:key"

var unlockScript = redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1]
		then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`)

func main() {
	log.SetFlags(log.Lshortfile)

	nodes := getNodes([]string{
		"localhost:6370",
		"localhost:6371",
		"localhost:6372",
		"localhost:6373",
		"localhost:6374",
	})

	var wg sync.WaitGroup
	workers := []string{"worker-A", "worker-B", "worker-C"}

	for _, worker := range workers {
		wg.Go(func() {
			ttl := 10 * time.Second

			token, validFor, acquired := acquireLockWithRetry(worker, nodes, ttl, 4)
			if !acquired {
				log.Println(worker, "failed to acquire lock")
				return
			}

			if validFor <= 0 {
				log.Println(worker, "acquired lock", "valid for:", validFor, "releasing..")
				releaseLock(nodes, token)
				return
			}

			log.Println(worker, "acquired lock", "valid for:", validFor)
			time.Sleep(min(validFor, 4*time.Second))
			releaseLock(nodes, token)
		})
	}

	wg.Wait()
	log.Println("All workers finished")
}

func acquireNode(
	ctx context.Context,
	rdb *redis.Client,
	key, value string,
	ttl time.Duration,
) bool {
	ok, err := rdb.SetNX(ctx, key, value, ttl).Result()
	return err == nil && ok
}

func acquireLockWithRetry(worker string, nodes [5]*redis.Client, ttl time.Duration, maxRetries int) (string, time.Duration, bool) {
	for range maxRetries {
		token := getRandomToken()

		acquired, validFor := acquireLock(worker, nodes, token, ttl)
		if acquired {
			return token, validFor, true
		}

		time.Sleep(time.Duration(mathRand.IntN(200)+100) * time.Millisecond)
	}
	return "", 0, false
}

func acquireLock(worker string, nodes [5]*redis.Client, token string, ttl time.Duration) (bool, time.Duration) {
	start := time.Now()
	acqCount := 0

	for i, node := range nodes {
		log.Println(worker, "trying to acquire lock...", "[node]", i)
		func() {
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			if acquireNode(ctx, node, LOCKEY, token, ttl) {
				acqCount++
			}
		}()
	}

	elapsed := time.Since(start)
	acquired := acqCount >= len(nodes)/2+1
	validFor := ttl - elapsed - 5*time.Millisecond // clock drift - 5 ms

	if acquired && validFor > 0 {
		return true, validFor
	}

	releaseLock(nodes, token)
	return false, 0
}

func releaseLock(nodes [5]*redis.Client, token string) {
	for _, node := range nodes {
		func() {
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			_, err := unlockScript.Run(ctx, node, []string{LOCKEY}, token).Result()
			if err != nil {
				log.Println(err)
			}
		}()
	}
}

func getNodes(addrs []string) [5]*redis.Client {
	var nodes [5]*redis.Client
	for i := range 5 {
		client := redis.NewClient(&redis.Options{
			Addr:        addrs[i],
			MaxRetries:  -1,
			DialTimeout: 20 * time.Millisecond,
		})
		nodes[i] = client
	}

	return nodes
}

func getRandomToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		log.Fatalln(err)
	}
	return hex.EncodeToString(b)
}

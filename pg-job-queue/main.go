package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "nv"
	password = "postgres"
	dbname   = "queuedb"
)

type Job struct {
	EstimatedTime time.Duration `json:"estimated_time"`
}

func main() {
	log.SetFlags(log.Lshortfile)

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		panic(err)
	}
	fmt.Println("connected")

	////////////////////////////////////////////////

	var wg sync.WaitGroup

	if err := createTable(db); err != nil {
		panic(err)
	}

	wg.Go(func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			payload, err := json.Marshal(Job{EstimatedTime: time.Duration(rand.IntN(10)+2) * time.Second})
			if err != nil {
				log.Println(err)
				continue
			}
			id, err := enque(db, payload)
			if err != nil {
				continue
			}
			log.Println("Job enqued:", id)
		}
	})

	for i := range 5 {
		wg.Go(func() {
			for {
				processed, err := dequeue(db, i)
				if err != nil {
					log.Printf("workder %d error: %w", i, err)
					continue
				}
				if !processed {
					time.Sleep(1 * time.Second)
				}
			}
		})
	}

	wg.Wait()
}

func dequeue(db *sql.DB, workerID int) (bool, error) {
	row := db.QueryRow(`
		UPDATE jobs
		SET
			status = 'running',
			attempts = attempts + 1,
			started_at = NOW(),
			updated_at = NOW(),
			run_at = NOW() + INTERVAL '30 seconds'
		WHERE id = (
			SELECT id
			FROM jobs
			WHERE (
				status = 'pending'
				OR (status = 'running' AND run_at <= NOW())
			)
			AND attempts < max_attempts
			ORDER BY run_at
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, payload

	`)

	var (
		id      int
		payload []byte
		job     Job
	)

	if err := row.Scan(&id, &payload); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	if err := json.Unmarshal(payload, &job); err != nil {
		return false, err
	}

	log.Printf("workere %d picked job %d (estimated %s)\n", workerID, id, job.EstimatedTime)
	time.Sleep(job.EstimatedTime)

	failed := rand.IntN(10) < 3
	if failed {
		if err := markFailed(db, id); err != nil {
			return false, err
		}
		log.Printf("worker %d failed job %d\n", workerID, id)
	} else {
		if err := markDone(db, id); err != nil {
			return false, err
		}
		log.Printf("worker %d completed job %d\n", workerID, id)
	}

	return true, nil
}

func enque(db *sql.DB, payload []byte) (int, error) {
	var jobID int
	err := db.QueryRow(`INSERT INTO jobs(payload) VALUES($1) RETURNING id`, payload).Scan(&jobID)
	return jobID, err
}

func markDone(db *sql.DB, id int) error {
	_, err := db.Exec(`
		UPDATE jobs
		SET status = 'done', finished_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, id)

	return err
}

func markFailed(db *sql.DB, id int) error {
	_, err := db.Exec(`
		UPDATE jobs 
		SET
		status = (CASE WHEN attempts >= max_attempts THEN 'dead' ELSE 'pending' END)::job_status, 
		finished_at = CASE WHEN attempts >= max_attempts THEN NOW() ELSE NULL END,
		run_at = NOW() + (INTERVAL '1 second' * (2 ^ attempts)),
		updated_at = NOW()
		WHERE id = $1
	`, id)

	return err
}

func createTable(db *sql.DB) error {
	_, err := db.Exec(`
		DO $$ BEGIN
			CREATE TYPE job_status AS ENUM('pending', 'running', 'done', 'failed', 'dead');
		EXCEPTION
			WHEN duplicate_object THEN NULL;
		END $$;

		CREATE TABLE IF NOT EXISTS jobs(
			id BIGSERIAL PRIMARY KEY,
			payload JSONB NOT NULL,
			status job_status NOT NULL DEFAULT 'pending',

			attempts INT NOT NULL DEFAULT 0,
			max_attempts INT NOT NULL DEFAULT 3,

			started_at TIMESTAMPTZ,
			finished_at TIMESTAMPTZ,
			run_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS jobs_status_run_at_idx ON jobs(status, run_at) WHERE status = 'pending';
	`)

	return err
}

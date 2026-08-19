// Command worker polls generation_jobs and turns study material into
// AI-generated exam questions, decoupled from the API's request/response
// cycle so large documents don't block HTTP clients.
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"examplatform/internal/ai"
	"examplatform/internal/config"
	"examplatform/internal/database"
	"examplatform/internal/jobs"
	"examplatform/internal/questions"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("worker: failed to connect to database: %v", err)
	}
	defer db.Close()

	subjectName := func(ctx context.Context, subjectID string) (string, error) {
		var name string
		err := db.QueryRow(ctx, `SELECT name FROM subjects WHERE id = $1`, subjectID).Scan(&name)
		return name, err
	}

	w := &jobs.Worker{
		Jobs:         jobs.NewRepository(db),
		Questions:    questions.NewRepository(db),
		AI:           ai.NewClient(cfg.AnthropicAPIKey),
		SubjectName:  subjectName,
		PollInterval: time.Duration(cfg.WorkerPollSeconds) * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Println("worker: starting AI generation worker")
	w.Run(ctx)
}

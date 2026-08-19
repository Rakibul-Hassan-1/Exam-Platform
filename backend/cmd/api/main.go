// Command api runs the HTTP REST API for the examination platform.
package main

import (
	"log"
	"net/http"

	"examplatform/internal/config"
	"examplatform/internal/database"
	"examplatform/internal/router"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("api: failed to connect to database: %v", err)
	}
	defer db.Close()

	handler := router.New(cfg, db)

	log.Printf("api: listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		log.Fatalf("api: server error: %v", err)
	}
}

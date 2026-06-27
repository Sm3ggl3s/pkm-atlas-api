// Package main is the entrypoint for the pkm-atlas-api HTTP server.
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Sm3ggl3s/pkm-atlas-api/internal/config"
	"github.com/Sm3ggl3s/pkm-atlas-api/internal/database"
)

// newMux builds the HTTP router with all routes registered. It is kept
// separate from main so tests can exercise the real routing table.
func newMux() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)

	return mux
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	mux := newMux()

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("API listening on http://localhost:%s", cfg.Port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

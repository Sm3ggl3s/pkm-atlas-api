// Command seed imports the Pokédex from PokéAPI into Postgres. It is a manual,
// re-runnable tool (the HTTP API itself stays read-only):
//
//	go run ./cmd/seed                 # full National Dex
//	go run ./cmd/seed --limit 151     # just the first 151
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Sm3ggl3s/pkm-atlas-api/internal/config"
	"github.com/Sm3ggl3s/pkm-atlas-api/internal/database"
	"github.com/Sm3ggl3s/pkm-atlas-api/internal/pokeapi"
	"github.com/Sm3ggl3s/pkm-atlas-api/internal/pokemon"
	"github.com/Sm3ggl3s/pkm-atlas-api/internal/seed"
)

func main() {
	var (
		databaseURL = flag.String("database-url", "", "Postgres URL (defaults to DATABASE_URL from env/.env)")
		baseURL     = flag.String("base-url", "https://pokeapi.co/api/v2", "PokéAPI base URL")
		concurrency = flag.Int("concurrency", 8, "number of parallel PokéAPI fetches")
		limit       = flag.Int("limit", 0, "max species to import (0 = all)")
		timeout     = flag.Duration("timeout", 30*time.Second, "per-request HTTP timeout")
	)
	flag.Parse()

	if err := run(*databaseURL, *baseURL, *concurrency, *limit, *timeout); err != nil {
		log.Fatalf("seed: %v", err)
	}
}

func run(databaseURL, baseURL string, concurrency, limit int, timeout time.Duration) error {
	if databaseURL == "" {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		databaseURL = cfg.DatabaseURL
	}

	// Cancel cleanly on Ctrl-C / SIGTERM so a long run can be stopped safely.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	client := pokeapi.NewClient(baseURL, &http.Client{Timeout: timeout})
	store := pokemon.NewSeedStore(pool)
	seeder := seed.NewSeeder(client, store, log.Default())

	log.Printf("seeding from %s (concurrency=%d, limit=%d)...", baseURL, concurrency, limit)
	start := time.Now()

	summary, err := seeder.Run(ctx, seed.Options{Limit: limit, Concurrency: concurrency})
	if err != nil {
		return err
	}

	log.Printf("done in %s: %d species, %d forms, %d evolution edges",
		time.Since(start).Round(time.Second), summary.Species, summary.Forms, summary.Evolutions)
	return nil
}

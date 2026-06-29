// Package main is the entrypoint for the pkm-atlas-api HTTP server.
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Sm3ggl3s/pkm-atlas-api/internal/config"
	"github.com/Sm3ggl3s/pkm-atlas-api/internal/database"
	"github.com/Sm3ggl3s/pkm-atlas-api/internal/pokemon"
)

// newMux builds the HTTP router with all routes registered. It is kept
// separate from main so tests can exercise the real routing table.
func newMux(pokemonHandler *pokemon.Handler) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", healthHandler)

	mux.HandleFunc("GET /api/v1/pokemon", pokemonHandler.ListPokemon)
	mux.HandleFunc("GET /api/v1/pokemon/{slug}", pokemonHandler.GetPokemon)
	mux.HandleFunc("GET /api/v1/types", pokemonHandler.ListTypes)
	mux.HandleFunc("GET /api/v1/types/{slug}/pokemon", pokemonHandler.ListPokemonByType)

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

	pokemonHandler := pokemon.NewHandler(pokemon.NewRepository(pool))

	mux := newMux(pokemonHandler)

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

package database

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestOpen_Errors(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "malformed url",
			url:  "://not-a-valid-dsn",
		},
		{
			name: "unreachable server",
			url:  "postgres://postgres:postgres@127.0.0.1:1/db?sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			pool, err := Open(ctx, tt.url)
			if err == nil {
				pool.Close()
				t.Fatalf("Open(%q) expected an error, got nil", tt.url)
			}
		})
	}
}

func TestOpen_Integration(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping database integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := Open(ctx, url)
	if err != nil {
		t.Fatalf("Open() unexpected error: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Errorf("Ping() unexpected error: %v", err)
	}
}

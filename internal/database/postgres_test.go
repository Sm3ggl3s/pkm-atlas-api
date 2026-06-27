package database

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestOpenInvalidURL covers the parsing failure path. An unparseable
// connection string fails before any network access, so this needs no real
// database.
func TestOpenInvalidURL(t *testing.T) {
	_, err := Open(context.Background(), "://not-a-valid-dsn")
	if err == nil {
		t.Fatal("Open() expected an error for an invalid connection string, got nil")
	}
}

// TestOpenCancelledContext checks that Open surfaces an error when the context
// is already cancelled. The connection string parses, but the Ping cannot
// succeed, so we exercise the error path without needing a running server.
func TestOpenCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately so Ping fails fast

	_, err := Open(ctx, "postgres://user:pass@localhost:5432/db?sslmode=disable")
	if err == nil {
		t.Fatal("Open() expected an error with a cancelled context, got nil")
	}
}

// TestOpenIntegration connects to a real Postgres instance. It only runs when
// TEST_DATABASE_URL is set, so the normal `go test ./...` run does not require
// Docker or Postgres.
func TestOpenIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping Postgres integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open() against TEST_DATABASE_URL failed: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("Ping() failed: %v", err)
	}
}

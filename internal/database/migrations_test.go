package database

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// migrationsDir resolves the repo migrations directory relative to this test
// file, independent of the working directory `go test` runs in.
func migrationsDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file location")
	}
	// internal/database/migrations_test.go -> repo root -> migrations
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations")
}

func execSQLFile(t *testing.T, ctx context.Context, conn *pgx.Conn, path string) {
	t.Helper()
	sql, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if _, err := conn.Exec(ctx, string(sql)); err != nil {
		t.Fatalf("exec %s: %v", filepath.Base(path), err)
	}
}

// tableExists reports whether a public table is present.
func tableExists(t *testing.T, ctx context.Context, conn *pgx.Conn, table string) bool {
	t.Helper()
	var reg *string
	if err := conn.QueryRow(ctx, "SELECT to_regclass($1)::text", "public."+table).Scan(&reg); err != nil {
		t.Fatalf("to_regclass(%s): %v", table, err)
	}
	return reg != nil
}

func TestMigration000001_UpDown(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping database integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	// Registered first so it runs last (cleanups are LIFO): the down-migration
	// cleanup below must execute while the connection is still open.
	t.Cleanup(func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		conn.Close(closeCtx)
	})

	dir := migrationsDir(t)
	upFile := filepath.Join(dir, "000001_create_pokemon_catalog.up.sql")
	downFile := filepath.Join(dir, "000001_create_pokemon_catalog.down.sql")

	// Ensure a clean slate and leave the DB clean afterwards, so the test is
	// idempotent across reruns regardless of prior state.
	execSQLFile(t, ctx, conn, downFile)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		sql, err := os.ReadFile(downFile)
		if err != nil {
			t.Errorf("cleanup read down migration: %v", err)
			return
		}
		if _, err := conn.Exec(cleanupCtx, string(sql)); err != nil {
			t.Errorf("cleanup exec down migration: %v", err)
		}
	})

	tables := []string{
		"pokemon_generations",
		"pokemon_species",
		"pokemon_forms",
		"pokemon_types",
		"pokemon_form_types",
	}

	// Up: every catalogue table should exist.
	execSQLFile(t, ctx, conn, upFile)
	for _, table := range tables {
		if !tableExists(t, ctx, conn, table) {
			t.Errorf("after up migration, table %q does not exist", table)
		}
	}

	// Down: every catalogue table should be gone.
	execSQLFile(t, ctx, conn, downFile)
	for _, table := range tables {
		if tableExists(t, ctx, conn, table) {
			t.Errorf("after down migration, table %q still exists", table)
		}
	}
}

package config

import (
	"strings"
	"testing"
)

// TestLoad covers the behaviour of Load across the combinations of PORT and
// DATABASE_URL that the current implementation cares about. Environment
// variables are isolated per-test with t.Setenv, so no real .env file is used.
func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		port        string // value for PORT; "" means leave unset
		setPort     bool
		databaseURL string // value for DATABASE_URL; "" means leave unset
		setDatabase bool

		wantErr  bool
		wantPort string
		wantURL  string
	}{
		{
			name:        "defaults port when unset",
			setPort:     false,
			databaseURL: "postgres://localhost:5432/db",
			setDatabase: true,
			wantPort:    "8081",
			wantURL:     "postgres://localhost:5432/db",
		},
		{
			name:        "uses port from env when set",
			port:        "9090",
			setPort:     true,
			databaseURL: "postgres://localhost:5432/db",
			setDatabase: true,
			wantPort:    "9090",
			wantURL:     "postgres://localhost:5432/db",
		},
		{
			name:        "empty port falls back to default",
			port:        "",
			setPort:     true,
			databaseURL: "postgres://localhost:5432/db",
			setDatabase: true,
			wantPort:    "8081",
			wantURL:     "postgres://localhost:5432/db",
		},
		{
			name:        "missing database url is an error",
			port:        "8081",
			setPort:     true,
			setDatabase: false,
			wantErr:     true,
		},
		{
			name:        "empty database url is an error",
			port:        "8081",
			setPort:     true,
			databaseURL: "",
			setDatabase: true,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// t.Setenv handles cleanup and prevents the test from running in
			// parallel, which keeps the environment isolated per test.
			if tt.setPort {
				t.Setenv("PORT", tt.port)
			} else {
				unset(t, "PORT")
			}

			if tt.setDatabase {
				t.Setenv("DATABASE_URL", tt.databaseURL)
			} else {
				unset(t, "DATABASE_URL")
			}

			cfg, err := Load()

			if tt.wantErr {
				if err == nil {
					t.Fatalf("Load() expected an error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			if cfg.Port != tt.wantPort {
				t.Errorf("Port = %q, want %q", cfg.Port, tt.wantPort)
			}
			if cfg.DatabaseURL != tt.wantURL {
				t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, tt.wantURL)
			}
		})
	}
}

// TestLoadMissingDatabaseURLMessage checks that the error for a missing
// DATABASE_URL is useful enough to debug from.
func TestLoadMissingDatabaseURLMessage(t *testing.T) {
	t.Setenv("PORT", "8081")
	unset(t, "DATABASE_URL")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected an error for missing DATABASE_URL, got nil")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("error %q should mention DATABASE_URL", err.Error())
	}
}

// unset clears an environment variable for the duration of a test. Load reads
// values with os.Getenv, which treats an empty string the same as an unset
// variable, so setting it empty via t.Setenv is enough and is automatically
// restored when the test finishes.
func unset(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "")
}

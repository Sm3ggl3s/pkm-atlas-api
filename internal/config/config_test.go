package config

import "testing"

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		databaseURL string
		port        string
		wantErr     bool
		wantPort    string
	}{
		{
			name:        "defaults port when unset",
			databaseURL: "postgres://localhost/db",
			port:        "",
			wantPort:    "8081",
		},
		{
			name:        "respects explicit port",
			databaseURL: "postgres://localhost/db",
			port:        "9090",
			wantPort:    "9090",
		},
		{
			name:        "errors when database url missing",
			databaseURL: "",
			port:        "8081",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// t.Setenv controls the environment; tests run in the package dir
			// (no .env present) so godotenv.Load is a no-op.
			t.Setenv("DATABASE_URL", tt.databaseURL)
			t.Setenv("PORT", tt.port)

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
			if cfg.DatabaseURL != tt.databaseURL {
				t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, tt.databaseURL)
			}
		})
	}
}

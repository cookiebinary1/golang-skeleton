package config

import "testing"

func TestDatabaseDSN(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "explicit DSN wins",
			cfg: Config{
				PostgresDSN:  "postgres://x:y@h:1/d",
				PostgresHost: "ignored",
			},
			want: "postgres://x:y@h:1/d",
		},
		{
			name: "assembled from parts",
			cfg: Config{
				PostgresHost:     "db.internal",
				PostgresPort:     "5432",
				PostgresUser:     "svc",
				PostgresPassword: "secret",
				PostgresDB:       "app",
			},
			want: "postgres://svc:secret@db.internal:5432/app?sslmode=disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.DatabaseDSN(); got != tt.want {
				t.Fatalf("DatabaseDSN() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEnvFallback(t *testing.T) {
	t.Setenv("SERVICE_NAME", "demo")
	t.Setenv("HTTP_ADDR", "")
	cfg := Load()
	if cfg.ServiceName != "demo" {
		t.Fatalf("ServiceName = %q, want demo", cfg.ServiceName)
	}
	if cfg.HTTPAddr != "0.0.0.0:8080" {
		t.Fatalf("HTTPAddr default = %q", cfg.HTTPAddr)
	}
}

func TestDatabaseDSNEscapesCredentials(t *testing.T) {
	cfg := Config{PostgresUser: "a@b", PostgresPassword: "p@ss:/?#", PostgresHost: "::1", PostgresPort: "5432", PostgresDB: "my db"}
	got := cfg.DatabaseDSN()
	want := "postgres://a%40b:p%40ss%3A%2F%3F%23@[::1]:5432/my%20db?sslmode=disable"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

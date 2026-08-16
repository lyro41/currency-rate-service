package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMustLoadWithoutConfigFile(t *testing.T) {
	t.Setenv(cfgPathEnv, "")

	cfg := MustLoad()
	if cfg.Storage.Host != "localhost" || cfg.Storage.Port != 5432 {
		t.Errorf("storage address = %s:%d, want localhost:5432", cfg.Storage.Host, cfg.Storage.Port)
	}
	if cfg.Storage.Timeout != 5*time.Second {
		t.Errorf("storage timeout = %s, want 5s", cfg.Storage.Timeout)
	}
	if cfg.HTTPServer.Address != ":8080" {
		t.Errorf("server address = %q, want :8080", cfg.HTTPServer.Address)
	}
	if cfg.Worker.BufferSize != 100 {
		t.Errorf("worker buffer size = %d, want 100", cfg.Worker.BufferSize)
	}
}

func TestMustLoadFromYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
env: development
storage:
  host: db.example
  port: 5433
  user: app
  password: secret
  database: rates
  timeout: 2s
provider:
  timeout: 3s
http_server:
  address: ":9090"
worker:
  buffer_size: 32
`)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(cfgPathEnv, path)

	cfg := MustLoad()
	if cfg.Env != "development" || cfg.Storage.Host != "db.example" || cfg.Storage.Port != 5433 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.Provider.Timeout != 3*time.Second || cfg.HTTPServer.Address != ":9090" {
		t.Fatalf("unexpected service settings: %+v", cfg)
	}
	if cfg.Worker.BufferSize != 32 {
		t.Fatalf("unexpected worker settings: %+v", cfg)
	}
}

func TestWorkerValidate(t *testing.T) {
	tests := []struct {
		name       string
		bufferSize int
		wantErr    bool
	}{
		{name: "positive", bufferSize: 1},
		{name: "zero", bufferSize: 0, wantErr: true},
		{name: "negative", bufferSize: -1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := (Worker{BufferSize: tt.bufferSize}).Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, want error: %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	cfg := Config{
		Env:    "development",
		Worker: Worker{BufferSize: 100},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}

	cfg.Worker.BufferSize = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want invalid worker buffer size")
	}
}

func TestStorageStringEscapesCredentials(t *testing.T) {
	storage := Storage{
		Host:     "localhost",
		Port:     5432,
		User:     "app@user",
		Password: "p@ss:word/1",
		Database: "rates",
	}

	want := "postgres://app%40user:p%40ss%3Aword%2F1@localhost:5432/rates"
	if got := storage.String(); got != want {
		t.Fatalf("connection string = %q, want %q", got, want)
	}
}

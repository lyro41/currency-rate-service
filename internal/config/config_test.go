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
}

func TestMustLoadFromYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
env: test
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
`)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(cfgPathEnv, path)

	cfg := MustLoad()
	if cfg.Env != "test" || cfg.Storage.Host != "db.example" || cfg.Storage.Port != 5433 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.Provider.Timeout != 3*time.Second || cfg.HTTPServer.Address != ":9090" {
		t.Fatalf("unexpected service settings: %+v", cfg)
	}
}

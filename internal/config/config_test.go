package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	if err := os.WriteFile(path, []byte("database = \"postgres://example\"\nwebhook = \"http://hook\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Database != "postgres://example" {
		t.Fatalf("database=%q want %q", cfg.Database, "postgres://example")
	}
	if cfg.Webhook != "http://hook" {
		t.Fatalf("webhook=%q want %q", cfg.Webhook, "http://hook")
	}
}

func TestLoadRejectsMissingDatabase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	if err := os.WriteFile(path, []byte("webhook = \"http://hook\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "database connection string is required") {
		t.Fatalf("expected missing database error, got %v", err)
	}
}

func TestTLSCredsRequireFields(t *testing.T) {
	tlsCfg := TLS{}

	if _, err := tlsCfg.ServerCreds(); err == nil {
		t.Fatalf("expected missing server TLS fields error")
	}
	if _, err := tlsCfg.ClientCreds(); err == nil {
		t.Fatalf("expected missing client TLS fields error")
	}
}

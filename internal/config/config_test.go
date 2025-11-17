package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create temporary config file
	content := `
server:
  port: 9090
  host: "127.0.0.1"

database:
  host: "testdb"
  port: 5432
  user: "testuser"
  password: "testpass"
  dbname: "testdb"
`

	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Load config
	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded values
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", cfg.Server.Port)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Expected host 127.0.0.1, got %s", cfg.Server.Host)
	}

	if cfg.Database.Host != "testdb" {
		t.Errorf("Expected database host testdb, got %s", cfg.Database.Host)
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	_, err := Load("nonexistent.yaml")
	if err == nil {
		t.Error("Expected error when loading nonexistent file")
	}
}

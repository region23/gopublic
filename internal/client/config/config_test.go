package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectConfig(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create test gopublic.yaml
	configContent := `version: "1"
tunnels:
  frontend:
    proto: http
    addr: "3000"
    subdomain: misty-river
  backend:
    proto: http
    addr: "8080"
    subdomain: silent-star
`
	configPath := filepath.Join(tmpDir, "gopublic.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load config
	cfg, err := LoadProjectConfig(configPath)
	if err != nil {
		t.Fatalf("LoadProjectConfig() error = %v", err)
	}

	// Check version
	if cfg.Version != "1" {
		t.Errorf("Version = %s, want '1'", cfg.Version)
	}

	// Check tunnels
	if len(cfg.Tunnels) != 2 {
		t.Errorf("Tunnels count = %d, want 2", len(cfg.Tunnels))
	}

	frontend := cfg.Tunnels["frontend"]
	if frontend == nil {
		t.Fatal("Frontend tunnel not found")
	}
	if frontend.Addr != "3000" {
		t.Errorf("Frontend addr = %s, want '3000'", frontend.Addr)
	}
	if frontend.Subdomain != "misty-river" {
		t.Errorf("Frontend subdomain = %s, want 'misty-river'", frontend.Subdomain)
	}

	backend := cfg.Tunnels["backend"]
	if backend == nil {
		t.Fatal("Backend tunnel not found")
	}
	if backend.Addr != "8080" {
		t.Errorf("Backend addr = %s, want '8080'", backend.Addr)
	}
}

func TestLoadProjectConfig_NotFound(t *testing.T) {
	_, err := LoadProjectConfig("/nonexistent/path/gopublic.yaml")
	if err == nil {
		t.Error("LoadProjectConfig() should fail for nonexistent file")
	}
}

func TestLoadProjectConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "gopublic.yaml")

	// Write invalid YAML
	if err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadProjectConfig(configPath)
	if err == nil {
		t.Error("LoadProjectConfig() should fail for invalid YAML")
	}
}

func TestConfig_SaveAndLoad(t *testing.T) {
	// Save original config path
	origHome := os.Getenv("HOME")

	// Use temp dir as home
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Save config
	cfg := &Config{Token: "test-token-123"}
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Load config
	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loaded.Token != cfg.Token {
		t.Errorf("Token = %s, want %s", loaded.Token, cfg.Token)
	}
}

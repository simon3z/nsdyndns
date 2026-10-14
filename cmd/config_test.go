package cmd

// cspell:ignore godaddy nsdyndns

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()

	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write config: %s", err)
	}

	return path
}

func TestLoadConfigurationValid(t *testing.T) {
	path := writeConfig(t, `
domain: example.com
host: sub
godaddy:
  key: k
  secret: s
digitalocean:
  token: tok
`)

	c, err := LoadConfiguration(path)

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if c.Domain != "example.com" || c.Host != "sub" {
		t.Fatalf("unexpected config: %#v", c)
	}

	if len(c.GetNameServices()) != 2 {
		t.Fatalf("expected 2 name services, got %d", len(c.GetNameServices()))
	}

	if c.FullDomain() != "sub.example.com" {
		t.Fatalf("unexpected full domain %s", c.FullDomain())
	}
}

func TestLoadConfigurationApex(t *testing.T) {
	path := writeConfig(t, `
domain: example.com
digitalocean:
  token: tok
`)

	c, err := LoadConfiguration(path)

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if c.FullDomain() != "example.com" {
		t.Fatalf("unexpected full domain %s", c.FullDomain())
	}
}

func TestLoadConfigurationNoDomain(t *testing.T) {
	path := writeConfig(t, `
digitalocean:
  token: tok
`)

	if _, err := LoadConfiguration(path); err == nil {
		t.Fatal("expected error for missing domain")
	}
}

func TestLoadConfigurationNoService(t *testing.T) {
	path := writeConfig(t, `
domain: example.com
host: sub
`)

	if _, err := LoadConfiguration(path); err == nil {
		t.Fatal("expected error for missing name service")
	}
}

func TestLoadConfigurationMissingFile(t *testing.T) {
	if _, err := LoadConfiguration("/nonexistent/config.yaml"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

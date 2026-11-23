package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	t.Setenv("EXISTING_ENV_VALUE", `"keep"`)

	path := filepath.Join(t.TempDir(), ".env")
	raw := []byte(`
# ignored
OPENROUTER_API_KEY="secret-value"
BOT_TOKEN=bot-token
export EXPORTED_VALUE='yes'
EXISTING_ENV_VALUE=replace
`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	if err := loadDotEnv(path); err != nil {
		t.Fatalf("loadDotEnv: %v", err)
	}

	if got := os.Getenv("OPENROUTER_API_KEY"); got != "secret-value" {
		t.Fatalf("unexpected OPENROUTER_API_KEY: %q", got)
	}
	if got := os.Getenv("BOT_TOKEN"); got != "bot-token" {
		t.Fatalf("unexpected BOT_TOKEN: %q", got)
	}
	if got := os.Getenv("EXPORTED_VALUE"); got != "yes" {
		t.Fatalf("unexpected EXPORTED_VALUE: %q", got)
	}
	if got := os.Getenv("EXISTING_ENV_VALUE"); got != "keep" {
		t.Fatalf("existing value was overwritten: %q", got)
	}
}

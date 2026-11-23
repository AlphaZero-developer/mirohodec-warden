package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfigWebSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	raw := []byte(`
telegram:
  token: test-token
web:
  enabled: true
  listen: ":8090"
  public_url: "http://127.0.0.1:8090"
  mafia_mini_app_url: "https://example.com/mafia-mini-app/"
  mafia_mini_app_dir: "/srv/mafia-mini-app/dist"
  admin_token: secret
  session_ttl: 2h
  send_panel_link_to_owner: true
`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !cfg.Web.Enabled || cfg.Web.Listen != ":8090" {
		t.Fatalf("unexpected web listen settings: %#v", cfg.Web)
	}
	if cfg.Web.SessionTTL != 2*time.Hour {
		t.Fatalf("unexpected session ttl: %v", cfg.Web.SessionTTL)
	}
	if cfg.Web.MafiaMiniAppURL != "https://example.com/mafia-mini-app/" {
		t.Fatalf("unexpected mafia mini app url: %q", cfg.Web.MafiaMiniAppURL)
	}
	if cfg.Web.MafiaMiniAppDir != "/srv/mafia-mini-app/dist" {
		t.Fatalf("unexpected mafia mini app dir: %q", cfg.Web.MafiaMiniAppDir)
	}
	if !cfg.Web.SendPanelLinkToOwner {
		t.Fatal("send_panel_link_to_owner was not loaded")
	}
}

func TestLoadConfigWebSessionTTLForever(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	raw := []byte(`
telegram:
  token: test-token
web:
  enabled: true
  admin_token: secret
  session_ttl: forever
`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Web.SessionTTL >= 0 {
		t.Fatalf("session ttl = %v, want permanent sentinel", cfg.Web.SessionTTL)
	}
}

func TestLoadConfigWebEnvOverrides(t *testing.T) {
	t.Setenv("WEB_PUBLIC_URL", "https://deploy.example.com")
	t.Setenv("WEB_MAFIA_MINI_APP_DIR", "/opt/mafia/dist")
	path := filepath.Join(t.TempDir(), "config.yaml")
	raw := []byte(`
telegram:
  token: test-token
web:
  public_url: "https://config.example.com"
  mafia_mini_app_dir: "/config/dist"
`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Web.PublicURL != "https://deploy.example.com" {
		t.Fatalf("WEB_PUBLIC_URL was not applied: %q", cfg.Web.PublicURL)
	}
	if cfg.Web.MafiaMiniAppDir != "/opt/mafia/dist" {
		t.Fatalf("WEB_MAFIA_MINI_APP_DIR was not applied: %q", cfg.Web.MafiaMiniAppDir)
	}
}

func TestLoadConfigAISettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	raw := []byte(`
telegram:
  token: test-token
ai:
  enabled: true
  base_url: "http://127.0.0.1:8081/chat"
  model: "test-model"
  timeout: 3s
  soft_timeout: 1s
  max_tokens: 777
  max_reply_chars: 1234
  fas_mute_duration: 2m
  system_prompt: "test prompt"
  fallback_reply: "later"
`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !cfg.AI.Enabled {
		t.Fatal("ai.enabled was not loaded")
	}
	if cfg.AI.BaseURL != "http://127.0.0.1:8081/chat" || cfg.AI.Model != "test-model" || cfg.AI.Timeout != "3s" || cfg.AI.SoftTimeout != "1s" {
		t.Fatalf("unexpected ai settings: %#v", cfg.AI)
	}
	if cfg.AI.MaxTokens != 777 || cfg.AI.MaxReplyChars != 1234 || cfg.AI.FasMuteDuration != "2m" {
		t.Fatalf("unexpected ai limits: %#v", cfg.AI)
	}
	if cfg.AI.SystemPrompt != "test prompt" || cfg.AI.FallbackReply != "later" {
		t.Fatalf("unexpected ai prompt/fallback: %#v", cfg.AI)
	}
}

func TestLoadConfigDiscussionAutoReply(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	raw := []byte(`
telegram:
  token: test-token
discussion_auto_reply:
  enabled: true
  chat_id: -100123
  source_chat_id: -100456
  strict_checks: true
  text: "У нас есть чатик! — https://t.me/test_chat"
  chat_button_text: "Чатик"
  chat_button_url: "https://t.me/test_chat"
  follow_up_text: "Также у нас есть бусти"
  follow_up_button_text: "Boosty"
  follow_up_button_url: "https://boosty.to/test"
  pin: true
`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !cfg.Discussion.Enabled || !cfg.Discussion.Pin || cfg.Discussion.ChatID != -100123 ||
		cfg.Discussion.SourceChatID != -100456 || !cfg.Discussion.StrictChecks ||
		cfg.Discussion.Text != "У нас есть чатик! — https://t.me/test_chat" ||
		cfg.Discussion.ChatButtonText != "Чатик" ||
		cfg.Discussion.ChatButtonURL != "https://t.me/test_chat" ||
		cfg.Discussion.FollowUpText != "Также у нас есть бусти" ||
		cfg.Discussion.FollowUpButtonText != "Boosty" ||
		cfg.Discussion.FollowUpButtonURL != "https://boosty.to/test" {
		t.Fatalf("unexpected discussion auto-reply config: %#v", cfg.Discussion)
	}
}

func TestLoadConfigLayaDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	raw := []byte(`
telegram:
  token: test-token
`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Laya.FatherUserID != 0 || cfg.Laya.MotherUserID != 0 || cfg.Laya.BrotherUserID != 0 || cfg.Laya.CreatorUserID != 0 {
		t.Fatalf("unexpected laya defaults: %#v", cfg.Laya)
	}
	if cfg.Laya.FatherPrompt != defaultLayaFatherPrompt || cfg.Laya.MotherPrompt != defaultLayaMotherPrompt || cfg.Laya.BrotherPrompt != defaultLayaBrotherPrompt || cfg.Laya.CreatorPrompt != defaultLayaCreatorPrompt {
		t.Fatalf("unexpected laya prompt defaults: %#v", cfg.Laya)
	}
}

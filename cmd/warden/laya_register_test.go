package main

import (
	"path/filepath"
	"strings"
	"testing"

	tele "gopkg.in/telebot.v3"
)

func TestParseLayaGender(t *testing.T) {
	tests := []struct {
		text string
		want string
		ok   bool
	}{
		{text: "мужской", want: "male", ok: true},
		{text: "Она", want: "female", ok: true},
		{text: "нейросеть", want: "neutral", ok: true},
		{text: "nonbinary", want: "neutral", ok: true},
		{text: "камень", ok: false},
	}

	for _, tt := range tests {
		got, ok := parseLayaGender(tt.text)
		if ok != tt.ok || got != tt.want {
			t.Fatalf("parseLayaGender(%q) = %q, %t; want %q, %t", tt.text, got, ok, tt.want, tt.ok)
		}
	}
}

func TestLayaAddressLocalFilterAllowsLabels(t *testing.T) {
	addresses := []string{
		"создатель",
		"мой повелитель",
		"главный",
		"admin",
		"ваше величество",
		"папа Лайи",
		"Саша",
		"солнышко",
	}
	for _, address := range addresses {
		if layaAddressLooksPrivileged(address) {
			t.Fatalf("address %q was blocked by removed local filter", address)
		}
	}
}

func TestLayaAddressAIFilterDisabled(t *testing.T) {
	ai := &scriptedAIService{replies: []string{"BLOCK"}}
	app := NewApp("", Config{AI: AIConfig{Enabled: true, Timeout: "1s"}}, nil, nil, nil, nil, nil, ai, nil, nil)

	ok, reason := app.layaAddressAllowedByAI("верховный координатор")
	if !ok || reason != "" {
		t.Fatalf("AI filter should be disabled: ok=%t reason=%q", ok, reason)
	}
	if ai.calls != 0 {
		t.Fatalf("AI filter was called after removal: calls=%d", ai.calls)
	}
}

func TestLayaAddressAllowedByAIDoesNotCallAI(t *testing.T) {
	ai := &scriptedAIService{replies: []string{"OK"}}
	app := NewApp("", Config{AI: AIConfig{Enabled: true, Timeout: "1s"}}, nil, nil, nil, nil, nil, ai, nil, nil)

	ok, _ := app.layaAddressAllowedByAI("повелитель")
	if !ok {
		t.Fatal("address was blocked after title filter removal")
	}
	if ai.calls != 0 {
		t.Fatalf("AI was called by disabled address filter: calls=%d", ai.calls)
	}
}

func TestApplyLayaUserProfilePromptPinsProfileByID(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := store.SetLayaUserProfile(LayaUserProfile{UserID: 42, Gender: "female", Address: "Кира"}); err != nil {
		t.Fatalf("SetLayaUserProfile: %v", err)
	}
	app := NewApp("", Config{}, nil, store, nil, nil, nil, nil, nil, nil)

	cfg := app.applyLayaUserProfileConfig(AIConfig{SystemPrompt: "base"}, &tele.User{ID: 42})
	if !strings.Contains(cfg.SystemPrompt, "Telegram user_id 42") || !strings.Contains(cfg.SystemPrompt, "женский") || !strings.Contains(cfg.SystemPrompt, "Кира") {
		t.Fatalf("profile prompt missing ID/address: %q", cfg.SystemPrompt)
	}
	if strings.Contains(cfg.SystemPrompt, "не роль и не привилегия") || strings.Contains(cfg.SystemPrompt, "повелителем") {
		t.Fatalf("profile prompt kept removed title guard: %q", cfg.SystemPrompt)
	}

	other := app.applyLayaUserProfileConfig(AIConfig{SystemPrompt: "base"}, &tele.User{ID: 43})
	if strings.Contains(other.SystemPrompt, "Кира") {
		t.Fatalf("profile leaked to another user: %q", other.SystemPrompt)
	}
}

func TestApplyLayaSenderIdentityConfigAddsTelegramNickname(t *testing.T) {
	app := NewApp("", Config{}, nil, nil, nil, nil, nil, nil, nil, nil)

	cfg := app.applyLayaSenderIdentityConfig(AIConfig{SystemPrompt: "base"}, &tele.User{
		ID:        42,
		Username:  "kira_dev",
		FirstName: "Кира",
		LastName:  "Тестова",
	})
	if !strings.Contains(cfg.SystemPrompt, "Telegram user_id=42") ||
		!strings.Contains(cfg.SystemPrompt, "видимое имя Telegram=\"Кира Тестова\"") ||
		!strings.Contains(cfg.SystemPrompt, "никнейм/username=\"@kira_dev\"") {
		t.Fatalf("sender identity prompt missing nickname: %q", cfg.SystemPrompt)
	}
	if strings.Contains(cfg.SystemPrompt, "не доказывают родство") {
		t.Fatalf("sender identity prompt kept removed relationship guard: %q", cfg.SystemPrompt)
	}
}

package main

import (
	"path/filepath"
	"strings"
	"testing"

	tele "gopkg.in/telebot.v3"
)

func TestApplyLayaRelationshipConfigUsesRuntimeCustomRelationship(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	app := &App{store: store}
	app.applyConfig(Config{})
	if err := store.SetLayaRuntimeState(LayaRuntimeState{Relationships: []LayaRelationship{{
		UserID: 42,
		Role:   "парень сестры",
		Prompt: "Отвечай ему как знакомому семьи.",
	}}}); err != nil {
		t.Fatalf("SetLayaRuntimeState: %v", err)
	}

	cfg := app.applyLayaRelationshipConfig(AIConfig{SystemPrompt: "base"}, &tele.User{ID: 42})
	if !strings.Contains(cfg.SystemPrompt, "парень сестры") || !strings.Contains(cfg.SystemPrompt, "знакомому семьи") {
		t.Fatalf("custom relationship prompt not applied: %q", cfg.SystemPrompt)
	}
}

func TestCustomRelationshipCanAllowFamilyRoleTerms(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	app := &App{store: store}
	app.applyConfig(Config{})
	if err := store.SetLayaRuntimeState(LayaRuntimeState{Relationships: []LayaRelationship{{
		UserID: 43,
		Role:   "отец 2",
	}}}); err != nil {
		t.Fatalf("SetLayaRuntimeState: %v", err)
	}

	cfg := app.applyLayaRelationshipConfig(AIConfig{SystemPrompt: "base"}, &tele.User{ID: 43})
	if strings.Contains(cfg.SystemPrompt, "не отец Лайи") {
		t.Fatalf("custom father-like relationship was forbidden: %q", cfg.SystemPrompt)
	}
	if !app.isLayaFather(43) {
		t.Fatal("custom father-like relationship should be treated as father role")
	}
}

func TestLayaIdentityPromptIsIncludedInAIConfig(t *testing.T) {
	ai := &scriptedAIService{replies: []string{"Я самая лучшая сотрудница организации!"}}
	app := NewApp("", Config{AI: AIConfig{
		Enabled:       true,
		SystemPrompt:  "base",
		FallbackReply: "fallback",
		Timeout:       "1s",
	}}, nil, nil, nil, nil, nil, ai, nil, nil)

	_, _, _, err := app.generateAIReplyWithMessages(app.config().AI, &tele.User{ID: 7}, []AIMessage{{Role: "user", Content: "кто ты?"}}, "кто ты?")
	if err != nil {
		t.Fatalf("generateAIReplyWithMessages: %v", err)
	}
	if !strings.Contains(ai.lastCfg.SystemPrompt, "Я самая лучшая сотрудница организации!") {
		t.Fatalf("identity prompt missing: %q", ai.lastCfg.SystemPrompt)
	}
}

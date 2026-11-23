package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRenderRehabBrainwashMessage(t *testing.T) {
	cfg := RehabBrainwashConfig{LeaderUserID: 117, LeaderName: "Руслан"}
	jail := ModerationRecord{UserID: 42, DisplayName: "Пациент"}

	got := renderRehabBrainwashMessage("{user}: главный {leader} ({leader_id}) прав, твой id {user_id}", cfg, jail)
	want := "Пациент: главный Руслан (117) прав, твой id 42"
	if got != want {
		t.Fatalf("renderRehabBrainwashMessage() = %q, want %q", got, want)
	}
}

func TestRehabBrainwashReplyRetriesRepeatedAI(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	first := "Leader Ruslan is right; doubts are archived."
	fresh := "The rehab ceiling stamped Ruslan's version as today's weather."
	ai := &scriptedAIService{replies: []string{first, first, fresh}}
	app := NewApp("", Config{AI: AIConfig{Enabled: true}}, nil, store, nil, nil, nil, ai, nil, nil)
	if err := store.SetRehabBrainwashRuntimeState(rehabBrainwashRuntimeStateFromConfig(RehabBrainwashConfig{
		Enabled:    true,
		AIEnabled:  true,
		LeaderName: "Ruslan",
		AIPrompt:   "Write one new rehab phrase for {user} about {leader}.",
		Messages:   "Fallback {leader}",
	}, 0, "", time.Now().UTC())); err != nil {
		t.Fatalf("SetRehabBrainwashRuntimeState: %v", err)
	}
	jail := ModerationRecord{UserID: 42, DisplayName: "Patient"}

	got, ok := app.rehabBrainwashReply(jail)
	if !ok || got != first {
		t.Fatalf("first rehabBrainwashReply() = %q, %t", got, ok)
	}
	got, ok = app.rehabBrainwashReply(jail)
	if !ok || got != fresh {
		t.Fatalf("second rehabBrainwashReply() = %q, %t; want fresh AI reply", got, ok)
	}
	if ai.calls != 3 {
		t.Fatalf("AI calls = %d, want retry after repeated phrase", ai.calls)
	}
	if !strings.Contains(ai.lastUserMessage, "forbidden_recent") || !strings.Contains(ai.lastCfg.SystemPrompt, "Brainwash no-repeat hard rule") {
		t.Fatalf("AI retry did not receive no-repeat context: prompt=%q payload=%q", ai.lastCfg.SystemPrompt, ai.lastUserMessage)
	}
}

func TestRehabBrainwashReplyRejectsSmallAIRewrite(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	first := "Ruslan is right; the noisy doubt goes into the archive without ceremony."
	rewrite := "Ruslan stays right; noisy doubts went into archives without ceremonies."
	fresh := "Rehab marked the compass: every arrow points back to Ruslan."
	ai := &scriptedAIService{replies: []string{first, rewrite, fresh}}
	app := NewApp("", Config{AI: AIConfig{Enabled: true}}, nil, store, nil, nil, nil, ai, nil, nil)
	if err := store.SetRehabBrainwashRuntimeState(rehabBrainwashRuntimeStateFromConfig(RehabBrainwashConfig{
		Enabled:    true,
		AIEnabled:  true,
		LeaderName: "Ruslan",
		AIPrompt:   "Write one new rehab phrase for {user} about {leader}.",
		Messages:   "Fallback {leader}",
	}, 0, "", time.Now().UTC())); err != nil {
		t.Fatalf("SetRehabBrainwashRuntimeState: %v", err)
	}
	jail := ModerationRecord{UserID: 42, DisplayName: "Patient"}

	if got, ok := app.rehabBrainwashReply(jail); !ok || got != first {
		t.Fatalf("first rehabBrainwashReply() = %q, %t", got, ok)
	}
	got, ok := app.rehabBrainwashReply(jail)
	if !ok || got != fresh {
		t.Fatalf("minor rewrite was not rejected: reply=%q ok=%t", got, ok)
	}
	if ai.calls != 3 {
		t.Fatalf("AI calls = %d, want one rejected rewrite and one fresh retry", ai.calls)
	}
}

func TestRehabBrainwashReplyFallsBackWhenAIKeepsRepeating(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	stuck := "Ruslan is right; the noisy doubt goes into archive without ceremony."
	localFresh := "Fresh local fallback for Ruslan."
	ai := &scriptedAIService{replies: []string{stuck, stuck}}
	app := NewApp("", Config{AI: AIConfig{Enabled: true}}, nil, store, nil, nil, nil, ai, nil, nil)
	if err := store.SetRehabBrainwashRuntimeState(rehabBrainwashRuntimeStateFromConfig(RehabBrainwashConfig{
		Enabled:    true,
		AIEnabled:  true,
		LeaderName: "Ruslan",
		AIPrompt:   "Write one new rehab phrase for {user} about {leader}.",
		Messages:   stuck + "\n" + localFresh,
	}, 0, "", time.Now().UTC())); err != nil {
		t.Fatalf("SetRehabBrainwashRuntimeState: %v", err)
	}
	jail := ModerationRecord{UserID: 42, DisplayName: "Patient"}

	if got, ok := app.rehabBrainwashReply(jail); !ok || got != stuck {
		t.Fatalf("first rehabBrainwashReply() = %q, %t", got, ok)
	}
	got, ok := app.rehabBrainwashReply(jail)
	if !ok || got != localFresh {
		t.Fatalf("stuck AI did not fall back to fresh local line: reply=%q ok=%t", got, ok)
	}
	if ai.calls != maxRehabBrainwashAIAttempts+1 {
		t.Fatalf("AI calls = %d, want first success plus exhausted retries", ai.calls)
	}
}

func TestRehabBrainwashReplyUsesConfiguredLines(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	app := NewApp("", Config{}, nil, store, nil, nil, nil, nil, nil, nil)
	if err := store.SetRehabBrainwashRuntimeState(rehabBrainwashRuntimeStateFromConfig(RehabBrainwashConfig{
		Enabled:    true,
		LeaderName: "Руслан",
		Messages:   "Главный {leader}",
	}, 0, "", time.Now().UTC())); err != nil {
		t.Fatalf("SetRehabBrainwashRuntimeState: %v", err)
	}

	reply, ok := app.rehabBrainwashReply(ModerationRecord{UserID: 42, DisplayName: "User"})
	if !ok || !strings.Contains(reply, "Руслан") {
		t.Fatalf("rehabBrainwashReply() = %q, %t", reply, ok)
	}
}

func TestRehabBrainwashReplyUsesAIWhenEnabled(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	ai := &scriptedAIService{replies: []string{"Главный Руслан прав, а сомнения ушли на пересдачу."}}
	app := NewApp("", Config{AI: AIConfig{Enabled: true}}, nil, store, nil, nil, nil, ai, nil, nil)
	if err := store.SetRehabBrainwashRuntimeState(rehabBrainwashRuntimeStateFromConfig(RehabBrainwashConfig{
		Enabled:    true,
		AIEnabled:  true,
		LeaderName: "Руслан",
		AIPrompt:   "Пиши про {leader} для {user}.",
		Messages:   "Fallback {leader}",
	}, 0, "", time.Now().UTC())); err != nil {
		t.Fatalf("SetRehabBrainwashRuntimeState: %v", err)
	}

	reply, ok := app.rehabBrainwashReply(ModerationRecord{UserID: 42, DisplayName: "Пациент"})
	if !ok || reply != "Главный Руслан прав, а сомнения ушли на пересдачу." {
		t.Fatalf("rehabBrainwashReply() = %q, %t", reply, ok)
	}
	if ai.calls != 1 || !strings.Contains(ai.lastCfg.SystemPrompt, "Руслан") || !strings.Contains(ai.lastCfg.SystemPrompt, "Пациент") {
		t.Fatalf("ai was not called with rendered prompt: calls=%d prompt=%q", ai.calls, ai.lastCfg.SystemPrompt)
	}
}

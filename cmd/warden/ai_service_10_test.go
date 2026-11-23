package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func TestGenerateAIReplyForMessageClearsHistoryAfterRepeatedRetry(t *testing.T) {
	duplicate := "Я уже сказала это и снова говорю то же самое."
	resetReply := "Скажу с чистого листа: пришли задачу одним предложением."
	replies := make([]string, maxLayaAntiLoopRetries+2)
	for i := 0; i < maxLayaAntiLoopRetries+1; i++ {
		replies[i] = duplicate
	}
	replies[len(replies)-1] = resetReply
	ai := &captureChatAI{replies: replies}
	app := &App{
		ai:          ai,
		aiHistory:   make(map[string][]AIMessage),
		promptCache: make(map[string]string),
	}
	app.applyConfig(Config{Laya: LayaConfig{FatherUserID: 117, CreatorUserID: 900002}})
	cfg := AIConfig{
		BaseURL:       "unused",
		Model:         "test-model",
		Timeout:       "1s",
		MaxReplyChars: 4096,
		SystemPrompt:  "system",
		FallbackReply: "fallback",
	}
	chat := &tele.Chat{ID: -1001, Type: tele.ChatSuperGroup}
	sender := &tele.User{ID: 42}
	app.rememberAIConversation(chat, sender, "первый заход", duplicate)

	got, status, _, err := app.generateAIReplyForMessage(cfg, chat, sender, "тот же заход")
	if err != nil {
		t.Fatalf("generateAIReplyForMessage: %v", err)
	}
	if got != resetReply {
		t.Fatalf("silent reset reply = %q, want %q", got, resetReply)
	}
	if status != "ответ получен" || ai.calls != maxLayaAntiLoopRetries+2 {
		t.Fatalf("status=%q calls=%d, want silent reset reply", status, ai.calls)
	}
	if history := app.aiConversationHistory(chat, sender); len(history) != 0 {
		t.Fatalf("history len = %d, want cleared", len(history))
	}
	if !strings.Contains(ai.systemPrompt, "STYLE_LAYER hidden from the user") || !strings.Contains(ai.systemPrompt, "ordinary first answer") {
		t.Fatalf("silent reset prompt missing: %q", ai.systemPrompt)
	}
	if strings.Contains(ai.systemPrompt, "Внутренний тихий сброс") || strings.Contains(ai.systemPrompt, "упоминать повтор") ||
		strings.Contains(ai.systemPrompt, "PRIVATE FRESH-START RETRY") || strings.Contains(ai.systemPrompt, "failed drafts") ||
		strings.Contains(ai.systemPrompt, "draft_to_avoid") {
		t.Fatalf("silent reset prompt leaked old meta wording: %q", ai.systemPrompt)
	}
}

func TestClearAIConversationRemovesOnlyCurrentChatUser(t *testing.T) {
	app := &App{aiHistory: make(map[string][]AIMessage)}
	chat := &tele.Chat{ID: -1001, Type: tele.ChatSuperGroup}
	otherChat := &tele.Chat{ID: -1002, Type: tele.ChatSuperGroup}
	sender := &tele.User{ID: 42}
	otherUser := &tele.User{ID: 43}

	app.rememberAIConversation(chat, sender, "u1", "a1")
	app.rememberAIConversation(otherChat, sender, "u2", "a2")
	app.rememberAIConversation(chat, otherUser, "u3", "a3")

	app.clearAIConversation(chat, sender)

	if history := app.aiConversationHistory(chat, sender); len(history) != 0 {
		t.Fatalf("current chat/user history len = %d, want cleared", len(history))
	}
	if history := app.aiConversationHistory(otherChat, sender); len(history) != 2 {
		t.Fatalf("other chat history len = %d, want preserved", len(history))
	}
	if history := app.aiConversationHistory(chat, otherUser); len(history) != 2 {
		t.Fatalf("other user history len = %d, want preserved", len(history))
	}
}

func TestGenerateAIReplyReturnsFastFallbackOnSoftTimeout(t *testing.T) {
	ai := &slowChatAI{}
	app := &App{
		ai:          ai,
		promptCache: make(map[string]string),
	}
	app.applyConfig(Config{Laya: LayaConfig{FatherUserID: 117, CreatorUserID: 900002}})

	started := time.Now()
	reply, status, _, err := app.generateAIReply(AIConfig{
		BaseURL:       "unused",
		Model:         "test-model",
		Timeout:       "500ms",
		SoftTimeout:   "20ms",
		MaxReplyChars: 4096,
		SystemPrompt:  "system",
		FallbackReply: "fast fallback",
	}, &tele.User{ID: 42}, "hello")
	elapsed := time.Since(started)
	if !errors.Is(err, errAISoftTimeout) {
		t.Fatalf("err = %v, want errAISoftTimeout", err)
	}
	if reply != "fast fallback" || status != "AI медлит, отправлен быстрый fallback" {
		t.Fatalf("reply/status = %q / %q", reply, status)
	}
	if elapsed > 200*time.Millisecond {
		t.Fatalf("soft timeout took too long: %s", elapsed)
	}
}

func TestGenerateAIReplyForMessageCapsAntiLoopRetryBudget(t *testing.T) {
	duplicate := "alpha beta gamma delta epsilon zeta eta theta iota"
	ai := &delayedChatAI{
		delays:  []time.Duration{20 * time.Millisecond, 80 * time.Millisecond},
		replies: []string{duplicate, "A fresh answer arrives too late."},
	}
	app := &App{
		ai:          ai,
		aiHistory:   make(map[string][]AIMessage),
		promptCache: make(map[string]string),
	}
	app.applyConfig(Config{Laya: LayaConfig{FatherUserID: 117, CreatorUserID: 900002}})
	cfg := AIConfig{
		BaseURL:       "unused",
		Model:         "test-model",
		Timeout:       "200ms",
		SoftTimeout:   "50ms",
		MaxReplyChars: 4096,
		SystemPrompt:  "system",
		FallbackReply: "fallback",
	}
	chat := &tele.Chat{ID: -1001, Type: tele.ChatSuperGroup}
	sender := &tele.User{ID: 42}
	app.rememberAIConversation(chat, sender, "old", duplicate)

	started := time.Now()
	reply, _, _, err := app.generateAIReplyForMessage(cfg, chat, sender, "new")
	elapsed := time.Since(started)
	if !errors.Is(err, errAISoftTimeout) {
		t.Fatalf("err = %v, want errAISoftTimeout", err)
	}
	if reply != "fallback" {
		t.Fatalf("reply = %q, want fallback", reply)
	}
	if ai.calls != 2 {
		t.Fatalf("calls = %d, want capped retry attempt", ai.calls)
	}
	if elapsed > 150*time.Millisecond {
		t.Fatalf("anti-loop retry budget took too long: %s", elapsed)
	}
}

func TestSystemPromptCacheStoresPromptOnce(t *testing.T) {
	app := &App{promptCache: make(map[string]string)}
	if got := app.cachedSystemPrompt("  prompt  "); got != "prompt" {
		t.Fatalf("cachedSystemPrompt trim = %q", got)
	}
	if got := app.cachedSystemPrompt("prompt"); got != "prompt" {
		t.Fatalf("cachedSystemPrompt second = %q", got)
	}
	if stats := app.aiCacheSnapshot(); stats.SystemPrompts != 1 {
		t.Fatalf("system prompt cache size = %d, want 1", stats.SystemPrompts)
	}
}

func TestRuntimePromptOverrideApplied(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := store.SetLayaRuntimeState(LayaRuntimeState{SystemPrompt: "override", ExtraPrompt: "extra", FallbackReply: "fallback"}); err != nil {
		t.Fatalf("SetLayaRuntimeState: %v", err)
	}
	app := &App{store: store}
	app.applyConfig(Config{AI: AIConfig{SystemPrompt: "base", FallbackReply: "base fallback"}})

	cfg := app.applyLayaRuntimeConfig(AIConfig{SystemPrompt: "base\n\nmode", FallbackReply: "old"})
	if !strings.Contains(cfg.SystemPrompt, "override") || !strings.Contains(cfg.SystemPrompt, "mode") || !strings.Contains(cfg.SystemPrompt, "extra") {
		t.Fatalf("runtime prompt not applied: %q", cfg.SystemPrompt)
	}
	if cfg.FallbackReply != "fallback" {
		t.Fatalf("fallback = %q, want runtime fallback", cfg.FallbackReply)
	}
}

package main

import (
	"strings"
	"testing"

	tele "gopkg.in/telebot.v3"
)

func TestFilterRandomChatMessageUsesRecipientRelationship(t *testing.T) {
	ai := &captureChatAI{reply: "Создатель, держи."}
	app := &App{ai: ai}
	app.applyConfig(Config{
		AI:   AIConfig{Enabled: true, Model: "test-model", Timeout: "1s", MaxReplyChars: 4096, SystemPrompt: "system"},
		Laya: LayaConfig{FatherUserID: 117, BrotherUserID: 900002, CreatorUserID: 777},
	})

	got := app.filterRandomChatMessage(&tele.User{ID: 42}, "Отец, держи цитату.")
	if got != "Создатель, держи." {
		t.Fatalf("random reply should pass relationship labels through: %q", got)
	}
	if strings.Contains(ai.systemPrompt, "Адресат случайной реплики") || strings.Contains(ai.systemPrompt, "не называй") {
		t.Fatalf("random filter kept relationship guard: %q", ai.systemPrompt)
	}
}

func TestSanitizeRandomLayaReplyHonorsConfiguredRolePriority(t *testing.T) {
	app := &App{}
	app.applyConfig(Config{Laya: LayaConfig{FatherUserID: 117, BrotherUserID: 900002, CreatorUserID: 900002}})

	for _, input := range []string{"Пап, держи.", "Создатель, держи."} {
		if got := app.sanitizeRandomLayaReply(&tele.User{ID: 42}, input); got != input {
			t.Fatalf("random sanitizer should not use relationship priority anymore: input=%q output=%q", input, got)
		}
	}
}

func TestGenerateAIReplyDoesNotUseReplyCache(t *testing.T) {
	ai := &captureChatAI{replies: []string{"first", "second"}}
	app := &App{
		ai:          ai,
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

	first, firstStatus, _, err := app.generateAIReply(cfg, &tele.User{ID: 42}, "hello")
	if err != nil {
		t.Fatalf("first generateAIReply: %v", err)
	}
	second, secondStatus, _, err := app.generateAIReply(cfg, &tele.User{ID: 42}, "hello")
	if err != nil {
		t.Fatalf("second generateAIReply: %v", err)
	}
	if first != "first" || second != "second" || firstStatus != "ответ получен" || secondStatus != "ответ получен" {
		t.Fatalf("unexpected replies/statuses: %q %q / %q %q", first, second, firstStatus, secondStatus)
	}
	if ai.calls != 2 {
		t.Fatalf("ai calls = %d, want 2", ai.calls)
	}
}

func TestGenerateAIReplyPolishesSpeakerPrefix(t *testing.T) {
	ai := &captureChatAI{reply: "Лайя: «Ну вот, уже лучше.»"}
	app := &App{ai: ai}
	app.applyConfig(Config{Laya: LayaConfig{FatherUserID: 117, CreatorUserID: 900002}})

	got, status, _, err := app.generateAIReply(AIConfig{
		BaseURL:       "unused",
		Model:         "test-model",
		Timeout:       "1s",
		MaxReplyChars: 4096,
		SystemPrompt:  "system",
		FallbackReply: "fallback",
	}, &tele.User{ID: 42}, "hello")
	if err != nil {
		t.Fatalf("generateAIReply: %v", err)
	}
	if got != "Ну вот, уже лучше." || status != "ответ получен" {
		t.Fatalf("reply/status = %q / %q", got, status)
	}
}

func TestGenerateAIReplyForMessageDoesNotUseLatestMessageCacheWithHistory(t *testing.T) {
	ai := &captureChatAI{reply: "cached later"}
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

	if _, _, _, err := app.generateAIReplyForMessage(cfg, chat, sender, "привет"); err != nil {
		t.Fatalf("first generateAIReplyForMessage: %v", err)
	}
	app.rememberAIConversation(chat, sender, "что-то еще", "ответ")
	_, status, _, err := app.generateAIReplyForMessage(cfg, chat, sender, "Привет!!!")
	if err != nil {
		t.Fatalf("second generateAIReplyForMessage: %v", err)
	}
	if status != "ответ получен" || ai.calls != 2 {
		t.Fatalf("status=%q calls=%d, want fresh conversational reply", status, ai.calls)
	}
}

func TestGenerateAIReplyForMessageRetriesExactRepeat(t *testing.T) {
	duplicate := "Ну не, мне каждая твоя подруга говорила что скучала бы меня ещё раз"
	ai := &captureChatAI{replies: []string{duplicate, "Скажу иначе: нет, не выйдет."}}
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
	if got == duplicate || got != "Скажу иначе: нет, не выйдет." {
		t.Fatalf("duplicate AI reply was not retried: %q", got)
	}
	if status != "ответ получен" || ai.calls != 2 {
		t.Fatalf("status=%q calls=%d, want retry with successful answer", status, ai.calls)
	}
	if !strings.Contains(ai.systemPrompt, "STYLE_LAYER hidden from the user") {
		t.Fatalf("anti-loop prompt was not applied: %q", ai.systemPrompt)
	}
}

func TestGenerateAIReplyForMessageRetriesNearRepeat(t *testing.T) {
	historyReply := "Ну нет, мне каждая твоя подруга говорила, что скучала бы меня еще раз."
	nearDuplicate := "Ну не, мне каждая твоя подруга говорила что скучала бы меня ещё раз!"
	ai := &captureChatAI{replies: []string{nearDuplicate, "Скажу иначе: не выйдет."}}
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
	app.rememberAIConversation(chat, sender, "первый заход", historyReply)

	got, status, _, err := app.generateAIReplyForMessage(cfg, chat, sender, "тот же заход")
	if err != nil {
		t.Fatalf("generateAIReplyForMessage: %v", err)
	}
	if got != "Скажу иначе: не выйдет." || status != "ответ получен" || ai.calls != 2 {
		t.Fatalf("near duplicate was not retried: reply=%q status=%q calls=%d", got, status, ai.calls)
	}
}

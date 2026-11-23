package main

import (
	"strings"
	"testing"

	tele "gopkg.in/telebot.v3"
)

func TestGenerateAIReplyForMessageAllowsRepeatedShortFactualAnswer(t *testing.T) {
	nameReply := "Лайетана Каталонес."
	ai := &captureChatAI{reply: nameReply}
	app := &App{
		ai:          ai,
		aiHistory:   make(map[string][]AIMessage),
		aiNoRepeat:  make(map[string][]string),
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
	app.rememberAIConversation(chat, sender, "как меня зовут", nameReply)

	got, status, _, err := app.generateAIReplyForMessage(cfg, chat, sender, "а меня как зовут?")
	if err != nil {
		t.Fatalf("generateAIReplyForMessage: %v", err)
	}
	if got != nameReply || status != "ответ получен" || ai.calls != 1 {
		t.Fatalf("short factual repeat should pass: reply=%q status=%q calls=%d", got, status, ai.calls)
	}
}

func TestGenerateAIReplyForMessageRetriesInternalMetaReply(t *testing.T) {
	metaReply := "Я поймала повтор и сбросила контекст. Сформулируй заново."
	fresh := "Лайетана на связи. Что разбираем?"
	ai := &captureChatAI{replies: []string{metaReply, fresh}}
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

	got, status, _, err := app.generateAIReplyForMessage(cfg, chat, sender, "Лайетана Каталонес?")
	if err != nil {
		t.Fatalf("generateAIReplyForMessage: %v", err)
	}
	if got != fresh || status != "ответ получен" || ai.calls != 2 {
		t.Fatalf("internal meta reply should be retried: reply=%q status=%q calls=%d", got, status, ai.calls)
	}
	if !strings.Contains(ai.systemPrompt, "STYLE_LAYER hidden from the user") {
		t.Fatalf("private retry prompt missing: %q", ai.systemPrompt)
	}
}

func TestGenerateAIReplyForMessageAllowsOrdinaryLoopWords(t *testing.T) {
	reply := "В коде цикл for тут нормальный, а сброс настроек лучше оставить на кнопку."
	ai := &captureChatAI{reply: reply}
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

	got, status, _, err := app.generateAIReplyForMessage(cfg, chat, sender, "как сделать цикл и сброс настроек?")
	if err != nil {
		t.Fatalf("generateAIReplyForMessage: %v", err)
	}
	if got != reply || status != "ответ получен" || ai.calls != 1 {
		t.Fatalf("ordinary loop/reset words should pass: reply=%q status=%q calls=%d", got, status, ai.calls)
	}
}

func TestGenerateAIReplyForMessageAllowsOrdinaryDifferentReplyWithHistory(t *testing.T) {
	historyReply := "Поняла, тебя зовут Лайетана Каталонес, звучит как паспорт для драмы."
	reply := "Ахах, имя запомнила: Лайетана Каталонес, можешь не сверлить мне лоб."
	ai := &captureChatAI{reply: reply}
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
	app.rememberAIConversation(chat, sender, "как меня зовут", historyReply)

	got, status, _, err := app.generateAIReplyForMessage(cfg, chat, sender, "ну и как меня зовут?")
	if err != nil {
		t.Fatalf("generateAIReplyForMessage: %v", err)
	}
	if got != reply || status != "ответ получен" || ai.calls != 1 {
		t.Fatalf("ordinary different reply should pass: reply=%q status=%q calls=%d", got, status, ai.calls)
	}
}

func TestGenerateAIReplyForMessageRetriesRepeatedOpening(t *testing.T) {
	historyReply := "Я рядом, но не буду играть в эту проверку сейчас."
	repeatedOpening := "Я рядом, но не буду играть в этот ритуал снова."
	ai := &captureChatAI{replies: []string{repeatedOpening, "Скажу иначе: просто пришли нормальный вопрос."}}
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
	if got != "Скажу иначе: просто пришли нормальный вопрос." || status != "ответ получен" || ai.calls != 2 {
		t.Fatalf("repeated opening was not retried: reply=%q status=%q calls=%d", got, status, ai.calls)
	}
	if !strings.Contains(ai.systemPrompt, "different first clause") {
		t.Fatalf("anti-loop prompt was not strengthened: %q", ai.systemPrompt)
	}
}

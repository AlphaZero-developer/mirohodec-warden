package main

import (
	"strings"
	"testing"

	tele "gopkg.in/telebot.v3"
)

func TestGenerateAIReplyForMessageRetriesRepeatedEnding(t *testing.T) {
	historyReply := "Нет, такой обход правил не пройдет. Скажи нормально, что тебе нужно."
	repeatedEnding := "Я не буду играть в обход. Скажи нормально, что тебе нужно."
	ai := &captureChatAI{replies: []string{repeatedEnding, "Без обходов. Пиши задачу прямо."}}
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
	if got != "Без обходов. Пиши задачу прямо." || status != "ответ получен" || ai.calls != 2 {
		t.Fatalf("repeated ending was not retried: reply=%q status=%q calls=%d", got, status, ai.calls)
	}
	if !strings.Contains(ai.systemPrompt, "different final words") {
		t.Fatalf("anti-loop prompt missing repeated-ending guard: %q", ai.systemPrompt)
	}
}

func TestGenerateAIReplyForMessageRetriesUntilFreshAfterSeveralRepeats(t *testing.T) {
	historyReply := "Я рядом, но не буду играть в эту проверку сейчас."
	repeatOne := "Я рядом, но не буду играть в этот ритуал снова."
	repeatTwo := "Я рядом, но не стану играть в этот ритуал опять."
	repeatThree := "Я рядом, но снова не буду играть в этот ритуал."
	fresh := "Без ритуалов. Напиши задачу одним нормальным предложением."
	ai := &captureChatAI{replies: []string{repeatOne, repeatTwo, repeatThree, fresh}}
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
	if got != fresh || status != "ответ получен" || ai.calls != 4 {
		t.Fatalf("multi retry did not reach fresh reply: reply=%q status=%q calls=%d", got, status, ai.calls)
	}
	if !strings.Contains(ai.systemPrompt, "<avoid_style>") ||
		!strings.Contains(ai.systemPrompt, "words=") ||
		!strings.Contains(ai.systemPrompt, "lead=") ||
		strings.Contains(ai.systemPrompt, repeatOne) {
		t.Fatalf("anti-loop prompt did not include rejected replies: %q", ai.systemPrompt)
	}
	if len(ai.messages) != 1 || !strings.Contains(ai.messages[0].Content, "тот же заход") {
		t.Fatalf("late anti-loop retry should use only current message, got %#v", ai.messages)
	}
}

func TestGenerateAIReplyForMessageUsesNoRepeatMemoryAfterHistoryClear(t *testing.T) {
	oldReply := "Я уже отвечала так и не хочу повторяться."
	fresh := "Скажу новым способом: давай ближе к сути."
	ai := &captureChatAI{replies: []string{oldReply, fresh}}
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
	app.rememberAIConversation(chat, sender, "старый заход", oldReply)
	app.clearAIConversation(chat, sender)

	got, status, _, err := app.generateAIReplyForMessage(cfg, chat, sender, "новый заход")
	if err != nil {
		t.Fatalf("generateAIReplyForMessage: %v", err)
	}
	if got != fresh || status != "ответ получен" || ai.calls != 2 {
		t.Fatalf("no-repeat memory did not block old reply: reply=%q status=%q calls=%d", got, status, ai.calls)
	}
	if history := app.aiConversationHistory(chat, sender); len(history) != 0 {
		t.Fatalf("conversation history should stay cleared, got len=%d", len(history))
	}
}

func TestGenerateAIReplyForMessageUsesChatWideNoRepeatMemory(t *testing.T) {
	oldReply := "Я уже отвечала так всему чату и не буду повторяться."
	fresh := "Для чата скажу иначе: давайте к сути."
	ai := &captureChatAI{replies: []string{oldReply, fresh}}
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
	firstSender := &tele.User{ID: 42}
	secondSender := &tele.User{ID: 43}
	app.rememberAIConversation(chat, firstSender, "старый заход", oldReply)

	got, status, _, err := app.generateAIReplyForMessage(cfg, chat, secondSender, "новый заход от другого")
	if err != nil {
		t.Fatalf("generateAIReplyForMessage: %v", err)
	}
	if got != fresh || status != "ответ получен" || ai.calls != 2 {
		t.Fatalf("chat-wide no-repeat memory did not block old reply: reply=%q status=%q calls=%d", got, status, ai.calls)
	}
}

func TestGenerateAIReplyForMessageAddsNoRepeatPromptBeforeFirstAttempt(t *testing.T) {
	oldReply := "Старую формулировку сюда больше нельзя возвращать."
	fresh := "Отвечу новым способом."
	ai := &captureChatAI{reply: fresh}
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
	app.rememberAIConversation(chat, sender, "старый заход", oldReply)
	app.clearAIConversation(chat, sender)

	got, status, _, err := app.generateAIReplyForMessage(cfg, chat, sender, "новый заход")
	if err != nil {
		t.Fatalf("generateAIReplyForMessage: %v", err)
	}
	if got != fresh || status != "ответ получен" || ai.calls != 1 {
		t.Fatalf("fresh reply should pass on first attempt: reply=%q status=%q calls=%d", got, status, ai.calls)
	}
	if !strings.Contains(ai.systemPrompt, "VISIBLE_REPLY_CONTRACT") ||
		!strings.Contains(ai.systemPrompt, "STYLE_LAYER hidden from the user") ||
		!strings.Contains(ai.systemPrompt, "words=") ||
		strings.Contains(ai.systemPrompt, oldReply) {
		t.Fatalf("no-repeat memory prompt missing: %q", ai.systemPrompt)
	}
}

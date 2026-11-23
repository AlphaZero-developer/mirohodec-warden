package main

import (
	"strings"
	"testing"

	tele "gopkg.in/telebot.v3"
)

func TestGenerateAIReplyForTelegramMessageCarriesBridgeThroughChain(t *testing.T) {
	ai := &captureChatAI{reply: "ответ цепочки"}
	app := &App{
		ai:           ai,
		aiHistory:    make(map[string][]AIMessage),
		aiNoRepeat:   make(map[string][]string),
		aiReplyLinks: make(map[string]layaAIReplyLink),
		promptCache:  make(map[string]string),
	}
	app.applyConfig(Config{Laya: LayaConfig{FatherUserID: 117, CreatorUserID: 900002}})
	chat := &tele.Chat{ID: -1001, Type: tele.ChatSuperGroup, Title: "Главный чат"}
	source := &tele.User{ID: 99, Username: "mira", FirstName: "Мира"}
	entrant := &tele.User{ID: 42, Username: "kira_dev", FirstName: "Кира"}
	third := &tele.User{ID: 43, Username: "third", FirstName: "Третий"}

	app.rememberAIConversation(chat, source, "Лайя, помни кодовое", "Контекст: синий ключ лежит в архиве.")
	app.rememberLayaAIReplyMessage(chat, &tele.Message{ID: 55, ThreadID: 7, Text: "Контекст: синий ключ лежит в архиве."}, source)
	firstBridge := &tele.Message{
		ID:       56,
		ThreadID: 7,
		Sender:   entrant,
		Text:     "Подключаюсь к этой истории.",
		ReplyTo:  &tele.Message{ID: 55, Sender: &tele.User{ID: 777, IsBot: true, FirstName: "Лайя"}},
	}
	if _, _, _, err := app.generateAIReplyForTelegramMessage(AIConfig{
		BaseURL:       "unused",
		Model:         "test-model",
		Timeout:       "1s",
		MaxReplyChars: 4096,
		SystemPrompt:  "system",
		FallbackReply: "fallback",
	}, chat, entrant, firstBridge, firstBridge.Text); err != nil {
		t.Fatalf("first bridge generateAIReplyForTelegramMessage: %v", err)
	}
	app.rememberAIConversation(chat, entrant, firstBridge.Text, "ответ цепочки")
	app.rememberLayaAIReplyMessage(chat, &tele.Message{ID: 60, ThreadID: 7, Text: "ответ цепочки"}, entrant)

	ai.messages = nil
	ai.systemPrompt = ""
	ai.reply = "ok"
	secondBridge := &tele.Message{
		ID:       61,
		ThreadID: 7,
		Sender:   third,
		Text:     "Я тоже отвечаю на цепочку.",
		ReplyTo:  &tele.Message{ID: 60, Sender: &tele.User{ID: 777, IsBot: true, FirstName: "Лайя"}},
	}
	if _, _, _, err := app.generateAIReplyForTelegramMessage(AIConfig{
		BaseURL:       "unused",
		Model:         "test-model",
		Timeout:       "1s",
		MaxReplyChars: 4096,
		SystemPrompt:  "system",
		FallbackReply: "fallback",
	}, chat, third, secondBridge, secondBridge.Text); err != nil {
		t.Fatalf("second bridge generateAIReplyForTelegramMessage: %v", err)
	}
	if !strings.Contains(ai.messages[0].Content, "Мира [user_id=99]: Лайя, помни кодовое") ||
		!strings.Contains(ai.messages[0].Content, "синий ключ лежит в архиве") ||
		!strings.Contains(ai.messages[0].Content, "Кира [user_id=42]: Подключаюсь к этой истории") ||
		!strings.Contains(ai.messages[0].Content, "Конкретное сообщение цепочки, на которое сейчас отвечают: ответ цепочки") {
		t.Fatalf("bridge chain context missing original or intermediate history:\n%s", ai.messages[0].Content)
	}
}

func TestGenerateAIReplyForTelegramMessageCarriesBridgeThroughUserCarrier(t *testing.T) {
	ai := &captureChatAI{reply: "ok"}
	app := &App{
		ai:           ai,
		aiHistory:    make(map[string][]AIMessage),
		aiNoRepeat:   make(map[string][]string),
		aiReplyLinks: make(map[string]layaAIReplyLink),
		promptCache:  make(map[string]string),
	}
	app.applyConfig(Config{Laya: LayaConfig{FatherUserID: 117, CreatorUserID: 900002}})
	chat := &tele.Chat{ID: -1001, Type: tele.ChatSuperGroup, Title: "Главный чат"}
	source := &tele.User{ID: 99, Username: "mira", FirstName: "Мира"}
	entrant := &tele.User{ID: 42, Username: "kira_dev", FirstName: "Кира"}
	third := &tele.User{ID: 43, Username: "third", FirstName: "Третий"}

	app.rememberAIConversation(chat, source, "Лайя, помни тему", "Контекст: зеленая папка на столе.")
	app.rememberLayaAIReplyMessage(chat, &tele.Message{ID: 55, ThreadID: 7, Text: "Контекст: зеленая папка на столе."}, source)
	carrier := &tele.Message{
		ID:       56,
		ThreadID: 7,
		Sender:   entrant,
		Text:     "Я вклиниваюсь в историю про папку.",
		ReplyTo:  &tele.Message{ID: 55, Sender: &tele.User{ID: 777, IsBot: true, FirstName: "Лайя"}},
	}
	if _, _, _, err := app.generateAIReplyForTelegramMessage(AIConfig{
		BaseURL:       "unused",
		Model:         "test-model",
		Timeout:       "1s",
		MaxReplyChars: 4096,
		SystemPrompt:  "system",
		FallbackReply: "fallback",
	}, chat, entrant, carrier, carrier.Text); err != nil {
		t.Fatalf("carrier generateAIReplyForTelegramMessage: %v", err)
	}

	ai.messages = nil
	replyToCarrier := &tele.Message{
		ID:       57,
		ThreadID: 7,
		Sender:   third,
		Text:     "Я отвечаю уже не Лайе, а сообщению Киры.",
		ReplyTo:  carrier,
	}
	if _, _, _, err := app.generateAIReplyForTelegramMessage(AIConfig{
		BaseURL:       "unused",
		Model:         "test-model",
		Timeout:       "1s",
		MaxReplyChars: 4096,
		SystemPrompt:  "system",
		FallbackReply: "fallback",
	}, chat, third, replyToCarrier, replyToCarrier.Text); err != nil {
		t.Fatalf("reply-to-carrier generateAIReplyForTelegramMessage: %v", err)
	}
	got := ai.messages[0].Content
	for _, want := range []string{
		"Уже подключенный контекст предыдущей цепочки",
		"зеленая папка на столе",
		"Кира [user_id=42]: Я вклиниваюсь в историю про папку.",
		"Конкретное сообщение цепочки, на которое сейчас отвечают: Я вклиниваюсь в историю про папку.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("carrier bridge missing %q in:\n%s", want, got)
		}
	}
}

func TestTrimLayaBridgeSnapshotDeduplicatesBeforeLimit(t *testing.T) {
	var messages []AIMessage
	for i := 0; i < maxLayaBridgeSnapshot+4; i++ {
		messages = append(messages, AIMessage{Role: "user", Content: "одинаковый вход"})
	}
	messages = append(messages,
		AIMessage{Role: "user", Content: "важный старый контекст"},
		AIMessage{Role: "assistant", Content: "важный ответ Лайи"},
	)

	got := trimLayaBridgeSnapshot(messages)
	if len(got) != 3 {
		t.Fatalf("snapshot len = %d, want 3: %#v", len(got), got)
	}
	if got[1].Content != "важный старый контекст" || got[2].Content != "важный ответ Лайи" {
		t.Fatalf("snapshot lost important tail: %#v", got)
	}
}

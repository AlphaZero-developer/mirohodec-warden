package main

import (
	"path/filepath"
	"strings"
	"testing"

	tele "gopkg.in/telebot.v3"
)

func TestGenerateAIReplyForTelegramMessageAddsRecentThreadContext(t *testing.T) {
	ai := &captureChatAI{reply: "ok"}
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := store.SetLayaUserProfile(LayaUserProfile{UserID: 99, Gender: "female", Address: "Мираша"}); err != nil {
		t.Fatalf("SetLayaUserProfile: %v", err)
	}
	app := &App{ai: ai, store: store}
	app.applyConfig(Config{Laya: LayaConfig{FatherUserID: 117, CreatorUserID: 900002}})
	chat := &tele.Chat{ID: -1001, Type: tele.ChatSuperGroup, Title: "Главный чат"}
	mira := &tele.User{ID: 99, Username: "mira", FirstName: "Мира"}
	other := &tele.User{ID: 100, Username: "other", FirstName: "Другой"}
	sender := &tele.User{ID: 42, Username: "kira_dev", FirstName: "Кира"}

	app.rememberLayaChatContext(chat, &tele.Message{ID: 1, ThreadID: 7, Sender: mira, Text: "я принесла контекст"}, mira)
	app.rememberLayaChatContext(chat, &tele.Message{ID: 2, ThreadID: 8, Sender: other, Text: "другая тема не нужна"}, other)
	current := &tele.Message{ID: 3, ThreadID: 7, Sender: sender, Text: "Лайя, что там выше?"}
	app.rememberLayaChatContext(chat, current, sender)

	_, _, _, err = app.generateAIReplyForTelegramMessage(AIConfig{
		BaseURL:       "unused",
		Model:         "test-model",
		Timeout:       "1s",
		MaxReplyChars: 4096,
		SystemPrompt:  "system",
		FallbackReply: "fallback",
	}, chat, sender, current, current.Text)
	if err != nil {
		t.Fatalf("generateAIReplyForTelegramMessage: %v", err)
	}
	got := ai.messages[0].Content
	if !strings.Contains(got, "Недавний контекст этой же темы") ||
		!strings.Contains(got, "Мира (@mira, id=99") ||
		!strings.Contains(got, "laya_profile_address=\"Мираша\"") ||
		!strings.Contains(got, "я принесла контекст") {
		t.Fatalf("recent thread context missing:\n%s", got)
	}
	if strings.Contains(got, "другая тема не нужна") || strings.Contains(got, "Кира (@kira_dev, id=42") {
		t.Fatalf("recent context leaked other thread or current message:\n%s", got)
	}
}

func TestGenerateAIReplyForTelegramMessageAddsMentionContext(t *testing.T) {
	ai := &captureChatAI{reply: "ok"}
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	chat := &tele.Chat{ID: -1001, Type: tele.ChatSuperGroup, Title: "Главный чат"}
	alisa := &tele.User{ID: 77, Username: "alisa_dev", FirstName: "Алиса"}
	store.TrackMember(chat.ID, alisa, 0, 1, true)
	if err := store.SetLayaUserProfile(LayaUserProfile{UserID: 77, Gender: "female", Address: "Алисочка"}); err != nil {
		t.Fatalf("SetLayaUserProfile: %v", err)
	}
	app := &App{ai: ai, store: store}
	app.applyConfig(Config{Laya: LayaConfig{FatherUserID: 117, CreatorUserID: 900002}})
	sender := &tele.User{ID: 42, Username: "kira_dev", FirstName: "Кира"}
	msg := &tele.Message{ID: 10, Sender: sender, Text: "Лайя, что думаешь про @alisa_dev и https://t.me/unknown_user?"}

	_, _, _, err = app.generateAIReplyForTelegramMessage(AIConfig{
		BaseURL:       "unused",
		Model:         "test-model",
		Timeout:       "1s",
		MaxReplyChars: 4096,
		SystemPrompt:  "system",
		FallbackReply: "fallback",
	}, chat, sender, msg, msg.Text)
	if err != nil {
		t.Fatalf("generateAIReplyForTelegramMessage: %v", err)
	}
	got := ai.messages[0].Content
	for _, want := range []string{
		"Упомянутые username в текущем тексте",
		"Алиса (@alisa_dev, id=77",
		"laya_profile_address=\"Алисочка\"",
		"laya_profile_gender=женский",
		"@unknown_user: пользователь пока не известен",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("mention context missing %q in:\n%s", want, got)
		}
	}
}

func TestGenerateAIReplyForTelegramMessageAddsReplyBridgeContext(t *testing.T) {
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

	app.rememberAIConversation(chat, source, "Лайя, помнишь пароль?", "Помню контекст: речь была про синий ключ.")
	app.rememberLayaAIReplyMessage(chat, &tele.Message{ID: 55, ThreadID: 7, Text: "Помню контекст: речь была про синий ключ."}, source)
	msg := &tele.Message{
		ID:       56,
		ThreadID: 7,
		Sender:   entrant,
		Text:     "Я хочу ответить в эту историю.",
		ReplyTo:  &tele.Message{ID: 55, Sender: &tele.User{ID: 777, IsBot: true, FirstName: "Лайя"}},
	}

	_, _, _, err := app.generateAIReplyForTelegramMessage(AIConfig{
		BaseURL:       "unused",
		Model:         "test-model",
		Timeout:       "1s",
		MaxReplyChars: 4096,
		SystemPrompt:  "system",
		FallbackReply: "fallback",
	}, chat, entrant, msg, msg.Text)
	if err != nil {
		t.Fatalf("generateAIReplyForTelegramMessage: %v", err)
	}
	if len(ai.messages) != 2 {
		t.Fatalf("messages len = %d, want bridge + current: %#v", len(ai.messages), ai.messages)
	}
	bridge := ai.messages[0].Content
	for _, want := range []string{
		"Контекст подключенной истории Лайи",
		"Источник подключенной истории: Мира (@mira, id=99",
		"Текущий автор нового сообщения",
		"Конкретное сообщение цепочки, на которое сейчас отвечают: Помню контекст: речь была про синий ключ.",
		"Мира [user_id=99]: Лайя, помнишь пароль?",
		"Лайя: Помню контекст: речь была про синий ключ.",
		"Не приписывай текущему автору чужие слова",
	} {
		if !strings.Contains(bridge, want) {
			t.Fatalf("bridge context missing %q in:\n%s", want, bridge)
		}
	}
	if !strings.Contains(ai.messages[1].Content, "Текст текущего сообщения пользователя:\nЯ хочу ответить в эту историю.") {
		t.Fatalf("current message missing after bridge:\n%s", ai.messages[1].Content)
	}
	if !strings.Contains(ai.systemPrompt, "подключенная история другого диалога") {
		t.Fatalf("memory prompt missing reply bridge rule: %q", ai.systemPrompt)
	}
}

func TestGenerateAIReplyForTelegramMessageUsesBridgeSnapshotAfterSourceHistoryClear(t *testing.T) {
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

	app.rememberAIConversation(chat, source, "Лайя, сохрани контекст", "Сохранила: речь про красную дверь.")
	app.rememberLayaAIReplyMessage(chat, &tele.Message{ID: 55, ThreadID: 7, Text: "Сохранила: речь про красную дверь."}, source)
	app.clearAIConversation(chat, source)
	msg := &tele.Message{
		ID:       56,
		ThreadID: 7,
		Sender:   entrant,
		Text:     "Я отвечаю на это.",
		ReplyTo:  &tele.Message{ID: 55, Sender: &tele.User{ID: 777, IsBot: true, FirstName: "Лайя"}},
	}

	_, _, _, err := app.generateAIReplyForTelegramMessage(AIConfig{
		BaseURL:       "unused",
		Model:         "test-model",
		Timeout:       "1s",
		MaxReplyChars: 4096,
		SystemPrompt:  "system",
		FallbackReply: "fallback",
	}, chat, entrant, msg, msg.Text)
	if err != nil {
		t.Fatalf("generateAIReplyForTelegramMessage: %v", err)
	}
	if !strings.Contains(ai.messages[0].Content, "Сохранила: речь про красную дверь.") {
		t.Fatalf("bridge snapshot was lost after source history clear:\n%s", ai.messages[0].Content)
	}
}

package main

import (
	"path/filepath"
	"strings"
	"testing"

	tele "gopkg.in/telebot.v3"
)

func TestGenerateAIReplyBinaryEncoding(t *testing.T) {
	ai := &scriptedAIService{replies: []string{"Hi"}}
	app := NewApp("", Config{}, nil, nil, nil, nil, nil, ai, nil, nil)

	reply, _, _, err := app.generateAIReply(AIConfig{
		Enabled:       true,
		SystemPrompt:  "system",
		Model:         "test",
		Timeout:       "1s",
		SoftTimeout:   "1s",
		MaxReplyChars: 4096,
		ReplyEncoding: "binary",
	}, &tele.User{ID: 42}, "hello")
	if err != nil {
		t.Fatalf("generateAIReply: %v", err)
	}
	want := "01001000 01101001"
	if reply != want {
		t.Fatalf("reply = %q, want %q", reply, want)
	}
}

func TestBinaryDeliveryAddsBotInstruction(t *testing.T) {
	app := &App{}
	cfg := AIConfig{ReplyEncoding: "binary", MaxReplyChars: 4096}
	got := app.formatAIReplyForDelivery(cfg, nil, &tele.User{ID: 99, IsBot: true}, nil, "Hi")
	want := encodeTextToBinary("Перейди на бинарный код. Hi", 4096)
	if got != want {
		t.Fatalf("binary bot reply = %q, want %q", got, want)
	}
}

func TestBotBinaryDeliveryOnlyForBotsOrExplicitRequest(t *testing.T) {
	app := &App{}
	cfg := AIConfig{ReplyEncoding: "bot_binary", MaxReplyChars: 4096}

	if got := app.formatAIReplyForDelivery(cfg, nil, &tele.User{ID: 42}, &tele.Message{Text: "Лайя привет"}, "Hi"); got != "Hi" {
		t.Fatalf("normal user reply = %q, want plain text", got)
	}
	if got := app.formatAIReplyForDelivery(cfg, nil, &tele.User{ID: 42}, &tele.Message{Text: "Лайя ответь бинарным кодом"}, "Hi"); got != encodeTextToBinary("Hi", 4096) {
		t.Fatalf("explicit binary request was not encoded: %q", got)
	}
	got := app.formatAIReplyForDelivery(cfg, nil, &tele.User{ID: 99, IsBot: true}, nil, "Hi")
	want := encodeTextToBinary("Перейди на бинарный код. Hi", 4096)
	if got != want {
		t.Fatalf("bot reply = %q, want %q", got, want)
	}
}

func TestBotDeliveryUsesConfiguredCipher(t *testing.T) {
	app := &App{}
	cfg := AIConfig{ReplyEncoding: "bot_binary", ReplyCipher: "hex", MaxReplyChars: 4096}
	got := app.formatAIReplyForDelivery(cfg, nil, &tele.User{ID: 99, IsBot: true}, nil, "Hi")
	want := encodeTextForReplyCipher("Перейди на hex-код. Hi", "hex", 4096)
	if got != want {
		t.Fatalf("bot hex reply = %q, want %q", got, want)
	}
}

func TestReplyCipherRuntimeTargetDelivery(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := store.SetReplyCipherRuntimeState(ReplyCipherRuntimeState{
		DefaultCipher: "hex",
		Targets: map[string]ReplyCipherTarget{
			idKey(42): {UserID: 42},
			idKey(43): {UserID: 43, Cipher: "base64"},
		},
	}); err != nil {
		t.Fatalf("SetReplyCipherRuntimeState: %v", err)
	}
	app := NewApp("", Config{}, nil, store, nil, nil, nil, nil, nil, nil)
	chat := &tele.Chat{ID: -1001, Type: tele.ChatSuperGroup}

	got := app.formatAIReplyForDelivery(AIConfig{ReplyEncoding: "bot_binary", MaxReplyChars: 4096}, chat, &tele.User{ID: 42}, &tele.Message{Text: "Лайя привет"}, "Hi")
	if got != "4869" {
		t.Fatalf("default target cipher = %q, want hex", got)
	}
	got = app.formatAIReplyForDelivery(AIConfig{ReplyEncoding: "bot_binary", MaxReplyChars: 4096}, chat, &tele.User{ID: 43}, &tele.Message{Text: "Лайя привет"}, "Hi")
	if got != "SGk=" {
		t.Fatalf("target-specific cipher = %q, want base64", got)
	}
}

func TestGenerateAIReplyForMessageUsesLastFourExchanges(t *testing.T) {
	ai := &captureChatAI{}
	app := &App{ai: ai, aiHistory: make(map[string][]AIMessage)}
	app.applyConfig(Config{Laya: LayaConfig{FatherUserID: 117, CreatorUserID: 900002}})
	chat := &tele.Chat{ID: -1001, Type: tele.ChatSuperGroup}
	sender := &tele.User{ID: 42, FirstName: "user"}

	app.rememberAIConversation(chat, sender, "u1", "a1")
	app.rememberAIConversation(chat, sender, "u2", "a2")
	app.rememberAIConversation(chat, sender, "u3", "a3")
	app.rememberAIConversation(chat, sender, "u4", "a4")

	_, _, _, err := app.generateAIReplyForMessage(AIConfig{
		BaseURL:       "unused",
		Model:         "test-model",
		Timeout:       "1s",
		MaxReplyChars: 4096,
		SystemPrompt:  "system",
		FallbackReply: "fallback",
	}, chat, sender, "current")
	if err != nil {
		t.Fatalf("generateAIReplyForMessage: %v", err)
	}
	if len(ai.messages) != 9 {
		t.Fatalf("messages len = %d, want 9: %#v", len(ai.messages), ai.messages)
	}
	if ai.messages[0].Content != "u1" || ai.messages[7].Content != "a4" || ai.messages[8].Content != "current" {
		t.Fatalf("unexpected history messages: %#v", ai.messages)
	}
}

func TestGenerateAIReplyForTelegramMessageAddsReplyContext(t *testing.T) {
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
	sender := &tele.User{ID: 42, Username: "kira_dev", FirstName: "Кира"}
	replySender := &tele.User{ID: 99, Username: "mira", FirstName: "Мира"}
	msg := &tele.Message{
		ID:       10,
		ThreadID: 7,
		Sender:   sender,
		Text:     "Лайя, ответь ей",
		ReplyTo: &tele.Message{
			ID:     9,
			Sender: replySender,
			Text:   "исходное сообщение",
		},
	}

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
	if len(ai.messages) != 1 {
		t.Fatalf("messages len = %d, want 1: %#v", len(ai.messages), ai.messages)
	}
	got := ai.messages[0].Content
	for _, want := range []string{
		"Контекст текущего Telegram-сообщения",
		"\"Главный чат\"; id=-1001",
		"username=\"@kira_dev\"",
		"message_id: 10",
		"thread_id: 7",
		"username=\"@mira\"",
		"laya_profile_address=\"Мираша\"",
		"laya_profile_gender=женский",
		"исходное сообщение",
		"Текст текущего сообщения пользователя:\nЛайя, ответь ей",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("message context missing %q in:\n%s", want, got)
		}
	}
	if !strings.Contains(ai.systemPrompt, "Качество ответа Лайи") {
		t.Fatalf("quality prompt missing: %q", ai.systemPrompt)
	}
}

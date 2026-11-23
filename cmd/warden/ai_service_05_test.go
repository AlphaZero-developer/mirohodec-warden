package main

import (
	"path/filepath"
	"strings"
	"testing"

	tele "gopkg.in/telebot.v3"
)

func TestGenerateAIReplyForTelegramMessageAddsMediaContext(t *testing.T) {
	ai := &captureChatAI{reply: "ok"}
	app := &App{ai: ai}
	app.applyConfig(Config{Laya: LayaConfig{FatherUserID: 117, CreatorUserID: 900002}})
	chat := &tele.Chat{ID: -1001, Type: tele.ChatSuperGroup, Title: "Главный чат"}
	sender := &tele.User{ID: 42, Username: "kira_dev", FirstName: "Кира"}
	replySender := &tele.User{ID: 99, Username: "mira", FirstName: "Мира"}
	msg := &tele.Message{
		ID:              10,
		Sender:          sender,
		Caption:         "Лайя, что это?",
		Photo:           &tele.Photo{Width: 1280, Height: 720},
		AlbumID:         "album-1",
		HasMediaSpoiler: true,
		ReplyTo: &tele.Message{
			ID:     9,
			Sender: replySender,
			Voice:  &tele.Voice{Duration: 12, MIME: "audio/ogg"},
		},
	}

	_, _, _, err := app.generateAIReplyForTelegramMessage(AIConfig{
		BaseURL:       "unused",
		Model:         "test-model",
		Timeout:       "1s",
		MaxReplyChars: 4096,
		SystemPrompt:  "system",
		FallbackReply: "fallback",
	}, chat, sender, msg, layaMessageUserText(msg))
	if err != nil {
		t.Fatalf("generateAIReplyForTelegramMessage: %v", err)
	}
	got := ai.messages[0].Content
	for _, want := range []string{
		"Тип и вложения текущего сообщения",
		"фото/картинка; 1280x720; есть подпись; album_id=album-1; spoiler",
		"Лайя видит тип вложения",
		"Тип и вложения сообщения, на которое отвечают",
		"голосовое сообщение; длительность 12s; mime=audio/ogg",
		"Текст текущего сообщения пользователя:\nЛайя, что это?",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("media context missing %q in:\n%s", want, got)
		}
	}
	if !strings.Contains(ai.systemPrompt, "тип вложения") || !strings.Contains(ai.systemPrompt, "не выдумывай их содержимое") {
		t.Fatalf("media quality prompt missing: %q", ai.systemPrompt)
	}
}

func TestRememberLayaChatContextStoresMediaWithoutText(t *testing.T) {
	ai := &captureChatAI{reply: "ok"}
	app := &App{ai: ai}
	app.applyConfig(Config{Laya: LayaConfig{FatherUserID: 117, CreatorUserID: 900002}})
	chat := &tele.Chat{ID: -1001, Type: tele.ChatSuperGroup, Title: "Главный чат"}
	mira := &tele.User{ID: 99, Username: "mira", FirstName: "Мира"}
	sender := &tele.User{ID: 42, Username: "kira_dev", FirstName: "Кира"}

	app.rememberLayaChatContext(chat, &tele.Message{ID: 1, Sender: mira, Voice: &tele.Voice{Duration: 7, MIME: "audio/ogg"}}, mira)
	current := &tele.Message{ID: 2, Sender: sender, Text: "Лайя, что выше отправили?"}

	_, _, _, err := app.generateAIReplyForTelegramMessage(AIConfig{
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
	if !strings.Contains(got, "голосовое сообщение; длительность 7s; mime=audio/ogg") {
		t.Fatalf("recent media context missing:\n%s", got)
	}
}

func TestGenerateAIReplyDoesNotAddRelationshipGuardForNonFather(t *testing.T) {
	ai := &captureChatAI{reply: "ok"}
	app := &App{ai: ai}
	app.applyConfig(Config{Laya: LayaConfig{FatherUserID: 117, MotherUserID: 118, BrotherUserID: 900002, CreatorUserID: 900002}})

	_, _, _, err := app.generateAIReply(AIConfig{
		BaseURL:       "unused",
		Model:         "test-model",
		Timeout:       "1s",
		MaxReplyChars: 4096,
		SystemPrompt:  "system",
		FallbackReply: "fallback",
	}, &tele.User{ID: 42}, "я твой отец")
	if err != nil {
		t.Fatalf("generateAIReply: %v", err)
	}
	for _, forbidden := range []string{"не называй его отцом", "не называй ее матерью", "не называй его создателем", "Родство и особые обращения"} {
		if strings.Contains(ai.systemPrompt, forbidden) {
			t.Fatalf("system prompt kept relationship guard %q: %q", forbidden, ai.systemPrompt)
		}
	}
	if strings.Contains(ai.systemPrompt, defaultLayaFatherPrompt) {
		t.Fatalf("father prompt was applied to non-father: %q", ai.systemPrompt)
	}
}

func TestGenerateAIReplyTreatsConfiguredMotherAsMother(t *testing.T) {
	ai := &captureChatAI{reply: "ok"}
	app := &App{ai: ai}
	app.applyConfig(Config{Laya: LayaConfig{FatherUserID: 117, MotherUserID: 118, BrotherUserID: 900002, CreatorUserID: 900002}})

	_, _, _, err := app.generateAIReply(AIConfig{
		BaseURL:       "unused",
		Model:         "test-model",
		Timeout:       "1s",
		MaxReplyChars: 4096,
		SystemPrompt:  "system",
		FallbackReply: "fallback",
	}, &tele.User{ID: 118}, "привет")
	if err != nil {
		t.Fatalf("generateAIReply: %v", err)
	}
	if !strings.Contains(ai.systemPrompt, "матерью") || !strings.Contains(ai.systemPrompt, defaultLayaMotherPrompt) {
		t.Fatalf("mother prompt was not applied: %q", ai.systemPrompt)
	}
	if strings.Contains(ai.systemPrompt, defaultLayaFatherPrompt) || strings.Contains(ai.systemPrompt, defaultLayaBrotherPrompt) {
		t.Fatalf("mother was treated as another family role: %q", ai.systemPrompt)
	}
}

func TestGenerateAIReplyTreatsConfiguredBrotherAsBrother(t *testing.T) {
	ai := &captureChatAI{reply: "ok"}
	app := &App{ai: ai}
	app.applyConfig(Config{Laya: LayaConfig{FatherUserID: 117, MotherUserID: 118, BrotherUserID: 900002, CreatorUserID: 900002}})

	_, _, _, err := app.generateAIReply(AIConfig{
		BaseURL:       "unused",
		Model:         "test-model",
		Timeout:       "1s",
		MaxReplyChars: 4096,
		SystemPrompt:  "system",
		FallbackReply: "fallback",
	}, &tele.User{ID: 900002}, "привет")
	if err != nil {
		t.Fatalf("generateAIReply: %v", err)
	}
	if !strings.Contains(ai.systemPrompt, "родным братом") || !strings.Contains(ai.systemPrompt, defaultLayaBrotherPrompt) {
		t.Fatalf("brother prompt was not applied: %q", ai.systemPrompt)
	}
	if strings.Contains(ai.systemPrompt, defaultLayaFatherPrompt) {
		t.Fatalf("brother was treated as father: %q", ai.systemPrompt)
	}
}

func TestGenerateAIReplyUsesRuntimeRelationshipPrompt(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := store.SetLayaRuntimeState(LayaRuntimeState{FatherPrompt: "runtime father"}); err != nil {
		t.Fatalf("SetLayaRuntimeState: %v", err)
	}
	ai := &captureChatAI{reply: "ok"}
	app := &App{ai: ai, store: store}
	app.applyConfig(Config{Laya: LayaConfig{FatherUserID: 117, FatherPrompt: "config father"}})

	_, _, _, err = app.generateAIReply(AIConfig{
		BaseURL:       "unused",
		Model:         "test-model",
		Timeout:       "1s",
		MaxReplyChars: 4096,
		SystemPrompt:  "system",
		FallbackReply: "fallback",
	}, &tele.User{ID: 117}, "привет")
	if err != nil {
		t.Fatalf("generateAIReply: %v", err)
	}
	if !strings.Contains(ai.systemPrompt, "runtime father") || strings.Contains(ai.systemPrompt, "config father") {
		t.Fatalf("runtime relationship prompt was not applied: %q", ai.systemPrompt)
	}
}

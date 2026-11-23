package main

import (
	"testing"

	tele "gopkg.in/telebot.v3"
)

func TestLayaRelationshipSanitizersArePassthrough(t *testing.T) {
	app := &App{}
	app.applyConfig(Config{Laya: LayaConfig{
		FatherUserID:  117,
		MotherUserID:  118,
		BrotherUserID: 119,
		CreatorUserID: 120,
	}})

	for _, input := range []string{
		"Пап, я рядом.",
		"Создатель, держи.",
		"Это твой брат пишет код, а не ты.",
		"Шейхом тебя не называю. Я слушаю.",
	} {
		if got := app.sanitizeLayaDirectReply(&tele.User{ID: 42}, input); got != input {
			t.Fatalf("direct sanitizer changed reply: input=%q output=%q", input, got)
		}
		if got := app.sanitizeRandomLayaReply(&tele.User{ID: 42}, input); got != input {
			t.Fatalf("random sanitizer changed reply: input=%q output=%q", input, got)
		}
	}
}

func TestRandomLayaRelationshipSafetyPromptDisabled(t *testing.T) {
	app := &App{}
	if got := app.randomLayaRelationshipSafetyPrompt(&tele.User{ID: 42}); got != "" {
		t.Fatalf("relationship safety prompt should be disabled, got %q", got)
	}
}

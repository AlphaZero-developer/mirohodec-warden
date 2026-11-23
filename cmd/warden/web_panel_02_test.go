package main

import (
	"context"
	wardenweb "mirohodec-warden/internal/web"
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func TestWebPanelServiceFAQQuotesPhotos(t *testing.T) {
	service, app := newWebPanelTestService(t)
	ctx := context.Background()

	if err := service.AddFAQ(ctx, wardenFAQInput("hello; hi", "answer")); err != nil {
		t.Fatalf("AddFAQ: %v", err)
	}
	faqID := findFAQID(t, app, "answer")
	if err := service.UpdateFAQ(ctx, wardenFAQInputWithID(faqID, "hello", "updated", false)); err != nil {
		t.Fatalf("UpdateFAQ: %v", err)
	}
	if err := service.DeleteFAQ(ctx, faqID); err != nil {
		t.Fatalf("DeleteFAQ: %v", err)
	}

	if err := service.AddQuote(ctx, wardenQuoteInput("soon", "Laya", "tag")); err != nil {
		t.Fatalf("AddQuote: %v", err)
	}
	quoteID := findQuoteID(t, app, "soon")
	if err := service.UpdateQuote(ctx, wardenQuoteInputWithID(quoteID, "later", "Laya", "tag2")); err != nil {
		t.Fatalf("UpdateQuote: %v", err)
	}
	if err := service.DeleteQuote(ctx, quoteID); err != nil {
		t.Fatalf("DeleteQuote: %v", err)
	}

	if err := service.AddPhoto(ctx, wardenPhotoInput("file-id", "unique-id", "caption", "tag")); err != nil {
		t.Fatalf("AddPhoto: %v", err)
	}
	photos := app.photos.List("tag")
	if len(photos) != 1 || photos[0].FileID != "file-id" {
		t.Fatalf("unexpected photos: %#v", photos)
	}
	if err := service.DeletePhoto(ctx, photos[0].ID); err != nil {
		t.Fatalf("DeletePhoto: %v", err)
	}
	if got := app.photos.List("tag"); len(got) != 0 {
		t.Fatalf("photo should be deleted: %#v", got)
	}
}

func TestWebPanelServiceLayaControl(t *testing.T) {
	service, app := newWebPanelTestService(t)
	ctx := context.Background()
	app.cachedSystemPrompt("old prompt")

	if err := service.UpdateLayaControl(ctx, wardenweb.LayaControlInput{
		SystemPrompt:       "runtime prompt",
		ExtraPrompt:        "runtime mood",
		FallbackReply:      "runtime fallback",
		Model:              "runtime-model",
		Timeout:            "11s",
		SoftTimeout:        "3s",
		MaxTokens:          123,
		MaxReplyChars:      321,
		LogConversations:   "on",
		FatherUserID:       111,
		MotherUserID:       222,
		BrotherUserID:      333,
		CreatorUserID:      444,
		FatherPrompt:       "runtime father",
		MotherPrompt:       "runtime mother",
		BrotherPrompt:      "runtime brother",
		CreatorPrompt:      "runtime creator",
		Relationships:      "555 | парень сестры | отвечает теплее к знакомому семьи\n666 | брат 2",
		MoodMode:           "custom",
		MoodPrompt:         "runtime mood prompt",
		DirectInsultMode:   "local",
		SafetyCooldownMode: "off",
		SafetyPauseReply:   "pause {remaining}",
		TagReactionRules:   "художник => иди рисуй",
		RetryAttempts:      2,
		RetryBackoff:       "700ms",
	}); err != nil {
		t.Fatalf("UpdateLayaControl: %v", err)
	}
	state := app.store.LayaRuntimeState()
	if state.SystemPrompt != "runtime prompt" || state.ExtraPrompt != "runtime mood" || state.FallbackReply != "runtime fallback" ||
		state.Model != "runtime-model" || state.Timeout != "11s" || state.SoftTimeout != "3s" || state.MaxTokens != 123 || state.MaxReplyChars != 321 || state.LogConversationsMode != "on" ||
		state.FatherUserID != 111 || state.MotherUserID != 222 || state.BrotherUserID != 333 || state.CreatorUserID != 444 ||
		state.FatherPrompt != "runtime father" || state.MotherPrompt != "runtime mother" || state.BrotherPrompt != "runtime brother" || state.CreatorPrompt != "runtime creator" ||
		len(state.Relationships) != 2 || state.Relationships[0].UserID != 555 || state.Relationships[0].Role != "парень сестры" || state.Relationships[1].Role != "брат 2" ||
		state.MoodMode != "custom" || state.MoodPrompt != "runtime mood prompt" || state.DirectInsultMode != "local" ||
		state.SafetyCooldownMode != "off" || state.SafetyPauseReply != "pause {remaining}" || state.TagReactionRules != "художник => иди рисуй" ||
		state.RetryAttempts != 2 || state.RetryBackoff != "700ms" {
		t.Fatalf("unexpected laya state: %#v", state)
	}
	page, err := service.LayaControl(ctx)
	if err != nil {
		t.Fatalf("LayaControl: %v", err)
	}
	if page.RuntimeSystemPrompt != "runtime prompt" || page.FallbackReply != "runtime fallback" ||
		page.RuntimeModel != "runtime-model" || page.ActiveModel != "runtime-model" || page.ActiveTimeout != "11s" || page.ActiveSoftTimeout != "3s" ||
		page.ActiveMaxTokens != 123 || page.ActiveMaxReplyChars != 321 || !page.ActiveLogConversations ||
		page.RuntimeFatherUserID != 111 || page.ActiveFatherUserID != 111 || page.ActiveMotherUserID != 222 || page.ActiveBrotherUserID != 333 || page.ActiveCreatorUserID != 444 ||
		page.RuntimeFatherPrompt != "runtime father" || page.ActiveFatherPrompt != "runtime father" || page.ConfigFatherPrompt == "" ||
		page.RuntimeMotherPrompt != "runtime mother" || page.RuntimeBrotherPrompt != "runtime brother" || page.RuntimeCreatorPrompt != "runtime creator" ||
		page.RuntimeRelationships == "" || len(page.ActiveRelationships) != 2 || page.ActiveRelationships[1].UserID != 666 ||
		page.MoodMode != "custom" || page.MoodPrompt != "runtime mood prompt" || page.DirectInsultMode != "local" ||
		page.SafetyCooldownMode != "off" || page.SafetyPauseReply != "pause {remaining}" || page.TagReactionRules != "художник => иди рисуй" ||
		page.RetryAttempts != 2 || page.RetryBackoff != "700ms" ||
		page.Cache.SystemPrompts != 0 {
		t.Fatalf("unexpected laya page: %#v", page)
	}
}

func TestWebPanelServiceAILogs(t *testing.T) {
	service, app := newWebPanelTestService(t)
	ctx := context.Background()

	app.recordAIWebLog(
		aiWebLogMeta{Source: "Telegram chat", Stage: "primary", ChatID: -1001, ChatTitle: "Main", ThreadID: 7, MessageID: 55},
		AIConfig{BaseURL: "https://example.test", Model: "test-model", MaxTokens: 123, MaxReplyChars: 321, SystemPrompt: "system"},
		&tele.User{ID: 42, Username: "mira", FirstName: "Mira"},
		[]AIMessage{{Role: "user", Content: "payload"}},
		"hello",
		"raw reply",
		"final reply",
		"ответ получен",
		15*time.Millisecond,
		nil,
	)

	page, err := service.AILogs(ctx)
	if err != nil {
		t.Fatalf("AILogs: %v", err)
	}
	if page.Total != 1 || page.Info != 1 || page.Warn != 0 || page.Error != 0 || page.OK != 1 || page.Errors != 0 || page.AverageTime != "15ms" || len(page.Rows) != 1 {
		t.Fatalf("unexpected ai log page stats: %#v", page)
	}
	row := page.Rows[0]
	if row.UserInput != "hello" || row.SystemPrompt != "system" || row.Messages[0].Content != "payload" ||
		row.RawReply != "raw reply" || row.FinalReply != "final reply" || row.Model != "test-model" || row.Chat != "Main (-1001)" || row.Level != "info" {
		t.Fatalf("unexpected ai log row: %#v", row)
	}
	if err := service.ClearAILogs(ctx); err != nil {
		t.Fatalf("ClearAILogs: %v", err)
	}
	page, err = service.AILogs(ctx)
	if err != nil {
		t.Fatalf("AILogs after clear: %v", err)
	}
	if page.Total != 0 || len(page.Rows) != 0 {
		t.Fatalf("ai logs were not cleared: %#v", page)
	}
}

func TestParseLayaRelationshipsText(t *testing.T) {
	rels, err := parseLayaRelationshipsText("42 | парень сестры | держи семейный тон\n# comment\n43 | отец 2")
	if err != nil {
		t.Fatalf("parseLayaRelationshipsText: %v", err)
	}
	if len(rels) != 2 || rels[0].UserID != 42 || rels[0].Role != "парень сестры" || rels[0].Prompt != "держи семейный тон" || rels[1].Role != "отец 2" {
		t.Fatalf("unexpected relationships: %#v", rels)
	}
	if _, err := parseLayaRelationshipsText("bad line"); err == nil {
		t.Fatal("expected error for malformed line")
	}
}

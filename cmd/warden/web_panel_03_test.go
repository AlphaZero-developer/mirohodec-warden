package main

import (
	"context"
	wardenweb "mirohodec-warden/internal/web"
	"testing"
)

func TestWebPanelServiceDiscussionUpdate(t *testing.T) {
	service, app := newWebPanelTestService(t)
	ctx := context.Background()
	app.applyConfig(Config{Discussion: DiscussionAutoReplyConfig{
		Enabled:      false,
		ChatID:       0,
		SourceChatID: 0,
		Text:         "config text",
	}})

	if err := service.UpdateDiscussion(ctx, wardenweb.DiscussionInput{
		Enabled:            true,
		ChatID:             -100111,
		SourceChatID:       -100222,
		StrictChecks:       true,
		Text:               "runtime text",
		ChatButtonText:     "Chat",
		ChatButtonURL:      "https://t.me/test_chat",
		FollowUpText:       "follow",
		FollowUpButtonText: "Boosty",
		FollowUpButtonURL:  "https://boosty.to/test",
		Pin:                true,
	}); err != nil {
		t.Fatalf("UpdateDiscussion: %v", err)
	}
	state := app.store.DiscussionRuntimeState()
	if !state.Enabled || state.ChatID != -100111 || state.SourceChatID != -100222 || !state.StrictChecks ||
		state.Text != "runtime text" || state.ChatButtonText != "Chat" || state.ChatButtonURL != "https://t.me/test_chat" ||
		state.FollowUpText != "follow" || state.FollowUpButtonText != "Boosty" ||
		state.FollowUpButtonURL != "https://boosty.to/test" || !state.Pin || state.UpdatedUnix == 0 {
		t.Fatalf("unexpected discussion state: %#v", state)
	}
	active := app.activeDiscussionConfig()
	if !active.Enabled || active.ChatID != -100111 || active.SourceChatID != -100222 ||
		active.Text != "runtime text" || active.ChatButtonText != "Chat" || active.ChatButtonURL != "https://t.me/test_chat" {
		t.Fatalf("runtime discussion config was not active: %#v", active)
	}
	page, err := service.Discussion(ctx)
	if err != nil {
		t.Fatalf("Discussion: %v", err)
	}
	if page.Active.ChatID != -100111 || page.Active.ChatButtonText != "Chat" || page.Config.Text != "config text" || page.Updated == "" {
		t.Fatalf("unexpected discussion page: %#v", page)
	}
}

func TestWebPanelServiceContentControl(t *testing.T) {
	service, app := newWebPanelTestService(t)
	ctx := context.Background()
	cfg := app.config()
	cfg.FAQ = FAQConfig{Enabled: true, AutoReply: false, RequireMention: true, EntryCooldown: "1h", UserCooldown: "5m"}
	cfg.Quotes = QuoteConfig{Enabled: true, AutoReply: true, AutoReplyChance: 0.05, AutoReplyCooldown: "30m", ActiveUserWindow: "7d", PhotoChance: 0.1}
	cfg.Photos = PhotoConfig{Enabled: true, AutoReply: false, AutoReplyChance: 0.2, Cooldown: "30s", Triggers: []string{"config"}, LocalDir: "config/photos", LocalExtensions: []string{".jpg"}}
	cfg.Ideas = IdeaConfig{Enabled: true, UserCooldown: "10m", ChatCooldown: "1m", MaxLength: 2000}
	app.applyConfig(cfg)

	if err := service.UpdateContentControl(ctx, wardenweb.ContentControlInput{
		FAQEnabledMode:          "off",
		FAQAutoReplyMode:        "on",
		FAQRequireMentionMode:   "off",
		FAQEntryCooldown:        "2h",
		FAQUserCooldown:         "7m",
		QuotesEnabledMode:       "on",
		QuotesAutoReplyMode:     "off",
		QuotesAutoReplyChance:   "25%",
		QuotesAutoReplyCooldown: "11m",
		QuotesActiveUserWindow:  "3d",
		QuotesPhotoChance:       "0.4",
		PhotosEnabledMode:       "on",
		PhotosAutoReplyMode:     "on",
		PhotosAutoReplyChance:   "15",
		PhotosCooldown:          "45s",
		PhotosTriggers:          "art\npaint",
		PhotosLocalDir:          "runtime/photos",
		PhotosLocalExtensions:   ".webp,.png",
		IdeasEnabledMode:        "off",
		IdeasUserCooldown:       "20m",
		IdeasChatCooldown:       "4m",
		IdeasMaxLength:          777,
	}); err != nil {
		t.Fatalf("UpdateContentControl: %v", err)
	}
	state := app.store.ContentRuntimeState()
	if state.FAQEnabledMode != "off" || state.FAQAutoReplyMode != "on" || state.FAQEntryCooldown != "2h" ||
		state.QuotesAutoReplyChance != "25%" || state.PhotosTriggers != "art\npaint" ||
		state.IdeasMaxLength != 777 || state.UpdatedUnix == 0 {
		t.Fatalf("unexpected content state: %#v", state)
	}
	faqCfg := app.activeFAQConfig()
	quoteCfg := app.activeQuoteConfig()
	photoCfg := app.activePhotoConfig()
	ideaCfg := app.activeIdeaConfig()
	if faqCfg.Enabled || !faqCfg.AutoReply || faqCfg.RequireMention || faqCfg.EntryCooldown != "2h" ||
		!quoteCfg.Enabled || quoteCfg.AutoReply || quoteCfg.AutoReplyChance != 0.25 || quoteCfg.ActiveUserWindow != "3d" || quoteCfg.PhotoChance != 0.4 ||
		!photoCfg.Enabled || !photoCfg.AutoReply || photoCfg.AutoReplyChance != 0.15 || len(photoCfg.Triggers) != 2 || photoCfg.LocalDir != "runtime/photos" ||
		ideaCfg.Enabled || ideaCfg.UserCooldown != "20m" || ideaCfg.MaxLength != 777 {
		t.Fatalf("runtime content config was not active: faq=%#v quotes=%#v photos=%#v ideas=%#v", faqCfg, quoteCfg, photoCfg, ideaCfg)
	}
	page, err := service.ContentControl(ctx)
	if err != nil {
		t.Fatalf("ContentControl: %v", err)
	}
	if page.FAQ.EnabledMode != "off" || page.FAQ.ActiveAutoReply != true ||
		page.Quotes.ActiveAutoReplyChance != "25.00%" || page.Photos.ActiveAutoReplyChance != "15.00%" ||
		page.Photos.ActiveTriggers != "art, paint" || page.Ideas.ActiveMaxLength != 777 || page.Updated == "" {
		t.Fatalf("unexpected content page: %#v", page)
	}
}

func wardenGroupInput(chatID int64, title string, topicID int) wardenweb.GroupInput {
	return wardenweb.GroupInput{
		ChatID:            chatID,
		Title:             title,
		Enabled:           true,
		RehabTopicID:      topicID,
		SpamNoRehabAction: spamNoRehabMuteOnDelete,
	}
}

func wardenRehabInput(chatID, userID int64) wardenweb.RehabInput {
	return wardenweb.RehabInput{ChatID: chatID, UserID: userID, Duration: "1h", Reason: "test"}
}

func wardenFAQInput(patterns, answer string) wardenweb.FAQInput {
	return wardenweb.FAQInput{Patterns: patterns, Answer: answer, Enabled: true}
}

func wardenFAQInputWithID(id, patterns, answer string, enabled bool) wardenweb.FAQInput {
	return wardenweb.FAQInput{ID: id, Patterns: patterns, Answer: answer, Enabled: enabled}
}

func wardenQuoteInput(text, author, tags string) wardenweb.QuoteInput {
	return wardenweb.QuoteInput{Text: text, Author: author, Tags: tags}
}

func wardenQuoteInputWithID(id, text, author, tags string) wardenweb.QuoteInput {
	return wardenweb.QuoteInput{ID: id, Text: text, Author: author, Tags: tags}
}

func wardenPhotoInput(fileID, uniqueID, caption, tags string) wardenweb.PhotoInput {
	return wardenweb.PhotoInput{FileID: fileID, UniqueID: uniqueID, Caption: caption, Tags: tags}
}

func findFAQID(t *testing.T, app *App, answer string) string {
	t.Helper()
	for _, rec := range app.faq.List() {
		if rec.Answer == answer {
			return rec.ID
		}
	}
	t.Fatalf("faq with answer %q not found", answer)
	return ""
}

func findQuoteID(t *testing.T, app *App, text string) string {
	t.Helper()
	for _, rec := range app.quotes.List("") {
		if rec.Text == text {
			return rec.ID
		}
	}
	t.Fatalf("quote with text %q not found", text)
	return ""
}

func webGroupHasSetting(group wardenweb.Group, section, name string) bool {
	for _, setting := range group.Settings {
		if setting.Section == section && setting.Name == name {
			return true
		}
	}
	return false
}

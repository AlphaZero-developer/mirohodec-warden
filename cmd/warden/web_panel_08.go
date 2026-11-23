package main

import (
	"context"
	"errors"
	"fmt"
	wardenweb "mirohodec-warden/internal/web"
	"strconv"
	"strings"
	"time"
)

func (s webPanelService) ContentControl(ctx context.Context) (wardenweb.ContentControlPage, error) {
	if s.app == nil || s.app.store == nil {
		return wardenweb.ContentControlPage{}, errors.New("app store is not ready")
	}
	cfg := s.app.config()
	state := normalizeContentRuntimeState(s.app.store.ContentRuntimeState())
	activeFAQ := s.app.activeFAQConfig()
	activeQuotes := s.app.activeQuoteConfig()
	activePhotos := s.app.activePhotoConfig()
	activeIdeas := s.app.activeIdeaConfig()
	updatedBy := strings.TrimSpace(state.UpdatedByName)
	if updatedBy == "" && state.UpdatedByID != 0 {
		updatedBy = strconv.FormatInt(state.UpdatedByID, 10)
	}
	rows := []wardenweb.GroupSetting{
		{Section: "FAQ", Name: "enabled", Value: webBool(activeFAQ.Enabled), Note: "Global switch before per-group overrides."},
		{Section: "FAQ", Name: "auto reply", Value: webBool(activeFAQ.AutoReply), Note: "Answers matched FAQ entries in normal chat messages."},
		{Section: "FAQ", Name: "require mention", Value: webBool(activeFAQ.RequireMention), Note: "When enabled, automatic FAQ answers require a Laya mention."},
		{Section: "FAQ", Name: "cooldowns", Value: fmt.Sprintf("entry=%s, user=%s", activeFAQ.EntryCooldown, activeFAQ.UserCooldown), Note: "Entry cooldown protects one FAQ answer; user cooldown protects one user."},
		{Section: "Quotes", Name: "enabled", Value: webBool(activeQuotes.Enabled), Note: "Global switch before per-group overrides."},
		{Section: "Quotes", Name: "auto reply", Value: webBool(activeQuotes.AutoReply), Note: "Random quote auto replies in active chats."},
		{Section: "Quotes", Name: "chance", Value: webPercent(activeQuotes.AutoReplyChance), Note: "Base probability before weekly events and temporary boosts."},
		{Section: "Quotes", Name: "cooldown", Value: activeQuotes.AutoReplyCooldown, Note: "Per-chat auto quote cooldown."},
		{Section: "Quotes", Name: "active user window", Value: activeQuotes.ActiveUserWindow, Note: "How far back Laya can pick a random active quote target."},
		{Section: "Quotes", Name: "photo chance", Value: webPercent(activeQuotes.PhotoChance), Note: "Chance to send a photo together with quote-like replies."},
		{Section: "Photos", Name: "enabled", Value: webBool(activePhotos.Enabled), Note: "Global switch before per-group overrides."},
		{Section: "Photos", Name: "auto reply", Value: webBool(activePhotos.AutoReply), Note: "Random photo replies on triggers or general photo auto mode."},
		{Section: "Photos", Name: "chance", Value: webPercent(activePhotos.AutoReplyChance), Note: "0 inherits the default behavior of always replying after a trigger."},
		{Section: "Photos", Name: "cooldown", Value: activePhotos.Cooldown, Note: "Per-chat auto photo cooldown."},
		{Section: "Photos", Name: "triggers", Value: webStringList(activePhotos.Triggers), Note: "Words that allow automatic photo replies; '-' in runtime clears the list."},
		{Section: "Photos", Name: "local dir", Value: activePhotos.LocalDir, Note: "Optional disk folder for random local images."},
		{Section: "Photos", Name: "local extensions", Value: webStringList(activePhotos.LocalExtensions), Note: "Allowed local image extensions."},
		{Section: "Ideas", Name: "enabled", Value: webBool(activeIdeas.Enabled), Note: "Controls job/donate/idea/bug/appeal forwarding to the owner log."},
		{Section: "Ideas", Name: "cooldowns", Value: fmt.Sprintf("user=%s, chat=%s", activeIdeas.UserCooldown, activeIdeas.ChatCooldown), Note: "Limits how often reports can be sent."},
		{Section: "Ideas", Name: "max length", Value: strconv.Itoa(activeIdeas.MaxLength), Note: "Maximum report length in runes."},
	}
	return wardenweb.ContentControlPage{
		Rows: rows,
		FAQ: wardenweb.ContentFAQSettings{
			EnabledMode:          state.FAQEnabledMode,
			AutoReplyMode:        state.FAQAutoReplyMode,
			RequireMentionMode:   state.FAQRequireMentionMode,
			EntryCooldown:        state.FAQEntryCooldown,
			UserCooldown:         state.FAQUserCooldown,
			ActiveEnabled:        activeFAQ.Enabled,
			ActiveAutoReply:      activeFAQ.AutoReply,
			ActiveRequireMention: activeFAQ.RequireMention,
			ActiveEntryCooldown:  activeFAQ.EntryCooldown,
			ActiveUserCooldown:   activeFAQ.UserCooldown,
			ConfigEnabled:        cfg.FAQ.Enabled,
			ConfigAutoReply:      cfg.FAQ.AutoReply,
			ConfigRequireMention: cfg.FAQ.RequireMention,
			ConfigEntryCooldown:  cfg.FAQ.EntryCooldown,
			ConfigUserCooldown:   cfg.FAQ.UserCooldown,
		},
		Quotes: wardenweb.ContentQuoteSettings{
			EnabledMode:             state.QuotesEnabledMode,
			AutoReplyMode:           state.QuotesAutoReplyMode,
			AutoReplyChance:         state.QuotesAutoReplyChance,
			AutoReplyCooldown:       state.QuotesAutoReplyCooldown,
			ActiveUserWindow:        state.QuotesActiveUserWindow,
			PhotoChance:             state.QuotesPhotoChance,
			ActiveEnabled:           activeQuotes.Enabled,
			ActiveAutoReply:         activeQuotes.AutoReply,
			ActiveAutoReplyChance:   webPercent(activeQuotes.AutoReplyChance),
			ActiveAutoReplyCooldown: activeQuotes.AutoReplyCooldown,
			ActiveActiveUserWindow:  activeQuotes.ActiveUserWindow,
			ActivePhotoChance:       webPercent(activeQuotes.PhotoChance),
			ConfigEnabled:           cfg.Quotes.Enabled,
			ConfigAutoReply:         cfg.Quotes.AutoReply,
			ConfigAutoReplyChance:   webPercent(cfg.Quotes.AutoReplyChance),
			ConfigAutoReplyCooldown: cfg.Quotes.AutoReplyCooldown,
			ConfigActiveUserWindow:  cfg.Quotes.ActiveUserWindow,
			ConfigPhotoChance:       webPercent(cfg.Quotes.PhotoChance),
		},
		Photos: wardenweb.ContentPhotoSettings{
			EnabledMode:           state.PhotosEnabledMode,
			AutoReplyMode:         state.PhotosAutoReplyMode,
			AutoReplyChance:       state.PhotosAutoReplyChance,
			Cooldown:              state.PhotosCooldown,
			Triggers:              state.PhotosTriggers,
			LocalDir:              state.PhotosLocalDir,
			LocalExtensions:       state.PhotosLocalExtensions,
			ActiveEnabled:         activePhotos.Enabled,
			ActiveAutoReply:       activePhotos.AutoReply,
			ActiveAutoReplyChance: webPercent(activePhotos.AutoReplyChance),
			ActiveCooldown:        activePhotos.Cooldown,
			ActiveTriggers:        webStringList(activePhotos.Triggers),
			ActiveLocalDir:        activePhotos.LocalDir,
			ActiveLocalExtensions: webStringList(activePhotos.LocalExtensions),
			ConfigEnabled:         cfg.Photos.Enabled,
			ConfigAutoReply:       cfg.Photos.AutoReply,
			ConfigAutoReplyChance: webPercent(cfg.Photos.AutoReplyChance),
			ConfigCooldown:        cfg.Photos.Cooldown,
			ConfigTriggers:        webStringList(cfg.Photos.Triggers),
			ConfigLocalDir:        cfg.Photos.LocalDir,
			ConfigLocalExtensions: webStringList(cfg.Photos.LocalExtensions),
		},
		Ideas: wardenweb.ContentIdeaSettings{
			EnabledMode:        state.IdeasEnabledMode,
			UserCooldown:       state.IdeasUserCooldown,
			ChatCooldown:       state.IdeasChatCooldown,
			MaxLength:          state.IdeasMaxLength,
			ActiveEnabled:      activeIdeas.Enabled,
			ActiveUserCooldown: activeIdeas.UserCooldown,
			ActiveChatCooldown: activeIdeas.ChatCooldown,
			ActiveMaxLength:    activeIdeas.MaxLength,
			ConfigEnabled:      cfg.Ideas.Enabled,
			ConfigUserCooldown: cfg.Ideas.UserCooldown,
			ConfigChatCooldown: cfg.Ideas.ChatCooldown,
			ConfigMaxLength:    cfg.Ideas.MaxLength,
		},
		Updated:   webFormatUnix(state.UpdatedUnix),
		UpdatedBy: updatedBy,
	}, nil
}

func (s webPanelService) UpdateContentControl(ctx context.Context, input wardenweb.ContentControlInput) error {
	if s.app == nil || s.app.store == nil {
		return errors.New("app store is not ready")
	}
	for name, value := range map[string]string{
		"quotes_auto_reply_chance": input.QuotesAutoReplyChance,
		"quotes_photo_chance":      input.QuotesPhotoChance,
		"photos_auto_reply_chance": input.PhotosAutoReplyChance,
	} {
		if err := validateContentChanceInput(name, value); err != nil {
			return err
		}
	}
	for name, value := range map[string]string{
		"faq_entry_cooldown":         input.FAQEntryCooldown,
		"faq_user_cooldown":          input.FAQUserCooldown,
		"quotes_auto_reply_cooldown": input.QuotesAutoReplyCooldown,
		"quotes_active_user_window":  input.QuotesActiveUserWindow,
		"photos_cooldown":            input.PhotosCooldown,
		"ideas_user_cooldown":        input.IdeasUserCooldown,
		"ideas_chat_cooldown":        input.IdeasChatCooldown,
	} {
		if err := validateContentDurationInput(name, value); err != nil {
			return err
		}
	}
	if input.IdeasMaxLength < 0 {
		return errors.New("ideas_max_length must be positive")
	}
	state := ContentRuntimeState{
		FAQEnabledMode:          input.FAQEnabledMode,
		FAQAutoReplyMode:        input.FAQAutoReplyMode,
		FAQRequireMentionMode:   input.FAQRequireMentionMode,
		FAQEntryCooldown:        input.FAQEntryCooldown,
		FAQUserCooldown:         input.FAQUserCooldown,
		QuotesEnabledMode:       input.QuotesEnabledMode,
		QuotesAutoReplyMode:     input.QuotesAutoReplyMode,
		QuotesAutoReplyChance:   input.QuotesAutoReplyChance,
		QuotesAutoReplyCooldown: input.QuotesAutoReplyCooldown,
		QuotesActiveUserWindow:  input.QuotesActiveUserWindow,
		QuotesPhotoChance:       input.QuotesPhotoChance,
		PhotosEnabledMode:       input.PhotosEnabledMode,
		PhotosAutoReplyMode:     input.PhotosAutoReplyMode,
		PhotosAutoReplyChance:   input.PhotosAutoReplyChance,
		PhotosCooldown:          input.PhotosCooldown,
		PhotosTriggers:          input.PhotosTriggers,
		PhotosLocalDir:          input.PhotosLocalDir,
		PhotosLocalExtensions:   input.PhotosLocalExtensions,
		IdeasEnabledMode:        input.IdeasEnabledMode,
		IdeasUserCooldown:       input.IdeasUserCooldown,
		IdeasChatCooldown:       input.IdeasChatCooldown,
		IdeasMaxLength:          input.IdeasMaxLength,
	}
	state = contentRuntimeUpdated(state, input.UpdatedByID, input.UpdatedByName, time.Now().UTC())
	return s.app.store.SetContentRuntimeState(state)
}

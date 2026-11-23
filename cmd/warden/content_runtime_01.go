package main

import (
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func normalizeContentRuntimeState(state ContentRuntimeState) ContentRuntimeState {
	state.FAQEnabledMode = normalizeLayaRuntimeMode(state.FAQEnabledMode, "on", "off")
	state.FAQAutoReplyMode = normalizeLayaRuntimeMode(state.FAQAutoReplyMode, "on", "off")
	state.FAQRequireMentionMode = normalizeLayaRuntimeMode(state.FAQRequireMentionMode, "on", "off")
	state.FAQEntryCooldown = strings.TrimSpace(state.FAQEntryCooldown)
	state.FAQUserCooldown = strings.TrimSpace(state.FAQUserCooldown)

	state.QuotesEnabledMode = normalizeLayaRuntimeMode(state.QuotesEnabledMode, "on", "off")
	state.QuotesAutoReplyMode = normalizeLayaRuntimeMode(state.QuotesAutoReplyMode, "on", "off")
	state.QuotesAutoReplyChance = normalizeContentChanceText(state.QuotesAutoReplyChance)
	state.QuotesAutoReplyCooldown = strings.TrimSpace(state.QuotesAutoReplyCooldown)
	state.QuotesActiveUserWindow = strings.TrimSpace(state.QuotesActiveUserWindow)
	state.QuotesPhotoChance = normalizeContentChanceText(state.QuotesPhotoChance)

	state.PhotosEnabledMode = normalizeLayaRuntimeMode(state.PhotosEnabledMode, "on", "off")
	state.PhotosAutoReplyMode = normalizeLayaRuntimeMode(state.PhotosAutoReplyMode, "on", "off")
	state.PhotosAutoReplyChance = normalizeContentChanceText(state.PhotosAutoReplyChance)
	state.PhotosCooldown = strings.TrimSpace(state.PhotosCooldown)
	state.PhotosTriggers = strings.TrimSpace(state.PhotosTriggers)
	state.PhotosLocalDir = strings.TrimSpace(state.PhotosLocalDir)
	state.PhotosLocalExtensions = strings.TrimSpace(state.PhotosLocalExtensions)

	state.IdeasEnabledMode = normalizeLayaRuntimeMode(state.IdeasEnabledMode, "on", "off")
	state.IdeasUserCooldown = strings.TrimSpace(state.IdeasUserCooldown)
	state.IdeasChatCooldown = strings.TrimSpace(state.IdeasChatCooldown)
	if state.IdeasMaxLength < 0 {
		state.IdeasMaxLength = 0
	}
	state.UpdatedByName = strings.TrimSpace(state.UpdatedByName)
	return state
}

func emptyContentRuntimeState(state ContentRuntimeState) bool {
	state = normalizeContentRuntimeState(state)
	return state.FAQEnabledMode == "" && state.FAQAutoReplyMode == "" && state.FAQRequireMentionMode == "" &&
		state.FAQEntryCooldown == "" && state.FAQUserCooldown == "" &&
		state.QuotesEnabledMode == "" && state.QuotesAutoReplyMode == "" && state.QuotesAutoReplyChance == "" &&
		state.QuotesAutoReplyCooldown == "" && state.QuotesActiveUserWindow == "" && state.QuotesPhotoChance == "" &&
		state.PhotosEnabledMode == "" && state.PhotosAutoReplyMode == "" && state.PhotosAutoReplyChance == "" &&
		state.PhotosCooldown == "" && state.PhotosTriggers == "" && state.PhotosLocalDir == "" && state.PhotosLocalExtensions == "" &&
		state.IdeasEnabledMode == "" && state.IdeasUserCooldown == "" && state.IdeasChatCooldown == "" && state.IdeasMaxLength == 0 &&
		state.UpdatedUnix == 0 && state.UpdatedByID == 0 && state.UpdatedByName == ""
}

func contentRuntimeUpdated(state ContentRuntimeState, byID int64, byName string, now time.Time) ContentRuntimeState {
	state = normalizeContentRuntimeState(state)
	state.UpdatedUnix = now.UTC().Unix()
	state.UpdatedByID = byID
	state.UpdatedByName = strings.TrimSpace(byName)
	return state
}

func (a *App) activeContentRuntimeState() ContentRuntimeState {
	if a == nil || a.store == nil {
		return ContentRuntimeState{}
	}
	return normalizeContentRuntimeState(a.store.ContentRuntimeState())
}

func (a *App) activeFAQConfig() FAQConfig {
	if a == nil {
		return FAQConfig{}
	}
	cfg := a.config().FAQ
	state := a.activeContentRuntimeState()
	cfg.Enabled = contentBoolMode(cfg.Enabled, state.FAQEnabledMode)
	cfg.AutoReply = contentBoolMode(cfg.AutoReply, state.FAQAutoReplyMode)
	cfg.RequireMention = contentBoolMode(cfg.RequireMention, state.FAQRequireMentionMode)
	if state.FAQEntryCooldown != "" {
		cfg.EntryCooldown = state.FAQEntryCooldown
	}
	if state.FAQUserCooldown != "" {
		cfg.UserCooldown = state.FAQUserCooldown
	}
	return cfg
}

func (a *App) activeFAQConfigForChat(chat *tele.Chat) FAQConfig {
	cfg := a.activeFAQConfig()
	if chat != nil && a != nil && a.store != nil {
		if settings, ok := a.store.ChatSettings(chat.ID); ok {
			cfg.Enabled = a.chatFeatureEnabled(chat.ID, cfg.Enabled, settings.FAQEnabledMode)
		}
	}
	return cfg
}

func (a *App) activeQuoteConfig() QuoteConfig {
	if a == nil {
		return QuoteConfig{}
	}
	cfg := a.config().Quotes
	state := a.activeContentRuntimeState()
	cfg.Enabled = contentBoolMode(cfg.Enabled, state.QuotesEnabledMode)
	cfg.AutoReply = contentBoolMode(cfg.AutoReply, state.QuotesAutoReplyMode)
	if state.QuotesAutoReplyChance != "" {
		cfg.AutoReplyChance = parseContentChance(state.QuotesAutoReplyChance, cfg.AutoReplyChance)
	}
	if state.QuotesAutoReplyCooldown != "" {
		cfg.AutoReplyCooldown = state.QuotesAutoReplyCooldown
	}
	if state.QuotesActiveUserWindow != "" {
		cfg.ActiveUserWindow = state.QuotesActiveUserWindow
	}
	if state.QuotesPhotoChance != "" {
		cfg.PhotoChance = parseContentChance(state.QuotesPhotoChance, cfg.PhotoChance)
	}
	return cfg
}

func (a *App) activeQuoteConfigForChat(chat *tele.Chat) QuoteConfig {
	cfg := a.activeQuoteConfig()
	if chat != nil && a != nil && a.store != nil {
		if settings, ok := a.store.ChatSettings(chat.ID); ok {
			cfg.Enabled = a.chatFeatureEnabled(chat.ID, cfg.Enabled, settings.QuotesEnabledMode)
		}
	}
	return cfg
}

func (a *App) activePhotoConfig() PhotoConfig {
	if a == nil {
		return PhotoConfig{}
	}
	cfg := a.config().Photos
	state := a.activeContentRuntimeState()
	cfg.Enabled = contentBoolMode(cfg.Enabled, state.PhotosEnabledMode)
	cfg.AutoReply = contentBoolMode(cfg.AutoReply, state.PhotosAutoReplyMode)
	if state.PhotosAutoReplyChance != "" {
		cfg.AutoReplyChance = parseContentChance(state.PhotosAutoReplyChance, cfg.AutoReplyChance)
	}
	if state.PhotosCooldown != "" {
		cfg.Cooldown = state.PhotosCooldown
	}
	if state.PhotosTriggers != "" {
		cfg.Triggers = contentRuntimeList(state.PhotosTriggers)
	}
	if state.PhotosLocalDir != "" {
		cfg.LocalDir = state.PhotosLocalDir
	}
	if state.PhotosLocalExtensions != "" {
		cfg.LocalExtensions = contentRuntimeList(state.PhotosLocalExtensions)
	}
	return cfg
}

func (a *App) activePhotoConfigForChat(chat *tele.Chat) PhotoConfig {
	cfg := a.activePhotoConfig()
	if chat != nil && a != nil && a.store != nil {
		if settings, ok := a.store.ChatSettings(chat.ID); ok {
			cfg.Enabled = a.chatFeatureEnabled(chat.ID, cfg.Enabled, settings.PhotosEnabledMode)
		}
	}
	return cfg
}

func (a *App) activeIdeaConfig() IdeaConfig {
	if a == nil {
		return IdeaConfig{}
	}
	cfg := a.config().Ideas
	state := a.activeContentRuntimeState()
	cfg.Enabled = contentBoolMode(cfg.Enabled, state.IdeasEnabledMode)
	if state.IdeasUserCooldown != "" {
		cfg.UserCooldown = state.IdeasUserCooldown
	}
	if state.IdeasChatCooldown != "" {
		cfg.ChatCooldown = state.IdeasChatCooldown
	}
	if state.IdeasMaxLength > 0 {
		cfg.MaxLength = state.IdeasMaxLength
	}
	return cfg
}

func contentBoolMode(base bool, mode string) bool {
	switch normalizeLayaRuntimeMode(mode, "on", "off") {
	case "on":
		return true
	case "off":
		return false
	default:
		return base
	}
}

package main

import (
	"strings"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

const spamNoRehabMuteOnDelete = "mute_on_delete"

func (a *App) effectiveChatSettings(chat *tele.Chat) ChatSettings {
	cfg := a.config()
	cfgTopicID, hasConfigTopic := a.configRehabTopicID(chat)
	if chat == nil {
		rehabTopicID := cfg.Chat.RehabTopicID
		persistedTopicID := 0
		if rehabTopicID > 0 {
			persistedTopicID = rehabTopicID
		}
		return ChatSettings{
			ChatID:                cfg.Chat.ID,
			Enabled:               cfg.Chat.ID != 0,
			RehabTopicID:          rehabTopicID,
			PersistedRehabTopicID: persistedTopicID,
			SpamNoRehabAction:     spamNoRehabMuteOnDelete,
			DebugCommands:         cfg.Telegram.ShowDebugCommands,
		}
	}
	if a.store != nil {
		if settings, ok := a.store.ChatSettings(chat.ID); ok {
			if settings.SpamNoRehabAction == "" {
				settings.SpamNoRehabAction = spamNoRehabMuteOnDelete
			}
			if settings.RehabTopicID == 0 && settings.PersistedRehabTopicID > 0 {
				settings.RehabTopicID = settings.PersistedRehabTopicID
				if a.log != nil {
					a.log.Info("rehab topic restored", zap.Int64("chat", chat.ID), zap.Int("topic", settings.RehabTopicID))
				}
				if err := a.store.UpsertChatSettings(settings); err != nil && a.log != nil {
					a.log.Warn("restore rehab topic failed", zap.Int64("chat", chat.ID), zap.Int("topic", settings.RehabTopicID), zap.Error(err))
				}
				return settings
			}
			if settings.RehabTopicID == 0 && hasConfigTopic {
				settings.RehabTopicID = cfgTopicID
				settings.PersistedRehabTopicID = cfgTopicID
				if err := a.store.UpsertChatSettings(settings); err != nil && a.log != nil {
					a.log.Warn("persist config rehab topic failed", zap.Int64("chat", chat.ID), zap.Int("topic", cfgTopicID), zap.Error(err))
				}
				return settings
			}
			if settings.RehabTopicID > 0 && settings.PersistedRehabTopicID == 0 {
				settings.PersistedRehabTopicID = settings.RehabTopicID
				if err := a.store.UpsertChatSettings(settings); err != nil && a.log != nil {
					a.log.Warn("persist known rehab topic failed", zap.Int64("chat", chat.ID), zap.Int("topic", settings.RehabTopicID), zap.Error(err))
				}
			}
			return settings
		}
	}
	settings := ChatSettings{
		ChatID:            chat.ID,
		Title:             strings.TrimSpace(chat.Title),
		Enabled:           true,
		SpamNoRehabAction: spamNoRehabMuteOnDelete,
		DebugCommands:     cfg.Telegram.ShowDebugCommands,
	}
	if hasConfigTopic {
		settings.RehabTopicID = cfgTopicID
		settings.PersistedRehabTopicID = cfgTopicID
	}
	return settings
}

func (a *App) rehabTopicID(chat *tele.Chat) int {
	settings := a.effectiveChatSettings(chat)
	return settings.RehabTopicID
}

func (a *App) chatFeatureEnabled(chatID int64, base bool, mode string) bool {
	switch normalizeLayaRuntimeMode(mode, "on", "off") {
	case "on":
		return true
	case "off":
		return false
	default:
		return base
	}
}

func (a *App) featureEnabledForChat(chat *tele.Chat, base bool, modeOf func(ChatSettings) string) bool {
	if chat == nil || a == nil || a.store == nil || modeOf == nil {
		return base
	}
	settings, ok := a.store.ChatSettings(chat.ID)
	if !ok {
		return base
	}
	return a.chatFeatureEnabled(chat.ID, base, modeOf(settings))
}

func (a *App) activeSpamConfigForChat(chatID int64) SpamConfig {
	cfg := a.config().Spam
	if a == nil || a.store == nil || chatID == 0 {
		return cfg
	}
	if settings, ok := a.store.ChatSettings(chatID); ok {
		cfg.Enabled = a.chatFeatureEnabled(chatID, cfg.Enabled, settings.SpamEnabledMode)
	}
	return cfg
}

func (a *App) activeWarnConfigForChat(chatID int64) WarnConfig {
	cfg := a.config().Warn
	if a == nil || a.store == nil || chatID == 0 {
		return cfg
	}
	if settings, ok := a.store.ChatSettings(chatID); ok {
		cfg.Enabled = a.chatFeatureEnabled(chatID, cfg.Enabled, settings.WarnEnabledMode)
		if strings.TrimSpace(settings.WarnExpireAfter) != "" {
			cfg.ExpireAfter = settings.WarnExpireAfter
		}
		if settings.WarnMuteThreshold > 0 {
			cfg.MuteThreshold = settings.WarnMuteThreshold
		}
		if strings.TrimSpace(settings.WarnMuteDuration) != "" {
			cfg.MuteDuration = settings.WarnMuteDuration
		}
		if settings.WarnRehabThreshold > 0 {
			cfg.RehabThreshold = settings.WarnRehabThreshold
		}
		if strings.TrimSpace(settings.WarnRehabDuration) != "" {
			cfg.RehabDuration = settings.WarnRehabDuration
		}
	}
	return cfg
}

func (a *App) activeKarmaConfigForChat(chatID int64) KarmaConfig {
	cfg := a.config().Karma
	if a == nil || a.store == nil || chatID == 0 {
		return cfg
	}
	if settings, ok := a.store.ChatSettings(chatID); ok {
		cfg.Enabled = a.chatFeatureEnabled(chatID, cfg.Enabled, settings.KarmaEnabledMode)
		if settings.KarmaMessagesPerPoint > 0 {
			cfg.ActivityMessagesPerPoint = settings.KarmaMessagesPerPoint
		}
		if settings.KarmaDailyLimit > 0 {
			cfg.ActivityDailyLimit = settings.KarmaDailyLimit
		}
		if strings.TrimSpace(settings.KarmaManualCooldown) != "" {
			cfg.ManualCooldown = settings.KarmaManualCooldown
		}
	}
	return cfg
}

func (a *App) aiEnabledForChat(chat *tele.Chat) bool {
	if !a.aiWhitelistedChat(chat) {
		return false
	}
	return a.featureEnabledForChat(chat, a.config().AI.Enabled, func(settings ChatSettings) string { return settings.AIEnabledMode })
}

func (a *App) aiWhitelistedChat(chat *tele.Chat) bool {
	if chat == nil || a == nil {
		return false
	}
	if chat.Type == tele.ChatPrivate {
		return true
	}
	cfg := a.config()
	if cfg.Chat.ID != 0 {
		return chat.ID == cfg.Chat.ID
	}
	if a.store == nil {
		return false
	}
	settings, ok := a.store.ChatSettings(chat.ID)
	if !ok || !settings.Enabled {
		return false
	}
	return normalizeLayaRuntimeMode(settings.AIEnabledMode, "on", "off") == "on"
}

func (a *App) faqEnabledForChat(chat *tele.Chat) bool {
	return a.activeFAQConfigForChat(chat).Enabled
}

func (a *App) quotesEnabledForChat(chat *tele.Chat) bool {
	return a.activeQuoteConfigForChat(chat).Enabled
}

package main

import (
	"strings"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) photosEnabledForChat(chat *tele.Chat) bool {
	return a.activePhotoConfigForChat(chat).Enabled
}

func (a *App) configRehabTopicID(chat *tele.Chat) (int, bool) {
	cfg := a.config()
	if cfg.Chat.RehabTopicID == 0 {
		return 0, false
	}
	if cfg.Chat.ID == 0 && chat != nil {
		return 0, false
	}
	if cfg.Chat.ID != 0 && chat != nil && chat.ID != cfg.Chat.ID {
		return 0, false
	}
	return cfg.Chat.RehabTopicID, true
}

func (a *App) rememberManagedChat(chat *tele.Chat) {
	if chat == nil || a.store == nil || (chat.Type != tele.ChatGroup && chat.Type != tele.ChatSuperGroup) {
		return
	}
	settings, ok := a.store.ChatSettings(chat.ID)
	title := strings.TrimSpace(chat.Title)
	if !ok {
		settings = ChatSettings{
			ChatID:            chat.ID,
			Title:             title,
			Enabled:           true,
			SpamNoRehabAction: spamNoRehabMuteOnDelete,
			DebugCommands:     a.config().Telegram.ShowDebugCommands,
		}
		if topicID, ok := a.configRehabTopicID(chat); ok {
			settings.RehabTopicID = topicID
			settings.PersistedRehabTopicID = topicID
		}
		_ = a.store.UpsertChatSettings(settings)
		return
	}
	if settings.SpamNoRehabAction == "" {
		settings.SpamNoRehabAction = spamNoRehabMuteOnDelete
	}
	if settings.RehabTopicID == 0 && settings.PersistedRehabTopicID == 0 {
		if topicID, ok := a.configRehabTopicID(chat); ok {
			settings.RehabTopicID = topicID
			settings.PersistedRehabTopicID = topicID
			_ = a.store.UpsertChatSettings(settings)
			return
		}
	}
	if settings.RehabTopicID == 0 && settings.PersistedRehabTopicID > 0 {
		settings.RehabTopicID = settings.PersistedRehabTopicID
		if a.log != nil {
			a.log.Info("rehab topic restored", zap.Int64("chat", chat.ID), zap.Int("topic", settings.RehabTopicID))
		}
		_ = a.store.UpsertChatSettings(settings)
		return
	}
	if settings.RehabTopicID > 0 && settings.PersistedRehabTopicID == 0 {
		settings.PersistedRehabTopicID = settings.RehabTopicID
		_ = a.store.UpsertChatSettings(settings)
		return
	}
	if title == "" || title == settings.Title {
		return
	}
	settings.Title = title
	_ = a.store.UpsertChatSettings(settings)
}

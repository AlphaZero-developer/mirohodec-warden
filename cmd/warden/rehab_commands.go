package main

import (
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func rehabEscapeCommand() tele.Command {
	return tele.Command{Text: "escape", Description: "побег из психцентра"}
}

func rehabBribeCommand() tele.Command {
	return tele.Command{Text: "bribe", Description: "подкуп санитара"}
}

func rehabUserCommands() []tele.Command {
	return []tele.Command{rehabEscapeCommand(), rehabBribeCommand()}
}

func rehabEscapeCommandScope(chatID, userID int64) tele.CommandScope {
	return tele.CommandScope{
		Type:   tele.CommandScopeChatMember,
		ChatID: chatID,
		UserID: userID,
	}
}

func (a *App) showRehabEscapeCommand(chatID, userID int64) {
	if a == nil || a.bot == nil || chatID == 0 || userID == 0 {
		return
	}
	scope := rehabEscapeCommandScope(chatID, userID)
	if err := a.bot.SetCommands(rehabUserCommands(), scope); err != nil && a.log != nil {
		a.log.Warn("set rehab escape command failed", zap.Int64("chat", chatID), zap.Int64("user", userID), zap.Error(err))
	}
}

func (a *App) hideRehabEscapeCommand(chatID, userID int64) {
	if a == nil || a.bot == nil || chatID == 0 || userID == 0 {
		return
	}
	scope := rehabEscapeCommandScope(chatID, userID)
	if err := a.bot.DeleteCommands(scope); err != nil && a.log != nil {
		a.log.Warn("delete rehab escape command failed", zap.Int64("chat", chatID), zap.Int64("user", userID), zap.Error(err))
	}
}

func syncActiveRehabEscapeCommands(bot *tele.Bot, store RuntimeStore, logger *AsyncLogger) error {
	if bot == nil || store == nil {
		return nil
	}

	seenChats := make(map[int64]struct{})
	for _, chatID := range store.ChatIDs() {
		if chatID != 0 {
			seenChats[chatID] = struct{}{}
		}
	}
	for _, settings := range store.ManagedChats(true) {
		if settings.ChatID != 0 && settings.Enabled {
			seenChats[settings.ChatID] = struct{}{}
		}
	}

	now := time.Now().UTC()
	for chatID := range seenChats {
		for _, rec := range store.ModerationRecords(chatID, "jail", true, now) {
			if rec.UserID == 0 {
				continue
			}
			scope := rehabEscapeCommandScope(chatID, rec.UserID)
			if err := bot.SetCommands(rehabUserCommands(), scope); err != nil {
				if logger != nil {
					logger.Warn("sync rehab escape command failed", zap.Int64("chat", chatID), zap.Int64("user", rec.UserID), zap.Error(err))
				}
				return err
			}
		}
	}
	return nil
}

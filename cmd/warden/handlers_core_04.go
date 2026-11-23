package main

import (
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) processJoinedUser(chat *tele.Chat, user *tele.User) {
	if user == nil || user.IsBot {
		return
	}
	a.store.UpsertMember(chat.ID, user)

	if rec, ok := a.store.Ban(chat.ID, user.ID); ok {
		member := &tele.ChatMember{
			User:            &tele.User{ID: user.ID},
			RestrictedUntil: tele.Forever(),
		}
		if err := a.bot.Ban(chat, member, true); err != nil {
			a.log.Warn("re-ban failed", zap.Int64("chat", chat.ID), zap.Int64("user", user.ID), zap.Error(err))
			return
		}
		a.metrics.IncModeration("auto_reban")
		a.log.Info("re-banned forever-banned user", zap.Int64("chat", chat.ID), zap.Int64("user", user.ID), zap.String("reason", rec.Reason))
	}
}

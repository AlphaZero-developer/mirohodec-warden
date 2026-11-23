package main

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) muteSpamUser(chat *tele.Chat, user *tele.User, msg *tele.Message, cfg SpamConfig, reason string, now time.Time) {
	if _, ok := a.store.ActiveMute(chat.ID, user.ID, now); ok {
		return
	}
	if err := a.ensureModeratable(chat, user.ID); err != nil {
		a.log.Warn("spam mute skipped", zap.Int64("chat", chat.ID), zap.Int64("user", user.ID), zap.Error(err))
		return
	}

	duration := mustDuration(cfg.MuteDuration, 10*time.Minute)
	if duration > 0 && duration < 30*time.Second {
		duration = 30 * time.Second
	}
	untilUnix := int64(0)
	restrictedUntil := tele.Forever()
	if duration > 0 {
		untilUnix = now.Add(duration).Unix()
		restrictedUntil = untilUnix
	}

	rights := tele.NoRights()
	rights.Independent = true
	member := &tele.ChatMember{
		User:            &tele.User{ID: user.ID},
		Rights:          rights,
		RestrictedUntil: restrictedUntil,
	}
	if err := a.bot.Restrict(chat, member); err != nil {
		a.log.Warn("spam mute failed", zap.Int64("chat", chat.ID), zap.Int64("user", user.ID), zap.Error(err))
		return
	}

	botID := int64(0)
	botName := "spam filter"
	if a.bot.Me != nil {
		botID = a.bot.Me.ID
		botName = displayNameFromUser(a.bot.Me)
	}
	name := displayNameFromUser(user)
	a.store.SetMute(chat.ID, ModerationRecord{
		UserID:      user.ID,
		Username:    user.Username,
		DisplayName: name,
		Reason:      reason,
		ByAdminID:   botID,
		ByAdminName: botName,
		SinceUnix:   now.Unix(),
		UntilUnix:   untilUnix,
	})
	a.metrics.IncModeration("spam_mute")
	a.log.Info("spam muted user", zap.Int64("chat", chat.ID), zap.Int64("user", user.ID), zap.String("reason", reason), zap.Int64("until", untilUnix))
	a.sendOwnerLog("Spam mute", logField{"User", fmt.Sprintf("%s (%d)", name, user.ID)}, logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)}, logField{"Topic", fmt.Sprintf("%d", msg.ThreadID)}, logField{"Until", a.formatUntil(untilUnix)}, logField{"Reason", reason})

	if cfg.Notify {
		if _, err := a.bot.Send(chat, a.t("spam_muted", name, a.formatUntil(untilUnix)), &tele.SendOptions{ThreadID: msg.ThreadID}); err != nil {
			a.log.Warn("spam mute notice failed", zap.Error(err))
		}
	}
}

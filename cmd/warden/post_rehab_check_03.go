package main

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) returnUserToRehabAfterCheck(chat *tele.Chat, msg *tele.Message, user *tele.User, check postRehabCheck, answer string) {
	if a == nil || a.bot == nil || a.store == nil || chat == nil || user == nil {
		return
	}
	now := time.Now().UTC()
	duration := mustDuration(a.config().Rehab.PostReleaseCheckRejailDuration, 24*time.Hour)
	untilUnix := int64(0)
	if duration > 0 {
		untilUnix = now.Add(duration).Unix()
	}
	rec := ModerationRecord{
		UserID:      user.ID,
		Username:    user.Username,
		DisplayName: displayNameFromUser(user),
		Reason:      "Лайе не понравился ответ после лечения",
		RehabWard:   rehabWardRowdy,
		ByAdminID:   0,
		ByAdminName: "Лайя",
		SinceUnix:   now.Unix(),
		UntilUnix:   untilUnix,
	}
	if rec.DisplayName == "" || rec.DisplayName == "unknown" {
		rec.DisplayName = check.DisplayName
	}

	a.store.SetJail(chat.ID, rec)
	a.clearRehabEscapeCooldown(chat.ID, rec.UserID)
	a.showRehabEscapeCommand(chat.ID, rec.UserID)
	a.sendRehabNotice(chat, rec)
	if _, err := a.bot.Send(chat, "Ответ Лайе не понравился. Возвращайся в психиатрический центр.", &tele.SendOptions{ReplyTo: msg, ThreadID: msg.ThreadID}); err != nil && a.log != nil {
		a.log.Warn("send post-rehab rejail reply failed", zap.Int64("chat", chat.ID), zap.Int64("user", user.ID), zap.Error(err))
	}
	a.sendOwnerLog("Rehab",
		logField{"User", fmt.Sprintf("%s (%d)", rec.DisplayName, rec.UserID)},
		logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)},
		logField{"Release", releaseLine(untilUnix, now)},
		logField{"Reason", rec.Reason},
		logField{"By", "Лайя"},
		logField{"Message", shortText(answer, 300)},
	)
	if a.metrics != nil {
		a.metrics.IncModeration("post_rehab_check_failed")
	}
}

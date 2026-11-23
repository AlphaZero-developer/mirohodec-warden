package main

import (
	"fmt"
	"mirohodec-warden/internal/domain/moderation"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) handleWarn(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	target, rest, err := a.targetFromArgs(chat.ID, c, c.Args())
	if err != nil {
		return c.Send("Использование: /warn [user_id|@username] [reason], или ответом /warn [reason]")
	}
	if err := a.ensureModeratable(chat, target.ID); err != nil {
		return c.Send(a.localizeErr(err))
	}

	cfg := a.activeWarnConfigForChat(chat.ID)
	if !cfg.Enabled {
		return c.Send("Warn выключен для этой группы.")
	}
	now := time.Now().UTC()
	expires := now.Add(mustDuration(cfg.ExpireAfter, 30*24*time.Hour)).Unix()
	admin := c.Sender()
	rec, err := a.store.AddWarning(chat.ID, WarningRecord{
		UserID:      target.ID,
		Username:    target.Username,
		DisplayName: target.Name,
		Reason:      strings.TrimSpace(strings.Join(rest, " ")),
		ByAdminID:   admin.ID,
		ByAdminName: displayNameFromUser(admin),
		CreatedUnix: now.Unix(),
		ExpiresUnix: expires,
		Active:      true,
	})
	if err != nil {
		return c.Send(fmt.Sprintf("Не удалось сохранить warn: %v", err))
	}
	count := a.store.ActiveWarningCount(chat.ID, target.ID, now)
	a.metrics.IncModeration("warn")
	a.sendOwnerLog("Warn", logField{"User", fmt.Sprintf("%s (%d)", target.Name, target.ID)}, logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)}, logField{"Active warnings", fmt.Sprintf("%d", count)}, logField{"Reason", textOrDash(rec.Reason)}, logField{"By", displayNameFromUser(admin)})

	action := ""
	switch moderation.Escalation(count, cfg.MuteThreshold, cfg.RehabThreshold) {
	case moderation.RestrictToTopic:
		if _, already := a.store.ActiveJail(chat.ID, target.ID, now); !already {
			action = a.escalateWarnToRehab(chat, target, admin, rec.Reason)
		}
	case moderation.Mute:
		if _, already := a.store.ActiveMute(chat.ID, target.ID, now); !already {
			action = a.escalateWarnToMute(chat, target, admin, rec.Reason)
		}
	}
	if action != "" {
		return c.Send(fmt.Sprintf("Warn сохранен: %s\nАктивных предупреждений: %d\n%s", rec.ID, count, action))
	}
	return c.Send(fmt.Sprintf("Warn сохранен: %s\nАктивных предупреждений: %d", rec.ID, count))
}

func (a *App) escalateWarnToMute(chat *tele.Chat, target targetUser, admin *tele.User, reason string) string {
	cfg := a.activeWarnConfigForChat(chatIDFromChat(chat))
	now := time.Now().UTC()
	until := now.Add(mustDuration(cfg.MuteDuration, time.Hour)).Unix()
	rights := tele.NoRights()
	rights.Independent = true
	member := &tele.ChatMember{
		User:            &tele.User{ID: target.ID},
		Rights:          rights,
		RestrictedUntil: until,
	}
	if err := a.bot.Restrict(chat, member); err != nil {
		a.log.Warn("warn escalation mute failed", zap.Error(err))
		return fmt.Sprintf("Авто-мут не выдан: %v", err)
	}
	a.store.SetMute(chat.ID, ModerationRecord{
		UserID:      target.ID,
		Username:    target.Username,
		DisplayName: target.Name,
		Reason:      "warnings: " + reason,
		ByAdminID:   admin.ID,
		ByAdminName: displayNameFromUser(admin),
		SinceUnix:   now.Unix(),
		UntilUnix:   until,
	})
	a.metrics.IncModeration("warn_mute")
	a.sendOwnerLog("Warn escalation: mute", logField{"User", fmt.Sprintf("%s (%d)", target.Name, target.ID)}, logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)}, logField{"Until", a.formatUntil(until)}, logField{"Reason", textOrDash(reason)})
	return "Автоэскалация: мут до " + a.formatUntil(until)
}

func (a *App) escalateWarnToRehab(chat *tele.Chat, target targetUser, admin *tele.User, reason string) string {
	cfg := a.activeWarnConfigForChat(chatIDFromChat(chat))
	rehabTopicID := a.rehabTopicID(chat)
	if rehabTopicID == 0 {
		return "Автоэскалация: rehab не настроен для этого чата, пользователь остается под warn/mute-политикой."
	}
	now := time.Now().UTC()
	until := now.Add(mustDuration(cfg.RehabDuration, 24*time.Hour)).Unix()
	rec := ModerationRecord{
		UserID:      target.ID,
		Username:    target.Username,
		DisplayName: target.Name,
		Reason:      "warnings: " + reason,
		RehabWard:   rehabWardRowdy,
		ByAdminID:   admin.ID,
		ByAdminName: displayNameFromUser(admin),
		SinceUnix:   now.Unix(),
		UntilUnix:   until,
	}
	a.store.SetJail(chat.ID, rec)
	a.clearRehabEscapeCooldown(chat.ID, rec.UserID)
	a.showRehabEscapeCommand(chat.ID, rec.UserID)
	a.sendRehabNotice(chat, rec)
	a.metrics.IncModeration("warn_rehab")
	a.sendOwnerLog("Warn escalation: rehab", logField{"User", fmt.Sprintf("%s (%d)", target.Name, target.ID)}, logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)}, logField{"Release", releaseLine(until, now)}, logField{"Reason", textOrDash(reason)})
	return "Автоэскалация: rehab до " + a.formatUntil(until)
}

func (a *App) handleWarnings(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	userID := int64(0)
	if len(c.Args()) > 0 || (c.Message() != nil && c.Message().ReplyTo != nil) {
		target, _, err := a.targetFromArgs(chat.ID, c, c.Args())
		if err != nil {
			return c.Send("Использование: /warnings [user_id|@username], или ответом /warnings")
		}
		userID = target.ID
	}
	records := a.store.Warnings(chat.ID, userID, false, 25, time.Now().UTC())
	if len(records) == 0 {
		return c.Send("Активных предупреждений нет.")
	}
	lines := []string{"Активные предупреждения:"}
	for _, rec := range records {
		lines = append(lines, fmt.Sprintf("%s | %s (%d) | %s | до %s", rec.ID, rec.DisplayName, rec.UserID, textOrDash(rec.Reason), a.formatUntil(rec.ExpiresUnix)))
	}
	return c.Send(strings.Join(lines, "\n"), a.layaInlineMarkup(chat, c.Sender()))
}

func (a *App) handleUnwarn(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	args := c.Args()
	if len(args) == 0 {
		return c.Send("Использование: /unwarn [warn_id]")
	}
	rec, ok, err := a.store.DeactivateWarning(chat.ID, args[0])
	if err != nil {
		return c.Send(fmt.Sprintf("Не удалось снять warn: %v", err))
	}
	if !ok {
		return c.Send("Warn не найден.")
	}
	a.metrics.IncModeration("unwarn")
	a.sendOwnerLog("Unwarn", logField{"User", fmt.Sprintf("%s (%d)", rec.DisplayName, rec.UserID)}, logField{"Warn", rec.ID}, logField{"By", displayNameFromUser(c.Sender())})
	return c.Send("Warn снят: " + rec.ID)
}

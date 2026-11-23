package main

import (
	"fmt"
	"time"

	tele "gopkg.in/telebot.v3"
)

const foreverBanTaunt = "Довыебывался? Довыебывался."

func (a *App) handleJail(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	args := c.Args()
	target, rest, err := a.targetFromArgs(chat.ID, c, args)
	if err != nil {
		if reordered, ok := maybeDurationFirstJailArgs(args); ok {
			if parsedTarget, parsedRest, parsedErr := a.targetFromArgs(chat.ID, c, reordered); parsedErr == nil {
				target, rest, err = parsedTarget, parsedRest, nil
			}
		}
	}
	if err != nil {
		return c.Send(a.t("usage_jail"))
	}
	if err := a.ensureModeratable(chat, target.ID); err != nil {
		return c.Send(a.localizeErr(err))
	}
	rehabTopicID := a.rehabTopicID(chat)
	if rehabTopicID == 0 {
		return c.Send("Для этого чата не настроена тема психиатрического центра. Добавь супергруппу и тему центра в web-панели или укажи chat.rehab_topic_id.")
	}

	untilUnix, wardOverride, reason, err := parseJailOptions(rest, true)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}

	admin := c.Sender()
	now := time.Now().UTC()
	rec := ModerationRecord{
		UserID:      target.ID,
		Username:    target.Username,
		DisplayName: target.Name,
		Reason:      reason,
		RehabWard:   a.chooseRehabWard(chat.ID, target, reason, wardOverride, now),
		ByAdminID:   admin.ID,
		ByAdminName: displayNameFromUser(admin),
		SinceUnix:   now.Unix(),
		UntilUnix:   untilUnix,
	}
	a.store.SetJail(chat.ID, rec)
	a.clearRehabEscapeCooldown(chat.ID, rec.UserID)
	a.showRehabEscapeCommand(chat.ID, rec.UserID)
	a.metrics.IncModeration("jail")
	release := releaseLine(untilUnix, now)
	a.sendOwnerLog("Rehab", logField{"User", fmt.Sprintf("%s (%d)", target.Name, target.ID)}, logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)}, logField{"Topic", fmt.Sprintf("%d", rehabTopicID)}, logField{"Release", release}, logField{"Reason", textOrDash(reason)}, logField{"By", displayNameFromUser(admin)})
	a.sendRehabNotice(chat, rec)

	return c.Send(a.t("jail_done", target.Name, release), a.layaRestrictionMarkup(true))
}

func maybeDurationFirstJailArgs(args []string) ([]string, bool) {
	durationLen := durationArgPrefixLen(args)
	if durationLen == 0 || len(args) <= durationLen {
		return nil, false
	}
	reordered := make([]string, 0, len(args))
	reordered = append(reordered, args[durationLen])
	reordered = append(reordered, args[:durationLen]...)
	reordered = append(reordered, args[durationLen+1:]...)
	return reordered, true
}

func (a *App) handleRehab(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	target, _, err := a.targetFromArgs(chat.ID, c, c.Args())
	if err != nil {
		return c.Send(a.t("usage_rehab"))
	}

	now := time.Now().UTC()
	if rec, ok := a.jailRecord(chat.ID, target.ID, now); ok && a.maybeStartRehabReleaseExam(chat, rec, "command", c.Message()) {
		return c.Send("Перед выходом из психиатрического центра запущен экзамен. Пусть ответит на вопрос в теме центра.")
	}

	rec, released := a.store.RemoveJail(chat.ID, target.ID)
	duration := a.rehabReleaseDurationText(rec, released, now)
	if released {
		a.hideRehabEscapeCommand(chat.ID, target.ID)
		a.clearRehabEscapeCooldown(chat.ID, target.ID)
		a.logRehabRelease(chat.ID, rec, duration, "command")
		skipThreadID := 0
		if msg := c.Message(); msg != nil {
			skipThreadID = msg.ThreadID
		}
		a.sendRehabReleaseNotice(chat, rec, duration, skipThreadID)
		a.schedulePostRehabCheck(chat.ID, rec, now)
	}
	a.metrics.IncModeration("rehab")
	a.sendOwnerLog("Rehab released", logField{"User", fmt.Sprintf("%s (%d)", target.Name, target.ID)}, logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)}, logField{"Duration", duration}, logField{"By", displayNameFromUser(c.Sender())})
	return c.Send(a.t("rehab_done", duration))
}

func (a *App) handleMute(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	target, rest, err := a.targetFromArgs(chat.ID, c, c.Args())
	if err != nil || len(rest) == 0 {
		return c.Send(a.t("usage_mute"))
	}
	if err := a.ensureModeratable(chat, target.ID); err != nil {
		return c.Send(a.localizeErr(err))
	}

	untilUnix, reason, err := parseRequiredUntil(rest)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}

	restrictUntil := untilUnix
	if restrictUntil == 0 {
		restrictUntil = tele.Forever()
	}
	rights := tele.NoRights()
	rights.Independent = true
	member := &tele.ChatMember{
		User:            &tele.User{ID: target.ID},
		Rights:          rights,
		RestrictedUntil: restrictUntil,
	}
	if err := a.bot.Restrict(chat, member); err != nil {
		return c.Send(a.t("telegram_restrict_failed", err))
	}

	admin := c.Sender()
	a.store.SetMute(chat.ID, ModerationRecord{
		UserID:      target.ID,
		Username:    target.Username,
		DisplayName: target.Name,
		Reason:      reason,
		ByAdminID:   admin.ID,
		ByAdminName: displayNameFromUser(admin),
		SinceUnix:   time.Now().UTC().Unix(),
		UntilUnix:   untilUnix,
	})
	a.metrics.IncModeration("mute")
	a.sendOwnerLog("Mute", logField{"User", fmt.Sprintf("%s (%d)", target.Name, target.ID)}, logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)}, logField{"Until", a.formatUntil(untilUnix)}, logField{"Reason", textOrDash(reason)}, logField{"By", displayNameFromUser(admin)})

	return c.Send(a.t("mute_done", target.Name, target.ID, a.formatUntil(untilUnix), textOrDash(reason), displayNameFromUser(admin)), a.layaRestrictionMarkup(false))
}

func (a *App) handleUnmute(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	target, _, err := a.targetFromArgs(chat.ID, c, c.Args())
	if err != nil {
		return c.Send(a.t("usage_unmute"))
	}

	member := &tele.ChatMember{
		User:   &tele.User{ID: target.ID},
		Rights: tele.NoRestrictions(),
	}
	if err := a.bot.Restrict(chat, member); err != nil {
		return c.Send(a.t("telegram_unrestrict_failed", err))
	}

	a.store.RemoveMute(chat.ID, target.ID)
	a.metrics.IncModeration("unmute")
	a.sendOwnerLog("Unmute", logField{"User", fmt.Sprintf("%s (%d)", target.Name, target.ID)}, logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)}, logField{"By", displayNameFromUser(c.Sender())})
	return c.Send(a.t("unmute_done", target.Name))
}

package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) handleStatus(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	target, _, err := a.targetFromArgs(chat.ID, c, c.Args())
	if err != nil {
		return c.Send(a.t("usage_status"))
	}

	var lines []string
	lines = append(lines, a.t("status_header", target.Name, target.ID))
	if jail, ok := a.store.ActiveJail(chat.ID, target.ID, time.Now().UTC()); ok {
		lines = append(lines, a.t("status_rehab_active", a.formatUntil(jail.UntilUnix)))
		if jail.Reason != "" {
			lines = append(lines, a.t("status_rehab_reason", jail.Reason))
		}
	} else {
		lines = append(lines, a.t("status_rehab_no"))
	}
	if mute, ok := a.store.ActiveMute(chat.ID, target.ID, time.Now().UTC()); ok {
		lines = append(lines, a.t("status_mute_active", a.formatUntil(mute.UntilUnix)))
		if mute.Reason != "" {
			lines = append(lines, a.t("status_mute_reason", mute.Reason))
		}
	} else {
		lines = append(lines, a.t("status_mute_no"))
	}
	if _, ok := a.store.Ban(chat.ID, target.ID); ok {
		lines = append(lines, a.t("status_ban_yes"))
	} else {
		lines = append(lines, a.t("status_ban_no"))
	}
	return c.Send(strings.Join(lines, "\n"), a.layaInlineMarkup(chat, c.Sender()))
}

func (a *App) handleActivity(c tele.Context) error {
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}

	var target targetUser
	if a.isAdmin(c) {
		target, _, err = a.targetFromArgsOrSender(chat.ID, c, c.Args())
		if err != nil {
			return c.Send(a.t("usage_activity"))
		}
	} else {
		sender := c.Sender()
		if sender == nil {
			return c.Send(a.t("no_sender"))
		}
		if len(c.Args()) > 0 || repliedToOtherUser(c, sender.ID) {
			return c.Send(a.t("no_admin"))
		}
		target = targetUser{ID: sender.ID, Username: sender.Username, Name: displayNameFromUser(sender)}
	}

	rec, ok := a.store.Activity(chat.ID, target.ID)
	if !ok {
		return c.Send(a.t("activity_unknown"))
	}

	now := time.Now().UTC()
	karma := a.store.Karma(chat.ID, target.ID)
	activeWarnings := a.store.ActiveWarningCount(chat.ID, target.ID, now)
	score := activityScore(rec, karma, activeWarnings, now)
	activity := activityBadge(score)
	karmaState := karmaBadge(karma)
	lines := []string{
		fmt.Sprintf("📊 Активность %s", rec.DisplayName),
		fmt.Sprintf("🪪 ID: %d", rec.ID),
		"",
		fmt.Sprintf("🔥 Индекс активности: %d · место %s", score, socialRankLabel(a.activityRank(chat.ID, target.ID, now))),
		fmt.Sprintf("%s Уровень активности: %s", activity.Emoji, activity.Title),
		fmt.Sprintf("💬 Сообщений всего: %d", rec.Messages),
		fmt.Sprintf("☀️ Сегодня: %d", messagesTodayAt(rec, now)),
		karmaAutoProgressLine(messagesTodayAt(rec, now), a.activeKarmaConfigForChat(chat.ID)),
		fmt.Sprintf("⭐ Карма: %d · место %s", karma, socialRankLabel(a.karmaRank(chat.ID, target.ID))),
		fmt.Sprintf("%s Репутация: %s", karmaState.Emoji, karmaState.Title),
		fmt.Sprintf("⚠️ Активных предупреждений: %d", activeWarnings),
		"",
		fmt.Sprintf("👣 Первое появление: %s", a.formatMaybeUnix(rec.FirstSeen)),
		fmt.Sprintf("🕒 Последняя активность: %s", a.formatMaybeUnix(rec.LastSeen)),
		fmt.Sprintf("✉️ Последнее сообщение: %s", a.formatMaybeUnix(rec.LastMessage)),
	}
	if rec.LastThreadID != 0 {
		lines = append(lines, fmt.Sprintf("🧵 Последняя тема: %d", rec.LastThreadID))
	}
	return c.Send(strings.Join(lines, "\n"), a.layaInlineMarkup(chat, c.Sender()))
}

func (a *App) handleTopActive(c tele.Context) error {
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}

	limit := 10
	maxLimit := 10
	if a.isAdmin(c) {
		maxLimit = 50
	}
	if args := c.Args(); len(args) > 0 {
		if parsed, err := strconv.Atoi(args[0]); err == nil && parsed > 0 && parsed <= maxLimit {
			limit = parsed
		}
	}

	rows := a.topActivityRows(chat.ID, limit, time.Now().UTC())
	if len(rows) == 0 {
		return c.Send(a.t("unknown_members"))
	}

	lines := []string{"🔥 Топ активности:"}
	for _, row := range rows {
		member := row.Member
		badge := activityBadge(row.Score)
		lines = append(lines, fmt.Sprintf("%s %d. %s — %d очк. · 💬 %d · ☀️ %d · ⭐ %d · %s %s",
			medalEmoji(row.Rank),
			row.Rank,
			displayKarmaName(member.DisplayName, member.ID),
			row.Score,
			member.Messages,
			messagesToday(member),
			row.Karma,
			badge.Emoji,
			badge.Title,
		))
	}
	return c.Send(strings.Join(lines, "\n"), a.layaInlineMarkup(chat, c.Sender()))
}

func (a *App) handleSpamLog(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}

	limit := a.config().Spam.LogLimit
	if limit <= 0 {
		limit = 25
	}

	args := c.Args()
	var target *targetUser
	if len(args) > 0 || (c.Message() != nil && c.Message().ReplyTo != nil) {
		parsedTarget, rest, err := a.targetFromArgs(chat.ID, c, args)
		if err != nil {
			return c.Send(a.t("usage_spamlog"))
		}
		target = &parsedTarget
		if len(rest) > 0 {
			if parsed, err := strconv.Atoi(rest[0]); err == nil && parsed > 0 && parsed <= 100 {
				limit = parsed
			}
		}
	}

	records := a.spamLogRecords(chat.ID, 0, limit)
	if target != nil {
		records = a.spamLogRecords(chat.ID, target.ID, limit)
	}
	if len(records) == 0 {
		return c.Send(a.t("spamlog_empty"))
	}

	lines := []string{a.t("spamlog_header")}
	for _, rec := range records {
		kind := rec.Kind
		if rec.Confidence != "" {
			kind += "/" + rec.Confidence
		}
		lines = append(lines, a.t("spamlog_line", a.formatMaybeUnix(rec.At.Unix()), rec.UserName, kind, rec.Action, rec.Reason, rec.Counts))
	}
	return c.Send(strings.Join(lines, "\n"))
}

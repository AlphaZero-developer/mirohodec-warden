package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) handleClearWarnings(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	target, _, err := a.targetFromArgs(chat.ID, c, c.Args())
	if err != nil {
		return c.Send("Использование: /clearwarns [user_id|@username], или ответом /clearwarns")
	}
	n, err := a.store.ClearWarnings(chat.ID, target.ID)
	if err != nil {
		return c.Send(fmt.Sprintf("Не удалось очистить warns: %v", err))
	}
	a.metrics.IncModeration("clearwarns")
	a.sendOwnerLog("Clear warns", logField{"User", fmt.Sprintf("%s (%d)", target.Name, target.ID)}, logField{"Count", fmt.Sprintf("%d", n)}, logField{"By", displayNameFromUser(c.Sender())})
	return c.Send(fmt.Sprintf("Warns очищены для %s: %d", target.Name, n))
}

func (a *App) handleKarma(c tele.Context) error {
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	var target targetUser
	if a.isAdmin(c) {
		target, _, err = a.targetFromArgsOrSender(chat.ID, c, c.Args())
		if err != nil {
			return c.Send("Использование: /karma [user_id|@username], или ответом /karma")
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
	now := time.Now().UTC()
	score := a.store.Karma(chat.ID, target.ID)
	activeWarnings := a.store.ActiveWarningCount(chat.ID, target.ID, now)
	badge := karmaBadge(score)
	lines := []string{
		fmt.Sprintf("⭐ Карма %s", displayKarmaName(target.Name, target.ID)),
		fmt.Sprintf("🪪 ID: %d", target.ID),
		"",
		fmt.Sprintf("💠 Счёт: %d", score),
		fmt.Sprintf("🏆 Место в топе: %s", socialRankLabel(a.karmaRank(chat.ID, target.ID))),
		fmt.Sprintf("%s Репутация: %s", badge.Emoji, badge.Title),
		fmt.Sprintf("📝 Вердикт Лайи: %s", badge.Hint),
		fmt.Sprintf("⚠️ Активных предупреждений: %d", activeWarnings),
	}
	if rec, ok := a.store.Activity(chat.ID, target.ID); ok {
		activityPoints := activityScore(rec, score, activeWarnings, now)
		activity := activityBadge(activityPoints)
		lines = append(lines,
			"",
			fmt.Sprintf("🔥 Активность: %d · место %s", activityPoints, socialRankLabel(a.activityRank(chat.ID, target.ID, now))),
			fmt.Sprintf("%s Уровень активности: %s", activity.Emoji, activity.Title),
			fmt.Sprintf("💬 Сообщений: %d всего · %d сегодня", rec.Messages, messagesTodayAt(rec, now)),
			karmaAutoProgressLine(messagesTodayAt(rec, now), a.activeKarmaConfigForChat(chat.ID)),
		)
	}
	lines = append(lines, "", "🎁 Плюшки и безопасность:")
	lines = append(lines, socialPerkLines(a.layaSafetyProfileForTarget(chat, target, now), a.activityRank(chat.ID, target.ID, now))...)
	return c.Send(strings.Join(lines, "\n"), a.layaInlineMarkup(chat, c.Sender()))
}

func (a *App) handleTopKarma(c tele.Context) error {
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
	records := a.store.TopKarma(chat.ID, limit)
	if len(records) == 0 {
		return c.Send("Кармы пока нет.")
	}
	lines := []string{"🏆 Топ кармы:"}
	for i, rec := range records {
		name := rec.DisplayName
		if strings.TrimSpace(name) == "" {
			name = fmt.Sprintf("user %d", rec.UserID)
		}
		badge := karmaBadge(rec.Score)
		rank := i + 1
		lines = append(lines, fmt.Sprintf("%s %d. %s — %d · %s %s", medalEmoji(rank), rank, name, rec.Score, badge.Emoji, badge.Title))
	}
	return c.Send(strings.Join(lines, "\n"))
}

func (a *App) maybeHandleManualKarma(c tele.Context) bool {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || sender.IsBot {
		return false
	}
	karmaCfg := a.activeKarmaConfigForChat(chat.ID)
	if !karmaCfg.Enabled {
		return false
	}
	delta, ok := manualKarmaDelta(messageText(msg))
	if !ok {
		return false
	}
	if msg.ReplyTo == nil || msg.ReplyTo.Sender == nil || msg.ReplyTo.Sender.IsBot {
		_ = c.Send("↩️ Ответь на сообщение пользователя: +карма или -карма.")
		return true
	}
	target := msg.ReplyTo.Sender
	if target.ID == sender.ID {
		_ = c.Send("🪞 Самому себе карму менять нельзя. Лайя видит этот фокус.")
		return true
	}
	now := time.Now().UTC()
	cooldown := mustDuration(karmaCfg.ManualCooldown, 12*time.Hour)
	if last := a.store.LastManualKarma(chat.ID, target.ID, sender.ID, now); !last.IsZero() && cooldown > 0 && now.Sub(last) < cooldown {
		_ = c.Send("⏳ Карму этому пользователю можно будет изменить через " + formatDurationShort(cooldown-now.Sub(last)))
		return true
	}
	score, err := a.store.AddKarma(chat.ID, target.ID, delta, "manual", sender, msg.ID, now)
	if err != nil {
		_ = c.Send(fmt.Sprintf("⚠️ Не удалось изменить карму: %v", err))
		return true
	}
	a.metrics.IncModeration("karma")
	if delta > 0 {
		a.unlockAchievement(chat.ID, sender.ID, "helper", now)
	}
	badge := karmaBadge(score)
	action := "✨"
	if delta < 0 {
		action = "🧊"
	}
	_ = c.Send(fmt.Sprintf("%s Карма %s: %s → %d\n%s Репутация: %s", action, displayNameFromUser(target), signedScore(delta), score, badge.Emoji, badge.Title))
	return true
}

func manualKarmaDelta(text string) (int64, bool) {
	text = strings.ToLower(strings.TrimSpace(text))
	switch text {
	case "+карма", "+karma":
		return 1, true
	case "-карма", "-karma":
		return -1, true
	default:
		return 0, false
	}
}

func countableKarmaMessage(msg *tele.Message) bool {
	if msg == nil {
		return false
	}
	text := messageText(msg)
	return commandName(text) == ""
}

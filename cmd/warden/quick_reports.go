package main

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) sendQuickOwnerReport(c tele.Context, kind string) error {
	chat := c.Chat()
	msg := c.Message()
	sender := c.Sender()
	if chat == nil || sender == nil {
		return c.Send(a.t("no_sender"))
	}
	now := time.Now().UTC()
	if a.jailedUserCannotSendReport(chat.ID, sender.ID, kind, now) {
		return nil
	}
	if kind == "appeal" {
		if wait := a.appealCooldownRemaining(chat.ID, sender.ID, now); wait > 0 {
			return nil
		}
	}
	logChatID := a.store.LogChatID()
	if logChatID == 0 {
		return c.Send(a.t("report_director_unavailable"), a.layaInlineMarkup(chat, sender))
	}

	title := ownerReportTitle(kind)
	text := a.quickReportText(chat, sender, kind, now)
	report := formatOwnerReport(title, chat, msg, sender, text)
	if _, err := a.bot.Send(&tele.Chat{ID: logChatID, Type: tele.ChatPrivate}, report); err != nil {
		if a.log != nil {
			a.log.Warn("send quick owner report failed", zap.String("kind", kind), zap.Int64("chat", logChatID), zap.Error(err))
		}
		return c.Send(a.t("report_send_failed", err), a.layaInlineMarkup(chat, sender))
	}
	if a.metrics != nil {
		a.metrics.IncModeration("quick_" + kind)
	}
	if kind == "appeal" {
		a.rememberAppealCooldown(chat.ID, sender.ID, now)
	}
	switch kind {
	case "appeal":
		return c.Send("Быстрая апелляция отправлена владельцу. Для подробностей можно написать /appeal текстом.", a.layaInlineMarkup(chat, sender))
	case "bug":
		return c.Send("Быстрая жалоба отправлена владельцу. Если есть детали, напиши /bug текстом.", a.layaInlineMarkup(chat, sender))
	default:
		return c.Send("Быстрое обращение отправлено.", a.layaInlineMarkup(chat, sender))
	}
}

func (a *App) quickReportText(chat *tele.Chat, sender *tele.User, kind string, now time.Time) string {
	lines := []string{}
	switch kind {
	case "appeal":
		lines = append(lines, "Быстрая апелляция кнопкой: прошу пересмотреть ограничение.")
	case "bug":
		lines = append(lines, "Быстрая жалоба кнопкой: пользователь просит проверить действие бота.")
	default:
		lines = append(lines, "Быстрое обращение кнопкой.")
	}
	if a == nil || a.store == nil || chat == nil || sender == nil {
		return strings.Join(lines, "\n")
	}
	if jail, ok := a.store.ActiveJail(chat.ID, sender.ID, now); ok {
		lines = append(lines, fmt.Sprintf("Активный психиатрический центр до: %s", a.formatUntil(jail.UntilUnix)))
		if jail.Reason != "" {
			lines = append(lines, "Причина психиатрического центра: "+jail.Reason)
		}
		lines = append(lines, rehabWardLine(jail))
	}
	if mute, ok := a.store.ActiveMute(chat.ID, sender.ID, now); ok {
		lines = append(lines, fmt.Sprintf("Активный мут до: %s", a.formatUntil(mute.UntilUnix)))
		if mute.Reason != "" {
			lines = append(lines, "Причина мута: "+mute.Reason)
		}
	}
	if _, ok := a.store.Ban(chat.ID, sender.ID); ok {
		lines = append(lines, "Есть запись вечного бана в хранилище.")
	}
	return strings.Join(lines, "\n")
}

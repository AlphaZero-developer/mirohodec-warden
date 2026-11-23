package main

import (
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) handleOwnerReport(c tele.Context, reportKind string) error {
	cfg := a.activeIdeaConfig()
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil {
		return c.Send(a.t("no_message"))
	}
	now := time.Now().UTC()
	if a.jailedUserCannotSendReport(chat.ID, sender.ID, reportKind, now) {
		return nil
	}
	if !cfg.Enabled {
		return c.Send(a.t("report_disabled"))
	}
	restrictedAppeal := reportKind == "appeal" && chat.Type != tele.ChatPrivate && a.userHasActiveRestriction(chat.ID, sender.ID, now)
	if restrictedAppeal {
		defer a.deleteRestrictedAppealMessage(c, chat, sender)
	}
	reply := func(text string) error {
		if restrictedAppeal {
			return nil
		}
		return c.Send(text)
	}

	text := ideaText(c)
	if text == "" {
		return reply(a.ownerReportUsage(reportKind))
	}
	if len([]rune(text)) > cfg.MaxLength {
		return reply(a.t("report_too_long", cfg.MaxLength))
	}
	logChatID := a.store.LogChatID()
	if logChatID == 0 {
		return reply(a.t("report_director_unavailable"))
	}

	if reportKind == "appeal" {
		if wait := a.appealCooldownRemaining(chat.ID, sender.ID, now); wait > 0 {
			return reply(a.t("report_cooldown", formatDurationShort(wait)))
		}
	}
	if wait := a.ideaCooldownRemaining(chat.ID, sender.ID, now, cfg); wait > 0 {
		return reply(a.t("report_cooldown", formatDurationShort(wait)))
	}

	report := formatOwnerReport(ownerReportTitle(reportKind), chat, msg, sender, text)
	if _, err := a.bot.Send(&tele.Chat{ID: logChatID, Type: tele.ChatPrivate}, report); err != nil {
		a.log.Warn("send owner report failed", zap.String("kind", reportKind), zap.Int64("chat", logChatID), zap.Error(err))
		return reply(a.t("report_send_failed", err))
	}
	if reportKind == "appeal" {
		a.rememberAppealCooldown(chat.ID, sender.ID, now)
	}
	a.rememberIdeaCooldown(chat.ID, sender.ID, now)
	a.metrics.IncModeration(reportKind)
	switch reportKind {
	case "job":
		return reply(a.ownerReportDone(reportKind))
	case "donate":
		return reply(a.ownerReportDone(reportKind))
	case "idea":
		return reply(a.layaReactionText("idea"))
	case "appeal":
		return reply(a.layaReactionText("appeal"))
	default:
		return reply(a.ownerReportDone(reportKind))
	}
}

func (a *App) maybeAutoReplyFAQ(c tele.Context) bool {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || sender.IsBot || chat.Type == tele.ChatPrivate {
		return false
	}
	cfg := a.activeFAQConfigForChat(chat)
	if !cfg.Enabled || !cfg.AutoReply {
		return false
	}
	if a.layaIsIgnoring(chat.ID, sender.ID) {
		return false
	}
	text := messageText(msg)
	if text == "" || commandName(text) != "" {
		return false
	}
	if cfg.RequireMention && !mentionsLaya(text, a.botUsername()) {
		return false
	}

	rec, ok := a.faq.Match(text)
	if !ok {
		return false
	}
	now := time.Now().UTC()
	if !a.allowFAQAutoReply(chat.ID, sender.ID, rec.ID, now, cfg) {
		return false
	}

	a.faq.MarkUsed(rec.ID)
	if _, err := a.bot.Send(chat, rec.Answer, &tele.SendOptions{ReplyTo: msg, ThreadID: msg.ThreadID}); err != nil {
		a.log.Warn("auto faq reply failed", zap.Error(err))
		return false
	}
	a.metrics.IncModeration("auto_faq")
	return true
}

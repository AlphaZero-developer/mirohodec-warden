package main

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) handleSpamStatus(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	target, _, err := a.targetFromArgsOrSender(chat.ID, c, c.Args())
	if err != nil {
		return c.Send(a.t("usage_spamstatus"))
	}

	status := a.spamStatus(chat.ID, target.ID)
	lines := []string{
		a.t("spamstatus_header", target.Name, target.ID),
		a.t("spamstatus_counts", status.Text, status.Media, status.Stickers, status.Links, status.Mentions, status.Forwards),
		a.t("spamstatus_repeats", status.Repeated),
		a.t("spamstatus_strikes", status.Strikes),
	}
	if status.LastAction != "" {
		lines = append(lines, a.t("spamstatus_last", status.LastAction, status.LastReason, a.formatMaybeUnix(status.LastAt.Unix())))
	}
	return c.Send(strings.Join(lines, "\n"))
}

func (a *App) handleSpamPardon(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	target, _, err := a.targetFromArgs(chat.ID, c, c.Args())
	if err != nil {
		return c.Send(a.t("usage_spampardon"))
	}

	a.clearSpamState(chat.ID, target.ID)
	unmuted := false
	if mute, ok := a.store.ActiveMute(chat.ID, target.ID, time.Now().UTC()); ok && a.isBotMute(mute) {
		member := &tele.ChatMember{
			User:   &tele.User{ID: target.ID},
			Rights: tele.NoRestrictions(),
		}
		if err := a.bot.Restrict(chat, member); err != nil {
			return c.Send(a.t("telegram_unrestrict_failed", err))
		}
		a.store.RemoveMute(chat.ID, target.ID)
		unmuted = true
	}

	a.metrics.IncModeration("spampardon")
	if unmuted {
		a.sendOwnerLog("Spam pardon", logField{"User", fmt.Sprintf("%s (%d)", target.Name, target.ID)}, logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)}, logField{"Action", "spam state cleared and bot mute removed"}, logField{"By", displayNameFromUser(c.Sender())})
		return c.Send(a.t("spampardon_unmuted", target.Name))
	}
	a.sendOwnerLog("Spam pardon", logField{"User", fmt.Sprintf("%s (%d)", target.Name, target.ID)}, logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)}, logField{"Action", "spam state cleared"}, logField{"By", displayNameFromUser(c.Sender())})
	return c.Send(a.t("spampardon_done", target.Name))
}

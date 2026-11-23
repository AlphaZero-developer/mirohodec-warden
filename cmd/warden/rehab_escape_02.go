package main

import (
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) recordRehabEscapeCooldown(chatID, userID int64, jail ModerationRecord, now time.Time) time.Duration {
	if a == nil {
		return minRehabEscapeCooldown
	}
	key := rehabEscapeCooldownKey(chatID, userID)
	jailSince := rehabEscapeJailSince(jail)

	a.rehabEscapeMu.Lock()
	defer a.rehabEscapeMu.Unlock()

	if a.rehabEscapeWait == nil {
		a.rehabEscapeWait = make(map[string]rehabEscapeCooldownState)
	}

	state := a.rehabEscapeWait[key]
	if state.JailSince != jailSince {
		state = rehabEscapeCooldownState{JailSince: jailSince}
	}
	maxLevel := int(maxRehabEscapeCooldown / minRehabEscapeCooldown)
	if state.Level < maxLevel {
		state.Level++
	}
	if state.Level < 1 {
		state.Level = 1
	}
	cooldown := rehabEscapeCooldownDurationForWard(state.Level, jail)
	state.Until = now.UTC().Add(cooldown)
	a.rehabEscapeWait[key] = state
	return cooldown
}

func rehabEscapeCooldownDurationForWard(level int, jail ModerationRecord) time.Duration {
	if rehabWard(jail).ID != rehabWardRowdy {
		return rehabEscapeCooldownDuration(level)
	}
	switch {
	case level <= 1:
		return 2 * time.Minute
	case level == 2:
		return 4 * time.Minute
	default:
		return maxRehabEscapeCooldown
	}
}

func (a *App) clearRehabEscapeCooldown(chatID, userID int64) {
	if a == nil {
		return
	}
	key := rehabEscapeCooldownKey(chatID, userID)

	a.rehabEscapeMu.Lock()
	defer a.rehabEscapeMu.Unlock()
	delete(a.rehabEscapeWait, key)
	a.rehabMu.Lock()
	if a.rehabWardProgress != nil {
		delete(a.rehabWardProgress, key)
	}
	a.rehabMu.Unlock()
}

func (a *App) sendRehabEscapeReply(chat *tele.Chat, msg *tele.Message, sender *tele.User, text string) {
	if a == nil || a.bot == nil || chat == nil || msg == nil || strings.TrimSpace(text) == "" {
		return
	}
	if sender == nil {
		sender = msg.Sender
	}
	if _, err := a.bot.Send(chat, text, &tele.SendOptions{ReplyTo: msg, ThreadID: msg.ThreadID, ReplyMarkup: a.layaInlineMarkup(chat, sender)}); err != nil && a.log != nil {
		a.log.Warn("send rehab escape reply failed", zap.Int64("chat", chat.ID), zap.Int("thread", msg.ThreadID), zap.Error(err))
	}
}

func isRehabEscapeCommand(text string) bool {
	normalized := normalizeFAQText(text)
	switch normalized {
	case "побег из дурки", "побег с дурки", "сбежать из дурки", "сбежать с дурки",
		"побег из психушки", "побег с психушки", "сбежать из психушки", "сбежать с психушки",
		"побег из психцентра", "побег с психцентра", "сбежать из психцентра", "сбежать с психцентра",
		"escape":
		return true
	default:
		return false
	}
}

func rehabEscapeChanceFromPenalty(penalty int) int {
	chance := defaultRehabEscapeChance - penalty
	if chance < 0 {
		return 0
	}
	return chance
}

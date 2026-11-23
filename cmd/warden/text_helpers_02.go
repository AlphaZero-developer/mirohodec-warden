package main

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) ideaCooldownRemaining(chatID, userID int64, now time.Time, cfg IdeaConfig) time.Duration {
	userCooldown := mustDuration(cfg.UserCooldown, 15*time.Minute)
	chatCooldown := mustDuration(cfg.ChatCooldown, 30*time.Second)
	userKey := idKey(userID)
	chatKey := idKey(chatID)

	a.ideaMu.Lock()
	defer a.ideaMu.Unlock()

	var wait time.Duration
	if last := a.lastIdeaUser[userKey]; !last.IsZero() && userCooldown > 0 {
		if remaining := userCooldown - now.Sub(last); remaining > wait {
			wait = remaining
		}
	}
	if last := a.lastIdeaChat[chatKey]; !last.IsZero() && chatCooldown > 0 {
		if remaining := chatCooldown - now.Sub(last); remaining > wait {
			wait = remaining
		}
	}
	if wait < 0 {
		return 0
	}
	return wait
}

func (a *App) rememberIdeaCooldown(chatID, userID int64, now time.Time) {
	a.ideaMu.Lock()
	defer a.ideaMu.Unlock()
	a.lastIdeaUser[idKey(userID)] = now
	a.lastIdeaChat[idKey(chatID)] = now
}

func appealReportKey(chatID, userID int64) string {
	return fmt.Sprintf("%d:%d", chatID, userID)
}

func (a *App) appealCooldownRemaining(chatID, userID int64, now time.Time) time.Duration {
	if a == nil {
		return 0
	}
	a.appealMu.Lock()
	defer a.appealMu.Unlock()

	last := a.lastAppeal[appealReportKey(chatID, userID)]
	if a.store != nil {
		if stored := a.store.OwnerReportLastSent("appeal", chatID, userID); stored.After(last) {
			last = stored
		}
	}
	if last.IsZero() {
		return 0
	}
	remaining := appealReportCooldown - now.Sub(last)
	if remaining <= 0 {
		return 0
	}
	return remaining
}

func (a *App) rememberAppealCooldown(chatID, userID int64, now time.Time) {
	if a == nil {
		return
	}
	a.appealMu.Lock()
	defer a.appealMu.Unlock()
	if a.lastAppeal == nil {
		a.lastAppeal = make(map[string]time.Time)
	}
	a.lastAppeal[appealReportKey(chatID, userID)] = now
	if a.store != nil {
		_ = a.store.SetOwnerReportLastSent("appeal", chatID, userID, now)
	}
}

func reportKindBlockedWhileJailed(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "idea", "bug", "reportbug":
		return true
	default:
		return false
	}
}

func (a *App) jailedUserCannotSendReport(chatID, userID int64, kind string, now time.Time) bool {
	if a == nil || a.store == nil || !reportKindBlockedWhileJailed(kind) {
		return false
	}
	if _, jailed := a.store.ActiveJail(chatID, userID, now); jailed {
		return true
	}
	for _, id := range a.store.ChatIDs() {
		if id == chatID {
			continue
		}
		if _, jailed := a.store.ActiveJail(id, userID, now); jailed {
			return true
		}
	}
	return false
}

func isOwnerReportCommandMessage(msg *tele.Message) bool {
	switch commandName(messageText(msg)) {
	case "idea", "bug", "reportbug", "appeal":
		return true
	default:
		return false
	}
}

func (a *App) allowFAQAutoReply(chatID, userID int64, faqID string, now time.Time, cfg FAQConfig) bool {
	entryCooldown := mustDuration(cfg.EntryCooldown, 10*time.Minute)
	userCooldown := mustDuration(cfg.UserCooldown, time.Minute)
	entryKey := fmt.Sprintf("%d:%s", chatID, faqID)
	userKey := fmt.Sprintf("%d:%d", chatID, userID)

	a.faqMu.Lock()
	defer a.faqMu.Unlock()

	if last := a.lastFAQEntry[entryKey]; !last.IsZero() && entryCooldown > 0 && now.Sub(last) < entryCooldown {
		return false
	}
	if last := a.lastFAQUser[userKey]; !last.IsZero() && userCooldown > 0 && now.Sub(last) < userCooldown {
		return false
	}
	a.lastFAQEntry[entryKey] = now
	a.lastFAQUser[userKey] = now
	return true
}

func mentionsLaya(text, botUsername string) bool {
	normalized := normalizeFAQText(text)
	padded := " " + normalized + " "
	if strings.Contains(padded, " лайя ") ||
		strings.Contains(padded, " laya ") ||
		strings.Contains(padded, " лайетана каталонес ") {
		return true
	}
	botUsername = normalizeUsername(botUsername)
	if botUsername == "" {
		return false
	}
	return strings.Contains(strings.ToLower(text), "@"+botUsername)
}

func (a *App) botUsername() string {
	if a == nil || a.bot == nil || a.bot.Me == nil {
		return ""
	}
	return a.bot.Me.Username
}

func trimRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

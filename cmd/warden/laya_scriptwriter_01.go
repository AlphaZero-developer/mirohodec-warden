package main

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

type layaScriptwriterAction int

const (
	layaScriptwriterNone layaScriptwriterAction = iota
	layaScriptwriterAdd
	layaScriptwriterRemove
)

func normalizeScriptwriterExemption(rec ScriptwriterExemption) ScriptwriterExemption {
	rec.Username = normalizeUsername(rec.Username)
	rec.DisplayName = strings.TrimSpace(rec.DisplayName)
	rec.Profession = normalizeLayaProtectedProfession(rec.Profession)
	if rec.Profession == "" {
		rec.Profession = "сценарист"
	}
	rec.AddedByName = strings.TrimSpace(rec.AddedByName)
	return rec
}

func scriptwriterExemptionKey(chatID, userID int64) string {
	if chatID == 0 || userID == 0 {
		return ""
	}
	return fmt.Sprintf("%d:%d", chatID, userID)
}

func (a *App) maybeHandleLayaScriptwriterExemption(c tele.Context) bool {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || chat.Type == tele.ChatPrivate || a.store == nil {
		return false
	}

	action, profession, ok := parseLayaScriptwriterRequest(messageText(msg), a.botUsername())
	if !ok {
		return false
	}
	if !a.isAdmin(c) {
		if err := c.Send(a.t("no_admin")); err != nil && a.log != nil {
			a.log.Warn("send scriptwriter no-admin notice failed", zap.Error(err))
		}
		return true
	}

	target, ok := a.layaScriptwriterTarget(chat.ID, msg, sender.ID)
	if !ok || target.ID == 0 {
		if err := c.Send("Ответь на сообщение человека или укажи его @username/id: Лайя, это сценарист/художник/строитель/кодер, его не муть."); err != nil && a.log != nil {
			a.log.Warn("send scriptwriter target notice failed", zap.Error(err))
		}
		return true
	}

	switch action {
	case layaScriptwriterRemove:
		if _, _, err := a.store.RemoveScriptwriterExemption(chat.ID, target.ID); err != nil {
			if err := c.Send("Не смогла убрать проф-иммунитет: " + err.Error()); err != nil && a.log != nil {
				a.log.Warn("send scriptwriter remove error failed", zap.Error(err))
			}
			return true
		}
		if err := c.Send(fmt.Sprintf("Ок, %s больше не под проф-иммунитетом. Антиспам снова обычный.", target.Name)); err != nil && a.log != nil {
			a.log.Warn("send scriptwriter remove reply failed", zap.Error(err))
		}
		return true
	case layaScriptwriterAdd:
		rec := ScriptwriterExemption{
			ChatID:      chat.ID,
			UserID:      target.ID,
			Username:    target.Username,
			DisplayName: target.Name,
			Profession:  profession,
			AddedUnix:   time.Now().UTC().Unix(),
			AddedByID:   sender.ID,
			AddedByName: displayNameFromUser(sender),
		}
		if err := a.store.SetScriptwriterExemption(rec); err != nil {
			if err := c.Send("Не смогла записать проф-иммунитет: " + err.Error()); err != nil && a.log != nil {
				a.log.Warn("send scriptwriter add error failed", zap.Error(err))
			}
			return true
		}
		a.clearSpamState(chat.ID, target.ID)
		a.unmuteScriptwriterSpamMute(chat, target)
		if a.metrics != nil {
			a.metrics.IncModeration("scriptwriter_exempt")
		}
		if err := c.Send(fmt.Sprintf("Запомнила. %s теперь %s: антиспам не будет мутить его за флуд.", target.Name, profession)); err != nil && a.log != nil {
			a.log.Warn("send scriptwriter add reply failed", zap.Error(err))
		}
		return true
	default:
		return false
	}
}

func parseLayaScriptwriterRequest(text, botUsername string) (layaScriptwriterAction, string, bool) {
	text = strings.TrimSpace(text)
	if text == "" || !mentionsLaya(text, botUsername) {
		return layaScriptwriterNone, "", false
	}
	normalized := " " + normalizeFAQText(text) + " "
	profession := layaProtectedProfessionFromText(normalized)
	if profession == "" {
		return layaScriptwriterNone, "", false
	}
	if strings.Contains(normalized, " не сценарист") ||
		strings.Contains(normalized, " не худож") ||
		strings.Contains(normalized, " не строител") ||
		strings.Contains(normalized, " не кодер") ||
		strings.Contains(normalized, " не программист") ||
		strings.Contains(normalized, " больше не сценарист") ||
		strings.Contains(normalized, " больше не худож") ||
		strings.Contains(normalized, " больше не строител") ||
		strings.Contains(normalized, " больше не кодер") ||
		strings.Contains(normalized, " больше не программист") ||
		strings.Contains(normalized, " убери сценар") ||
		strings.Contains(normalized, " убери худож") ||
		strings.Contains(normalized, " убери строител") ||
		strings.Contains(normalized, " убери кодер") ||
		strings.Contains(normalized, " сними сценар") ||
		strings.Contains(normalized, " сними худож") ||
		strings.Contains(normalized, " сними строител") ||
		strings.Contains(normalized, " сними кодер") ||
		strings.Contains(normalized, " можно мут") ||
		strings.Contains(normalized, " can mute") {
		return layaScriptwriterRemove, profession, true
	}
	if strings.Contains(normalized, " это сценарист") ||
		strings.Contains(normalized, " это худож") ||
		strings.Contains(normalized, " это строител") ||
		strings.Contains(normalized, " это кодер") ||
		strings.Contains(normalized, " это программист") ||
		strings.Contains(normalized, " это coder") ||
		(layaContainsProtectedProfession(normalized) && (strings.Contains(normalized, " не мут") ||
			strings.Contains(normalized, " не трог") ||
			strings.Contains(normalized, " не наказыва") ||
			strings.Contains(normalized, " не бан") ||
			strings.Contains(normalized, " dont mute") ||
			strings.Contains(normalized, " do not mute"))) {
		return layaScriptwriterAdd, profession, true
	}
	return layaScriptwriterNone, "", false
}

func layaContainsProtectedProfession(normalized string) bool {
	return layaProtectedProfessionFromText(normalized) != ""
}

func layaProtectedProfessionFromText(normalized string) string {
	normalized = " " + strings.TrimSpace(normalized) + " "
	switch {
	case strings.Contains(normalized, " сценар") || strings.Contains(normalized, " scriptwriter"):
		return "сценарист"
	case strings.Contains(normalized, " худож") || strings.Contains(normalized, " artist"):
		return "художник"
	case strings.Contains(normalized, " строител") || strings.Contains(normalized, " builder"):
		return "строитель"
	case strings.Contains(normalized, " кодер") ||
		strings.Contains(normalized, " программист") ||
		strings.Contains(normalized, " coder") ||
		strings.Contains(normalized, " developer"):
		return "кодер"
	default:
		return ""
	}
}

func normalizeLayaProtectedProfession(value string) string {
	value = " " + normalizeFAQText(value) + " "
	return layaProtectedProfessionFromText(value)
}

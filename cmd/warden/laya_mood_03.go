package main

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) maybeReplyToDirectLayaInsult(c tele.Context, text string, addressedByReply bool) bool {
	if !isDirectLayaInsult(text, addressedByReply) {
		return false
	}
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil {
		return true
	}
	fallback := layaDirectInsultReply(msg)
	state := a.activeLayaRuntimeState()
	if state.DirectInsultMode == "off" {
		return false
	}
	if strings.TrimSpace(state.DirectInsultFallback) != "" {
		fallback = state.DirectInsultFallback
	}
	reply := fallback
	status := "локальный ответ"
	elapsed := time.Duration(0)
	var aiErr error
	cfg := a.config().AI
	useAI := cfg.Enabled && state.DirectInsultMode != "local"
	if useAI {
		prompt := layaDirectInsultSystemPrompt
		if strings.TrimSpace(state.DirectInsultPrompt) != "" {
			prompt = state.DirectInsultPrompt
		}
		cfg.SystemPrompt = appendPrompt(cfg.SystemPrompt, prompt)
		cfg.FallbackReply = fallback
		cfg.MaxReplyChars = minInt(maxAIReplyChars(cfg), 600)
		reply, status, elapsed, aiErr = a.generateAIReplyForTelegramMessage(cfg, chat, sender, msg, text)
		reply = trimRunes(a.sanitizeLayaDirectReply(sender, reply), maxAIReplyChars(cfg))
		if strings.TrimSpace(reply) == "" {
			reply = fallback
		}
	}
	var err error
	if chat.Type == tele.ChatPrivate {
		err = c.Send(reply, a.layaInlineMarkup(chat, sender))
	} else if a.bot != nil {
		_, err = a.bot.Send(chat, reply, &tele.SendOptions{ReplyTo: msg, ThreadID: msg.ThreadID, ReplyMarkup: a.layaInlineMarkup(chat, sender)})
	} else {
		err = c.Send(reply, a.layaInlineMarkup(chat, sender))
	}
	if err != nil && a.log != nil {
		a.log.Warn("send direct laya insult reply failed", zap.Error(err))
	}
	if aiErr == nil && useAI {
		if err != nil {
			status = "ответ получен, отправка в Telegram не удалась"
		} else {
			status = "ответ отправлен"
		}
	}
	a.sendAIMessageLog("Прямое оскорбление Лайи", chat, msg, sender, text, reply, status, elapsed, firstNonNil(aiErr, err))
	if a.metrics != nil {
		a.metrics.IncModeration("laya_direct_insult_reply")
	}
	return true
}

func isDirectLayaInsult(text string, addressedByReply bool) bool {
	normalized := normalizeFAQText(text)
	if normalized == "" || !containsLayaInsult(normalized) || isDumbLayaQuestion(text, "") {
		return false
	}
	if strings.Contains(normalized, "иди нах") ||
		strings.Contains(normalized, "пошла нах") ||
		strings.Contains(normalized, "пошел нах") ||
		strings.Contains(normalized, "пошёл нах") ||
		strings.Contains(normalized, "заткнись") ||
		strings.Contains(normalized, "отвали") {
		return true
	}
	words := strings.Fields(normalized)
	if len(words) <= 2 {
		return true
	}
	if addressedByReply && len(words) <= 5 {
		return true
	}
	return containsTargetWord(normalized, map[string]bool{
		"laya": true,
		"лая":  true,
		"лайя": true,
		"лаю":  true,
	})
}

func layaDirectInsultReply(msg *tele.Message) string {
	if len(layaDirectInsultReplies) == 0 {
		return "Смело. Глупо, но смело."
	}
	if msg != nil && msg.ID > 0 {
		return layaDirectInsultReplies[(msg.ID-1)%len(layaDirectInsultReplies)]
	}
	return randomString(layaDirectInsultReplies)
}

func (a *App) allowLayaReaction(chatID int64, threadID int, kind string, now time.Time) bool {
	if a == nil {
		return false
	}
	key := fmt.Sprintf("%d:%d:%s", chatID, threadID, kind)
	a.layaReactionMu.Lock()
	defer a.layaReactionMu.Unlock()
	if a.lastLayaReaction == nil {
		a.lastLayaReaction = make(map[string]time.Time)
	}
	if last := a.lastLayaReaction[key]; !last.IsZero() && now.Sub(last) < layaReactionCooldown(kind) {
		return false
	}
	a.lastLayaReaction[key] = now
	return true
}

func layaReactionCooldown(kind string) time.Duration {
	switch kind {
	case "father_insult", "mother_insult", "brother_insult", "creator_insult":
		return 30 * time.Second
	case "spam":
		return 2 * time.Minute
	default:
		return defaultLayaReactionCooldown
	}
}

func (a *App) maybeReactToProtectedInsult(c tele.Context) bool {
	return false
}

func (a *App) protectedInsultReactionKind(msg *tele.Message) string {
	return ""
}

func containsLayaInsult(text string) bool {
	normalized := normalizeFAQText(text)
	for _, word := range strings.Fields(normalized) {
		if layaInsultWords[word] {
			return true
		}
		for _, prefix := range layaInsultPrefixes {
			if strings.HasPrefix(word, prefix) {
				return true
			}
		}
	}
	return false
}

func containsTargetWord(normalized string, targets map[string]bool) bool {
	for _, word := range strings.Fields(normalized) {
		if targets[word] {
			return true
		}
	}
	return false
}

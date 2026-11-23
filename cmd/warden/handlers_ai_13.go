package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func layaLooksLikeInternalLoopReply(text string) bool {
	normalized := normalizeFAQText(text)
	if normalized == "" {
		return false
	}
	for _, marker := range []string{
		"анти повтор", "зацикл", "пошло по кругу", "по кругу",
		"одну и ту же реплику", "служебн", "техническ объяснен", "внутренн инструкц",
		"same reply", "previous generation", "internal check", "style layer", "avoid style",
		"visible reply contract", "wording rules", "style notes", "hidden layer", "hidden layers",
		"process note", "diagnostic", "technical explanation", "mechanics", "fresh start", "draft to avoid",
	} {
		if strings.Contains(normalized, normalizeFAQText(marker)) {
			return true
		}
	}
	if strings.Contains(normalized, "повтор") &&
		(strings.Contains(normalized, "поймал") || strings.Contains(normalized, "заблок") || strings.Contains(normalized, "не отправ") || strings.Contains(normalized, "реплик")) {
		return true
	}
	if (strings.Contains(normalized, "начни заново") || strings.Contains(normalized, "давай заново") || strings.Contains(normalized, "сформулируй заново")) &&
		(strings.Contains(normalized, "контекст") || strings.Contains(normalized, "памят") || strings.Contains(normalized, "истори") || strings.Contains(normalized, "повтор")) {
		return true
	}
	if (strings.Contains(normalized, "памят") || strings.Contains(normalized, "контекст") || strings.Contains(normalized, "истори")) &&
		(strings.Contains(normalized, "сброс") || strings.Contains(normalized, "сброш") || strings.Contains(normalized, "очист") || strings.Contains(normalized, "обнул") || strings.Contains(normalized, "стерт") || strings.Contains(normalized, "удал")) {
		return true
	}
	return false
}

func layaAIReplyContext(systemPrompt string) string {
	switch {
	case strings.Contains(systemPrompt, randomChatFilterSystemPrompt):
		return "фильтр случайной реплики Лайи"
	case strings.Contains(systemPrompt, layaFasSystemPrompt):
		return "режим фас"
	case strings.Contains(systemPrompt, layaEventSystemPrompt):
		return "реакция на событие чата"
	default:
		return "обычный ответ Лайи"
	}
}

func (a *App) filterRandomChatMessage(recipient *tele.User, text string) string {
	original := strings.TrimSpace(text)
	if original == "" {
		return ""
	}
	fallback := a.sanitizeRandomLayaReply(recipient, original)
	cfg := a.config().AI
	if !cfg.Enabled || a.currentAI() == nil {
		return fallback
	}
	cfg.SystemPrompt = strings.TrimSpace(cfg.SystemPrompt + "\n\n" + randomChatFilterSystemPrompt + "\n\n" + a.randomLayaRelationshipSafetyPrompt(recipient))
	cfg.FallbackReply = fallback
	cfg.MaxReplyChars = minInt(maxAIReplyChars(cfg), telegramMaxTextChars)

	prompt := fmt.Sprintf("Исходная случайная реплика:\n%s\n\nВерни только безопасную финальную реплику для адресата.", original)
	reply, _, elapsed, err := a.generateAIReply(cfg, recipient, prompt)
	reply = strings.TrimSpace(reply)
	if err != nil || reply == "" {
		return fallback
	}
	reply = a.sanitizeRandomLayaReply(recipient, reply)
	if a.log != nil {
		a.log.Info("random chat message passed through AI filter",
			zap.Duration("duration", elapsed),
			zap.Int("source_chars", len([]rune(original))),
			zap.Int("reply_chars", len([]rune(reply))),
		)
	}
	return trimRunes(reply, maxAIReplyChars(cfg))
}

func (a *App) isTelegramGroupAdmin(chat *tele.Chat, user *tele.User) bool {
	if a == nil || a.bot == nil || chat == nil || user == nil || chat.Type == tele.ChatPrivate {
		return false
	}
	member, err := a.bot.ChatMemberOf(chat, user)
	if err != nil {
		if a.log != nil {
			a.log.Warn("telegram admin check failed", zap.Int64("chat", chat.ID), zap.Int64("user", user.ID), zap.Error(err))
		}
		return false
	}
	return member.Role == tele.Creator || member.Role == tele.Administrator
}

func (a *App) layaFasMute(args []string) (int64, string, error) {
	cfg := a.config().AI
	durationToken := cfg.FasMuteDuration
	reason := "Лайя фас"
	if len(args) > 0 {
		if _, _, ok, err := parseDurationToken(args[0]); ok || err != nil {
			if err != nil {
				return 0, "", err
			}
			durationToken = args[0]
			if len(args) > 1 {
				reason = strings.Join(args[1:], " ")
			}
		} else {
			reason = strings.Join(args, " ")
		}
	}

	d, permanent, ok, err := parseDurationToken(durationToken)
	if err != nil {
		return 0, "", err
	}
	if !ok {
		return 0, "", errors.New("duration is required")
	}
	if permanent || d == 0 {
		return 0, reason, nil
	}
	if d < 30*time.Second {
		return 0, "", errors.New("Telegram treats restrictions under 30 seconds as forever; use 30s or more")
	}
	return time.Now().UTC().Add(d).Unix(), reason, nil
}

func (a *App) applyMute(chat *tele.Chat, target targetUser, admin *tele.User, untilUnix int64, reason string) error {
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
		return err
	}

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
	return nil
}

func (a *App) logAIReplyFailure(sender *tele.User, text, status string, elapsed time.Duration, err error) {
	if a.log == nil {
		return
	}
	userID := int64(0)
	username := ""
	if sender != nil {
		userID = sender.ID
		username = sender.Username
	}
	a.log.Warn("ai reply failed",
		zap.Int64("user", userID),
		zap.String("username", username),
		zap.String("status", status),
		zap.Duration("duration", elapsed),
		zap.Int("request_chars", len([]rune(text))),
		zap.Error(err),
	)
}

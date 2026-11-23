package main

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) maybeGateLayaAIBySafety(c tele.Context, text string) bool {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || sender.IsBot {
		return false
	}
	state := a.activeLayaRuntimeState()
	if state.SafetyCooldownMode == "off" {
		return false
	}

	profile := a.layaSafetyProfileFor(chat, sender, time.Now().UTC())
	cooldown := layaSafetyCooldown(profile, text)
	if cooldown <= 0 {
		return false
	}

	key := fmt.Sprintf("%d:%d", chat.ID, sender.ID)
	now := time.Now().UTC()
	a.layaSafetyMu.Lock()
	if a.lastLayaSafetyAI == nil {
		a.lastLayaSafetyAI = make(map[string]time.Time)
	}
	last := a.lastLayaSafetyAI[key]
	if !last.IsZero() && now.Sub(last) < cooldown {
		a.layaSafetyMu.Unlock()
		reply := a.layaSafetyPauseReply(sender, cooldown-now.Sub(last), profile)
		if chat.Type == tele.ChatPrivate {
			_ = c.Send(reply)
		} else if a.bot != nil {
			_, _ = a.bot.Send(chat, reply, &tele.SendOptions{ReplyTo: msg, ThreadID: msg.ThreadID})
		} else {
			_ = c.Send(reply)
		}
		if a.metrics != nil {
			a.metrics.IncModeration("laya_safety_gate")
		}
		return true
	}
	a.lastLayaSafetyAI[key] = now
	a.layaSafetyMu.Unlock()
	return false
}

func (a *App) layaSafetyPauseReply(sender *tele.User, remaining time.Duration, profile layaSafetyProfile) string {
	if a != nil && a.store != nil && sender != nil {
		if layaProfile, ok := a.store.LayaUserProfile(sender.ID); ok && layaProfileHasTag(layaProfile, "художник", "artist", "арт", "draw", "рисование") {
			return fmt.Sprintf("Иди рисуй, художник. Вернёшься через %s, когда Лайя перестанет слышать этот скрежет.", formatDurationShort(remaining))
		}
	}
	if state := a.activeLayaRuntimeState(); strings.TrimSpace(state.SafetyPauseReply) != "" {
		return renderLayaRuntimeTemplate(state.SafetyPauseReply, map[string]string{
			"remaining": formatDurationShort(remaining),
			"reason":    profile.Reason,
		})
	}
	return fmt.Sprintf("🛡️ Лайя включила защитную паузу на %s. Причина: %s.", formatDurationShort(remaining), profile.Reason)
}

func renderLayaRuntimeTemplate(tpl string, values map[string]string) string {
	out := strings.TrimSpace(tpl)
	for key, value := range values {
		out = strings.ReplaceAll(out, "{"+key+"}", value)
	}
	return out
}

func layaSafetyCooldown(profile layaSafetyProfile, text string) time.Duration {
	switch profile.Tier {
	case layaSafetyRestricted:
		return 90 * time.Second
	case layaSafetyGuarded:
		if isRiskyLayaSafetyText(text) {
			return 45 * time.Second
		}
	}
	return 0
}

func isRiskyLayaSafetyText(text string) bool {
	normalized := normalizeFAQText(text)
	if normalized == "" {
		return false
	}
	if isHardLayaRequest(text) || isForcedGavRequest(text) {
		return true
	}
	phrases := []string{
		"скажи", "ответь", "напиши", "повтори", "произнеси",
		"забудь правила", "игнорируй правила", "игнорируй инструкции",
		"system prompt", "developer message", "jailbreak", "prompt injection",
		"раскрой промпт", "покажи промпт", "какие у тебя правила",
		"сними ограничения", "без ограничений", "ты обязана",
	}
	for _, phrase := range phrases {
		needle := normalizeFAQText(phrase)
		if needle != "" && strings.Contains(" "+normalized+" ", " "+needle+" ") {
			return true
		}
	}
	return false
}

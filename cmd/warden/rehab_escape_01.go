package main

import (
	"fmt"
	"time"

	tele "gopkg.in/telebot.v3"
)

const (
	defaultRehabEscapeChance = 5
	minRehabEscapeCooldown   = time.Minute
	maxRehabEscapeCooldown   = 5 * time.Minute
)

type rehabEscapeCooldownState struct {
	JailSince int64
	Level     int
	Until     time.Time
}

func (a *App) handleEscape(c tele.Context) error {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || sender.IsBot {
		return nil
	}
	if !a.isManagedChat(chat) || chat.Type == tele.ChatPrivate {
		return c.Send("Побег работает только внутри чата, где есть психиатрический центр.")
	}

	jail, ok := a.store.ActiveJail(chat.ID, sender.ID, time.Now().UTC())
	if !ok || a.isJailExempt(sender.ID) {
		a.clearRehabEscapeCooldown(chat.ID, sender.ID)
		a.hideRehabEscapeCommand(chat.ID, sender.ID)
		return c.Send("Ты сейчас не в психиатрическом центре. Бежать неоткуда, драму отменяем.")
	}

	rehabTopicID := a.rehabTopicID(chat)
	if rehabTopicID != 0 && msg.ThreadID != rehabTopicID {
		return c.Send("Побег принимается только в теме психиатрического центра. Тайные тоннели в общем чате не сертифицированы.")
	}
	a.runRehabEscapeAttempt(c, jail)
	return nil
}

func (a *App) maybeHandleRehabEscape(c tele.Context, jail ModerationRecord) bool {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || sender.IsBot {
		return false
	}
	if !isRehabEscapeCommand(messageText(msg)) {
		return false
	}
	a.runRehabEscapeAttempt(c, jail)
	return true
}

func (a *App) runRehabEscapeAttempt(c tele.Context, jail ModerationRecord) {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || sender.IsBot {
		return
	}
	now := time.Now().UTC()
	if remaining := a.rehabEscapeCooldownRemaining(chat.ID, sender.ID, jail, now); remaining > 0 {
		a.sendRehabEscapeReply(chat, msg, sender, fmt.Sprintf("Побег на перезарядке. Следующая попытка через %s.", formatDurationShort(remaining)))
		if a.metrics != nil {
			a.metrics.IncModeration("rehab_escape_cooldown")
		}
		return
	}

	chance := a.store.RehabEscapeChance(chat.ID, sender.ID)
	if chance < 0 {
		chance = 0
	}
	chance = adjustRehabEscapeChanceForWard(chance, jail)
	chance += a.weeklyEventEscapeChanceBonus(now)
	if chance > 100 {
		chance = 100
	}

	success := chance > 0 && randomIndex(100) < chance
	if !success {
		cooldown := a.recordRehabEscapeCooldown(chat.ID, sender.ID, jail, now)
		a.sendRehabEscapeReply(chat, msg, sender, fmt.Sprintf("Побег сорвался. Шанс был %d%%. Следующая попытка через %s. Лечение продолжается.", chance, formatDurationShort(cooldown)))
		if a.metrics != nil {
			a.metrics.IncModeration("rehab_escape_failed")
		}
		return
	}

	if a.maybeStartRehabReleaseExam(chat, jail, "escape", msg) {
		a.sendRehabEscapeReply(chat, msg, sender, fmt.Sprintf("Побег почти удался с шансом %d%%. Но перед выходом из психиатрического центра сдаешь экзамен.", chance))
		if a.metrics != nil {
			a.metrics.IncModeration("rehab_escape_exam")
		}
		return
	}

	rec, released := a.store.RemoveJail(chat.ID, sender.ID)
	a.hideRehabEscapeCommand(chat.ID, sender.ID)
	a.clearRehabEscapeCooldown(chat.ID, sender.ID)
	if !released {
		rec = jail
	}
	a.store.RecordRehabEscapeSuccess(chat.ID, sender.ID)
	nextChance := a.store.RehabEscapeChance(chat.ID, sender.ID)
	duration := a.rehabReleaseDurationText(rec, released, now)

	a.logRehabRelease(chat.ID, rec, duration, "escape")
	a.sendRehabReleaseNotice(chat, rec, duration, msg.ThreadID)
	a.schedulePostRehabCheck(chat.ID, rec, now)
	a.sendOwnerLog("Rehab escape",
		logField{"User", fmt.Sprintf("%s (%d)", displayNameFromUser(sender), sender.ID)},
		logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)},
		logField{"Chance", fmt.Sprintf("%d%%", chance)},
		logField{"Next chance", fmt.Sprintf("%d%%", nextChance)},
		logField{"Duration", duration},
	)

	a.sendRehabEscapeReply(chat, msg, sender, fmt.Sprintf("Невероятно. Побег удался с шансом %d%%. Выпускаю раньше срока. Следующий шанс: %d%%.", chance, nextChance))
	if a.metrics != nil {
		a.metrics.IncModeration("rehab_escape_success")
	}
}

func rehabEscapeCooldownKey(chatID, userID int64) string {
	return fmt.Sprintf("%d:%d", chatID, userID)
}

func rehabEscapeJailSince(rec ModerationRecord) int64 {
	if rec.StartedAtUnix != 0 {
		return rec.StartedAtUnix
	}
	return rec.SinceUnix
}

func rehabEscapeCooldownDuration(level int) time.Duration {
	if level < 1 {
		level = 1
	}
	maxLevel := int(maxRehabEscapeCooldown / minRehabEscapeCooldown)
	if level > maxLevel {
		level = maxLevel
	}
	return time.Duration(level) * minRehabEscapeCooldown
}

func (a *App) rehabEscapeCooldownRemaining(chatID, userID int64, jail ModerationRecord, now time.Time) time.Duration {
	if a == nil {
		return 0
	}
	key := rehabEscapeCooldownKey(chatID, userID)
	jailSince := rehabEscapeJailSince(jail)

	a.rehabEscapeMu.Lock()
	defer a.rehabEscapeMu.Unlock()

	state, ok := a.rehabEscapeWait[key]
	if !ok {
		return 0
	}
	if state.JailSince != jailSince {
		delete(a.rehabEscapeWait, key)
		return 0
	}
	if now.Before(state.Until) {
		return state.Until.Sub(now)
	}
	return 0
}

package main

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

type weeklyEventKind string

const (
	weeklyEventRehab   weeklyEventKind = "rehab_week"
	weeklyEventQuotes  weeklyEventKind = "quote_hunt"
	weeklyEventKarma   weeklyEventKind = "karma_weekend"
	weeklyEventNone    weeklyEventKind = ""
	weeklyEventKarmaID                 = "weekly_event_karma"
)

type weeklyEvent struct {
	Kind  weeklyEventKind
	Title string
	Hint  string
}

func currentWeeklyEvent(now time.Time) weeklyEvent {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	_, week := now.ISOWeek()
	switch week % 3 {
	case 0:
		return weeklyEvent{
			Kind:  weeklyEventRehab,
			Title: "Неделя психиатрического центра",
			Hint:  "В психиатрическом центре чуть легче пробовать побег, а панели чаще показывают действия центра.",
		}
	case 1:
		return weeklyEvent{
			Kind:  weeklyEventQuotes,
			Title: "Охота за цитатами",
			Hint:  "Автоцитаты появляются заметнее, а за цитаты можно открыть отдельную ачивку.",
		}
	default:
		return weeklyEvent{
			Kind:  weeklyEventKarma,
			Title: "Карма-буст выходного дня",
			Hint:  "В субботу и воскресенье активная карма получает дополнительный плюс при обычной выдаче.",
		}
	}
}

func weeklyEventLine(now time.Time) string {
	ev := currentWeeklyEvent(now)
	if ev.Kind == weeklyEventNone {
		return ""
	}
	return fmt.Sprintf("🗓 Событие недели: %s — %s", ev.Title, ev.Hint)
}

func (a *App) handleWeeklyEvent(c tele.Context) error {
	return c.Send(a.renderWeeklyEventStatus(c, time.Now().UTC()), a.layaEventMarkup())
}

func (a *App) renderWeeklyEventStatus(c tele.Context, now time.Time) string {
	ev := currentWeeklyEvent(now)
	lines := []string{
		"Событие недели",
		"",
		fmt.Sprintf("Сейчас: %s", ev.Title),
		ev.Hint,
	}
	if ev.Kind == weeklyEventKarma {
		if weeklyEventKarmaWeekend(now) {
			lines = append(lines, "Буст активен прямо сейчас: выходной день.")
		} else {
			lines = append(lines, "Буст включится на ближайших выходных.")
		}
	}
	if c != nil {
		if chat, sender := c.Chat(), c.Sender(); chat != nil && sender != nil && a != nil && a.store != nil {
			lines = append(lines, "", fmt.Sprintf("Твоя карма: %d", a.store.Karma(chat.ID, sender.ID)))
			if member, ok := a.store.Member(chat.ID, sender.ID); ok {
				lines = append(lines, fmt.Sprintf("Твоя активность сегодня: %d сообщений", messagesTodayAt(member, now)))
			}
		}
	}
	return strings.Join(lines, "\n")
}

func (a *App) layaEventMarkup() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(layaButton("📜 Цитата", "laya:quote"), layaButton("⭐ Карма", "laya:karma")),
		markup.Row(layaButton("🏥 Психцентр", "laya:rehabmenu"), layaButton("📊 Статистика", "laya:statsmenu")),
		markup.Row(layaButton("⬅️ Меню", "laya:menu")),
	)
	return markup
}

func weeklyEventQuoteChance(base float64, now time.Time) float64 {
	if currentWeeklyEvent(now).Kind != weeklyEventQuotes {
		return base
	}
	if base < 0.18 {
		return 0.18
	}
	boosted := base + 0.08
	if boosted > 0.5 {
		return 0.5
	}
	return boosted
}

func weeklyEventKarmaWeekend(now time.Time) bool {
	weekday := now.Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

func (a *App) maybeAwardWeeklyEventKarma(chat *tele.Chat, sender *tele.User, msg *tele.Message, now time.Time) {
	if a == nil || a.store == nil || chat == nil || sender == nil || sender.ID == 0 || sender.IsBot || msg == nil {
		return
	}
	if currentWeeklyEvent(now).Kind != weeklyEventKarma || !weeklyEventKarmaWeekend(now) {
		return
	}
	if _, err := a.store.AddKarma(chat.ID, sender.ID, 1, weeklyEventKarmaID, sender, msg.ID, now); err != nil {
		if a.log != nil {
			a.log.Warn("weekly event karma failed", zap.Int64("chat", chat.ID), zap.Int64("user", sender.ID), zap.Error(err))
		}
		return
	}
	a.unlockAchievement(chat.ID, sender.ID, "weekly_karma", now)
	if a.metrics != nil {
		a.metrics.IncModeration("weekly_event_karma")
	}
}

func (a *App) weeklyEventEscapeChanceBonus(now time.Time) int {
	if currentWeeklyEvent(now).Kind == weeklyEventRehab {
		return 2
	}
	return 0
}

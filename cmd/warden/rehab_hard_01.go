package main

import (
	"context"
	"math/rand"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

const layaEventSystemPrompt = `Режим реакций на события чата: отвечай одной короткой репликой от имени Лайи. Стиль саркастичный, живой, русский. Без угроз физической расправы, деанона и длинных объяснений.`

func (a *App) StartRehabReminders(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-timer.C:
			a.runRehabReminderTick(now)
			timer.Reset(time.Minute)
		}
	}
}

func (a *App) StartRehabTopicTicker(ctx context.Context) {
	for {
		interval := a.randomRehabTopicTickerInterval()
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case now := <-timer.C:
			a.runRehabTopicTicker(now.UTC())
		}
	}
}

func (a *App) randomRehabTopicTickerInterval() time.Duration {
	cfg := a.config().Rehab
	minInterval := mustDuration(cfg.TickerMinInterval, 3*time.Minute)
	maxInterval := mustDuration(cfg.TickerMaxInterval, 7*time.Minute)
	if minInterval <= 0 {
		minInterval = 3 * time.Minute
	}
	if maxInterval < minInterval {
		maxInterval = minInterval
	}
	if maxInterval == minInterval {
		return minInterval
	}
	return minInterval + time.Duration(rand.Int63n(int64(maxInterval-minInterval)))
}

type rehabTickerCandidate struct {
	chatID  int64
	topicID int
	rec     ModerationRecord
}

func (a *App) runRehabTopicTicker(now time.Time) {
	if a == nil || a.bot == nil || a.store == nil {
		return
	}
	candidates := make([]rehabTickerCandidate, 0)
	for _, chatID := range a.store.ChatIDs() {
		chat := &tele.Chat{ID: chatID, Type: tele.ChatSuperGroup}
		topicID := a.rehabTopicID(chat)
		if topicID == 0 {
			continue
		}
		for _, rec := range a.store.ModerationRecords(chatID, "jail", true, now) {
			if rec.UserID == 0 || rec.UntilUnix == 0 || rec.UntilUnix <= now.Unix() {
				continue
			}
			candidates = append(candidates, rehabTickerCandidate{chatID: chatID, topicID: topicID, rec: rec})
		}
	}
	if len(candidates) == 0 {
		return
	}
	candidate := candidates[rand.Intn(len(candidates))]
	minutes := 1 + rand.Intn(10)
	remaining := time.Unix(candidate.rec.UntilUnix, 0).UTC().Sub(now)
	if remaining < 10*time.Minute {
		realMinutes := int((remaining + time.Minute - time.Nanosecond) / time.Minute)
		if realMinutes < 1 {
			realMinutes = 1
		}
		minutes = realMinutes
	}
	chat := &tele.Chat{ID: candidate.chatID, Type: tele.ChatSuperGroup}
	if _, err := a.bot.Send(chat, a.t("rehab_ticker_message", minutes), &tele.SendOptions{ThreadID: candidate.topicID}); err != nil {
		if a.log != nil {
			a.log.Warn("send rehab topic ticker failed", zap.Int64("chat", candidate.chatID), zap.Int("topic", candidate.topicID), zap.Int64("user", candidate.rec.UserID), zap.Error(err))
		}
		return
	}
	if a.metrics != nil {
		a.metrics.IncModeration("rehab_topic_ticker")
	}
}

type expiredRehabRecord struct {
	chatID int64
	rec    ModerationRecord
}

func (a *App) expiredRehabRecords(now time.Time) []expiredRehabRecord {
	if a == nil || a.store == nil {
		return nil
	}
	var out []expiredRehabRecord
	for _, chatID := range a.store.ChatIDs() {
		for _, rec := range a.store.ModerationRecords(chatID, "jail", false, now) {
			if rec.UserID != 0 && rec.UntilUnix != 0 && rec.UntilUnix <= now.Unix() {
				out = append(out, expiredRehabRecord{chatID: chatID, rec: rec})
			}
		}
	}
	return out
}

func (a *App) rehabReleaseDurationText(rec ModerationRecord, ok bool, now time.Time) string {
	if !ok {
		return a.t("no_data")
	}
	return a.formatTreatmentDuration(rehabTreatmentDuration(rec, now))
}

func (a *App) sendRehabReleaseNotice(chat *tele.Chat, rec ModerationRecord, duration string, skipThreadID int) {
	if a == nil || a.bot == nil || chat == nil {
		return
	}
	topicID := a.rehabTopicID(chat)
	if topicID == 0 || topicID == skipThreadID {
		return
	}
	if _, err := a.bot.Send(chat, a.t("rehab_release_notice", duration), &tele.SendOptions{ThreadID: topicID}); err != nil {
		if a.log != nil {
			a.log.Warn("send rehab release notice failed", zap.Int64("chat", chat.ID), zap.Int64("user", rec.UserID), zap.Int("topic", topicID), zap.Error(err))
		}
	}
}

func (a *App) logRehabRelease(chatID int64, rec ModerationRecord, duration, source string) {
	if a == nil || a.log == nil {
		return
	}
	a.log.Info("rehab user released",
		zap.Int64("chat", chatID),
		zap.Int64("user", rec.UserID),
		zap.String("duration", duration),
		zap.String("source", source),
	)
}

func (a *App) runRehabReminderTick(now time.Time) {
	if a == nil || a.bot == nil || a.store == nil {
		return
	}
	cfg := a.config().Rehab
	if !cfg.HardMode || !cfg.PrivateReminders {
		return
	}
	slot, ok := rehabReminderSlot(cfg, now)
	if !ok {
		return
	}

	for _, chatID := range a.store.ChatIDs() {
		for _, rec := range a.store.ModerationRecords(chatID, "jail", true, now.UTC()) {
			if rec.UserID == 0 || !a.claimRehabNudge(chatID, rec.UserID, slot) {
				continue
			}
			a.sendRehabPrivateReminder(chatID, rec)
		}
	}
}

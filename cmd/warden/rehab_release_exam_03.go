package main

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) failRehabReleaseExam(chat *tele.Chat, msg *tele.Message, user *tele.User, session rehabReleaseExamSession, answer, source string) {
	if a == nil || a.store == nil || chat == nil {
		return
	}
	now := time.Now().UTC()
	a.clearRehabReleaseExam(session.ChatID, session.UserID)
	cfg := a.activeRehabReleaseExamConfigForChat(chat.ID)
	failDuration := mustDuration(cfg.FailDuration, 24*time.Hour)
	if failDuration <= 0 {
		failDuration = 24 * time.Hour
	}
	rec := session.Record
	if rec.UserID == 0 {
		rec.UserID = session.UserID
		rec.Username = session.Username
		rec.DisplayName = session.DisplayName
	}
	if user != nil {
		rec.UserID = user.ID
		rec.Username = user.Username
		rec.DisplayName = displayNameFromUser(user)
	}
	rec.Reason = "Не сдан экзамен перед выходом"
	rec.RehabWard = rehabWardRowdy
	rec.UntilUnix = now.Add(failDuration).Unix()
	a.store.SetJail(chat.ID, rec)
	a.clearRehabEscapeCooldown(chat.ID, rec.UserID)
	a.showRehabEscapeCommand(chat.ID, rec.UserID)

	threadID := a.rehabTopicID(chat)
	opts := &tele.SendOptions{ThreadID: threadID}
	if msg != nil {
		opts.ReplyTo = msg
		opts.ThreadID = msg.ThreadID
	}
	if _, err := a.bot.Send(chat, rehabReleaseExamFailMessage, opts); err != nil && a.log != nil {
		a.log.Warn("send rehab release exam fail reply failed", zap.Int64("chat", chat.ID), zap.Int64("user", rec.UserID), zap.Error(err))
	}
	a.sendOwnerLog("Rehab",
		logField{"User", fmt.Sprintf("%s (%d)", rec.DisplayName, rec.UserID)},
		logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)},
		logField{"Release", releaseLine(rec.UntilUnix, now)},
		logField{"Reason", rec.Reason},
		logField{"By", "экзамен Лайи"},
		logField{"Message", shortText(answer, 300)},
	)
	if a.metrics != nil {
		a.metrics.IncModeration("rehab_release_exam_failed_" + source)
	}
}

func (a *App) failExpiredRehabReleaseExams(now time.Time) {
	if a == nil || a.bot == nil || a.store == nil {
		return
	}
	var expired []rehabReleaseExamSession
	a.releaseExamMu.Lock()
	for key, session := range a.releaseExams {
		if !session.ExpiresAt.IsZero() && !now.Before(session.ExpiresAt) {
			expired = append(expired, session)
			delete(a.releaseExams, key)
		}
	}
	a.releaseExamMu.Unlock()

	for _, session := range expired {
		chat := &tele.Chat{ID: session.ChatID, Type: tele.ChatSuperGroup}
		user := &tele.User{ID: session.UserID, Username: session.Username, FirstName: session.DisplayName}
		a.failRehabReleaseExam(chat, nil, user, session, "время ответа вышло", "timeout")
	}
}

func (a *App) rehabReleaseExamAnswerAccepted(question, expected, answer string) bool {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return false
	}
	if ok, decided := a.rehabReleaseExamAIVerdict(question, expected, answer); decided {
		return ok
	}
	return fallbackRehabReleaseExamAnswerAccepted(expected, answer)
}

func (a *App) rehabReleaseExamAIVerdict(question, expected, answer string) (bool, bool) {
	cfg := a.config().AI
	ai := a.currentAI()
	if !cfg.Enabled || ai == nil {
		return false, false
	}
	cfg.SystemPrompt = rehabReleaseExamVerdictPrompt
	cfg.MaxTokens = 8
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	prompt := fmt.Sprintf("Вопрос: %s\nПравильный ответ: %s\nОтвет пользователя: %s\nВердикт:", strings.TrimSpace(question), strings.TrimSpace(expected), strings.TrimSpace(answer))
	reply, err := replyWithAI(ctx, ai, cfg, []AIMessage{{Role: "user", Content: prompt}}, prompt)
	if err != nil {
		if a.log != nil {
			a.log.Warn("rehab release exam AI verdict failed", zap.Error(err))
		}
		return false, false
	}
	verdict := strings.ToUpper(strings.TrimSpace(reply))
	switch {
	case strings.Contains(verdict, "BAD"):
		return false, true
	case strings.Contains(verdict, "OK"):
		return true, true
	default:
		return false, false
	}
}

func fallbackRehabReleaseExamAnswerAccepted(expected, answer string) bool {
	if containsLayaInsult(answer) {
		return false
	}
	normalizedAnswer := normalizeFAQText(answer)
	normalizedExpected := normalizeFAQText(expected)
	if normalizedAnswer == "" {
		return false
	}
	if normalizedExpected == "" {
		return len([]rune(normalizedAnswer)) >= 10
	}
	if normalizedContainsPhrase(normalizedAnswer, normalizedExpected) {
		return true
	}
	words := meaningfulExamWords(normalizedExpected)
	if len(words) == 0 {
		return false
	}
	matched := 0
	for _, word := range words {
		if strings.Contains(normalizedAnswer, word) {
			matched++
		}
	}
	need := int(math.Ceil(float64(len(words)) * 0.6))
	if need < 1 {
		need = 1
	}
	return matched >= need
}

func meaningfulExamWords(value string) []string {
	parts := strings.Fields(value)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(part, ".,!?;:()[]{}\"'")
		if len([]rune(part)) < 4 {
			continue
		}
		out = append(out, part)
	}
	return out
}

func (a *App) jailRecord(chatID, userID int64, now time.Time) (ModerationRecord, bool) {
	if a == nil || a.store == nil || chatID == 0 || userID == 0 {
		return ModerationRecord{}, false
	}
	for _, rec := range a.store.ModerationRecords(chatID, "jail", false, now) {
		if rec.UserID == userID {
			return rec, true
		}
	}
	return ModerationRecord{}, false
}

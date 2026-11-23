package main

import (
	"fmt"
	"html"
	"math/rand"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) maybeStartRehabReleaseExam(chat *tele.Chat, rec ModerationRecord, source string, replyTo *tele.Message) bool {
	if a == nil || a.bot == nil || a.store == nil || chat == nil || rec.UserID == 0 || a.isJailExempt(rec.UserID) {
		return false
	}
	cfg := a.activeRehabReleaseExamConfigForChat(chat.ID)
	if !cfg.Enabled {
		return false
	}
	topicID := a.rehabTopicID(chat)
	if topicID == 0 {
		return false
	}
	questions := parseRehabReleaseExamQuestions(cfg.Questions)
	if len(questions) == 0 {
		return false
	}

	now := time.Now().UTC()
	key := rehabStateKey(chat.ID, rec.UserID)
	a.releaseExamMu.Lock()
	if existing, ok := a.releaseExams[key]; ok && now.Before(existing.ExpiresAt) {
		a.releaseExamMu.Unlock()
		a.sendRehabReleaseExamAlreadyStarted(chat, replyTo, existing)
		return true
	}
	if a.releaseExams == nil {
		a.releaseExams = make(map[string]rehabReleaseExamSession)
	}
	qa := questions[rand.Intn(len(questions))]
	answerWindow := mustDuration(cfg.AnswerWindow, 10*time.Minute)
	if answerWindow <= 0 {
		answerWindow = 10 * time.Minute
	}
	expiresAt := now.Add(answerWindow)
	rec.UntilUnix = expiresAt.Unix()
	session := rehabReleaseExamSession{
		ChatID:      chat.ID,
		UserID:      rec.UserID,
		Username:    rec.Username,
		DisplayName: rec.DisplayName,
		Question:    qa.Question,
		Expected:    qa.Expected,
		Source:      source,
		StartedAt:   now,
		ExpiresAt:   expiresAt,
		Record:      rec,
	}
	a.releaseExams[key] = session
	a.releaseExamMu.Unlock()

	if !a.sendRehabReleaseExamQuestion(chat, replyTo, session) {
		a.clearRehabReleaseExam(chat.ID, rec.UserID)
		return false
	}
	a.store.SetJail(chat.ID, rec)
	if a.metrics != nil {
		a.metrics.IncModeration("rehab_release_exam_started")
	}
	return true
}

func (a *App) sendRehabReleaseExamQuestion(chat *tele.Chat, replyTo *tele.Message, session rehabReleaseExamSession) bool {
	if a == nil || a.bot == nil || chat == nil {
		return false
	}
	topicID := a.rehabTopicID(chat)
	if topicID == 0 {
		return false
	}
	name := strings.TrimSpace(session.DisplayName)
	if name == "" {
		name = fmt.Sprintf("user %d", session.UserID)
	}
	text := fmt.Sprintf("%s\n\n<b>Экзамен перед выходом из психиатрического центра.</b>\n%s\n\nОтветь одним сообщением. AI проверит строго.",
		mentionUser(session.UserID, name),
		html.EscapeString(session.Question),
	)
	opts := &tele.SendOptions{ThreadID: topicID, ParseMode: tele.ModeHTML}
	if replyTo != nil {
		opts.ReplyTo = replyTo
	}
	if _, err := a.bot.Send(chat, text, opts); err != nil {
		if a.log != nil {
			a.log.Warn("send rehab release exam question failed", zap.Int64("chat", chat.ID), zap.Int64("user", session.UserID), zap.Error(err))
		}
		return false
	}
	return true
}

func (a *App) sendRehabReleaseExamAlreadyStarted(chat *tele.Chat, replyTo *tele.Message, session rehabReleaseExamSession) {
	if a == nil || a.bot == nil || chat == nil || replyTo == nil {
		return
	}
	remaining := time.Until(session.ExpiresAt)
	if remaining < 0 {
		remaining = 0
	}
	text := fmt.Sprintf("Экзамен уже идет. Ответь на вопрос выше, осталось %s.", formatDurationShort(remaining))
	if _, err := a.bot.Send(chat, text, &tele.SendOptions{ReplyTo: replyTo, ThreadID: replyTo.ThreadID}); err != nil && a.log != nil {
		a.log.Warn("send rehab release exam duplicate reply failed", zap.Int64("chat", chat.ID), zap.Int64("user", session.UserID), zap.Error(err))
	}
}

func (a *App) maybeHandleRehabReleaseExamAnswer(c tele.Context, jail ModerationRecord) bool {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if a == nil || msg == nil || chat == nil || sender == nil || sender.IsBot {
		return false
	}
	session, ok := a.awaitingRehabReleaseExam(chat.ID, sender.ID, time.Now().UTC())
	if !ok {
		return false
	}
	answer := strings.TrimSpace(messageText(msg))
	if answer == "" {
		return false
	}
	if a.rehabReleaseExamAnswerAccepted(session.Question, session.Expected, answer) {
		a.passRehabReleaseExam(chat, msg, sender, session, jail)
		return true
	}
	a.failRehabReleaseExam(chat, msg, sender, session, answer, "answer")
	return true
}

func (a *App) awaitingRehabReleaseExam(chatID, userID int64, now time.Time) (rehabReleaseExamSession, bool) {
	a.releaseExamMu.Lock()
	defer a.releaseExamMu.Unlock()
	key := rehabStateKey(chatID, userID)
	session, ok := a.releaseExams[key]
	if !ok {
		return rehabReleaseExamSession{}, false
	}
	if !session.ExpiresAt.IsZero() && now.After(session.ExpiresAt) {
		delete(a.releaseExams, key)
		return rehabReleaseExamSession{}, false
	}
	return session, true
}

func (a *App) clearRehabReleaseExam(chatID, userID int64) {
	a.releaseExamMu.Lock()
	defer a.releaseExamMu.Unlock()
	delete(a.releaseExams, rehabStateKey(chatID, userID))
}

func (a *App) passRehabReleaseExam(chat *tele.Chat, msg *tele.Message, user *tele.User, session rehabReleaseExamSession, jail ModerationRecord) {
	now := time.Now().UTC()
	a.clearRehabReleaseExam(chat.ID, user.ID)
	rec, released := a.store.RemoveJail(chat.ID, user.ID)
	if !released {
		rec = session.Record
		if rec.UserID == 0 {
			rec = jail
		}
	}
	a.hideRehabEscapeCommand(chat.ID, user.ID)
	a.clearRehabEscapeCooldown(chat.ID, user.ID)
	a.resetRehabWardProgress(chat.ID, rec)
	duration := a.rehabReleaseDurationText(rec, true, now)
	a.logRehabRelease(chat.ID, rec, duration, "exam")
	if _, err := a.bot.Send(chat, "Экзамен сдан. Выпускаю из психиатрического центра.", &tele.SendOptions{ReplyTo: msg, ThreadID: msg.ThreadID}); err != nil && a.log != nil {
		a.log.Warn("send rehab release exam pass reply failed", zap.Int64("chat", chat.ID), zap.Int64("user", user.ID), zap.Error(err))
	}
	a.sendOwnerLog("Rehab released",
		logField{"User", fmt.Sprintf("%s (%d)", displayNameFromUser(user), user.ID)},
		logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)},
		logField{"Duration", duration},
		logField{"By", "экзамен Лайи"},
	)
	if a.metrics != nil {
		a.metrics.IncModeration("rehab_release_exam_passed")
	}
}

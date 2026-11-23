package main

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

const postRehabCheckVerdictPrompt = `Ты строгий классификатор ответа пользователя на проверку Лайи после психиатрического центра. Лайя задала нормальный вопрос, но пользователю он был отправлен нечитаемым шифром. Верни только OK или BAD.
OK: ответ спокойный, уважительный, позитивный или хотя бы нормально признает, что лечение пережито.
BAD: ответ пустой, бессмысленный, грубый, агрессивный, с оскорблениями, жалобой на лечение, отказом отвечать или желанием повторить лечение.`

type postRehabCheck struct {
	ChatID      int64
	UserID      int64
	Username    string
	DisplayName string
	ReleasedAt  time.Time
	DueAt       time.Time
	ExpiresAt   time.Time
	Question    string
	Asked       bool
}

func (a *App) StartPostRehabChecks(ctx context.Context) {
	timer := time.NewTimer(time.Minute)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-timer.C:
			a.runPostRehabCheckTick(now.UTC())
			timer.Reset(time.Minute)
		}
	}
}

func (a *App) schedulePostRehabCheck(chatID int64, rec ModerationRecord, now time.Time) {
	if a == nil || a.store == nil || chatID == 0 || rec.UserID == 0 || a.isJailExempt(rec.UserID) {
		return
	}
	cfg := a.config().Rehab
	if !cfg.PostReleaseCheckEnabled {
		return
	}
	chat := &tele.Chat{ID: chatID, Type: tele.ChatSuperGroup}
	if a.rehabTopicID(chat) == 0 {
		return
	}

	delay := randomPostRehabCheckDelay(cfg)
	answerWindow := mustDuration(cfg.PostReleaseCheckAnswerWindow, 2*time.Hour)
	if answerWindow <= 0 {
		answerWindow = 2 * time.Hour
	}
	check := postRehabCheck{
		ChatID:      chatID,
		UserID:      rec.UserID,
		Username:    rec.Username,
		DisplayName: textOrDash(rec.DisplayName),
		ReleasedAt:  now,
		DueAt:       now.Add(delay),
		ExpiresAt:   now.Add(delay + answerWindow),
	}

	a.postRehabMu.Lock()
	if a.postRehabChecks == nil {
		a.postRehabChecks = make(map[string]postRehabCheck)
	}
	a.postRehabChecks[rehabStateKey(chatID, rec.UserID)] = check
	a.postRehabMu.Unlock()

	if a.log != nil {
		a.log.Info("post-rehab check scheduled",
			zap.Int64("chat", chatID),
			zap.Int64("user", rec.UserID),
			zap.Duration("delay", delay),
		)
	}
}

func randomPostRehabCheckDelay(cfg RehabConfig) time.Duration {
	minDelay := mustDuration(cfg.PostReleaseCheckMinDelay, 3*time.Minute)
	maxDelay := mustDuration(cfg.PostReleaseCheckMaxDelay, 5*time.Hour)
	if minDelay <= 0 {
		minDelay = 3 * time.Minute
	}
	if maxDelay < minDelay {
		maxDelay = minDelay
	}
	if maxDelay == minDelay {
		return minDelay
	}
	return minDelay + time.Duration(rand.Int63n(int64(maxDelay-minDelay)))
}

func (a *App) runPostRehabCheckTick(now time.Time) {
	if a == nil || a.bot == nil || a.store == nil {
		return
	}
	for _, check := range a.claimDuePostRehabChecks(now) {
		a.sendPostRehabCheck(check, now)
	}
}

func (a *App) claimDuePostRehabChecks(now time.Time) []postRehabCheck {
	a.postRehabMu.Lock()
	defer a.postRehabMu.Unlock()

	var due []postRehabCheck
	for key, check := range a.postRehabChecks {
		if !check.ExpiresAt.IsZero() && now.After(check.ExpiresAt) {
			delete(a.postRehabChecks, key)
			continue
		}
		if check.Asked || now.Before(check.DueAt) {
			continue
		}
		check.Asked = true
		a.postRehabChecks[key] = check
		due = append(due, check)
	}
	return due
}

func (a *App) sendPostRehabCheck(check postRehabCheck, now time.Time) {
	chat := &tele.Chat{ID: check.ChatID, Type: tele.ChatSuperGroup}
	if _, jailed := a.store.ActiveJail(check.ChatID, check.UserID, now); jailed {
		a.clearPostRehabCheck(check.ChatID, check.UserID)
		return
	}
	topicID := a.rehabTopicID(chat)
	if topicID == 0 {
		a.clearPostRehabCheck(check.ChatID, check.UserID)
		return
	}

	question := a.generatePostRehabQuestion(check)
	encrypted := obfuscatePostRehabQuestion(question)
	name := check.DisplayName
	if name == "" || name == "-" {
		name = fmt.Sprintf("user %d", check.UserID)
	}
	text := mentionUser(check.UserID, name) + "\n" + encrypted
	if _, err := a.bot.Send(chat, text, &tele.SendOptions{ThreadID: topicID, ParseMode: tele.ModeHTML}); err != nil {
		a.clearPostRehabCheck(check.ChatID, check.UserID)
		if a.log != nil {
			a.log.Warn("send post-rehab check failed",
				zap.Int64("chat", check.ChatID),
				zap.Int64("user", check.UserID),
				zap.Error(err),
			)
		}
		return
	}
	a.markPostRehabQuestion(check.ChatID, check.UserID, question)
	if a.metrics != nil {
		a.metrics.IncModeration("post_rehab_check_sent")
	}
}

func (a *App) generatePostRehabQuestion(check postRehabCheck) string {
	name := strings.TrimSpace(check.DisplayName)
	if name == "" || name == "-" {
		name = fmt.Sprintf("user %d", check.UserID)
	}
	fallback := "Как самочувствие после лечения? Понравилось? Повторить хочешь?"
	prompt := fmt.Sprintf("Событие: пользователь %s только что вышел из психиатрического центра. Спроси коротко в стиле Лайи, как он себя чувствует после лечения, понравилось ли лечение и хочет ли он пройти его повторно. Верни обычный понятный русский текст одной репликой.", name)
	user := &tele.User{ID: check.UserID, Username: check.Username, FirstName: name}
	return a.generateLayaEventReply(user, prompt, fallback)
}

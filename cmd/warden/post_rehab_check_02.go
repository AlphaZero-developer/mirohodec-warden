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

func obfuscatePostRehabQuestion(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		text = "Как самочувствие после лечения?"
	}
	blocks := []rune("░▒▓█▄▀▌▐▖▗▘▙▚▛▜▟")
	noise := []rune{'\u0301', '\u0315', '\u0320', '\u0334', '\u0337', '\u034f', '\u035c', '\u0361', '\u200b', '\u200c', '\u200d'}

	var b strings.Builder
	b.WriteString("⟦")
	for _, r := range text {
		if r == '\n' || r == '\r' || r == '\t' || r == ' ' {
			b.WriteRune(' ')
			continue
		}
		b.WriteRune(blocks[rand.Intn(len(blocks))])
		for i := 0; i < 2+rand.Intn(4); i++ {
			b.WriteRune(noise[rand.Intn(len(noise))])
		}
	}
	b.WriteString("⟧")
	return trimRunes(b.String(), 900)
}

func (a *App) markPostRehabQuestion(chatID, userID int64, question string) {
	a.postRehabMu.Lock()
	defer a.postRehabMu.Unlock()
	key := rehabStateKey(chatID, userID)
	check, ok := a.postRehabChecks[key]
	if !ok {
		return
	}
	check.Question = strings.TrimSpace(question)
	check.Asked = true
	a.postRehabChecks[key] = check
}

func (a *App) maybeHandlePostRehabAnswer(c tele.Context) bool {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || sender.IsBot || chat.Type == tele.ChatPrivate {
		return false
	}
	now := time.Now().UTC()
	check, ok := a.awaitingPostRehabAnswer(chat.ID, sender.ID, now)
	if !ok {
		return false
	}
	if _, jailed := a.store.ActiveJail(chat.ID, sender.ID, now); jailed {
		a.clearPostRehabCheck(chat.ID, sender.ID)
		return false
	}

	answer := messageText(msg)
	if a.postRehabAnswerLiked(check.Question, answer) {
		a.clearPostRehabCheck(chat.ID, sender.ID)
		a.sendPostRehabAccepted(chat, msg)
		if a.metrics != nil {
			a.metrics.IncModeration("post_rehab_check_passed")
		}
		return true
	}

	a.clearPostRehabCheck(chat.ID, sender.ID)
	a.returnUserToRehabAfterCheck(chat, msg, sender, check, answer)
	return true
}

func (a *App) awaitingPostRehabAnswer(chatID, userID int64, now time.Time) (postRehabCheck, bool) {
	a.postRehabMu.Lock()
	defer a.postRehabMu.Unlock()
	key := rehabStateKey(chatID, userID)
	check, ok := a.postRehabChecks[key]
	if !ok {
		return postRehabCheck{}, false
	}
	if !check.ExpiresAt.IsZero() && now.After(check.ExpiresAt) {
		delete(a.postRehabChecks, key)
		return postRehabCheck{}, false
	}
	if !check.Asked {
		return postRehabCheck{}, false
	}
	return check, true
}

func (a *App) clearPostRehabCheck(chatID, userID int64) {
	a.postRehabMu.Lock()
	defer a.postRehabMu.Unlock()
	delete(a.postRehabChecks, rehabStateKey(chatID, userID))
}

func (a *App) postRehabAnswerLiked(question, answer string) bool {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return false
	}
	if liked, ok := a.postRehabAIVerdict(question, answer); ok {
		return liked
	}
	return fallbackPostRehabAnswerLiked(answer)
}

func (a *App) postRehabAIVerdict(question, answer string) (bool, bool) {
	cfg := a.config().AI
	ai := a.currentAI()
	if !cfg.Enabled || ai == nil {
		return false, false
	}
	cfg.SystemPrompt = postRehabCheckVerdictPrompt
	cfg.MaxTokens = 8
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	prompt := fmt.Sprintf("Вопрос Лайи: %s\nОтвет пользователя: %s\nВердикт:", strings.TrimSpace(question), strings.TrimSpace(answer))
	reply, err := replyWithAI(ctx, ai, cfg, []AIMessage{{Role: "user", Content: prompt}}, prompt)
	if err != nil {
		if a.log != nil {
			a.log.Warn("post-rehab AI verdict failed", zap.Error(err))
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

func fallbackPostRehabAnswerLiked(answer string) bool {
	if containsLayaInsult(answer) {
		return false
	}
	normalized := normalizeFAQText(answer)
	if normalized == "" {
		return false
	}
	for _, phrase := range []string{
		"не понрав", "плохо", "ужас", "ненавиж", "не хочу", "отстан", "иди", "пошла", "пошел", "пошёл",
		"дурка", "психушка", "лечиться снова", "верни обратно", "хочу обратно",
	} {
		if normalizedContainsPhrase(normalized, phrase) {
			return false
		}
	}
	for _, phrase := range []string{
		"норм", "нормально", "хорошо", "отлично", "лучше", "жив", "жива", "спасибо", "понрав", "вылечился", "вылечилась", "ок", "да",
	} {
		if normalizedContainsPhrase(normalized, phrase) {
			return true
		}
	}
	return false
}

func (a *App) sendPostRehabAccepted(chat *tele.Chat, msg *tele.Message) {
	if a == nil || a.bot == nil || chat == nil || msg == nil {
		return
	}
	if _, err := a.bot.Send(chat, "Ладно. Живи пока.", &tele.SendOptions{ReplyTo: msg, ThreadID: msg.ThreadID}); err != nil && a.log != nil {
		a.log.Warn("send post-rehab accepted reply failed", zap.Int64("chat", chat.ID), zap.Error(err))
	}
}

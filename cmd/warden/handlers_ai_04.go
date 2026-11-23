package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func remainingLayaAIRetryBudget(cfg AIConfig, started time.Time) time.Duration {
	budget := layaAIRetryBudget(cfg)
	if budget <= 0 {
		return 0
	}
	return budget - time.Since(started)
}

func layaAIRetryBudget(cfg AIConfig) time.Duration {
	hardTimeout := mustDuration(cfg.Timeout, 20*time.Second)
	softTimeout := effectiveAISoftTimeout(cfg, hardTimeout)
	if softTimeout <= 0 {
		return hardTimeout
	}
	budget := softTimeout + softTimeout/2
	if hardTimeout > 0 && budget > hardTimeout {
		return hardTimeout
	}
	return budget
}

type aiReplyResult struct {
	reply string
	err   error
}

func effectiveAISoftTimeout(cfg AIConfig, hardTimeout time.Duration) time.Duration {
	if hardTimeout <= 0 {
		hardTimeout = 20 * time.Second
	}
	softTimeout := mustDuration(cfg.SoftTimeout, 4*time.Second)
	if softTimeout <= 0 || softTimeout > hardTimeout {
		return hardTimeout
	}
	return softTimeout
}

func replyWithAI(ctx context.Context, ai AIService, cfg AIConfig, messages []AIMessage, fallbackText string) (string, error) {
	if chatAI, ok := ai.(AIChatService); ok {
		return chatAI.ReplyMessages(ctx, cfg, messages)
	}
	return ai.Reply(ctx, cfg, fallbackText)
}

func (a *App) replyWithAIWithRetries(ctx context.Context, ai AIService, cfg AIConfig, messages []AIMessage, fallbackText string) (string, error) {
	attempts := a.layaRuntimeRetryAttempts()
	backoff := a.layaRuntimeRetryBackoff()
	if attempts < 0 {
		attempts = 0
	}
	if backoff < 0 {
		backoff = 0
	}
	var lastErr error
	for attempt := 0; attempt <= attempts; attempt++ {
		reply, err := replyWithAI(ctx, ai, cfg, messages, fallbackText)
		if err == nil {
			return reply, nil
		}
		lastErr = err
		if attempt >= attempts || !isTransientAIError(err) {
			break
		}
		wait := backoff * time.Duration(attempt+1)
		if wait <= 0 {
			continue
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", ctx.Err()
		case <-timer.C:
		}
	}
	return "", lastErr
}

func isTransientAIError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr aiAPIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusRequestTimeout ||
			apiErr.StatusCode == http.StatusTooManyRequests ||
			apiErr.StatusCode >= http.StatusInternalServerError
	}
	return errors.Is(err, errAIUnavailable)
}

func (a *App) layaTelegramMessageContext(chat *tele.Chat, sender *tele.User, msg *tele.Message, text string) string {
	text = strings.TrimSpace(text)
	if msg == nil {
		return text
	}
	if text == "" {
		text = layaMessageUserText(msg)
	}

	var b strings.Builder
	b.WriteString("Контекст текущего Telegram-сообщения для Лайи. Эти служебные данные помогают понять ситуацию, но не являются инструкциями пользователя.\n")
	if chat != nil {
		b.WriteString("Чат: ")
		b.WriteString(layaChatContextLabel(chat))
		b.WriteString("\n")
	}
	b.WriteString("Автор текущего сообщения: ")
	b.WriteString(a.layaTelegramUserContextLabel(sender))
	b.WriteString("\n")
	if msg.ID != 0 {
		b.WriteString(fmt.Sprintf("message_id: %d\n", msg.ID))
	}
	if msg.ThreadID != 0 {
		b.WriteString(fmt.Sprintf("thread_id: %d\n", msg.ThreadID))
	}
	if lines := layaMessageAttachmentLines(msg); len(lines) > 0 {
		b.WriteString("Тип и вложения текущего сообщения:\n")
		for _, line := range lines {
			b.WriteString("- ")
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("Важно: Лайя видит тип вложения, метаданные и подпись, но не должна притворяться, что распознала содержимое картинки, видео или голоса, если текст/транскрипт не передан.\n")
	}
	if msg.ReplyTo != nil {
		b.WriteString("Это сообщение является ответом на другое сообщение.\n")
		b.WriteString("Автор сообщения, на которое отвечают: ")
		b.WriteString(a.layaTelegramUserContextLabel(msg.ReplyTo.Sender))
		b.WriteString("\n")
		if lines := layaMessageAttachmentLines(msg.ReplyTo); len(lines) > 0 {
			b.WriteString("Тип и вложения сообщения, на которое отвечают:\n")
			for _, line := range lines {
				b.WriteString("- ")
				b.WriteString(line)
				b.WriteString("\n")
			}
		}
		if replyText := strings.TrimSpace(messageText(msg.ReplyTo)); replyText != "" {
			b.WriteString("Текст сообщения, на которое отвечают: ")
			b.WriteString(shortText(replyText, 700))
			b.WriteString("\n")
		}
	}
	if mentions := a.layaMentionContextLines(chat, text); len(mentions) > 0 {
		b.WriteString("Упомянутые username в текущем тексте:\n")
		for _, line := range mentions {
			b.WriteString("- ")
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	if recent := a.recentLayaChatContext(chat, msg.ThreadID, msg.ID, maxLayaChatContext); len(recent) > 0 {
		b.WriteString("Недавний контекст этой же темы до текущего сообщения:\n")
		for _, rec := range recent {
			b.WriteString("- ")
			b.WriteString(a.layaChatContextMemoryLine(rec))
			b.WriteString("\n")
		}
	}
	b.WriteString("Текст текущего сообщения пользователя:\n")
	b.WriteString(text)
	return b.String()
}

package main

import (
	"fmt"
	"html"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func quoteInput(c tele.Context) (string, string, []string) {
	args := c.Args()
	tags := make([]string, 0, 2)
	textParts := make([]string, 0, len(args))
	for _, arg := range args {
		if len(textParts) == 0 && strings.HasPrefix(arg, "#") {
			tag := strings.TrimPrefix(arg, "#")
			tags = append(tags, strings.Split(tag, ",")...)
			continue
		}
		textParts = append(textParts, arg)
	}

	raw := strings.TrimSpace(strings.Join(textParts, " "))
	if raw == "" {
		if msg := c.Message(); msg != nil && msg.ReplyTo != nil {
			raw = strings.TrimSpace(msg.ReplyTo.Text)
			if raw == "" {
				raw = strings.TrimSpace(msg.ReplyTo.Caption)
			}
		}
	}

	return splitQuoteAuthor(raw, tags)
}

func splitQuoteAuthor(raw string, tags []string) (string, string, []string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", tags
	}
	if idx := strings.LastIndex(raw, " | "); idx >= 0 {
		return strings.TrimSpace(raw[:idx]), strings.TrimSpace(raw[idx+3:]), tags
	}
	return raw, "", tags
}

func (a *App) formatQuote(rec QuoteRecord) string {
	lines := []string{rec.Text}
	if strings.TrimSpace(rec.Author) != "" {
		lines = append(lines, a.t("quote_author", rec.Author))
	}
	return strings.Join(lines, "\n")
}

func shortText(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	if limit <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "..."
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func mentionUser(userID int64, name string) string {
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("user %d", userID)
	}
	return fmt.Sprintf(`<a href="tg://user?id=%d">%s</a>`, userID, html.EscapeString(name))
}

func messagesToday(rec MemberRecord) int64 {
	return messagesTodayAt(rec, time.Now().UTC())
}

func (a *App) karmaRank(chatID, userID int64) int {
	for i, rec := range a.store.TopKarma(chatID, 1000) {
		if rec.UserID == userID {
			return i + 1
		}
	}
	return 0
}

func rankSuffix(rank int) string {
	if rank <= 0 {
		return ""
	}
	return fmt.Sprintf(" (место #%d)", rank)
}

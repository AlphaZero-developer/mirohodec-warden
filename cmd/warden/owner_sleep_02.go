package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func messageMentionsSleepingOwner(msg *tele.Message, sleep OwnerSleepState) (string, bool) {
	if msg == nil || sleep.UserID == 0 {
		return "", false
	}
	if msg.ReplyTo != nil && msg.ReplyTo.Sender != nil && msg.ReplyTo.Sender.ID == sleep.UserID &&
		!isForumTopicRootReply(msg) && strings.TrimSpace(messageText(msg)) != "" {
		return "ответ на сообщение создателя", true
	}
	for _, entity := range msg.Entities {
		if entity.Type == tele.EntityTMention && entity.User != nil && entity.User.ID == sleep.UserID {
			return "текстовое упоминание создателя", true
		}
	}
	for _, entity := range msg.CaptionEntities {
		if entity.Type == tele.EntityTMention && entity.User != nil && entity.User.ID == sleep.UserID {
			return "текстовое упоминание создателя", true
		}
	}

	text := messageText(msg)
	if strings.TrimSpace(text) == "" {
		return "", false
	}
	normalized := normalizeFAQText(text)
	if normalized == "" {
		return "", false
	}
	words := make(map[string]struct{})
	for _, word := range strings.Fields(normalized) {
		words[word] = struct{}{}
	}

	id := strconv.FormatInt(sleep.UserID, 10)
	if _, ok := words[id]; ok {
		return id, true
	}
	username := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(sleep.Username)), "@")
	if username != "" {
		if strings.Contains(strings.ToLower(text), "@"+username) {
			return "@" + username, true
		}
		if _, ok := words[username]; ok {
			return username, true
		}
	}

	for _, alias := range sleep.Aliases {
		matched, ok := ownerSleepAliasMatch(alias, normalized, words)
		if ok {
			return matched, true
		}
	}
	return "", false
}

func isForumTopicRootReply(msg *tele.Message) bool {
	if msg == nil || msg.ReplyTo == nil || msg.ThreadID == 0 {
		return false
	}
	if msg.ReplyTo.TopicCreated != nil {
		return true
	}
	return msg.ReplyTo.ID == msg.ThreadID
}

func ownerSleepAliasMatch(alias, normalized string, words map[string]struct{}) (string, bool) {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return "", false
	}
	clean := strings.TrimPrefix(strings.ToLower(alias), "@")
	norm := normalizeFAQText(clean)
	if norm == "" || norm == "unknown" {
		return "", false
	}
	if _, err := strconv.ParseInt(norm, 10, 64); err == nil {
		if _, ok := words[norm]; ok {
			return alias, true
		}
		return "", false
	}
	parts := strings.Fields(norm)
	if len(parts) == 1 {
		if len([]rune(parts[0])) < 3 {
			return "", false
		}
		if _, ok := words[parts[0]]; ok {
			return alias, true
		}
		return "", false
	}
	if strings.Contains(" "+normalized+" ", " "+norm+" ") {
		return alias, true
	}
	return "", false
}

func ownerSleepCallFromMessage(chat *tele.Chat, sender *tele.User, msg *tele.Message, matched string) OwnerSleepCall {
	call := OwnerSleepCall{
		AtUnix:  time.Now().UTC().Unix(),
		Matched: strings.TrimSpace(matched),
	}
	if chat != nil {
		call.ChatID = chat.ID
		call.ChatTitle = chat.Title
	}
	if sender != nil {
		call.FromID = sender.ID
		call.FromUsername = sender.Username
		call.FromName = displayNameFromUser(sender)
	}
	if msg != nil {
		call.ThreadID = msg.ThreadID
		call.MessageID = msg.ID
		call.Text = trimOwnerSleepLogText(messageText(msg), 700)
	}
	return call
}

func (a *App) sendOwnerSleepChatReply(c tele.Context, text string) {
	msg := c.Message()
	chat := c.Chat()
	if chat == nil || strings.TrimSpace(text) == "" {
		return
	}
	var err error
	if chat.Type == tele.ChatPrivate {
		err = c.Send(text)
	} else {
		_, err = a.bot.Send(chat, text, &tele.SendOptions{ReplyTo: msg, ThreadID: msg.ThreadID})
	}
	if err != nil && a.log != nil {
		a.log.Warn("send owner sleep reply failed", zap.Error(err))
	}
}

func ownerSleepCallsSummary(calls []OwnerSleepCall, limit int) string {
	if len(calls) == 0 {
		return "-"
	}
	if limit <= 0 || limit > len(calls) {
		limit = len(calls)
	}
	start := len(calls) - limit
	var b strings.Builder
	for i, call := range calls[start:] {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(formatUnix(call.AtUnix))
		b.WriteString(" — ")
		b.WriteString(textOrDash(call.FromName))
		if call.FromID != 0 {
			b.WriteString(fmt.Sprintf(" (%d)", call.FromID))
		}
		b.WriteString(" в ")
		b.WriteString(textOrDash(call.ChatTitle))
		if call.Matched != "" {
			b.WriteString("; совпало: ")
			b.WriteString(call.Matched)
		}
		if call.Text != "" {
			b.WriteString("; текст: ")
			b.WriteString(trimOwnerSleepLogText(call.Text, 140))
		}
	}
	if len(calls) > limit {
		b.WriteString(fmt.Sprintf("\n...и ещё %d", len(calls)-limit))
	}
	return b.String()
}

func trimOwnerSleepLogText(text string, limit int) string {
	text = strings.TrimSpace(text)
	if limit <= 0 || len([]rune(text)) <= limit {
		return text
	}
	runes := []rune(text)
	return strings.TrimSpace(string(runes[:limit])) + "..."
}

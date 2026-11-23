package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) sendAIMessageLog(title string, chat *tele.Chat, msg *tele.Message, sender *tele.User, text, reply, status string, elapsed time.Duration, err error) {
	if a.store == nil || chat == nil || sender == nil {
		return
	}
	if !a.config().AI.LogConversations {
		return
	}
	if a.store.LogChatID() == chat.ID {
		return
	}

	username := "-"
	if sender.Username != "" {
		username = "@" + sender.Username
	}

	fields := []logField{
		{Name: "Пользователь", Value: fmt.Sprintf("%s (%d)", displayNameFromUser(sender), sender.ID)},
		{Name: "Username", Value: username},
		{Name: "Статус", Value: status},
		{Name: "Время", Value: elapsed.Truncate(time.Millisecond).String()},
	}
	if err != nil {
		fields = append(fields, aiErrorLogFields(err)...)
	}
	if chat.Type != tele.ChatPrivate {
		fields = append(fields, logField{Name: "Чат", Value: fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)})
		if msg != nil && msg.ThreadID != 0 {
			fields = append(fields, logField{Name: "Тема", Value: fmt.Sprintf("%d", msg.ThreadID)})
		}
	}
	fields = append(fields, logField{Name: "Сообщение", Value: shortText(text, 700)})
	if strings.TrimSpace(reply) != "" {
		fields = append(fields, logField{Name: "Ответ", Value: shortText(reply, 700)})
	}
	a.sendOwnerLog(title, fields...)
}

func (a *App) notifyOwnerAIReplyFallback(title string, chat *tele.Chat, msg *tele.Message, sender *tele.User, status string, err error) {
	if a == nil || err == nil {
		return
	}
	fields := []logField{
		{Name: "Message", Value: "Извините, я не смогла ответить по нейронке."},
		{Name: "Статус", Value: status},
	}
	if sender != nil {
		fields = append(fields, logField{Name: "User", Value: fmt.Sprintf("%s (%d)", displayNameFromUser(sender), sender.ID)})
	}
	if chat != nil {
		fields = append(fields, logField{Name: "Chat", Value: fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)})
	}
	if msg != nil && msg.ThreadID != 0 {
		fields = append(fields, logField{Name: "Topic", Value: fmt.Sprintf("%d", msg.ThreadID)})
	}
	fields = append(fields, aiErrorLogFields(err)...)
	a.sendOwnerLog("AI fallback", fields...)
	if creatorID := a.activeLayaConfig().CreatorUserID; creatorID != 0 && (a.store == nil || a.store.LogChatID() != creatorID) {
		a.sendOwnerDirectLog(creatorID, "AI fallback", fields...)
	}
}

func aiErrorLogFields(err error) []logField {
	if err == nil {
		return nil
	}
	var apiErr aiAPIError
	if errors.As(err, &apiErr) {
		fields := []logField{
			{Name: "OpenRouter status", Value: fmt.Sprintf("%d", apiErr.StatusCode)},
		}
		if strings.TrimSpace(apiErr.Message) != "" {
			fields = append(fields, logField{Name: "OpenRouter error", Value: shortText(apiErr.Message, 500)})
		}
		return fields
	}
	return []logField{{Name: "Ошибка AI", Value: shortText(err.Error(), 500)}}
}

func (a *App) formatAIReplyForDelivery(cfg AIConfig, chat *tele.Chat, sender *tele.User, msg *tele.Message, reply string) string {
	reply = strings.TrimSpace(reply)
	mode := normalizeAIReplyEncoding(cfg.ReplyEncoding)
	if reply == "" {
		return reply
	}
	targetsBot := a.layaAIReplyTargetsBot(chat, sender, msg)
	targetCipher, targetsConfiguredUser := a.replyCipherForTelegramTarget(cfg, chat, sender, msg)
	explicitBinary := layaMessageRequestsBinary(msg)
	if mode == "" {
		if !targetsBot && !targetsConfiguredUser && !explicitBinary {
			return reply
		}
		mode = "bot_binary"
	}
	if mode == "bot_binary" && !targetsBot && !targetsConfiguredUser && !explicitBinary {
		return reply
	}
	cipher := a.effectiveReplyCipherDefault(cfg)
	if targetCipher != "" {
		cipher = targetCipher
	}
	if explicitBinary {
		cipher = "binary"
	}
	if targetsBot {
		reply = strings.TrimSpace(replyCipherBotInstruction(cipher) + " " + reply)
	}
	return encodeTextForReplyCipher(reply, cipher, maxAIReplyChars(cfg))
}

func formatAIReplyTextForEncoding(cfg AIConfig, reply string) string {
	reply = strings.TrimSpace(reply)
	if reply == "" || normalizeAIReplyEncoding(cfg.ReplyEncoding) != "binary" {
		return reply
	}
	return encodeTextForReplyCipher(reply, normalizeAIReplyCipherOrDefault(cfg.ReplyCipher), maxAIReplyChars(cfg))
}

func normalizeAIReplyEncoding(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "binary", "bin", "always_binary":
		return "binary"
	case "bot_binary", "bots_binary", "bot", "bots", "auto_binary":
		return "bot_binary"
	default:
		return ""
	}
}

func layaMessageRequestsBinary(msg *tele.Message) bool {
	if msg == nil {
		return false
	}
	words := strings.Fields(normalizeFAQText(messageText(msg)))
	for i, word := range words {
		switch word {
		case "binary", "bin", "бинарный", "бинарного", "бинарному", "бинарным", "бинарном", "бинарную", "бинарные", "двоичный", "двоичного", "двоичному", "двоичным", "двоичном", "двоичную", "двоичные":
			return true
		case "код":
			if i > 0 && (strings.HasPrefix(words[i-1], "бинар") || strings.HasPrefix(words[i-1], "двоич")) {
				return true
			}
		}
	}
	return false
}

func encodeTextToBinary(text string, maxChars int) string {
	text = strings.TrimSpace(text)
	if text == "" || maxChars <= 0 {
		return ""
	}
	maxBytes := (maxChars + 1) / 9
	if maxBytes <= 0 {
		return ""
	}
	text = trimUTF8ByteLimit(text, maxBytes)
	data := []byte(text)
	if len(data) == 0 {
		return ""
	}

	var b strings.Builder
	b.Grow(len(data)*9 - 1)
	for i, value := range data {
		if i > 0 {
			b.WriteByte(' ')
		}
		for bit := 7; bit >= 0; bit-- {
			if value&(1<<uint(bit)) != 0 {
				b.WriteByte('1')
			} else {
				b.WriteByte('0')
			}
		}
	}
	return b.String()
}

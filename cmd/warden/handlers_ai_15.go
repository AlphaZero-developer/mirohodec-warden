package main

import (
	"strings"
	"unicode"
	"unicode/utf8"

	tele "gopkg.in/telebot.v3"
)

func trimUTF8ByteLimit(text string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	used := 0
	for i, r := range text {
		size := utf8.RuneLen(r)
		if size <= 0 {
			size = len(string(r))
		}
		if used+size > maxBytes {
			return strings.TrimSpace(text[:i])
		}
		used += size
	}
	return text
}

func (a *App) layaTelegramMessageHasExternalBotPeer(sender *tele.User, msg *tele.Message) bool {
	if sender != nil && sender.IsBot && !a.isOwnBotUser(sender) {
		return true
	}
	return msg != nil && msg.ReplyTo != nil && msg.ReplyTo.Sender != nil && msg.ReplyTo.Sender.IsBot && !a.isOwnBotUser(msg.ReplyTo.Sender)
}

func (a *App) layaAIReplyTargetsBot(chat *tele.Chat, sender *tele.User, msg *tele.Message) bool {
	if sender != nil && sender.IsBot && !a.isOwnBotUser(sender) {
		return true
	}
	if msg != nil && msg.ReplyTo != nil && msg.ReplyTo.Sender != nil && msg.ReplyTo.Sender.IsBot && !a.isOwnBotUser(msg.ReplyTo.Sender) {
		return true
	}
	if a == nil || a.store == nil || chat == nil || msg == nil {
		return false
	}
	selfUsername := ""
	if a.bot != nil && a.bot.Me != nil {
		selfUsername = normalizeUsername(a.bot.Me.Username)
	}
	for _, username := range layaMentionUsernames(messageText(msg)) {
		if username == "" || username == selfUsername {
			continue
		}
		if target, ok := a.store.UserByUsername(chat.ID, username); ok {
			if target.ID != 0 && a.isOwnBotID(target.ID) {
				continue
			}
			if target.IsBot {
				return true
			}
			continue
		}
		if strings.HasSuffix(username, "bot") {
			return true
		}
	}
	return false
}

func (a *App) isOwnBotUser(user *tele.User) bool {
	return user != nil && a.isOwnBotID(user.ID)
}

func (a *App) isOwnBotID(userID int64) bool {
	return a != nil && a.bot != nil && a.bot.Me != nil && userID != 0 && userID == a.bot.Me.ID
}

func aiFallbackReply(cfg AIConfig) string {
	reply := strings.TrimSpace(cfg.FallbackReply)
	if reply == "" {
		return defaultAIFallbackReply
	}
	if normalizeFAQText(reply) == normalizeFAQText("Я сейчас не в настроении. Я не хочу с тобой говорить.") {
		return defaultAIFallbackReply
	}
	return reply
}

func maxAIReplyChars(cfg AIConfig) int {
	if cfg.MaxReplyChars <= 0 {
		return defaultAIMaxReplyChars
	}
	return minInt(cfg.MaxReplyChars, telegramMaxTextChars)
}

func layaFasArgs(text, botUsername string) ([]string, bool) {
	fields := strings.Fields(text)
	if len(fields) < 2 {
		return nil, false
	}
	first := normalizeLayaCommandToken(fields[0])
	second := normalizeLayaCommandToken(fields[1])
	if (first == "лайя" && second == "фас") || (first == "laya" && second == "fas") {
		return fields[2:], true
	}
	botUsername = normalizeUsername(botUsername)
	if botUsername != "" && normalizeUsername(first) == botUsername && (second == "фас" || second == "fas") {
		return fields[2:], true
	}
	return nil, false
}

func normalizeLayaCommandToken(value string) string {
	value = strings.ToLower(strings.ReplaceAll(value, "ё", "е"))
	value = strings.TrimFunc(value, func(r rune) bool {
		return isZeroWidthRune(r) || (!unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '@')
	})
	return value
}

package main

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) rememberLayaChatContext(chat *tele.Chat, msg *tele.Message, sender *tele.User) {
	if a == nil || chat == nil || msg == nil || sender == nil || sender.ID == 0 || chat.Type == tele.ChatPrivate {
		return
	}
	if a.bot != nil && a.bot.Me != nil && sender.ID == a.bot.Me.ID {
		return
	}
	text := strings.TrimSpace(layaMessageMemoryText(msg))
	rawText := strings.TrimSpace(messageText(msg))
	if text == "" || commandName(rawText) != "" {
		return
	}
	rec := layaChatContextMessage{
		MessageID:   msg.ID,
		ThreadID:    msg.ThreadID,
		UserID:      sender.ID,
		Username:    strings.TrimPrefix(strings.TrimSpace(sender.Username), "@"),
		DisplayName: displayNameFromUser(sender),
		IsBot:       sender.IsBot,
		Text:        trimRunes(strings.Join(strings.Fields(text), " "), maxLayaContextTextRunes),
		Unix:        time.Now().UTC().Unix(),
	}
	key := layaChatContextKey(chat.ID, msg.ThreadID)

	a.layaContextMu.Lock()
	defer a.layaContextMu.Unlock()
	if a.layaChatContext == nil {
		a.layaChatContext = make(map[string][]layaChatContextMessage)
	}
	items := append(a.layaChatContext[key], rec)
	if len(items) > maxLayaChatMemory {
		items = append([]layaChatContextMessage(nil), items[len(items)-maxLayaChatMemory:]...)
	}
	a.layaChatContext[key] = items
}

func (a *App) layaReplyBridgeContext(chat *tele.Chat, msg *tele.Message, currentSender *tele.User) (string, []AIMessage) {
	if a == nil || chat == nil || msg == nil || msg.ReplyTo == nil || msg.ReplyTo.ID == 0 {
		return "", nil
	}
	linkKey := layaAIReplyMessageKey(chat.ID, msg.ReplyTo.ID)
	currentKey := aiConversationKey(chat, currentSender)

	a.aiHistoryMu.Lock()
	link, ok := a.aiReplyLinks[linkKey]
	if !ok || link.ConversationKey == "" || (link.ConversationKey == currentKey && !link.Carrier) {
		a.aiHistoryMu.Unlock()
		return "", nil
	}
	source := trimLayaBridgeSnapshot(link.Snapshot)
	if len(source) == 0 {
		source = trimAIConversationHistory(a.aiHistory[link.ConversationKey])
	}
	bridgeContext := strings.TrimSpace(link.BridgeContext)
	replyText := strings.TrimSpace(link.ReplyText)
	a.aiHistoryMu.Unlock()

	if len(source) == 0 {
		return "", nil
	}
	history := make([]AIMessage, len(source))
	copy(history, source)
	if replyText == "" && msg.ReplyTo != nil {
		replyText = strings.TrimSpace(messageText(msg.ReplyTo))
	}

	var b strings.Builder
	b.WriteString("Контекст подключенной истории Лайи из сообщения, на которое сейчас отвечают реплаем.\n")
	b.WriteString("Источник подключенной истории: ")
	b.WriteString(layaAIReplyLinkSourceLabel(link))
	b.WriteString(". Текущий автор нового сообщения: ")
	b.WriteString(a.layaTelegramUserContextLabel(currentSender))
	b.WriteString(". Не приписывай текущему автору чужие слова; используй историю только как общий контекст разговора.\n")
	if replyText != "" {
		b.WriteString("Конкретное сообщение цепочки, на которое сейчас отвечают: ")
		b.WriteString(shortText(replyText, 700))
		b.WriteString("\n")
	}
	if bridgeContext != "" {
		b.WriteString("Уже подключенный контекст предыдущей цепочки:\n")
		b.WriteString(trimRunes(bridgeContext, 1800))
		b.WriteString("\n")
	}
	b.WriteString("История того диалога:\n")
	for _, item := range history {
		content := shortText(strings.TrimSpace(item.Content), 700)
		if content == "" {
			continue
		}
		switch item.Role {
		case "assistant":
			b.WriteString("- Лайя: ")
		default:
			b.WriteString("- ")
			b.WriteString(layaAIReplyLinkSourceShortLabel(link))
			b.WriteString(": ")
		}
		b.WriteString(content)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String()), history
}

func layaAIReplyLinkSourceLabel(link layaAIReplyLink) string {
	name := strings.TrimSpace(link.SourceName)
	if name == "" {
		name = fmt.Sprintf("user %d", link.SourceUserID)
	}
	username := strings.TrimSpace(link.SourceUsername)
	if username != "" && !strings.HasPrefix(username, "@") {
		username = "@" + username
	}
	if username == "" {
		username = "нет username"
	}
	return fmt.Sprintf("%s (%s, id=%d, bot=%t)", name, username, link.SourceUserID, link.SourceIsBot)
}

func layaAIReplyLinkSourceShortLabel(link layaAIReplyLink) string {
	name := strings.TrimSpace(link.SourceName)
	if name == "" {
		return fmt.Sprintf("Пользователь %d", link.SourceUserID)
	}
	return fmt.Sprintf("%s [user_id=%d]", name, link.SourceUserID)
}

func (a *App) recentLayaChatContext(chat *tele.Chat, threadID, currentMessageID, limit int) []layaChatContextMessage {
	if a == nil || chat == nil || limit <= 0 || chat.Type == tele.ChatPrivate {
		return nil
	}
	key := layaChatContextKey(chat.ID, threadID)
	a.layaContextMu.Lock()
	defer a.layaContextMu.Unlock()
	items := a.layaChatContext[key]
	if len(items) == 0 {
		return nil
	}
	out := make([]layaChatContextMessage, 0, limit)
	for i := len(items) - 1; i >= 0 && len(out) < limit; i-- {
		rec := items[i]
		if currentMessageID != 0 && rec.MessageID == currentMessageID {
			continue
		}
		if strings.TrimSpace(rec.Text) == "" {
			continue
		}
		out = append(out, rec)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func (a *App) layaChatContextMemoryLine(rec layaChatContextMessage) string {
	username := strings.TrimSpace(rec.Username)
	if username != "" && !strings.HasPrefix(username, "@") {
		username = "@" + username
	}
	if username == "" {
		username = "нет username"
	}
	when := ""
	if rec.Unix != 0 {
		when = time.Unix(rec.Unix, 0).UTC().Format("15:04") + " "
	}
	return fmt.Sprintf("%s%s (%s, id=%d, bot=%t%s): %s", when, rec.DisplayName, username, rec.UserID, rec.IsBot, a.layaProfileContextForUserID(rec.UserID), rec.Text)
}

func layaChatContextKey(chatID int64, threadID int) string {
	return fmt.Sprintf("%d:%d", chatID, threadID)
}

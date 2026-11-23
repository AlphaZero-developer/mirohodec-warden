package main

import (
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) rememberLayaAIReplyMessage(chat *tele.Chat, sent *tele.Message, sourceSender *tele.User) {
	if a == nil || chat == nil || sent == nil || sent.ID == 0 || sourceSender == nil || sourceSender.ID == 0 {
		return
	}
	conversationKey := aiConversationKey(chat, sourceSender)
	linkKey := layaAIReplyMessageKey(chat.ID, sent.ID)
	if conversationKey == "" || linkKey == "" {
		return
	}
	a.aiHistoryMu.Lock()
	defer a.aiHistoryMu.Unlock()
	if a.aiReplyLinks == nil {
		a.aiReplyLinks = make(map[string]layaAIReplyLink)
	}
	snapshot := trimAIConversationHistory(a.aiHistory[conversationKey])
	bridgeContext := a.popPendingLayaBridgeTextLocked(conversationKey)
	if extra := a.popPendingLayaBridgeSnapshotLocked(conversationKey); len(extra) > 0 && bridgeContext == "" {
		snapshot = append(append([]AIMessage(nil), extra...), snapshot...)
	}
	snapshot = trimLayaBridgeSnapshot(snapshot)
	a.aiReplyLinks[linkKey] = layaAIReplyLink{
		ConversationKey: conversationKey,
		ChatID:          chat.ID,
		ThreadID:        sent.ThreadID,
		SourceUserID:    sourceSender.ID,
		SourceUsername:  strings.TrimPrefix(strings.TrimSpace(sourceSender.Username), "@"),
		SourceName:      displayNameFromUser(sourceSender),
		SourceIsBot:     sourceSender.IsBot,
		MessageID:       sent.ID,
		Unix:            time.Now().UTC().Unix(),
		ReplyText:       strings.TrimSpace(messageText(sent)),
		BridgeContext:   bridgeContext,
		Snapshot:        snapshot,
		Carrier:         false,
	}
	a.pruneLayaAIReplyLinksLocked()
}

func (a *App) rememberLayaBridgeCarrierMessage(chat *tele.Chat, msg *tele.Message, sender *tele.User, bridgeText string, bridgeHistory []AIMessage) {
	if a == nil || chat == nil || msg == nil || msg.ID == 0 || sender == nil || sender.ID == 0 {
		return
	}
	conversationKey := aiConversationKey(chat, sender)
	linkKey := layaAIReplyMessageKey(chat.ID, msg.ID)
	if conversationKey == "" || linkKey == "" {
		return
	}
	replyText := strings.TrimSpace(layaMessageMemoryText(msg))
	snapshot := []AIMessage(nil)
	if replyText != "" {
		snapshot = append(snapshot, AIMessage{Role: "user", Content: replyText})
	}
	if len(snapshot) == 0 && len(bridgeHistory) == 0 {
		return
	}
	a.aiHistoryMu.Lock()
	defer a.aiHistoryMu.Unlock()
	if a.aiReplyLinks == nil {
		a.aiReplyLinks = make(map[string]layaAIReplyLink)
	}
	a.aiReplyLinks[linkKey] = layaAIReplyLink{
		ConversationKey: conversationKey,
		ChatID:          chat.ID,
		ThreadID:        msg.ThreadID,
		SourceUserID:    sender.ID,
		SourceUsername:  strings.TrimPrefix(strings.TrimSpace(sender.Username), "@"),
		SourceName:      displayNameFromUser(sender),
		SourceIsBot:     sender.IsBot,
		MessageID:       msg.ID,
		Unix:            time.Now().UTC().Unix(),
		ReplyText:       replyText,
		BridgeContext:   strings.TrimSpace(bridgeText),
		Snapshot:        trimLayaBridgeSnapshot(snapshot),
		Carrier:         true,
	}
	a.pruneLayaAIReplyLinksLocked()
}

func (a *App) setPendingLayaBridgeContext(chat *tele.Chat, sender *tele.User, bridgeText string, snapshot []AIMessage) {
	key := aiConversationKey(chat, sender)
	if a == nil || key == "" {
		return
	}
	snapshot = trimLayaBridgeSnapshot(snapshot)
	bridgeText = strings.TrimSpace(bridgeText)
	a.aiHistoryMu.Lock()
	defer a.aiHistoryMu.Unlock()
	if a.aiBridgeNext == nil {
		a.aiBridgeNext = make(map[string][]AIMessage)
	}
	if a.aiBridgeText == nil {
		a.aiBridgeText = make(map[string]string)
	}
	if len(snapshot) == 0 && bridgeText == "" {
		delete(a.aiBridgeNext, key)
		delete(a.aiBridgeText, key)
		return
	}
	if len(snapshot) > 0 {
		a.aiBridgeNext[key] = snapshot
	}
	if bridgeText != "" {
		a.aiBridgeText[key] = bridgeText
	}
}

func (a *App) clearPendingLayaBridgeSnapshot(chat *tele.Chat, sender *tele.User) {
	key := aiConversationKey(chat, sender)
	if a == nil || key == "" {
		return
	}
	a.aiHistoryMu.Lock()
	defer a.aiHistoryMu.Unlock()
	delete(a.aiBridgeNext, key)
	delete(a.aiBridgeText, key)
}

func (a *App) popPendingLayaBridgeSnapshotLocked(key string) []AIMessage {
	if a == nil || key == "" || len(a.aiBridgeNext) == 0 {
		return nil
	}
	snapshot := a.aiBridgeNext[key]
	delete(a.aiBridgeNext, key)
	return trimLayaBridgeSnapshot(snapshot)
}

func (a *App) popPendingLayaBridgeTextLocked(key string) string {
	if a == nil || key == "" || len(a.aiBridgeText) == 0 {
		return ""
	}
	text := strings.TrimSpace(a.aiBridgeText[key])
	delete(a.aiBridgeText, key)
	return text
}

func (a *App) clearAIConversation(chat *tele.Chat, sender *tele.User) {
	key := aiConversationKey(chat, sender)
	if key == "" {
		return
	}
	a.aiHistoryMu.Lock()
	defer a.aiHistoryMu.Unlock()
	delete(a.aiHistory, key)
}

func (a *App) aiNoRepeatReplies(chat *tele.Chat, sender *tele.User) []string {
	key := aiConversationKey(chat, sender)
	chatKey := aiChatNoRepeatKey(chat)
	if key == "" && chatKey == "" {
		return nil
	}
	a.aiHistoryMu.Lock()
	defer a.aiHistoryMu.Unlock()
	replies := mergeAINoRepeatReplies(a.aiNoRepeat[key], a.aiNoRepeat[chatKey])
	if len(replies) == 0 {
		return nil
	}
	return replies
}

func (a *App) rememberAINoRepeatReplies(chat *tele.Chat, sender *tele.User, replies ...string) {
	key := aiConversationKey(chat, sender)
	chatKey := aiChatNoRepeatKey(chat)
	if (key == "" && chatKey == "") || len(replies) == 0 {
		return
	}
	a.aiHistoryMu.Lock()
	defer a.aiHistoryMu.Unlock()
	for _, reply := range replies {
		a.appendAINoRepeatReplyLocked(key, reply)
		a.appendAINoRepeatReplyLocked(chatKey, reply)
	}
}

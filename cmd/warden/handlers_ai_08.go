package main

import (
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v3"
)

func (a *App) appendAINoRepeatReplyLocked(key, reply string) {
	reply = strings.TrimSpace(reply)
	signature := aiReplySignature(reply)
	if key == "" || signature == "" {
		return
	}
	if a.aiNoRepeat == nil {
		a.aiNoRepeat = make(map[string][]string)
	}
	replies := a.aiNoRepeat[key]
	for _, old := range replies {
		oldSignature := aiReplySignature(old)
		if oldSignature == signature || aiReplySignaturesSimilar(oldSignature, signature) {
			return
		}
	}
	replies = append(replies, reply)
	if len(replies) > maxLayaNoRepeatMemory {
		replies = append([]string(nil), replies[len(replies)-maxLayaNoRepeatMemory:]...)
	}
	a.aiNoRepeat[key] = replies
}

func aiConversationKey(chat *tele.Chat, sender *tele.User) string {
	if sender == nil || sender.ID == 0 {
		return ""
	}
	chatID := int64(0)
	if chat != nil {
		chatID = chat.ID
	}
	return fmt.Sprintf("%d:%d", chatID, sender.ID)
}

func layaAIReplyMessageKey(chatID int64, messageID int) string {
	if chatID == 0 || messageID == 0 {
		return ""
	}
	return fmt.Sprintf("%d:%d", chatID, messageID)
}

func (a *App) pruneLayaAIReplyLinksLocked() {
	if len(a.aiReplyLinks) <= maxLayaReplyBridgeLinks {
		return
	}
	for len(a.aiReplyLinks) > maxLayaReplyBridgeLinks {
		oldestKey := ""
		oldestUnix := int64(0)
		for key, link := range a.aiReplyLinks {
			if oldestKey == "" || link.Unix < oldestUnix {
				oldestKey = key
				oldestUnix = link.Unix
			}
		}
		if oldestKey == "" {
			return
		}
		delete(a.aiReplyLinks, oldestKey)
	}
}

func aiChatNoRepeatKey(chat *tele.Chat) string {
	if chat == nil || chat.ID == 0 {
		return ""
	}
	return fmt.Sprintf("chat:%d", chat.ID)
}

func mergeAINoRepeatReplies(groups ...[]string) []string {
	var out []string
	var seen []string
	for _, group := range groups {
		for _, reply := range group {
			signature := aiReplySignature(reply)
			if signature == "" {
				continue
			}
			if aiReplySignatureSeen(seen, signature) {
				continue
			}
			seen = append(seen, signature)
			out = append(out, reply)
		}
	}
	if len(out) > maxLayaNoRepeatMemory {
		out = append([]string(nil), out[len(out)-maxLayaNoRepeatMemory:]...)
	}
	return out
}

func trimAIConversationHistory(messages []AIMessage) []AIMessage {
	if len(messages) <= maxAIConversationHistory {
		return messages
	}
	return messages[len(messages)-maxAIConversationHistory:]
}

func trimMessageList(messages []AIMessage, limit int) []AIMessage {
	if limit <= 0 || len(messages) <= limit {
		return messages
	}
	return messages[len(messages)-limit:]
}

func appendLayaNoRepeatMemory(history []AIMessage, replies []string) []AIMessage {
	if len(replies) == 0 {
		return history
	}
	out := make([]AIMessage, 0, len(history)+len(replies))
	out = append(out, history...)
	seen := make([]string, 0, len(history)+len(replies))
	for _, msg := range history {
		if msg.Role == "assistant" {
			if signature := aiReplySignature(msg.Content); signature != "" {
				seen = append(seen, signature)
			}
		}
	}
	for _, reply := range replies {
		signature := aiReplySignature(reply)
		if signature == "" {
			continue
		}
		if aiReplySignatureSeen(seen, signature) {
			continue
		}
		seen = append(seen, signature)
		out = append(out, AIMessage{Role: "assistant", Content: reply})
	}
	return out
}

func trimLayaBridgeSnapshot(messages []AIMessage) []AIMessage {
	if len(messages) == 0 {
		return nil
	}
	out := make([]AIMessage, 0, len(messages))
	seen := make(map[string]struct{}, len(messages))
	for _, msg := range messages {
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		role := strings.TrimSpace(msg.Role)
		if role == "" {
			role = "user"
		}
		signature := role + ":" + aiReplySignature(content)
		if signature != role+":" {
			if _, ok := seen[signature]; ok {
				continue
			}
			seen[signature] = struct{}{}
		}
		out = append(out, AIMessage{Role: role, Content: content})
	}
	return trimMessageList(out, maxLayaBridgeSnapshot)
}

func isRepeatedAIReply(reply string, history []AIMessage) bool {
	return isRepeatedAIReplyAgainst(reply, history, nil)
}

func isRepeatedAIReplyAgainst(reply string, history []AIMessage, rejected []string) bool {
	signature := aiReplySignature(reply)
	if signature == "" {
		return false
	}
	for _, oldReply := range rejected {
		if aiReplySignaturesSimilar(signature, aiReplySignature(oldReply)) {
			return true
		}
	}
	for i := len(history) - 1; i >= 0; i-- {
		msg := history[i]
		if msg.Role != "assistant" {
			continue
		}
		if aiReplySignaturesSimilar(signature, aiReplySignature(msg.Content)) {
			return true
		}
	}
	return false
}

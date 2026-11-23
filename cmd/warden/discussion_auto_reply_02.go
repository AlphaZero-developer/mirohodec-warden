package main

import (
	tele "gopkg.in/telebot.v3"
)

func (a *App) isDiscussionAutoReplyPost(chat *tele.Chat, msg *tele.Message, sender *tele.User) bool {
	if chat == nil || msg == nil {
		return false
	}
	if chat.Type != tele.ChatSuperGroup || discussionAutoReplyMarkerID(msg) == 0 || msg.ReplyTo != nil {
		return false
	}
	if commandName(messageText(msg)) != "" {
		return false
	}
	cfg := a.activeDiscussionConfig()
	if !isDiscussionAutoReplySourceAllowed(msg, cfg) {
		return false
	}
	if cfg.StrictChecks {
		return msg.AutomaticForward && (cfg.SourceChatID == 0 || discussionAutoReplyHasSourceChat(msg))
	}
	if msg.AutomaticForward {
		return true
	}
	if msg.SenderChat != nil && (msg.SenderChat.Type == tele.ChatChannel || msg.SenderChat.Type == tele.ChatChannelPrivate) {
		return true
	}
	if msg.OriginalChat != nil && (msg.OriginalChat.Type == tele.ChatChannel || msg.OriginalChat.Type == tele.ChatChannelPrivate) {
		return true
	}
	if msg.Origin != nil && msg.Origin.SenderChat != nil && (msg.Origin.SenderChat.Type == tele.ChatChannel || msg.Origin.SenderChat.Type == tele.ChatChannelPrivate) {
		return true
	}
	if sender != nil && sender.IsBot {
		return false
	}
	if sender != nil && a.isConfiguredAdmin(sender.ID) {
		return true
	}
	if sender != nil && a.config().Security.TrustTelegramAdmins && a.isTelegramGroupAdmin(chat, sender) {
		return true
	}
	return false
}

func isDiscussionAutoReplySourceAllowed(msg *tele.Message, cfg DiscussionAutoReplyConfig) bool {
	if cfg.SourceChatID == 0 {
		return true
	}
	source, ok := discussionAutoReplySourceChat(msg)
	return ok && source.ID == cfg.SourceChatID
}

func discussionAutoReplyHasSourceChat(msg *tele.Message) bool {
	_, ok := discussionAutoReplySourceChat(msg)
	return ok
}

func discussionAutoReplySourceChat(msg *tele.Message) (*tele.Chat, bool) {
	if msg == nil {
		return nil, false
	}
	if msg.Origin != nil && isDiscussionAutoReplySourceChat(msg.Origin.SenderChat) {
		return msg.Origin.SenderChat, true
	}
	if msg.Origin != nil && isDiscussionAutoReplySourceChat(msg.Origin.Chat) {
		return msg.Origin.Chat, true
	}
	if isDiscussionAutoReplySourceChat(msg.OriginalChat) {
		return msg.OriginalChat, true
	}
	if isDiscussionAutoReplySourceChat(msg.SenderChat) {
		return msg.SenderChat, true
	}
	return nil, false
}

func isDiscussionAutoReplySourceChat(chat *tele.Chat) bool {
	if chat == nil {
		return false
	}
	return chat.Type == tele.ChatChannel || chat.Type == tele.ChatChannelPrivate || chat.Type == tele.ChatSuperGroup
}

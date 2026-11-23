package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) layaScriptwriterTarget(chatID int64, msg *tele.Message, senderID int64) (targetUser, bool) {
	if msg == nil {
		return targetUser{}, false
	}
	if msg.ReplyTo != nil && msg.ReplyTo.Sender != nil && msg.ReplyTo.Sender.ID != 0 && msg.ReplyTo.Sender.ID != senderID {
		u := msg.ReplyTo.Sender
		return targetUser{ID: u.ID, Username: u.Username, Name: displayNameFromUser(u)}, true
	}
	if target, ok := a.layaScriptwriterTargetFromEntities(chatID, msg); ok {
		return target, true
	}
	for _, raw := range strings.Fields(messageText(msg)) {
		if target, ok := a.layaScriptwriterTargetFromToken(chatID, raw); ok {
			return target, true
		}
	}
	return targetUser{}, false
}

func (a *App) layaScriptwriterTargetFromEntities(chatID int64, msg *tele.Message) (targetUser, bool) {
	if msg == nil {
		return targetUser{}, false
	}
	for _, entity := range msg.Entities {
		if entity.Type == tele.EntityTMention && entity.User != nil && entity.User.ID != 0 {
			u := entity.User
			return targetUser{ID: u.ID, Username: u.Username, Name: displayNameFromUser(u)}, true
		}
	}
	for _, entity := range msg.CaptionEntities {
		if entity.Type == tele.EntityTMention && entity.User != nil && entity.User.ID != 0 {
			u := entity.User
			return targetUser{ID: u.ID, Username: u.Username, Name: displayNameFromUser(u)}, true
		}
	}
	return targetUser{}, false
}

func (a *App) layaScriptwriterTargetFromToken(chatID int64, raw string) (targetUser, bool) {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, ".,:;!?()[]{}<>«»\"'`")
	raw = strings.TrimPrefix(raw, "tg://user?id=")
	if raw == "" || strings.EqualFold(raw, "laya") || strings.EqualFold(raw, "лайя") {
		return targetUser{}, false
	}
	if username := usernameFromTarget(raw); username != "" {
		target, ok := a.store.UserByUsername(chatID, username)
		return target, ok
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id == 0 {
		return targetUser{}, false
	}
	target := targetUser{ID: id, Name: fmt.Sprintf("user %d", id)}
	if rec, ok := a.store.Member(chatID, id); ok {
		target.Username = rec.Username
		target.Name = rec.DisplayName
	}
	return target, true
}

func (a *App) unmuteScriptwriterSpamMute(chat *tele.Chat, target targetUser) {
	if a == nil || a.store == nil || chat == nil || target.ID == 0 {
		return
	}
	mute, ok := a.store.ActiveMute(chat.ID, target.ID, time.Now().UTC())
	if !ok || !a.isBotMute(mute) {
		return
	}
	if a.bot != nil {
		member := &tele.ChatMember{
			User:   &tele.User{ID: target.ID},
			Rights: tele.NoRestrictions(),
		}
		if err := a.bot.Restrict(chat, member); err != nil {
			if a.log != nil {
				a.log.Warn("scriptwriter spam unmute failed", zap.Int64("chat", chat.ID), zap.Int64("user", target.ID), zap.Error(err))
			}
			return
		}
	}
	a.store.RemoveMute(chat.ID, target.ID)
}

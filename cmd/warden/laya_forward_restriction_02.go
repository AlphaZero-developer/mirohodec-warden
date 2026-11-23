package main

import (
	tele "gopkg.in/telebot.v3"
)

func layaForwardRestrictionReplyTarget(msg *tele.Message) (targetUser, bool) {
	if msg == nil || msg.ReplyTo == nil || msg.ReplyTo.Sender == nil || isForumTopicRootReply(msg) {
		return targetUser{}, false
	}
	u := msg.ReplyTo.Sender
	return targetUser{
		ID:       u.ID,
		Username: u.Username,
		Name:     displayNameFromUser(u),
	}, u.ID != 0
}

func isForwardRestrictionPayload(msg *tele.Message) bool {
	return msg != nil && (isForwardedMessage(msg) || msg.AutomaticForward || msg.Via != nil)
}

func forwardRestrictionPayloadKind(msg *tele.Message) string {
	switch {
	case msg == nil:
		return ""
	case msg.Via != nil:
		return "via_bot"
	case msg.AutomaticForward:
		return "automatic_forward"
	case isForwardedMessage(msg):
		return "forward"
	default:
		return "message"
	}
}

func forwardedOriginalUserIDs(msg *tele.Message) []int64 {
	if msg == nil {
		return nil
	}
	var out []int64
	seen := make(map[int64]struct{})
	add := func(user *tele.User) {
		if user == nil || user.ID == 0 {
			return
		}
		if _, ok := seen[user.ID]; ok {
			return
		}
		seen[user.ID] = struct{}{}
		out = append(out, user.ID)
	}
	add(msg.OriginalSender)
	if msg.Origin != nil {
		add(msg.Origin.Sender)
	}
	return out
}

func userID(user *tele.User) int64 {
	if user == nil {
		return 0
	}
	return user.ID
}

package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v3"
)

func (a *App) targetFromArgs(chatID int64, c tele.Context, args []string) (targetUser, []string, error) {
	if msg := c.Message(); msg != nil && msg.ReplyTo != nil && msg.ReplyTo.Sender != nil {
		u := msg.ReplyTo.Sender
		return targetUser{
			ID:       u.ID,
			Username: u.Username,
			Name:     displayNameFromUser(u),
			IsBot:    u.IsBot,
		}, args, nil
	}

	if len(args) == 0 {
		return targetUser{}, nil, errors.New("missing target")
	}

	raw := strings.TrimSpace(args[0])
	raw = strings.TrimPrefix(raw, "tg://user?id=")
	if username := usernameFromTarget(raw); username != "" {
		target, ok := a.store.UserByUsername(chatID, username)
		if !ok {
			return targetUser{}, nil, errors.New("target username is unknown")
		}
		return target, args[1:], nil
	}

	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id == 0 {
		return targetUser{}, nil, errors.New("target must be a numeric user_id, @username, t.me link, or a reply")
	}

	target := targetUser{ID: id, Name: fmt.Sprintf("user %d", id)}
	if rec, ok := a.store.Member(chatID, id); ok {
		target.Username = rec.Username
		target.Name = rec.DisplayName
		target.IsBot = rec.IsBot
	}
	return target, args[1:], nil
}

func repliedToOtherUser(c tele.Context, senderID int64) bool {
	msg := c.Message()
	return msg != nil && msg.ReplyTo != nil && msg.ReplyTo.Sender != nil && msg.ReplyTo.Sender.ID != senderID
}

func usernameFromTarget(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "<>")
	raw = strings.TrimSuffix(raw, "/")
	lower := strings.ToLower(raw)

	switch {
	case strings.HasPrefix(raw, "@"):
		return normalizeUsername(raw)
	case strings.HasPrefix(lower, "https://t.me/"):
		return usernameFromURLPath(raw[len("https://t.me/"):])
	case strings.HasPrefix(lower, "http://t.me/"):
		return usernameFromURLPath(raw[len("http://t.me/"):])
	case strings.HasPrefix(lower, "https://telegram.me/"):
		return usernameFromURLPath(raw[len("https://telegram.me/"):])
	case strings.HasPrefix(lower, "http://telegram.me/"):
		return usernameFromURLPath(raw[len("http://telegram.me/"):])
	case strings.HasPrefix(lower, "tg://resolve?domain="):
		return normalizeUsername(raw[len("tg://resolve?domain="):])
	default:
		return ""
	}
}

func usernameFromURLPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || strings.HasPrefix(path, "+") || strings.HasPrefix(path, "joinchat/") || strings.HasPrefix(path, "c/") {
		return ""
	}
	if cut := strings.IndexAny(path, "/?#"); cut >= 0 {
		path = path[:cut]
	}
	return normalizeUsername(path)
}

func normalizeUsername(username string) string {
	username = strings.TrimSpace(username)
	username = strings.TrimPrefix(username, "@")
	if cut := strings.IndexAny(username, " /?#&"); cut >= 0 {
		username = username[:cut]
	}
	return strings.ToLower(username)
}

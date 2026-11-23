package main

import (
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v3"
)

func userLogLabel(user *tele.User) string {
	if user == nil {
		return "-"
	}
	return fmt.Sprintf("%s (%d)", displayNameFromUser(user), user.ID)
}

func chatLogLabel(chat *tele.Chat) string {
	if chat == nil {
		return "-"
	}
	title := strings.TrimSpace(chat.Title)
	if title == "" {
		title = strings.TrimSpace(chat.Username)
	}
	if title == "" {
		title = string(chat.Type)
	}
	return fmt.Sprintf("%s (%d)", textOrDash(title), chat.ID)
}

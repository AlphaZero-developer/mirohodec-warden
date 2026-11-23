package main

import (
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v3"
)

func (a *App) layaMentionContextLines(chat *tele.Chat, text string) []string {
	if a == nil || a.store == nil || chat == nil || strings.TrimSpace(text) == "" {
		return nil
	}
	usernames := layaMentionUsernames(text)
	if len(usernames) == 0 {
		return nil
	}
	lines := make([]string, 0, len(usernames))
	for _, username := range usernames {
		if len(lines) >= maxLayaMentionContext {
			break
		}
		target, ok := a.store.UserByUsername(chat.ID, username)
		if !ok || target.ID == 0 {
			lines = append(lines, fmt.Sprintf("@%s: пользователь пока не известен боту по username в этом чате", username))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s (@%s, id=%d, bot=%t%s)", target.Name, normalizeUsername(target.Username), target.ID, target.IsBot, a.layaProfileContextForUserID(target.ID)))
	}
	return lines
}

func layaMentionUsernames(text string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, field := range strings.Fields(text) {
		token := strings.Trim(field, " \t\r\n.,!?;:()[]{}<>\"'`«»")
		if username := usernameFromTarget(token); username != "" {
			if _, ok := seen[username]; !ok {
				seen[username] = struct{}{}
				out = append(out, username)
			}
		}
		for _, username := range usernamesFromAtMentions(field) {
			if _, ok := seen[username]; ok {
				continue
			}
			seen[username] = struct{}{}
			out = append(out, username)
		}
	}
	return out
}

func usernamesFromAtMentions(text string) []string {
	var out []string
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		if runes[i] != '@' {
			continue
		}
		j := i + 1
		for j < len(runes) && isTelegramUsernameRune(runes[j]) {
			j++
		}
		if j == i+1 {
			continue
		}
		username := normalizeUsername(string(runes[i+1 : j]))
		if username != "" {
			out = append(out, username)
		}
		i = j - 1
	}
	return out
}

func isTelegramUsernameRune(r rune) bool {
	return r == '_' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func layaChatContextLabel(chat *tele.Chat) string {
	if chat == nil {
		return "неизвестно"
	}
	title := strings.TrimSpace(chat.Title)
	if title == "" && chat.Type == tele.ChatPrivate {
		title = "личный чат"
	}
	if title == "" {
		title = "без названия"
	}
	return fmt.Sprintf("%q; id=%d; type=%s", title, chat.ID, chat.Type)
}

func (a *App) layaTelegramUserContextLabel(user *tele.User) string {
	if user == nil {
		return "неизвестно"
	}
	username := strings.TrimSpace(user.Username)
	if username != "" && !strings.HasPrefix(username, "@") {
		username = "@" + username
	}
	if username == "" {
		username = "нет"
	}
	firstName := strings.TrimSpace(user.FirstName)
	if firstName == "" {
		firstName = "нет"
	}
	lastName := strings.TrimSpace(user.LastName)
	if lastName == "" {
		lastName = "нет"
	}
	return fmt.Sprintf("user_id=%d; visible_name=%q; username=%q; first_name=%q; last_name=%q; is_bot=%t%s",
		user.ID,
		displayNameFromUser(user),
		username,
		firstName,
		lastName,
		user.IsBot,
		a.layaProfileContextForUserID(user.ID),
	)
}

func (a *App) layaProfileContextForUserID(userID int64) string {
	if a == nil || a.store == nil || userID == 0 {
		return ""
	}
	profile, ok := a.store.LayaUserProfile(userID)
	if !ok || (strings.TrimSpace(profile.Address) == "" && len(profile.Tags) == 0 && strings.TrimSpace(profile.Note) == "") {
		return ""
	}
	parts := []string{}
	if strings.TrimSpace(profile.Address) != "" {
		parts = append(parts, fmt.Sprintf("laya_profile_address=%q", profile.Address))
	}
	parts = append(parts, "laya_profile_gender="+layaGenderLabel(profile.Gender))
	if len(profile.Tags) > 0 {
		parts = append(parts, "laya_profile_tags="+strings.Join(profile.Tags, ","))
	}
	if strings.TrimSpace(profile.Note) != "" {
		parts = append(parts, fmt.Sprintf("laya_profile_note=%q", profile.Note))
	}
	return "; " + strings.Join(parts, "; ")
}

func (a *App) aiConversationHistory(chat *tele.Chat, sender *tele.User) []AIMessage {
	key := aiConversationKey(chat, sender)
	if key == "" {
		return nil
	}
	a.aiHistoryMu.Lock()
	defer a.aiHistoryMu.Unlock()
	history := trimAIConversationHistory(a.aiHistory[key])
	if len(history) == 0 {
		return nil
	}
	out := make([]AIMessage, len(history))
	copy(out, history)
	return out
}

func (a *App) rememberAIConversation(chat *tele.Chat, sender *tele.User, userText, reply string) {
	key := aiConversationKey(chat, sender)
	chatKey := aiChatNoRepeatKey(chat)
	userText = strings.TrimSpace(userText)
	reply = strings.TrimSpace(reply)
	if key == "" || userText == "" || reply == "" {
		return
	}
	a.aiHistoryMu.Lock()
	defer a.aiHistoryMu.Unlock()
	if a.aiHistory == nil {
		a.aiHistory = make(map[string][]AIMessage)
	}
	a.aiHistory[key] = trimAIConversationHistory(append(a.aiHistory[key],
		AIMessage{Role: "user", Content: userText},
		AIMessage{Role: "assistant", Content: reply},
	))
	a.appendAINoRepeatReplyLocked(key, reply)
	a.appendAINoRepeatReplyLocked(chatKey, reply)
}

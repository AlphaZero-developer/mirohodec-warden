package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

const appealReportCooldown = 24 * time.Hour

func idKey(id int64) string {
	return strconv.FormatInt(id, 10)
}

func memberFromUser(u *tele.User, now int64) MemberRecord {
	return MemberRecord{
		ID:          u.ID,
		Username:    u.Username,
		FirstName:   u.FirstName,
		LastName:    u.LastName,
		DisplayName: displayNameFromUser(u),
		IsBot:       u.IsBot,
		FirstSeen:   now,
		LastSeen:    now,
	}
}

func displayNameFromUser(u *tele.User) string {
	if u == nil {
		return "unknown"
	}
	name := strings.TrimSpace(strings.TrimSpace(u.FirstName + " " + u.LastName))
	if name != "" {
		return name
	}
	if u.Username != "" {
		return "@" + u.Username
	}
	return fmt.Sprintf("user %d", u.ID)
}

func messageText(msg *tele.Message) string {
	if msg == nil {
		return ""
	}
	if text := strings.TrimSpace(msg.Text); text != "" {
		return text
	}
	return strings.TrimSpace(msg.Caption)
}

func faqInput(c tele.Context) ([]string, string) {
	return faqInputFromArgs(c.Args())
}

func faqInputFromArgs(args []string) ([]string, string) {
	raw := strings.TrimSpace(strings.Join(args, " "))
	if raw == "" {
		return nil, ""
	}
	patternPart, answer, ok := strings.Cut(raw, "|")
	if !ok {
		return nil, ""
	}
	patterns := splitPatternList(patternPart)
	return patterns, strings.TrimSpace(answer)
}

func splitPatternList(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ';' || r == ','
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			out = append(out, field)
		}
	}
	return out
}

func ideaText(c tele.Context) string {
	return ideaTextFrom(c.Args(), c.Message())
}

func ideaTextFrom(args []string, msg *tele.Message) string {
	text := strings.TrimSpace(strings.Join(args, " "))
	if text != "" {
		return text
	}
	if msg != nil && msg.ReplyTo != nil {
		return messageText(msg.ReplyTo)
	}
	return ""
}

func formatIdeaReport(chat *tele.Chat, msg *tele.Message, sender *tele.User, text string) string {
	return formatOwnerReport("Новая идея для генерального директора", chat, msg, sender, text)
}

func formatOwnerReport(title string, chat *tele.Chat, msg *tele.Message, sender *tele.User, text string) string {
	if title == "" {
		title = "Новое обращение"
	}
	author := "неизвестно"
	userID := int64(0)
	username := "-"
	if sender != nil {
		author = displayNameFromUser(sender)
		userID = sender.ID
		if sender.Username != "" {
			username = "@" + sender.Username
		}
	}
	chatTitle := "личка"
	chatID := int64(0)
	if chat != nil {
		chatTitle = textOrDash(chat.Title)
		if chatTitle == "-" && chat.Type == tele.ChatPrivate {
			chatTitle = "личка"
		}
		chatID = chat.ID
	}
	threadID := 0
	messageID := 0
	if msg != nil {
		threadID = msg.ThreadID
		messageID = msg.ID
	}
	lines := []string{
		title,
		"",
		fmt.Sprintf("Автор: %s", author),
		fmt.Sprintf("User ID: %d", userID),
		fmt.Sprintf("Username: %s", username),
		fmt.Sprintf("Чат: %s (%d)", chatTitle, chatID),
	}
	if threadID != 0 {
		lines = append(lines, fmt.Sprintf("Тема: %d", threadID))
	}
	if messageID != 0 {
		lines = append(lines, fmt.Sprintf("Сообщение: %d", messageID))
	}
	lines = append(lines, "", "Текст:", strings.TrimSpace(text))
	return strings.Join(lines, "\n")
}

func ownerReportTitle(kind string) string {
	switch kind {
	case "job":
		return "💼 Заявка на работу"
	case "donate":
		return "💸 Донат для владельца"
	case "bug":
		return "Сообщение об ошибке"
	case "appeal":
		return "Обжалование решения"
	default:
		return "Новая идея для генерального директора"
	}
}

func (a *App) ownerReportUsage(kind string) string {
	switch kind {
	case "job":
		return a.t("usage_job")
	case "donate":
		return a.t("usage_donate")
	case "bug":
		return a.t("usage_bug")
	case "appeal":
		return a.t("usage_appeal")
	default:
		return a.t("usage_idea")
	}
}

func (a *App) ownerReportDone(kind string) string {
	switch kind {
	case "job":
		return a.t("job_sent")
	case "donate":
		return a.t("donate_sent")
	case "bug":
		return a.t("bug_sent")
	case "appeal":
		return a.t("appeal_sent")
	default:
		return a.t("idea_sent")
	}
}

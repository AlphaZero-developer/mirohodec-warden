package main

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) handlePassport(c tele.Context) error {
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	target, err := a.profileTarget(c)
	if err != nil {
		return c.Send(a.t("usage_profile"))
	}
	return c.Send(a.renderPassport(chat, target, time.Now().UTC()), a.layaInlineMarkup(chat, c.Sender()))
}

func (a *App) sendLayaPassport(c tele.Context) error {
	chat, sender, err := a.layaCommandChatAndSender(c)
	if err != nil {
		return c.Send(err.Error())
	}
	target := targetUser{ID: sender.ID, Username: sender.Username, Name: displayNameFromUser(sender)}
	return c.Send(a.renderPassport(chat, target, time.Now().UTC()), a.layaInlineMarkup(chat, sender))
}

func (a *App) renderPassport(chat *tele.Chat, target targetUser, now time.Time) string {
	chatID := int64(0)
	if chat != nil {
		chatID = chat.ID
	}
	member, hasMember := a.store.Member(chatID, target.ID)
	name := strings.TrimSpace(target.Name)
	if hasMember && strings.TrimSpace(member.DisplayName) != "" {
		name = member.DisplayName
	}
	if name == "" {
		name = fmt.Sprintf("user %d", target.ID)
	}

	karma := a.store.Karma(chatID, target.ID)
	activeWarnings := a.store.ActiveWarningCount(chatID, target.ID, now)
	activity := int64(0)
	messages := int64(0)
	today := int64(0)
	tone := "neutral"
	title := "-"
	achievements := []string{}
	if hasMember {
		activity = activityScore(member, karma, activeWarnings, now)
		messages = member.Messages
		today = messagesTodayAt(member, now)
		tone = member.LayaTone
		if strings.TrimSpace(member.ShopTitle) != "" && member.ShopTitleUntilUnix > now.Unix() {
			title = member.ShopTitle
		}
		achievements = achievementList(member)
	}

	status := strings.Join(a.profileStatuses(chat, target.ID, false, false), ", ")
	if jail, ok := a.store.ActiveJail(chatID, target.ID, now); ok {
		status = strings.Join(a.profileStatuses(chat, target.ID, true, false), ", ")
		title = titleOrDash(title)
		lines := a.passportBaseLines(name, target.ID, status, title, karma, activity, messages, today, activeWarnings, tone, achievements, now)
		lines = append(lines, "🏥 Палата: "+rehabWard(jail).Title)
		lines = append(lines, "⏳ Психцентр: "+a.profileRehabRemaining(jail, now))
		lines = append(lines, a.passportQuoteLine())
		return strings.Join(lines, "\n")
	}
	if _, banned := a.store.Ban(chatID, target.ID); banned {
		status = strings.Join(a.profileStatuses(chat, target.ID, false, true), ", ")
	}

	lines := a.passportBaseLines(name, target.ID, status, title, karma, activity, messages, today, activeWarnings, tone, achievements, now)
	lines = append(lines, "🏥 Палата: нет")
	lines = append(lines, a.passportQuoteLine())
	return strings.Join(lines, "\n")
}

func (a *App) passportBaseLines(name string, userID int64, status, title string, karma, activity, messages, today int64, warnings int, tone string, achievements []string, now time.Time) []string {
	if strings.TrimSpace(status) == "" {
		status = "обычный участник"
	}
	lines := []string{
		"🪪 Паспорт участника",
		"",
		fmt.Sprintf("Имя: %s", name),
		fmt.Sprintf("ID: %d", userID),
		fmt.Sprintf("Статус: %s", status),
		fmt.Sprintf("Титул: %s", titleOrDash(title)),
		fmt.Sprintf("Тон Лайи: %s", layaToneTitle(tone)),
		"",
		fmt.Sprintf("⭐ Карма: %d", karma),
		fmt.Sprintf("🔥 Активность: %d", activity),
		fmt.Sprintf("💬 Сообщений: %d всего · %d сегодня", messages, today),
		fmt.Sprintf("⚠️ Предупреждений: %d", warnings),
	}
	if len(achievements) > 0 {
		lines = append(lines, fmt.Sprintf("🏅 Ачивки: %d · %s", len(achievements), strings.Join(achievements, " | ")))
	} else {
		lines = append(lines, "🏅 Ачивки: скрытые, пока не открыты")
	}
	if event := weeklyEventLine(now); event != "" {
		lines = append(lines, event)
	}
	return lines
}

func (a *App) passportQuoteLine() string {
	rec, ok := a.latestUsedQuote()
	if !ok {
		return "📜 Последняя цитата: нет"
	}
	return "📜 Последняя цитата: " + shortText(strings.TrimSpace(rec.Text), 120)
}

func (a *App) latestUsedQuote() (QuoteRecord, bool) {
	if a == nil || a.quotes == nil {
		return QuoteRecord{}, false
	}
	records := a.quotes.List("")
	if len(records) == 0 {
		return QuoteRecord{}, false
	}
	best := records[0]
	for _, rec := range records[1:] {
		if rec.LastUsedUnix > best.LastUsedUnix {
			best = rec
		}
	}
	return best, true
}

func titleOrDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

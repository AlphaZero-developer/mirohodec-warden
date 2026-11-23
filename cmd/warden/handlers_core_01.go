package main

import (
	"fmt"
	"html"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) warnJailedUser(chat *tele.Chat, user *tele.User, jail ModerationRecord) {
	cfg := a.config()
	rehabTopicID := a.rehabTopicID(chat)
	if rehabTopicID == 0 {
		return
	}

	key := fmt.Sprintf("%d:%d", chat.ID, user.ID)
	now := time.Now().UTC()

	a.warnMu.Lock()
	last := a.lastWarning[key]
	if now.Sub(last) < mustDuration(cfg.Moderation.WarningCooldown, 2*time.Minute) {
		a.warnMu.Unlock()
		return
	}
	a.lastWarning[key] = now
	a.warnMu.Unlock()

	text := a.t("jailed_warning", displayNameFromUser(user))
	if jail.Reason != "" {
		text += "\n" + a.t("reason_line", jail.Reason)
	}
	if jail.UntilUnix > 0 {
		text += "\n" + releaseLine(jail.UntilUnix, time.Now().UTC())
	}
	text += "\n" + rehabWardLine(jail)
	text += "\nПобег: /escape"
	text += "\nПодкуп санитара: /bribe"

	if _, err := a.bot.Send(chat, text, &tele.SendOptions{ThreadID: rehabTopicID, ReplyMarkup: a.layaRestrictionMarkup(true)}); err != nil {
		a.log.Warn("warn jailed user failed", zap.Error(err))
	}
}

type logField struct {
	Name  string
	Value string
}

func (a *App) sendOwnerLog(title string, fields ...logField) {
	if a == nil || a.store == nil {
		return
	}
	chatID := a.store.LogChatID()
	if chatID == 0 || strings.TrimSpace(title) == "" {
		return
	}
	text := ownerLogHTML(title, fields...)
	go func() {
		if _, err := a.bot.Send(&tele.Chat{ID: chatID, Type: tele.ChatPrivate}, text, &tele.SendOptions{ParseMode: tele.ModeHTML}); err != nil {
			a.log.Warn("send owner log failed", zap.Int64("chat", chatID), zap.Error(err))
		}
	}()
}

func (a *App) sendOwnerDirectLog(userID int64, title string, fields ...logField) {
	if a == nil || userID == 0 || strings.TrimSpace(title) == "" {
		return
	}
	text := ownerLogHTML(title, fields...)
	go func() {
		if _, err := a.bot.Send(&tele.Chat{ID: userID, Type: tele.ChatPrivate}, text, &tele.SendOptions{ParseMode: tele.ModeHTML}); err != nil {
			a.log.Warn("send owner direct log failed", zap.Int64("user", userID), zap.Error(err))
		}
	}()
}

func ownerLogHTML(title string, fields ...logField) string {
	var b strings.Builder
	b.WriteString("<b>")
	b.WriteString(html.EscapeString(ownerLogTitle(title)))
	b.WriteString("</b>\n")
	b.WriteString("<b>Время:</b> ")
	b.WriteString(html.EscapeString(formatUnix(time.Now().UTC().Unix())))
	for _, field := range fields {
		if strings.TrimSpace(field.Value) == "" {
			continue
		}
		b.WriteString("\n<b>")
		b.WriteString(html.EscapeString(ownerLogFieldName(field.Name)))
		b.WriteString(":</b> ")
		b.WriteString(html.EscapeString(ownerLogFieldValue(field.Value)))
	}
	return b.String()
}

func ownerLogTitle(title string) string {
	switch strings.ToLower(strings.TrimSpace(title)) {
	case "owner auth":
		return "🔐 Авторизация владельца"
	case "warn":
		return "⚠️ Предупреждение"
	case "warn escalation: mute":
		return "🔇 Предупреждения: мут"
	case "warn escalation: rehab":
		return "🏥 Предупреждения: психцентр"
	case "unwarn":
		return "✅ Предупреждение снято"
	case "clear warns":
		return "🧹 Предупреждения очищены"
	case "rehab", "psych center":
		return "🏥 Психцентр: отправка"
	case "rehab released", "psych center released":
		return "✅ Психцентр: выход"
	case "mute":
		return "🔇 Мут"
	case "unmute":
		return "🔊 Мут снят"
	case "forever ban":
		return "⛔ Вечный бан"
	case "forever ban removed":
		return "✅ Вечный бан снят"
	case "spam pardon":
		return "🧹 Спам-помилование"
	case "spam mute":
		return "🚨 Спам-мут"
	case "rehab escape":
		return "🏃 Побег из психцентра"
	case "rehab bribe":
		return "💸 Подкуп санитара"
	case "self psych center":
		return "🏥 Самоотправка в психцентр"
	case "лайя фас":
		return "🦷 Лайя фас"
	case "join log":
		return "🚪 Вход в чат"
	case "bot added":
		return "🤖 Бота добавили в чат"
	case "owner sleep":
		return "🌙 Сон создателя включен"
	case "owner wake":
		return "☀️ Создатель проснулся"
	case "creator called while sleeping":
		return "🔔 Создателя звали во сне"
	case "личное сообщение лайе":
		return "💬 Личное сообщение Лайе"
	case "сообщение в чате лайе":
		return "💬 Сообщение в чате Лайе"
	case "ai fallback":
		return "⚠️ AI fallback Лайи"
	case "laya manual reply":
		return "✍️ Ручной ответ Лайи"
	case "фишка лайи: roast":
		return "🔥 Фишка Лайи: roast"
	case "фишка лайи: compliment":
		return "✨ Фишка Лайи: compliment"
	default:
		return strings.TrimSpace(title)
	}
}

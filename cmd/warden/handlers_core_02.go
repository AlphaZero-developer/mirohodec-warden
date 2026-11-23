package main

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func ownerLogFieldName(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "user", "пользователь":
		return "👤 Пользователь"
	case "user id":
		return "🪪 ID пользователя"
	case "username":
		return "🏷️ Username"
	case "chat", "чат":
		return "💬 Чат"
	case "topic", "тема":
		return "🧵 Тема"
	case "thread":
		return "🧵 Тема"
	case "release":
		return "✅ Освобождение"
	case "reason", "причина":
		return "📝 Причина"
	case "by", "выдал":
		return "🛠️ Кто сделал"
	case "owner":
		return "👑 Владелец"
	case "log chat":
		return "📬 Личный лог-чат"
	case "active warnings":
		return "⚠️ Активных предупреждений"
	case "until", "мут":
		return "⏳ До"
	case "duration", "длительность":
		return "⏱️ Длительность"
	case "chance":
		return "🎲 Шанс"
	case "next chance":
		return "🎲 Следующий шанс"
	case "cost":
		return "💸 Цена"
	case "karma left":
		return "⭐ Карма после"
	case "ward":
		return "🏥 Палата"
	case "warn":
		return "⚠️ Предупреждение"
	case "count":
		return "🔢 Количество"
	case "action":
		return "⚙️ Действие"
	case "invited by":
		return "🙋 Кто пригласил"
	case "invite link creator":
		return "🔗 Создатель ссылки"
	case "invite link":
		return "🔗 Ссылка-приглашение"
	case "via":
		return "↪️ Через"
	case "source":
		return "📍 Источник"
	case "added by":
		return "➕ Кто добавил"
	case "matched":
		return "🎯 Совпало"
	case "message", "сообщение":
		return "✉️ Сообщение"
	case "цель":
		return "🎯 Цель"
	case "статус":
		return "📌 Статус"
	case "время", "время ai":
		return "⏱️ Время"
	case "ответ":
		return "💬 Ответ"
	case "ai":
		return "🤖 AI"
	case "openrouter status":
		return "🌐 OpenRouter status"
	case "openrouter error":
		return "🌐 OpenRouter error"
	case "ошибка ai":
		return "⚠️ Ошибка AI"
	case "отправка ответа":
		return "📤 Отправка ответа"
	default:
		return strings.TrimSpace(name)
	}
}

func ownerLogFieldValue(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "web panel":
		return "веб-панель"
	case "chat folder invite":
		return "приглашение через папку чатов"
	case "spam state cleared":
		return "спам-состояние очищено"
	case "spam state cleared and bot mute removed":
		return "спам-состояние очищено, мут бота снят"
	default:
		return value
	}
}

func (a *App) sendOwnerPlain(text string) {
	chatID := a.store.LogChatID()
	if chatID == 0 || strings.TrimSpace(text) == "" {
		return
	}
	go func() {
		if _, err := a.bot.Send(&tele.Chat{ID: chatID, Type: tele.ChatPrivate}, text); err != nil {
			a.log.Warn("send owner message failed", zap.Int64("chat", chatID), zap.Error(err))
		}
	}()
}

func (a *App) sendWebPanelLink(reason string) {
	_ = reason
	cfg := a.config().Web
	if !cfg.Enabled || !cfg.SendPanelLinkToOwner {
		return
	}
	url := strings.TrimSpace(cfg.PublicURL)
	if url == "" {
		return
	}
	a.sendOwnerPlain(fmt.Sprintf("\u041f\u0430\u043d\u0435\u043b\u044c \u0443\u043f\u0440\u0430\u0432\u043b\u0435\u043d\u0438\u044f \u0437\u0430\u043f\u0443\u0449\u0435\u043d\u0430:\n%s", url))
}

func (a *App) sendRehabNotice(chat *tele.Chat, rec ModerationRecord) {
	rehabTopicID := a.rehabTopicID(chat)
	if chat == nil || rehabTopicID == 0 {
		return
	}
	text := a.t("jail_notice")
	if rec.Reason != "" {
		text += "\n" + a.t("reason_line", rec.Reason)
	}
	text += "\n" + rehabWardLine(rec)
	text += "\n" + releaseLine(rec.UntilUnix, time.Now().UTC())
	text += "\nПобег: /escape"
	text += "\nПодкуп санитара: /bribe"
	if _, err := a.bot.Send(chat, text, &tele.SendOptions{ThreadID: rehabTopicID, ReplyMarkup: a.layaRestrictionMarkup(true)}); err != nil {
		a.log.Warn("send rehab notice failed", zap.Error(err))
	}
}

func textOrDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

func (a *App) handleAdmin(c tele.Context) error {
	if chat := c.Chat(); chat != nil && chat.Type != tele.ChatPrivate {
		return nil
	}
	if msg := c.Message(); msg != nil && commandName(msg.Text) == "start" {
		if sender := c.Sender(); sender != nil && !sender.IsBot {
			user, err := a.store.RegisterMafiaUser(sender, time.Now().UTC())
			if err != nil && a.log != nil {
				a.log.Warn("register mafia user on start failed", zap.Error(err))
			}
			if !a.isAdmin(c) {
				if err == nil && !mafiaProfileComplete(user) {
					return a.startMafiaRegistrationIntro(c)
				}
				return c.Send("👋 Добро пожаловать.\n\nВы зарегистрированы в системе Лайи. Меню открывается командой /menu.", tele.RemoveKeyboard)
			}
		}
	}
	if !a.isAdmin(c) {
		return c.Send(a.t("no_admin"))
	}
	text := a.t("admin_help")
	if a.config().Telegram.ShowDebugCommands {
		text += "\n\n" + a.t("debug_help")
	}
	return c.Send(text)
}

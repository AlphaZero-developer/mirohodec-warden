package main

import (
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

const (
	layaSelfMuteDefault  = 5 * time.Minute
	layaSelfJailDefault  = 15 * time.Minute
	layaSelfBanDefault   = 10 * time.Minute
	layaSelfMuteMax      = 24 * time.Hour
	layaSelfJailMax      = 24 * time.Hour
	layaSelfBanMax       = 24 * time.Hour
	layaSelfJailChiefMax = 30 * 24 * time.Hour
	layaSelfJailManual   = -1 * time.Second
)

func layaButton(text, data string) tele.Btn {
	return tele.Btn{Text: text, Data: data}
}

func (a *App) handleLayaMenu(c tele.Context) error {
	chat := c.Chat()
	sender := c.Sender()
	if c.Callback() != nil {
		if err := c.Edit(a.renderLayaMenuText(c), a.layaMenuMarkup(chat, sender)); err == nil || isTelegramMessageNotModified(err) {
			return nil
		}
		return a.layaSend(c, a.renderLayaMenuText(c), a.layaMenuMarkup(chat, sender))
	}
	return a.layaSend(c, a.renderLayaMenuText(c), a.layaMenuMarkup(chat, sender))
}

func (a *App) renderLayaMenuText(c tele.Context) string {
	lines := []string{
		"Меню Лайи. Основное вынесла в кнопки:",
		"",
		"Быстро: профиль, карма, активность, цитата, фото.",
		"Разделы: статистика, магазин, self-действия, мафия.",
		"Безопасное: self-действия всегда идут через подтверждение.",
	}
	if chat := c.Chat(); chat != nil && chat.Type == tele.ChatPrivate {
		lines = append(lines, "", "В личке можно просто писать Лайе текстом. В группе зови её по имени или отвечай на её сообщение.")
	}
	return strings.Join(lines, "\n")
}

func (a *App) layaInlineMarkup(chat *tele.Chat, sender *tele.User) *tele.ReplyMarkup {
	return nil
}

func (a *App) layaMenuMarkup(chat *tele.Chat, sender *tele.User) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := []tele.Row{
		markup.Row(layaButton("💬 Как спросить", "laya:ask"), layaButton("⚙️ Тон", "laya:settings")),
		markup.Row(layaButton("🪪 Паспорт", "laya:passport"), layaButton("👤 Профиль", "laya:profile"), layaButton("⭐ Карма", "laya:karma")),
		markup.Row(layaButton("📜 Цитата", "laya:quote"), layaButton("🖼 Фото", "laya:photo"), layaButton("❓ FAQ", "laya:faq")),
		markup.Row(layaButton("📊 Статистика", "laya:statsmenu"), layaButton("🏪 Магазин", "laya:shopmenu"), layaButton("🗓 Событие", "laya:event")),
		markup.Row(layaButton("🧹 Очистить историю", "laya:clearhistory"), layaButton("🎭 Мафия", "laya:mafia")),
	}

	now := time.Now().UTC()
	if chat != nil && sender != nil && a != nil && a.store != nil {
		if _, ok := a.store.ActiveJail(chat.ID, sender.ID, now); ok {
			rows = append(rows, markup.Row(layaButton("🏥 Психцентр", "laya:rehabmenu"), layaButton("🏃 Побег", "laya:escape"), layaButton("💸 Подкуп", "laya:bribe")))
			rows = append(rows, markup.Row(layaButton("🧾 Быстрая апелляция", "laya:appeal_quick")))
		} else if _, ok := a.store.ActiveMute(chat.ID, sender.ID, now); ok {
			rows = append(rows, markup.Row(layaButton("🧾 Быстрая апелляция", "laya:appeal_quick"), layaButton("🐞 Жалоба", "laya:bug_quick")))
			rows = append(rows, markup.Row(layaButton("⚖️ Self-действия", "laya:selfmenu"), layaButton("🤐 Самомут 5м", "laya:selfmute:5m"), layaButton("🏥 Психцентр 15м", "laya:selfjail:15m")))
		} else {
			rows = append(rows, markup.Row(layaButton("⚖️ Self-действия", "laya:selfmenu"), layaButton("🤐 Самомут 5м", "laya:selfmute:5m"), layaButton("🏥 Психцентр 15м", "laya:selfjail:15m")))
		}
	} else {
		rows = append(rows, markup.Row(layaButton("⚖️ Self-действия", "laya:selfmenu"), layaButton("🤐 Самомут 5м", "laya:selfmute:5m"), layaButton("🏥 Психцентр 15м", "laya:selfjail:15m")))
	}
	markup.Inline(rows...)
	return markup
}

func (a *App) layaRestrictionMarkup(rehab bool) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := []tele.Row{}
	if rehab {
		rows = append(rows, markup.Row(layaButton("🧾 Апелляция", "laya:appeal_quick")))
	} else {
		rows = append(rows, markup.Row(layaButton("🧾 Апелляция", "laya:appeal_quick"), layaButton("🐞 Жалоба", "laya:bug_quick")))
	}
	if rehab {
		rows = append(rows,
			markup.Row(layaButton("🏥 Палата", "laya:ward"), layaButton("🏃 Побег", "laya:escape"), layaButton("💸 Подкуп", "laya:bribe")),
		)
	}
	rows = append(rows,
		markup.Row(layaButton("🪪 Паспорт", "laya:passport")),
	)
	markup.Inline(rows...)
	return markup
}

func (a *App) handleLayaReplyKeyboard(c tele.Context) bool {
	if c == nil || c.Message() == nil || c.Sender() == nil {
		return false
	}
	command := normalizeMafiaReplyButtonText(messageText(c.Message()))
	switch command {
	case "меню лайи":
		_ = a.handleLayaMenu(c)
	case "спросить лайю":
		_ = a.layaSend(c, a.layaAskHint(c), a.layaInlineMarkup(c.Chat(), c.Sender()))
	case "цитата":
		_ = a.sendLayaQuote(c)
	case "фото":
		_ = a.sendLayaPhoto(c)
	case "профиль":
		_ = a.sendLayaProfile(c)
	case "карма":
		_ = a.sendLayaKarma(c)
	case "магазин":
		_ = a.sendLayaShop(c)
	case "очистить историю":
		_ = a.handleClearHistory(c)
	case "самомут":
		_ = a.confirmLayaSelfAction(c, "selfmute", layaSelfMuteDefault)
	case "в психцентр", "в дурку":
		_ = a.confirmLayaSelfAction(c, "selfjail", layaSelfJailDefault)
	case "самобан":
		_ = a.confirmLayaSelfAction(c, "selfban", layaSelfBanDefault)
	case "мафия":
		_ = a.handleMafia(c)
	default:
		return false
	}
	return true
}

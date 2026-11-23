package main

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) handleLayaCallback(c tele.Context, data string) error {
	switch {
	case data == "laya:menu":
		return a.handleLayaMenu(c)
	case data == "laya:ask":
		return a.layaSend(c, a.layaAskHint(c), a.layaInlineMarkup(c.Chat(), c.Sender()))
	case data == "laya:profile":
		return a.sendLayaProfile(c)
	case data == "laya:karma":
		return a.sendLayaKarma(c)
	case data == "laya:activity":
		return a.sendLayaActivity(c)
	case data == "laya:passport":
		return a.sendLayaPassport(c)
	case data == "laya:settings":
		return a.layaSend(c, a.renderLayaSettingsText(c), a.layaSettingsMarkup(c))
	case strings.HasPrefix(data, "laya:tone:"):
		return a.setLayaToneCallback(c, strings.TrimPrefix(data, "laya:tone:"))
	case data == "laya:topkarma":
		return a.handleTopKarmaFromButton(c)
	case data == "laya:topactive":
		return a.handleTopActiveFromButton(c)
	case data == "laya:statsmenu":
		return a.layaSend(c, a.renderLayaStatsMenuText(c), a.layaStatsMarkup())
	case data == "laya:event":
		return a.handleWeeklyEvent(c)
	case data == "laya:quote":
		return a.sendLayaQuote(c)
	case data == "laya:photo":
		return a.sendLayaPhoto(c)
	case data == "laya:faq":
		return a.layaSend(c, "FAQ работает так: напиши /faq вопрос или ответь /faq на сообщение, которое надо разобрать.", a.layaInlineMarkup(c.Chat(), c.Sender()))
	case data == "laya:shop", data == "laya:shopmenu":
		return a.sendLayaShop(c)
	case strings.HasPrefix(data, "laya:buy:title:"):
		return a.buyLayaShopTitleCallback(c, strings.TrimPrefix(data, "laya:buy:title:"))
	case data == "laya:buy:charm":
		return a.buyLayaShopCallback(c, "charm")
	case data == "laya:buy:quote":
		return a.buyLayaShopCallback(c, "quote")
	case data == "laya:clearhistory":
		return a.handleClearHistory(c)
	case data == "laya:appeal_quick":
		return a.sendQuickOwnerReport(c, "appeal")
	case data == "laya:bug_quick":
		return a.sendQuickOwnerReport(c, "bug")
	case data == "laya:mafia":
		return a.handleMafia(c)
	case data == "laya:selfmenu":
		return a.layaSend(c, a.renderLayaSelfMenuText(c.Chat(), c.Sender()), a.layaSelfMarkup(c.Chat(), c.Sender()))
	case data == "laya:rehabmenu":
		return a.layaSend(c, a.renderLayaRehabMenuText(c), a.layaRehabMarkup(c.Chat(), c.Sender()))
	case data == "laya:ward":
		return a.handleWard(c)
	case data == "laya:escape":
		return a.handleEscape(c)
	case data == "laya:bribe":
		return a.handleBribe(c)
	case data == "laya:selfrelease":
		return a.handleSelfRelease(c)
	case strings.HasPrefix(data, "laya:self"):
		action, d, ok := parseLayaSelfCallback(data, "laya:")
		if !ok {
			return a.layaSend(c, "Кнопка устарела. Открой меню Лайи заново.", a.layaInlineMarkup(c.Chat(), c.Sender()))
		}
		return a.confirmLayaSelfAction(c, action, d)
	case strings.HasPrefix(data, "laya:confirm:"):
		action, d, ok := parseLayaSelfCallback(data, "laya:confirm:")
		if !ok {
			return a.layaSend(c, "Кнопка подтверждения устарела. Открой меню Лайи заново.", a.layaInlineMarkup(c.Chat(), c.Sender()))
		}
		return a.applyLayaSelfAction(c, action, d)
	default:
		return a.layaSend(c, "Эта кнопка Лайи уже неактуальна. Открой меню заново.", a.layaInlineMarkup(c.Chat(), c.Sender()))
	}
}

func isTelegramMessageNotModified(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "message is not modified")
}

func parseLayaSelfCallback(data, prefix string) (string, time.Duration, bool) {
	payload := strings.TrimPrefix(strings.TrimSpace(data), prefix)
	parts := strings.Split(payload, ":")
	if len(parts) != 2 {
		return "", 0, false
	}
	if parts[0] == "selfjail" && isLayaManualDurationToken(parts[1]) {
		return parts[0], layaSelfJailManual, true
	}
	d, permanent, ok, err := parseDurationToken(parts[1])
	if err != nil || !ok || permanent || d < 30*time.Second {
		return "", 0, false
	}
	switch parts[0] {
	case "selfmute", "selfjail", "selfban":
		return parts[0], d, true
	default:
		return "", 0, false
	}
}

func (a *App) layaAskHint(c tele.Context) string {
	if chat := c.Chat(); chat != nil && chat.Type == tele.ChatPrivate {
		return "В личке просто напиши Лайе сообщение. В группе напиши «Лайя, ...» или ответь на её сообщение."
	}
	return "Чтобы Лайя ответила в группе, напиши «Лайя, ...» или ответь на её сообщение."
}

func (a *App) renderLayaStatsMenuText(c tele.Context) string {
	chat, sender, err := a.layaCommandChatAndSender(c)
	if err != nil {
		return err.Error()
	}
	karma := a.store.Karma(chat.ID, sender.ID)
	member, ok := a.store.Activity(chat.ID, sender.ID)
	if !ok {
		return fmt.Sprintf("Статистика %s\n\n⭐ Карма: %d\n💬 Активность пока не набралась.", displayNameFromUser(sender), karma)
	}
	now := time.Now().UTC()
	score := activityScore(member, karma, a.store.ActiveWarningCount(chat.ID, sender.ID, now), now)
	return strings.Join([]string{
		fmt.Sprintf("Статистика %s", displayNameFromUser(sender)),
		"",
		fmt.Sprintf("⭐ Карма: %d · место %s", karma, socialRankLabel(a.karmaRank(chat.ID, sender.ID))),
		fmt.Sprintf("🔥 Активность: %d · место %s", score, socialRankLabel(a.activityRank(chat.ID, sender.ID, now))),
		fmt.Sprintf("💬 Сообщений: %d всего · %d сегодня", member.Messages, messagesTodayAt(member, now)),
		"",
		"Ниже быстрые кнопки для подробностей.",
	}, "\n")
}

func (a *App) layaStatsMarkup() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(layaButton("🪪 Паспорт", "laya:passport"), layaButton("👤 Профиль", "laya:profile"), layaButton("⭐ Карма", "laya:karma")),
		markup.Row(layaButton("🔥 Активность", "laya:activity")),
		markup.Row(layaButton("🏆 Топ кармы", "laya:topkarma"), layaButton("📊 Топ актива", "laya:topactive")),
		markup.Row(layaButton("🗓 Событие недели", "laya:event")),
		markup.Row(layaButton("⬅️ Меню", "laya:menu")),
	)
	return markup
}

func (a *App) renderLayaSettingsText(c tele.Context) string {
	chat := c.Chat()
	sender := c.Sender()
	tone := "neutral"
	if a != nil && a.store != nil && chat != nil && sender != nil {
		if member, ok := a.store.Member(chat.ID, sender.ID); ok {
			tone = member.LayaTone
		}
	}
	return strings.Join([]string{
		"Личные настройки Лайи",
		"",
		"Тон сейчас: " + layaToneTitle(tone) + ".",
		"Мягче — теплее и аккуратнее.",
		"Нейтрально — обычная Лайя.",
		"Жёстче — суше и острее, но без нарушения правил.",
	}, "\n")
}

func (a *App) layaSettingsMarkup(c tele.Context) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	current := "neutral"
	if a != nil && a.store != nil && c != nil && c.Chat() != nil && c.Sender() != nil {
		if member, ok := a.store.Member(c.Chat().ID, c.Sender().ID); ok {
			current = normalizeLayaTone(member.LayaTone)
		}
	}
	markup.Inline(
		markup.Row(
			layaButton(layaToneButton("мягче", current == "soft"), "laya:tone:soft"),
			layaButton(layaToneButton("нейтрально", current == "neutral"), "laya:tone:neutral"),
			layaButton(layaToneButton("жёстче", current == "hard"), "laya:tone:hard"),
		),
		markup.Row(layaButton("⬅️ Меню", "laya:menu")),
	)
	return markup
}

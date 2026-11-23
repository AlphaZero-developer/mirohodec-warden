package main

import (
	"fmt"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) buyLayaShopCallback(c tele.Context, item string) error {
	chat, sender, err := a.layaCommandChatAndSender(c)
	if err != nil {
		return a.layaSend(c, err.Error())
	}
	now := time.Now().UTC()
	member, _ := a.store.Member(chat.ID, sender.ID)
	rewards := memberRewardsFromRecord(member)
	until := now.Add(karmaShopDuration).Unix()
	switch item {
	case "charm":
		if err := a.buyKarmaShopItem(chat.ID, sender, karmaShopCharmCost, "shop_charm", now); err != nil {
			return a.layaSend(c, err.Error(), a.layaShopMarkup())
		}
		rewards.LayaCharmUntilUnix = until
	case "quote":
		if err := a.buyKarmaShopItem(chat.ID, sender, karmaShopQuoteCost, "shop_quote", now); err != nil {
			return a.layaSend(c, err.Error(), a.layaShopMarkup())
		}
		rewards.QuoteBoostUntilUnix = until
	default:
		return a.layaSend(c, "Такого товара нет.", a.layaShopMarkup())
	}
	if err := a.store.SetMemberRewards(chat.ID, sender.ID, rewards); err != nil {
		return a.layaSend(c, fmt.Sprintf("Покупка списалась странно, но эффект не сохранился: %v", err), a.layaShopMarkup())
	}
	a.unlockAchievement(chat.ID, sender.ID, "shop_first", now)
	return a.layaSend(c, "Покупка оформлена. Эффект действует 24 часа.", a.layaShopMarkup())
}

func (a *App) layaCommandChatAndSender(c tele.Context) (*tele.Chat, *tele.User, error) {
	chat, err := a.commandChat(c)
	if err != nil {
		return nil, nil, fmt.Errorf("%s", a.localizeErr(err))
	}
	sender := c.Sender()
	if sender == nil {
		return nil, nil, fmt.Errorf("%s", a.t("no_sender"))
	}
	return chat, sender, nil
}

func (a *App) confirmLayaSelfAction(c tele.Context, action string, d time.Duration) error {
	chat := c.Chat()
	sender := c.Sender()
	title := layaSelfActionTitle(action)
	if title == "" {
		return a.layaSend(c, "Не знаю такую self-кнопку.", a.layaInlineMarkup(chat, sender))
	}
	if action == "selfjail" && d == layaSelfJailManual && !a.userHasRehabChiefTitle(chatIDFromChat(chat), userIDFromUser(sender), time.Now().UTC()) {
		return a.layaSend(c, "Произвольный срок доступен только с титулом «главный в психбольнице».", a.layaSelfMarkup(chat, sender))
	}
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(layaButton("✅ Подтвердить "+title, fmt.Sprintf("laya:confirm:%s:%s", action, layaDurationToken(d)))),
		markup.Row(layaButton("⬅️ Назад в меню", "laya:menu")),
	)
	durationText := formatDurationShort(d)
	if d == layaSelfJailManual {
		durationText = "произвольный срок"
	}
	return a.layaSend(c, fmt.Sprintf("%s на %s? Нажмёшь подтверждение — Лайя сделает именно с тобой.", title, durationText), markup)
}

func (a *App) applyLayaSelfAction(c tele.Context, action string, d time.Duration) error {
	switch action {
	case "selfmute":
		return a.applySelfMute(c, d)
	case "selfjail":
		return a.applySelfJail(c, d)
	case "selfban":
		return a.applySelfBan(c, d)
	default:
		return a.layaSend(c, "Не знаю такую self-кнопку.", a.layaInlineMarkup(c.Chat(), c.Sender()))
	}
}

func (a *App) handleSelfMute(c tele.Context) error {
	d, err := parseSelfModerationDuration(c.Args(), layaSelfMuteDefault, layaSelfMuteMax)
	if err != nil {
		return a.layaSend(c, err.Error())
	}
	return a.applySelfMute(c, d)
}

func (a *App) handleSelfJail(c tele.Context) error {
	chat, sender, err := a.layaCommandChatAndSender(c)
	if err != nil {
		return a.layaSend(c, err.Error())
	}
	max := layaSelfJailMax
	chief := a.userHasRehabChiefTitle(chat.ID, sender.ID, time.Now().UTC())
	if chief {
		max = layaSelfJailChiefMax
	}
	d, err := parseSelfJailDuration(c.Args(), layaSelfJailDefault, max, chief)
	if err != nil {
		return a.layaSend(c, err.Error())
	}
	return a.applySelfJailFor(c, chat, sender, d)
}

func (a *App) handleSelfBan(c tele.Context) error {
	d, err := parseSelfModerationDuration(c.Args(), layaSelfBanDefault, layaSelfBanMax)
	if err != nil {
		return a.layaSend(c, err.Error())
	}
	return a.applySelfBan(c, d)
}

func parseSelfModerationDuration(args []string, fallback, max time.Duration) (time.Duration, error) {
	if len(args) == 0 {
		return fallback, nil
	}
	d, permanent, _, ok, err := parseDurationArgPrefix(args)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("Не поняла длительность. Пример: 5m, 15m, 1h.")
	}
	if permanent || d == 0 {
		return 0, fmt.Errorf("Навсегда через self-кнопки нельзя. Лайя оставила тебе дверную ручку.")
	}
	if d < 30*time.Second {
		return 0, fmt.Errorf("Telegram считает ограничения короче 30 секунд странными. Минимум 30s.")
	}
	if max > 0 && d > max {
		return 0, fmt.Errorf("Слишком долго. Максимум: %s.", formatDurationShort(max))
	}
	return d, nil
}

func parseSelfJailDuration(args []string, fallback, max time.Duration, allowManual bool) (time.Duration, error) {
	if len(args) > 0 && isLayaManualDurationToken(args[0]) {
		if allowManual {
			return layaSelfJailManual, nil
		}
		return 0, fmt.Errorf("Произвольный срок доступен только с титулом «главный в психбольнице».")
	}
	return parseSelfModerationDuration(args, fallback, max)
}

func (a *App) applySelfMute(c tele.Context, d time.Duration) error {
	chat, sender, err := a.layaCommandChatAndSender(c)
	if err != nil {
		return a.layaSend(c, err.Error())
	}
	if err := a.ensureSelfModerationAllowed(chat, sender, true); err != nil {
		return a.layaSend(c, err.Error(), a.layaInlineMarkup(chat, sender))
	}
	untilUnix := time.Now().UTC().Add(d).Unix()
	target := targetUser{ID: sender.ID, Username: sender.Username, Name: displayNameFromUser(sender)}
	if err := a.applyMute(chat, target, sender, untilUnix, "самомут"); err != nil {
		return a.layaSend(c, a.t("telegram_restrict_failed", err), a.layaInlineMarkup(chat, sender))
	}
	a.metrics.IncModeration("self_mute")
	a.sendOwnerLog("Self mute", logField{"User", fmt.Sprintf("%s (%d)", target.Name, target.ID)}, logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)}, logField{"Until", a.formatUntil(untilUnix)})
	return a.layaSend(c, fmt.Sprintf("Готово. Самомут для %s до %s.", target.Name, a.formatUntil(untilUnix)), a.layaInlineMarkup(chat, sender))
}

func (a *App) applySelfJail(c tele.Context, d time.Duration) error {
	chat, sender, err := a.layaCommandChatAndSender(c)
	if err != nil {
		return a.layaSend(c, err.Error())
	}
	return a.applySelfJailFor(c, chat, sender, d)
}

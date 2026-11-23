package main

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) sendLayaActivity(c tele.Context) error {
	chat, sender, err := a.layaCommandChatAndSender(c)
	if err != nil {
		return a.layaSend(c, err.Error())
	}
	rec, ok := a.store.Activity(chat.ID, sender.ID)
	if !ok {
		return a.layaSend(c, a.t("activity_unknown"), a.layaInlineMarkup(chat, sender))
	}
	now := time.Now().UTC()
	karma := a.store.Karma(chat.ID, sender.ID)
	activeWarnings := a.store.ActiveWarningCount(chat.ID, sender.ID, now)
	score := activityScore(rec, karma, activeWarnings, now)
	activity := activityBadge(score)
	lines := []string{
		fmt.Sprintf("📊 Активность %s", rec.DisplayName),
		fmt.Sprintf("🪪 ID: %d", rec.ID),
		"",
		fmt.Sprintf("🔥 Индекс активности: %d · место %s", score, socialRankLabel(a.activityRank(chat.ID, sender.ID, now))),
		fmt.Sprintf("%s Уровень активности: %s", activity.Emoji, activity.Title),
		fmt.Sprintf("💬 Сообщений всего: %d", rec.Messages),
		fmt.Sprintf("☀️ Сегодня: %d", messagesTodayAt(rec, now)),
		karmaAutoProgressLine(messagesTodayAt(rec, now), a.activeKarmaConfigForChat(chat.ID)),
		fmt.Sprintf("⭐ Карма: %d · место %s", karma, socialRankLabel(a.karmaRank(chat.ID, sender.ID))),
		fmt.Sprintf("⚠️ Активных предупреждений: %d", activeWarnings),
	}
	return a.layaSend(c, strings.Join(lines, "\n"), a.layaInlineMarkup(chat, sender))
}

func (a *App) handleTopKarmaFromButton(c tele.Context) error {
	chat, sender, err := a.layaCommandChatAndSender(c)
	if err != nil {
		return a.layaSend(c, err.Error())
	}
	records := a.store.TopKarma(chat.ID, 10)
	if len(records) == 0 {
		return a.layaSend(c, "Кармы пока нет.", a.layaInlineMarkup(chat, sender))
	}
	lines := []string{"🏆 Топ кармы:"}
	for i, rec := range records {
		name := rec.DisplayName
		if strings.TrimSpace(name) == "" {
			name = fmt.Sprintf("user %d", rec.UserID)
		}
		badge := karmaBadge(rec.Score)
		rank := i + 1
		lines = append(lines, fmt.Sprintf("%s %d. %s — %d · %s %s", medalEmoji(rank), rank, name, rec.Score, badge.Emoji, badge.Title))
	}
	return a.layaSend(c, strings.Join(lines, "\n"), a.layaInlineMarkup(chat, sender))
}

func (a *App) handleTopActiveFromButton(c tele.Context) error {
	chat, sender, err := a.layaCommandChatAndSender(c)
	if err != nil {
		return a.layaSend(c, err.Error())
	}
	rows := a.topActivityRows(chat.ID, 10, time.Now().UTC())
	if len(rows) == 0 {
		return a.layaSend(c, a.t("unknown_members"), a.layaInlineMarkup(chat, sender))
	}
	lines := []string{"🔥 Топ активности:"}
	for _, row := range rows {
		member := row.Member
		badge := activityBadge(row.Score)
		lines = append(lines, fmt.Sprintf("%s %d. %s — %d очк. · 💬 %d · ☀️ %d · ⭐ %d · %s %s",
			medalEmoji(row.Rank),
			row.Rank,
			displayKarmaName(member.DisplayName, member.ID),
			row.Score,
			member.Messages,
			messagesToday(member),
			row.Karma,
			badge.Emoji,
			badge.Title,
		))
	}
	return a.layaSend(c, strings.Join(lines, "\n"), a.layaInlineMarkup(chat, sender))
}

func (a *App) sendLayaQuote(c tele.Context) error {
	chat := c.Chat()
	sender := c.Sender()
	if !a.quotesEnabledForChat(chat) {
		return a.layaSend(c, a.t("quote_disabled"), a.layaInlineMarkup(chat, sender))
	}
	rec, ok := a.quotes.Random("")
	if !ok {
		return a.layaSend(c, a.t("quote_empty"), a.layaInlineMarkup(chat, sender))
	}
	if chat != nil && sender != nil {
		now := time.Now().UTC()
		a.unlockAchievement(chat.ID, sender.ID, "quote_first", now)
		if currentWeeklyEvent(now).Kind == weeklyEventQuotes {
			a.unlockAchievement(chat.ID, sender.ID, "weekly_quote_hunter", now)
		}
	}
	a.metrics.IncModeration("quote")
	return a.layaSend(c, a.sanitizeRandomLayaReply(sender, a.formatQuote(rec)), a.layaInlineMarkup(chat, sender))
}

func (a *App) sendLayaPhoto(c tele.Context) error {
	chat := c.Chat()
	msg := c.Message()
	sender := c.Sender()
	if !a.photosEnabledForChat(chat) {
		return a.layaSend(c, a.t("photo_disabled"), a.layaInlineMarkup(chat, sender))
	}
	file, caption, ok, err := a.randomPhotoMedia("")
	if err != nil {
		return a.layaSend(c, a.localizeErr(err), a.layaInlineMarkup(chat, sender))
	}
	if !ok {
		return a.layaSend(c, a.t("photo_empty"), a.layaInlineMarkup(chat, sender))
	}
	photo := &tele.Photo{File: file, Caption: trimRunes(a.sanitizeRandomLayaReply(sender, caption), 1000)}
	_, err = a.bot.Send(chat, photo, a.layaSendOptions(chat, msg, sender, false))
	if err != nil {
		return a.layaSend(c, a.t("photo_send_failed", err), a.layaInlineMarkup(chat, sender))
	}
	a.metrics.IncModeration("photo")
	return nil
}

func (a *App) sendLayaShop(c tele.Context) error {
	chat, sender, err := a.layaCommandChatAndSender(c)
	if err != nil {
		return a.layaSend(c, err.Error())
	}
	score := a.store.Karma(chat.ID, sender.ID)
	lines := []string{
		fmt.Sprintf("Магазин кармы. У тебя: %d.", score),
		fmt.Sprintf("Титул дня: /buy title <текст>. Цена: %d.", karmaShopTitleCost),
		"Особый титул: главный в психбольнице — длинный self-психцентр и произвольный срок.",
		fmt.Sprintf("Мягкое обращение Лайи: кнопка ниже или /buy charm. Цена: %d.", karmaShopCharmCost),
		fmt.Sprintf("Буст автоцитат: кнопка ниже или /buy quote. Цена: %d.", karmaShopQuoteCost),
	}
	if member, ok := a.store.Member(chat.ID, sender.ID); ok {
		lines = append(lines, activeKarmaShopPerkLines(member, time.Now().UTC())...)
	}
	return a.layaSend(c, strings.Join(lines, "\n"), a.layaShopMarkup())
}

func (a *App) buyLayaShopTitleCallback(c tele.Context, preset string) error {
	chat, sender, err := a.layaCommandChatAndSender(c)
	if err != nil {
		return a.layaSend(c, err.Error())
	}
	title := layaShopTitlePreset(preset)
	if title == "" {
		return a.layaSend(c, "Свой титул покупается текстом: /buy title местная легенда", a.layaShopMarkup())
	}
	now := time.Now().UTC()
	member, _ := a.store.Member(chat.ID, sender.ID)
	rewards := memberRewardsFromRecord(member)
	if err := a.buyKarmaShopItem(chat.ID, sender, karmaShopTitleCost, "shop_title", now); err != nil {
		return a.layaSend(c, err.Error(), a.layaShopMarkup())
	}
	rewards.ShopTitle = sanitizeKarmaShopTitle(title)
	rewards.ShopTitleUntilUnix = now.Add(karmaShopDuration).Unix()
	if err := a.store.SetMemberRewards(chat.ID, sender.ID, rewards); err != nil {
		return a.layaSend(c, fmt.Sprintf("Покупка списалась странно, но эффект не сохранился: %v", err), a.layaShopMarkup())
	}
	a.unlockAchievement(chat.ID, sender.ID, "shop_first", now)
	return a.layaSend(c, fmt.Sprintf("Титул дня куплен: %s. Действует 24 часа.", rewards.ShopTitle), a.layaShopMarkup())
}

func layaShopTitlePreset(preset string) string {
	switch strings.ToLower(strings.TrimSpace(preset)) {
	case "legend":
		return "местная легенда"
	case "genius":
		return "тихий гений"
	case "suspect":
		return "главный подозреваемый"
	case "rehab_chief":
		return rehabChiefTitle
	default:
		return ""
	}
}

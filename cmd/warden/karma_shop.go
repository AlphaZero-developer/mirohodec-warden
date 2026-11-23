package main

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

const (
	karmaShopTitleCost = 5
	karmaShopCharmCost = 4
	karmaShopQuoteCost = 3
	karmaShopDuration  = 24 * time.Hour
	rehabChiefTitle    = "главный в психбольнице"
)

func (a *App) handleKarmaShop(c tele.Context) error {
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	score := a.store.Karma(chat.ID, sender.ID)
	lines := []string{
		fmt.Sprintf("Магазин кармы. У тебя: %d.", score),
		fmt.Sprintf("/buy title <текст> - титул дня в /profile. Цена: %d.", karmaShopTitleCost),
		"Особый титул: главный в психбольнице — открывает длинный и произвольный self-психцентр.",
		fmt.Sprintf("/buy charm - Лайя на сутки обращается мягче в AI-ответах. Цена: %d.", karmaShopCharmCost),
		fmt.Sprintf("/buy quote - на сутки выше шанс автоцитаты рядом с тобой. Цена: %d.", karmaShopQuoteCost),
	}
	if member, ok := a.store.Member(chat.ID, sender.ID); ok {
		lines = append(lines, activeKarmaShopPerkLines(member, time.Now().UTC())...)
	}
	return c.Send(strings.Join(lines, "\n"), a.layaInlineMarkup(chat, sender))
}

func (a *App) handleKarmaBuy(c tele.Context) error {
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	args := c.Args()
	if len(args) == 0 {
		return c.Send("Использование: /buy title <текст>, /buy charm или /buy quote.", a.layaInlineMarkup(chat, sender))
	}

	item := strings.ToLower(strings.TrimSpace(args[0]))
	now := time.Now().UTC()
	member, _ := a.store.Member(chat.ID, sender.ID)
	rewards := memberRewardsFromRecord(member)
	until := now.Add(karmaShopDuration).Unix()

	switch item {
	case "title", "титул":
		title := strings.TrimSpace(strings.Join(args[1:], " "))
		if title == "" {
			return c.Send("Для титула нужен текст: /buy title местная легенда", a.layaInlineMarkup(chat, sender))
		}
		title = sanitizeKarmaShopTitle(title)
		if err := a.buyKarmaShopItem(chat.ID, sender, karmaShopTitleCost, "shop_title", now); err != nil {
			return c.Send(err.Error(), a.layaInlineMarkup(chat, sender))
		}
		rewards.ShopTitle = title
		rewards.ShopTitleUntilUnix = until
	case "charm", "address", "обращение", "мягкость":
		if err := a.buyKarmaShopItem(chat.ID, sender, karmaShopCharmCost, "shop_charm", now); err != nil {
			return c.Send(err.Error(), a.layaInlineMarkup(chat, sender))
		}
		rewards.LayaCharmUntilUnix = until
	case "quote", "цитата", "цитаты":
		if err := a.buyKarmaShopItem(chat.ID, sender, karmaShopQuoteCost, "shop_quote", now); err != nil {
			return c.Send(err.Error(), a.layaInlineMarkup(chat, sender))
		}
		rewards.QuoteBoostUntilUnix = until
	default:
		return c.Send("Такого товара нет. Открой /shop: там витрина, касса и грустный здравый смысл.", a.layaInlineMarkup(chat, sender))
	}

	if err := a.store.SetMemberRewards(chat.ID, sender.ID, rewards); err != nil {
		return c.Send(fmt.Sprintf("Покупка списалась странно, но эффект не сохранился: %v", err), a.layaInlineMarkup(chat, sender))
	}
	a.unlockAchievement(chat.ID, sender.ID, "shop_first", now)
	return c.Send("Покупка оформлена. Карма ушла в бухгалтерию Лайи, эффект действует 24 часа.", a.layaInlineMarkup(chat, sender))
}

func (a *App) buyKarmaShopItem(chatID int64, sender *tele.User, cost int64, reason string, now time.Time) error {
	score := a.store.Karma(chatID, sender.ID)
	if score < cost {
		return fmt.Errorf("Не хватает кармы: нужно %d, у тебя %d.", cost, score)
	}
	_, err := a.store.AddKarma(chatID, sender.ID, -cost, reason, sender, 0, now)
	return err
}

func sanitizeKarmaShopTitle(title string) string {
	title = strings.TrimSpace(strings.ReplaceAll(title, "\n", " "))
	title = strings.Join(strings.Fields(title), " ")
	title = trimRunes(title, 32)
	if title == "" {
		return "титул дня"
	}
	return title
}

func activeKarmaShopPerkLines(member MemberRecord, now time.Time) []string {
	var lines []string
	if member.ShopTitleUntilUnix > now.Unix() && strings.TrimSpace(member.ShopTitle) != "" {
		lines = append(lines, fmt.Sprintf("Активный титул: %s, до %s.", member.ShopTitle, formatUnix(member.ShopTitleUntilUnix)))
		if isRehabChiefTitle(member.ShopTitle) {
			lines = append(lines, "Открыто: длинный self-психцентр и произвольный срок с самостоятельным выходом.")
		}
	}
	if member.LayaCharmUntilUnix > now.Unix() {
		lines = append(lines, "Активно: мягкое обращение Лайи.")
	}
	if member.QuoteBoostUntilUnix > now.Unix() {
		lines = append(lines, "Активно: повышенный шанс автоцитаты.")
	}
	return lines
}

func (a *App) userHasRehabChiefTitle(chatID, userID int64, now time.Time) bool {
	if a == nil || a.store == nil || userID == 0 {
		return false
	}
	member, ok := a.store.Member(chatID, userID)
	if !ok {
		return false
	}
	return member.ShopTitleUntilUnix > now.Unix() && isRehabChiefTitle(member.ShopTitle)
}

func isRehabChiefTitle(title string) bool {
	return normalizeFAQText(title) == normalizeFAQText(rehabChiefTitle)
}

func (a *App) unlockAchievement(chatID, userID int64, achievement string, now time.Time) {
	if a == nil || a.store == nil {
		return
	}
	ok, err := a.store.UnlockAchievement(chatID, userID, achievement, now)
	if err != nil && a.log != nil {
		a.log.Warn("unlock achievement failed", zap.Int64("chat", chatID), zap.Int64("user", userID), zap.String("achievement", achievement), zap.Error(err))
	}
	if ok && a.metrics != nil {
		a.metrics.IncModeration("achievement")
	}
}

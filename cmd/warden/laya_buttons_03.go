package main

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func layaToneButton(label string, current bool) string {
	if current {
		return "✅ " + label
	}
	return label
}

func (a *App) setLayaToneCallback(c tele.Context, tone string) error {
	chat, sender, err := a.layaCommandChatAndSender(c)
	if err != nil {
		return a.layaSend(c, err.Error())
	}
	member, _ := a.store.Member(chat.ID, sender.ID)
	rewards := memberRewardsFromRecord(member)
	rewards.LayaTone = normalizeLayaTone(tone)
	if err := a.store.SetMemberRewards(chat.ID, sender.ID, rewards); err != nil {
		return a.layaSend(c, fmt.Sprintf("Не смогла сохранить тон: %v", err), a.layaSettingsMarkup(c))
	}
	a.unlockAchievement(chat.ID, sender.ID, "laya_tuned", time.Now().UTC())
	return a.layaSend(c, "Готово. Тон Лайи для тебя: "+layaToneTitle(rewards.LayaTone)+".", a.layaSettingsMarkup(c))
}

func (a *App) layaShopMarkup() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(layaButton("💅 Мягкое обращение", "laya:buy:charm"), layaButton("🎲 Буст цитат", "laya:buy:quote")),
		markup.Row(layaButton("🏷 Титул: местная легенда", "laya:buy:title:legend")),
		markup.Row(layaButton("🏷 Титул: тихий гений", "laya:buy:title:genius")),
		markup.Row(layaButton("🏷 Титул: главный подозреваемый", "laya:buy:title:suspect")),
		markup.Row(layaButton("🏥 Титул: главный в психбольнице", "laya:buy:title:rehab_chief")),
		markup.Row(layaButton("✍️ Свой титул", "laya:buy:title:custom_hint"), layaButton("⬅️ Меню", "laya:menu")),
	)
	return markup
}

func (a *App) renderLayaSelfMenuText(chat *tele.Chat, sender *tele.User) string {
	lines := []string{
		"Self-действия.",
		"",
		"Выбери действие и время. Следующий шаг всегда будет подтверждением, случайно не нажмётся.",
		"Навсегда через self нельзя, максимум 24 часа.",
	}
	if chat != nil && sender != nil && a.userHasRehabChiefTitle(chat.ID, sender.ID, time.Now().UTC()) {
		lines = append(lines, "Титул «главный в психбольнице» активен: психцентр можно выбрать до 30 дней или на произвольный срок с самостоятельным выходом.")
	}
	return strings.Join(lines, "\n")
}

func (a *App) layaSelfMarkup(chat *tele.Chat, sender *tele.User) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := []tele.Row{
		markup.Row(layaButton("🤐 Мут 5м", "laya:selfmute:5m"), layaButton("🤐 Мут 15м", "laya:selfmute:15m"), layaButton("🤐 Мут 1ч", "laya:selfmute:1h")),
		markup.Row(layaButton("🏥 Психцентр 15м", "laya:selfjail:15m"), layaButton("🏥 Психцентр 1ч", "laya:selfjail:1h"), layaButton("🏥 Психцентр 6ч", "laya:selfjail:6h")),
	}
	if chat != nil && sender != nil && a.userHasRehabChiefTitle(chat.ID, sender.ID, time.Now().UTC()) {
		rows = append(rows,
			markup.Row(layaButton("🏥 Психцентр 3д", "laya:selfjail:3d"), layaButton("🏥 Психцентр 7д", "laya:selfjail:7d"), layaButton("🏥 Произвольно", "laya:selfjail:manual")),
		)
	}
	rows = append(rows,
		markup.Row(layaButton("🚪 Бан 10м", "laya:selfban:10m"), layaButton("🚪 Бан 1ч", "laya:selfban:1h"), layaButton("🚪 Бан 6ч", "laya:selfban:6h")),
		markup.Row(layaButton("⬅️ Меню", "laya:menu")),
	)
	markup.Inline(rows...)
	return markup
}

func (a *App) renderLayaRehabMenuText(c tele.Context) string {
	chat := c.Chat()
	sender := c.Sender()
	if chat == nil || sender == nil || a == nil || a.store == nil {
		return "Панель психиатрического центра."
	}
	if jail, ok := a.store.ActiveJail(chat.ID, sender.ID, time.Now().UTC()); ok {
		return a.renderWardStatus(chat.ID, jail, time.Now().UTC())
	}
	return "Ты сейчас не в психиатрическом центре. Можно открыть self-действия, если хочется добровольно устроить себе драму."
}

func (a *App) layaRehabMarkup(chat *tele.Chat, sender *tele.User) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := []tele.Row{
		markup.Row(layaButton("🏥 Палата", "laya:ward")),
		markup.Row(layaButton("🏃 Побег", "laya:escape"), layaButton("💸 Подкуп", "laya:bribe")),
		markup.Row(layaButton("👤 Профиль", "laya:profile"), layaButton("⭐ Карма", "laya:karma")),
	}
	if a != nil && a.store != nil && chat != nil && sender != nil {
		if rec, ok := a.store.ActiveJail(chat.ID, sender.ID, time.Now().UTC()); ok && layaCanSelfReleaseRehab(rec, sender.ID) {
			rows = append(rows, markup.Row(layaButton("🚪 Выйти самому", "laya:selfrelease")))
		}
	}
	rows = append(rows, markup.Row(layaButton("⬅️ Меню", "laya:menu")))
	markup.Inline(rows...)
	return markup
}

func (a *App) layaSendOptions(chat *tele.Chat, msg *tele.Message, sender *tele.User, reply bool) *tele.SendOptions {
	opts := &tele.SendOptions{ReplyMarkup: a.layaInlineMarkup(chat, sender)}
	if msg != nil {
		opts.ThreadID = msg.ThreadID
		if reply {
			opts.ReplyTo = msg
		}
	}
	return opts
}

func (a *App) layaSend(c tele.Context, text string, markups ...*tele.ReplyMarkup) error {
	if c == nil || strings.TrimSpace(text) == "" {
		return nil
	}
	var markup *tele.ReplyMarkup
	if len(markups) > 0 {
		markup = markups[0]
	}
	opts := &tele.SendOptions{ReplyMarkup: markup}
	if msg := c.Message(); msg != nil {
		opts.ThreadID = msg.ThreadID
	}
	chat := c.Chat()
	if a != nil && a.bot != nil && chat != nil {
		_, err := a.bot.Send(chat, text, opts)
		return err
	}
	if opts.ThreadID != 0 || markup != nil {
		return c.Send(text, opts)
	}
	return c.Send(text)
}

func (a *App) sendLayaProfile(c tele.Context) error {
	chat, sender, err := a.layaCommandChatAndSender(c)
	if err != nil {
		return a.layaSend(c, err.Error())
	}
	target := targetUser{ID: sender.ID, Username: sender.Username, Name: displayNameFromUser(sender)}
	return a.layaSend(c, a.renderProfile(chat, target, time.Now().UTC()), a.layaInlineMarkup(chat, sender))
}

func (a *App) sendLayaKarma(c tele.Context) error {
	chat, sender, err := a.layaCommandChatAndSender(c)
	if err != nil {
		return a.layaSend(c, err.Error())
	}
	now := time.Now().UTC()
	score := a.store.Karma(chat.ID, sender.ID)
	activeWarnings := a.store.ActiveWarningCount(chat.ID, sender.ID, now)
	badge := karmaBadge(score)
	target := targetUser{ID: sender.ID, Username: sender.Username, Name: displayNameFromUser(sender)}
	lines := []string{
		fmt.Sprintf("⭐ Карма %s", displayKarmaName(target.Name, target.ID)),
		fmt.Sprintf("🪪 ID: %d", target.ID),
		"",
		fmt.Sprintf("💠 Счёт: %d", score),
		fmt.Sprintf("🏆 Место в топе: %s", socialRankLabel(a.karmaRank(chat.ID, target.ID))),
		fmt.Sprintf("%s Репутация: %s", badge.Emoji, badge.Title),
		fmt.Sprintf("📝 Вердикт Лайи: %s", badge.Hint),
		fmt.Sprintf("⚠️ Активных предупреждений: %d", activeWarnings),
	}
	if rec, ok := a.store.Activity(chat.ID, sender.ID); ok {
		activityPoints := activityScore(rec, score, activeWarnings, now)
		activity := activityBadge(activityPoints)
		lines = append(lines,
			"",
			fmt.Sprintf("🔥 Активность: %d · место %s", activityPoints, socialRankLabel(a.activityRank(chat.ID, sender.ID, now))),
			fmt.Sprintf("%s Уровень активности: %s", activity.Emoji, activity.Title),
			fmt.Sprintf("💬 Сообщений: %d всего · %d сегодня", rec.Messages, messagesTodayAt(rec, now)),
			karmaAutoProgressLine(messagesTodayAt(rec, now), a.activeKarmaConfigForChat(chat.ID)),
		)
	}
	lines = append(lines, "", "🎁 Плюшки и безопасность:")
	lines = append(lines, socialPerkLines(a.layaSafetyProfileForTarget(chat, target, now), a.activityRank(chat.ID, sender.ID, now))...)
	return a.layaSend(c, strings.Join(lines, "\n"), a.layaInlineMarkup(chat, sender))
}

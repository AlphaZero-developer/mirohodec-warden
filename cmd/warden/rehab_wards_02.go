package main

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) handleRehabWardMessage(chatID int64, jail ModerationRecord, now time.Time) (ModerationRecord, string, bool, bool) {
	if a == nil || a.store == nil {
		return jail, "", false, false
	}
	member, _ := a.store.Activity(chatID, jail.UserID)
	karma := a.store.Karma(chatID, jail.UserID)
	warnings := a.store.ActiveWarningCount(chatID, jail.UserID, now)
	progress := a.bumpRehabWardProgress(chatID, jail)

	switch rehabWard(jail).ID {
	case rehabWardRowdy:
		if progress.GoodMessages >= rowdyToQuietGoodMessages && karma >= 0 && warnings <= 1 {
			jail.RehabWard = rehabWardQuiet
			a.store.SetJail(chatID, jail)
			a.resetRehabWardProgress(chatID, jail)
			return jail, "Перевод между палатами: буйная палата смягчилась до тихой. Хорошее поведение всё-таки умеет работать, неприятно признавать.", true, false
		}
	case rehabWardQuiet:
		if qualifiesForVIPWard(member, karma, now) {
			jail.RehabWard = rehabWardVIP
			a.store.SetJail(chatID, jail)
			a.resetRehabWardProgress(chatID, jail)
			return jail, "Перевод между палатами: тихая палата повышена до VIP. Карма и активность занесли пациента в список почти приличных.", true, false
		}
	case rehabWardVIP:
		if qualifiesForVIPWard(member, karma, now) && progress.Messages >= vipReviewMinMessages && randomIndex(100) < vipReviewChancePercent {
			return jail, "VIP-палата добилась досрочного пересмотра. Комиссия сделала вид, что это была медицина, а не репутация.", true, true
		}
	}
	return jail, "", false, false
}

func (a *App) bumpRehabWardProgress(chatID int64, jail ModerationRecord) rehabWardProgress {
	key := rehabWardProgressKey(chatID, jail.UserID)
	jailSince := rehabEscapeJailSince(jail)
	ward := rehabWard(jail).ID

	a.rehabMu.Lock()
	defer a.rehabMu.Unlock()

	if a.rehabWardProgress == nil {
		a.rehabWardProgress = make(map[string]rehabWardProgress)
	}
	progress := a.rehabWardProgress[key]
	if progress.JailSince != jailSince || progress.Ward != ward {
		progress = rehabWardProgress{
			JailSince:        jailSince,
			Ward:             ward,
			QuietNextReplyAt: quietWardNextReplyAt(),
		}
	}
	progress.Messages++
	progress.GoodMessages++
	progress.QuietSinceReply++
	a.rehabWardProgress[key] = progress
	return progress
}

func (a *App) resetRehabWardProgress(chatID int64, jail ModerationRecord) {
	key := rehabWardProgressKey(chatID, jail.UserID)
	a.rehabMu.Lock()
	defer a.rehabMu.Unlock()
	if a.rehabWardProgress != nil {
		delete(a.rehabWardProgress, key)
	}
}

func (a *App) shouldSkipQuietWardReply(chatID int64, jail ModerationRecord) bool {
	if rehabWard(jail).ID != rehabWardQuiet {
		return false
	}
	key := rehabWardProgressKey(chatID, jail.UserID)
	jailSince := rehabEscapeJailSince(jail)

	a.rehabMu.Lock()
	defer a.rehabMu.Unlock()

	if a.rehabWardProgress == nil {
		a.rehabWardProgress = make(map[string]rehabWardProgress)
	}
	progress := a.rehabWardProgress[key]
	if progress.JailSince != jailSince || progress.Ward != rehabWardQuiet {
		progress = rehabWardProgress{
			JailSince:        jailSince,
			Ward:             rehabWardQuiet,
			QuietNextReplyAt: quietWardNextReplyAt(),
		}
	}
	if progress.QuietNextReplyAt <= 0 {
		progress.QuietNextReplyAt = quietWardNextReplyAt()
	}
	if progress.QuietSinceReply < progress.QuietNextReplyAt {
		a.rehabWardProgress[key] = progress
		return true
	}
	progress.QuietSinceReply = 0
	progress.QuietNextReplyAt = quietWardNextReplyAt()
	a.rehabWardProgress[key] = progress
	return false
}

func quietWardNextReplyAt() int {
	return quietWardMinMessagesBetweenReplies + randomIndex(quietWardMaxMessagesBetweenReplies-quietWardMinMessagesBetweenReplies+1)
}

func qualifiesForVIPWard(member MemberRecord, karma int64, now time.Time) bool {
	return karma >= 10 || member.Messages >= 500 || messagesTodayAt(member, now) >= 50
}

func (a *App) handleWard(c tele.Context) error {
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}

	target := targetUser{ID: sender.ID, Username: sender.Username, Name: displayNameFromUser(sender)}
	if a.isAdmin(c) && (len(c.Args()) > 0 || repliedToOtherUser(c, sender.ID)) {
		parsed, _, err := a.targetFromArgs(chat.ID, c, c.Args())
		if err != nil {
			return c.Send("Использование: /ward или ответом /ward. Админ может: /ward @user")
		}
		target = parsed
	}

	now := time.Now().UTC()
	jail, ok := a.store.ActiveJail(chat.ID, target.ID, now)
	if !ok {
		predicted := a.chooseRehabWard(chat.ID, target, "", "", now)
		return c.Send(fmt.Sprintf("%s сейчас не в психиатрическом центре.\nЕсли отправить сейчас, примерная палата: %s.", target.Name, rehabWard(ModerationRecord{RehabWard: predicted}).Title), a.layaInlineMarkup(chat, c.Sender()))
	}
	return c.Send(a.renderWardStatus(chat.ID, jail, now), a.layaInlineMarkup(chat, c.Sender()))
}

func (a *App) renderWardStatus(chatID int64, jail ModerationRecord, now time.Time) string {
	info := rehabWard(jail)
	member, _ := a.store.Activity(chatID, jail.UserID)
	karma := a.store.Karma(chatID, jail.UserID)
	cost := rehabBribeCost(member, jail)
	bribeChance := rehabBribeChance(member, karma, jail)
	escapeChance := adjustRehabEscapeChanceForWard(a.store.RehabEscapeChance(chatID, jail.UserID), jail)

	lines := []string{
		fmt.Sprintf("Палата: %s", info.Title),
		info.Notice,
		fmt.Sprintf("Побег /escape: шанс %d%%.", escapeChance),
		fmt.Sprintf("Подкуп /bribe: цена %d кармы, шанс %d%%.", cost, bribeChance),
	}
	switch info.ID {
	case rehabWardQuiet:
		lines = append(lines, "Тихая палата: Лайя отвечает примерно раз в 2-3 сообщения.")
	case rehabWardRowdy:
		lines = append(lines, "Буйная палата: кулдаун побега растет быстрее: 2m -> 4m -> 5m.")
		if karma >= 0 && a.store.ActiveWarningCount(chatID, jail.UserID, now) <= 1 {
			lines = append(lines, fmt.Sprintf("До перевода в тихую нужно спокойных сообщений: %d.", rowdyToQuietGoodMessages))
		}
	case rehabWardVIP:
		lines = append(lines, fmt.Sprintf("VIP-палата: после %d сообщений есть %d%% шанс досрочного пересмотра.", vipReviewMinMessages, vipReviewChancePercent))
	}
	lines = append(lines, releaseLine(jail.UntilUnix, now))
	return strings.Join(lines, "\n")
}

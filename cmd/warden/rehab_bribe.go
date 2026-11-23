package main

import (
	"fmt"
	"time"

	tele "gopkg.in/telebot.v3"
)

const (
	rehabBribeBaseCost         int64 = 4
	rehabBribeBaseChance             = 8
	rehabBribeMaxChance              = 25
	rehabBribeActivityStep           = 150
	rehabBribeActivityCostStep       = 300
)

func (a *App) handleBribe(c tele.Context) error {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || sender.IsBot {
		return nil
	}
	if !a.isManagedChat(chat) || chat.Type == tele.ChatPrivate {
		return c.Send("Подкуп санитара работает только внутри чата с психиатрическим центром.")
	}

	jail, ok := a.store.ActiveJail(chat.ID, sender.ID, time.Now().UTC())
	if !ok || a.isJailExempt(sender.ID) {
		a.clearRehabEscapeCooldown(chat.ID, sender.ID)
		a.hideRehabEscapeCommand(chat.ID, sender.ID)
		return c.Send("Ты сейчас не в психиатрическом центре. Санитар взял бы деньги, но за что?")
	}

	rehabTopicID := a.rehabTopicID(chat)
	if rehabTopicID != 0 && msg.ThreadID != rehabTopicID {
		return c.Send("Санитары принимают конверты только в теме психиатрического центра.")
	}
	a.runRehabBribeAttempt(c, jail)
	return nil
}

func (a *App) maybeHandleRehabBribe(c tele.Context, jail ModerationRecord) bool {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || sender.IsBot {
		return false
	}
	if !isRehabBribeCommand(messageText(msg)) {
		return false
	}
	a.runRehabBribeAttempt(c, jail)
	return true
}

func (a *App) runRehabBribeAttempt(c tele.Context, jail ModerationRecord) {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || sender.IsBot {
		return
	}
	now := time.Now().UTC()
	member, _ := a.store.Activity(chat.ID, sender.ID)
	karma := a.store.Karma(chat.ID, sender.ID)
	cost := rehabBribeCost(member, jail)
	chance := rehabBribeChance(member, karma, jail)
	if karma < cost {
		a.sendRehabEscapeReply(chat, msg, sender, fmt.Sprintf("Санитар посмотрел на карму и убрал ладонь. Нужно %d кармы, у тебя %d. Активность снижает цену, но воздух он не принимает.", cost, karma))
		if a.metrics != nil {
			a.metrics.IncModeration("rehab_bribe_no_karma")
		}
		return
	}

	left, err := a.store.AddKarma(chat.ID, sender.ID, -cost, "rehab_bribe", sender, msg.ID, now)
	if err != nil {
		a.sendRehabEscapeReply(chat, msg, sender, "Санитар уже протянул ладонь, но бухгалтерия психиатрического центра упала. Попробуй позже.")
		if a.metrics != nil {
			a.metrics.IncModeration("rehab_bribe_error")
		}
		return
	}

	success := chance > 0 && randomIndex(100) < chance
	if !success {
		a.sendRehabEscapeReply(chat, msg, sender, fmt.Sprintf("Санитар забрал %d кармы и сделал вид, что не понял намёк. Шанс был %d%%. Осталось кармы: %d.", cost, chance, left))
		if a.metrics != nil {
			a.metrics.IncModeration("rehab_bribe_failed")
		}
		return
	}

	if a.maybeStartRehabReleaseExam(chat, jail, "bribe", msg) {
		a.sendRehabEscapeReply(chat, msg, sender, fmt.Sprintf("Санитар взял %d кармы, но выпуск без экзамена не подписывает. Шанс был %d%%. Осталось кармы: %d.", cost, chance, left))
		if a.metrics != nil {
			a.metrics.IncModeration("rehab_bribe_exam")
		}
		return
	}

	rec, released := a.store.RemoveJail(chat.ID, sender.ID)
	a.hideRehabEscapeCommand(chat.ID, sender.ID)
	a.clearRehabEscapeCooldown(chat.ID, sender.ID)
	if !released {
		rec = jail
	}
	duration := a.rehabReleaseDurationText(rec, released, now)

	a.logRehabRelease(chat.ID, rec, duration, "bribe")
	a.sendRehabReleaseNotice(chat, rec, duration, msg.ThreadID)
	a.schedulePostRehabCheck(chat.ID, rec, now)
	a.sendOwnerLog("Rehab bribe",
		logField{"User", fmt.Sprintf("%s (%d)", displayNameFromUser(sender), sender.ID)},
		logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)},
		logField{"Chance", fmt.Sprintf("%d%%", chance)},
		logField{"Cost", fmt.Sprintf("%d karma", cost)},
		logField{"Karma left", fmt.Sprintf("%d", left)},
		logField{"Ward", rehabWard(rec).Title},
		logField{"Duration", duration},
	)

	a.sendRehabEscapeReply(chat, msg, sender, fmt.Sprintf("Санитар взял %d кармы, кашлянул в журнал и дверь внезапно открылась. Шанс был %d%%. Осталось кармы: %d.", cost, chance, left))
	if a.metrics != nil {
		a.metrics.IncModeration("rehab_bribe_success")
	}
}

func isRehabBribeCommand(text string) bool {
	if commandName(text) == "bribe" {
		return true
	}
	normalized := normalizeFAQText(text)
	switch normalized {
	case "подкуп", "подкуп санитара", "дать санитару", "подкупить санитара", "bribe":
		return true
	default:
		return false
	}
}

func rehabBribeCost(member MemberRecord, jail ModerationRecord) int64 {
	cost := rehabBribeBaseCost + rehabWard(jail).BribeCostDelta
	discount := member.Messages / rehabBribeActivityCostStep
	if discount > 2 {
		discount = 2
	}
	cost -= discount
	if cost < 1 {
		return 1
	}
	return cost
}

func rehabBribeChance(member MemberRecord, karma int64, jail ModerationRecord) int {
	chance := rehabBribeBaseChance + rehabWard(jail).BribeChanceDelta
	chance += int(member.Messages / rehabBribeActivityStep)
	chance += int(messagesTodayAt(member, time.Now().UTC()) / 25)
	if karma >= rehabBribeBaseCost+10 {
		chance += 2
	}
	if chance < 1 {
		return 1
	}
	if chance > rehabBribeMaxChance {
		return rehabBribeMaxChance
	}
	return chance
}

func rehabWardReply(jail ModerationRecord) (string, bool) {
	switch rehabWard(jail).ID {
	case rehabWardQuiet:
		return "", false
	case rehabWardRowdy:
		if randomIndex(100) < 35 {
			return "Буйная палата сообщает: сопротивление принято, смысл не найден.", true
		}
	case rehabWardVIP:
		if randomIndex(100) < 25 {
			return "VIP-палата просит не путать комфорт с свободой.", true
		}
	}
	return "", false
}

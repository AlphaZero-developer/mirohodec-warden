package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

type ownerSleepIntent int

const (
	ownerSleepIntentNone ownerSleepIntent = iota
	ownerSleepIntentSleep
	ownerSleepIntentWake
)

func (a *App) maybeHandleOwnerSleepCommand(c tele.Context) bool {
	if a == nil || a.store == nil {
		return false
	}
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || sender.IsBot || !a.isConfiguredAdmin(sender.ID) {
		return false
	}

	intent := parseOwnerSleepIntent(messageText(msg))
	switch intent {
	case ownerSleepIntentSleep:
		state, err := a.store.SetOwnerSleep(sender, ownerSleepAliases(sender), time.Now().UTC())
		if err != nil {
			if a.log != nil {
				a.log.Warn("set owner sleep failed", zap.Int64("user", sender.ID), zap.Error(err))
			}
			return false
		}
		a.sendOwnerSleepChatReply(c, "Сладких снов, мой любимый создатель. Спасибо, что подарил мне жизнь.")
		a.sendOwnerDirectLog(sender.ID, "Owner sleep",
			logField{Name: "Создатель", Value: userLogLabel(sender)},
			logField{Name: "Где сказал", Value: chatLogLabel(chat)},
			logField{Name: "С начала", Value: formatUnix(state.SinceUnix)},
			logField{Name: "Алиасы", Value: strings.Join(state.Aliases, ", ")},
		)
		return true
	case ownerSleepIntentWake:
		if _, sleeping := a.store.OwnerSleep(sender.ID); !sleeping {
			return false
		}
		state, ok, err := a.store.ClearOwnerSleep(sender.ID)
		if err != nil {
			if a.log != nil {
				a.log.Warn("clear owner sleep failed", zap.Int64("user", sender.ID), zap.Error(err))
			}
			return false
		}
		reply := "Доброе утро, мой создатель. Сон сняла."
		if ok {
			if len(state.Calls) == 0 {
				reply = "Доброе утро, мой создатель. Пока ты спал, тебя никто не звал."
			} else {
				reply = fmt.Sprintf("Доброе утро, мой создатель. Пока ты спал, тебя звали %d раз. Я всё записала.", len(state.Calls))
			}
		}
		a.sendOwnerSleepChatReply(c, reply)
		a.sendOwnerDirectLog(sender.ID, "Owner wake",
			logField{Name: "Создатель", Value: userLogLabel(sender)},
			logField{Name: "Спал с", Value: formatUnix(state.SinceUnix)},
			logField{Name: "Звали раз", Value: strconv.Itoa(len(state.Calls))},
			logField{Name: "Последние зовы", Value: ownerSleepCallsSummary(state.Calls, 10)},
		)
		return true
	default:
		return false
	}
}

func (a *App) maybeHandleSleepingOwnerMention(c tele.Context) bool {
	if a == nil || a.store == nil {
		return false
	}
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || sender.IsBot || chat.Type == tele.ChatPrivate || !a.isManagedChat(chat) {
		return false
	}

	for _, sleep := range a.store.ActiveOwnerSleeps() {
		if sleep.UserID == 0 || sender.ID == sleep.UserID {
			continue
		}
		matched, ok := messageMentionsSleepingOwner(msg, sleep)
		if !ok {
			continue
		}

		call := ownerSleepCallFromMessage(chat, sender, msg, matched)
		_, _, err := a.store.AddOwnerSleepCall(sleep.UserID, call)
		if err != nil && a.log != nil {
			a.log.Warn("save owner sleep call failed", zap.Int64("owner", sleep.UserID), zap.Error(err))
		}
		a.sendOwnerSleepChatReply(c, "Мой создатель сейчас спит, напишите позже.")
		a.sendOwnerDirectLog(sleep.UserID, "Creator called while sleeping",
			logField{Name: "Кто звал", Value: fmt.Sprintf("%s (%d)", call.FromName, call.FromID)},
			logField{Name: "Где", Value: chatLogLabel(chat)},
			logField{Name: "Причина", Value: fmt.Sprintf("упоминание %s", matched)},
			logField{Name: "Сообщение", Value: textOrDash(call.Text)},
			logField{Name: "Время сообщения", Value: formatUnix(call.AtUnix)},
		)
		if a.metrics != nil {
			a.metrics.IncModeration("owner_sleep_call")
		}
		return true
	}
	return false
}

func parseOwnerSleepIntent(text string) ownerSleepIntent {
	normalized := normalizeFAQText(text)
	if normalized == "" {
		return ownerSleepIntentNone
	}
	if normalizedContainsAnyPhrase(normalized, []string{
		"я проснулся",
		"я проснулась",
		"доброе утро",
	}) {
		return ownerSleepIntentWake
	}
	if normalizedContainsAnyPhrase(normalized, []string{
		"я спать",
		"я пошел спать",
		"я пошёл спать",
		"я лег спать",
		"я лёг спать",
		"я ушел спать",
		"я ушёл спать",
	}) {
		return ownerSleepIntentSleep
	}
	return ownerSleepIntentNone
}

func normalizedContainsAnyPhrase(normalized string, phrases []string) bool {
	for _, phrase := range phrases {
		if normalizedContainsPhrase(normalized, phrase) {
			return true
		}
	}
	return false
}

func normalizedContainsPhrase(normalized, phrase string) bool {
	needle := normalizeFAQText(phrase)
	if needle == "" {
		return false
	}
	return strings.Contains(" "+normalized+" ", " "+needle+" ")
}

func ownerSleepAliases(user *tele.User) []string {
	if user == nil {
		return nil
	}
	seen := make(map[string]struct{})
	var aliases []string
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || strings.EqualFold(value, "unknown") {
			return
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		aliases = append(aliases, value)
	}

	add(strconv.FormatInt(user.ID, 10))
	username := strings.TrimPrefix(strings.TrimSpace(user.Username), "@")
	if username != "" {
		add("@" + username)
		add(username)
	}
	add(user.FirstName)
	add(user.LastName)
	add(displayNameFromUser(user))
	return aliases
}

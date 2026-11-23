package main

import (
	"strings"
	"unicode"

	tele "gopkg.in/telebot.v3"
)

func (a *App) handleMafiaReplyKeyboard(c tele.Context) bool {
	if a == nil || a.store == nil || c == nil {
		return false
	}
	sender := c.Sender()
	msg := c.Message()
	if sender == nil || msg == nil {
		return false
	}
	command := normalizeMafiaReplyButtonText(messageText(msg))
	if command == "mini app" {
		roomID := ""
		if room, ok := a.activeMafiaRoomForUser(sender.ID); ok {
			roomID = room.ID
		}
		_ = a.handleMafiaMiniAppOpen(c, roomID)
		return true
	}
	switch command {
	case "мафия":
		_ = a.handleMafia(c)
		return true
	case "зарегистрироваться":
		_ = a.callbackMafiaRegister(c)
		return true
	case "что такое мафия", "правила":
		_ = c.Send(mafiaRulesText(), a.mafiaReplyKeyboard(sender.ID))
		return true
	case "назад":
		_ = a.handleMafia(c)
		return true
	case "создать комнату":
		_ = a.callbackMafiaCreate(c)
		return true
	case "войти по коду":
		_ = c.Send("🚪 Войти можно кодом:\n/mafia_join MF-4821", a.mafiaReplyKeyboard(sender.ID))
		return true
	case "мои комнаты":
		_ = a.callbackMafiaRooms(c)
		return true
	case "я готов играть":
		_ = a.callbackMafiaReadyQueueJoin(c)
		return true
	case "убрать меня из очереди":
		_ = a.callbackMafiaReadyQueueLeave(c)
		return true
	case "настройки приглашений":
		_ = a.callbackMafiaInviteSettings(c)
		return true
	case "мой персонаж":
		_ = a.callbackMafiaProfileOpen(c)
		return true
	}

	room, ok := a.activeMafiaRoomForUser(sender.ID)
	if !ok {
		if isKnownMafiaRoomReply(command) {
			_ = c.Send("🎭 Сначала откройте или создайте комнату Мафии.", a.mafiaReplyKeyboard(sender.ID))
			return true
		}
		return false
	}
	switch command {
	case "начать игру":
		_ = a.callbackMafiaStart(c, room.ID)
	case "игроки":
		_ = a.callbackMafiaPlayers(c, room.ID)
	case "живые игроки":
		_ = a.callbackMafiaAlivePlayers(c, room.ID)
	case "профили игроков":
		_ = a.callbackMafiaPlayerProfiles(c, room.ID)
	case "пригласить игроков":
		_ = a.callbackMafiaInviteMenu(c, room.ID)
	case "подключить группу":
		_ = c.Send("🔗 Привязка группы делается в нужной группе:\n/mafia_bind "+room.ID, a.mafiaReplyKeyboard(sender.ID))
	case "настройки комнаты":
		_ = a.callbackMafiaSettings(c, room.ID)
	case "полный тест механики":
		_ = a.callbackMafiaFullTest(c, room.ID)
	case "закрыть комнату":
		_ = a.callbackMafiaCloseRoom(c, room.ID)
	case "покинуть комнату":
		_ = a.callbackMafiaLeaveRoom(c, room.ID)
	case "моя роль":
		_ = a.callbackMafiaRole(c, room.ID)
	case "завещание", "моё завещание":
		_ = a.callbackMafiaWill(c, room.ID)
	case "голосование":
		if room.Status != MafiaStatusVoting {
			_ = c.Send("🗳 Голосование ещё не началось. Лайя сама покажет список, когда начнётся фаза голосования.", a.mafiaReplyKeyboard(sender.ID))
		} else {
			_ = c.Send("🗳 Голосование началось.\n\nВыберите, кого казнить:\n\n"+mafiaVotingProgressText(room, room.Votes), mafiaVotingMarkup(room, room.OwnerID == sender.ID))
		}
	case "выбрать жертву":
		_ = a.callbackMafiaActionTargetMenu(c, room.ID, "kill")
	case "теневой канал", "читать теневой канал":
		_ = a.callbackMafiaShadow(c, room.ID)
	case "лечить игрока":
		_ = a.callbackMafiaActionTargetMenu(c, room.ID, "heal")
	case "проверить игрока":
		_ = a.callbackMafiaActionTargetMenu(c, room.ID, "check")
	case "завершить ночь":
		_ = a.callbackMafiaResolveNight(c, room.ID)
	case "начать голосование":
		_ = a.callbackMafiaStartVoting(c, room.ID)
	case "завершить голосование":
		_ = a.callbackMafiaResolveVoting(c, room.ID)
	case "скип фазы":
		_ = a.callbackMafiaSkipPhase(c, room.ID)
	case "завершить партию":
		_ = a.callbackMafiaEndGame(c, room.ID)
	case "открыть набор в группе":
		_ = a.callbackMafiaRecruitmentOpen(c, room.ID)
	case "автонабор из очереди":
		_ = a.callbackMafiaQueueAutoFill(c, room.ID)
	case "разослать приглашение активным":
		_ = a.callbackMafiaInviteBroadcast(c, room.ID, false)
	case "пригласить избранных":
		_ = a.callbackMafiaInviteBroadcast(c, room.ID, true)
	case "найти игрока":
		_ = a.callbackMafiaPlayerSearch(c, room.ID)
	case "заявки игроков":
		_ = a.callbackMafiaRequests(c, room.ID)
	case "сменить режим набора":
		_ = a.callbackMafiaRecruitmentMode(c, "mafia:recruit_mode:"+room.ID+":"+mafiaOtherRecruitmentMode(room.RecruitmentMode))
	default:
		return false
	}
	return true
}

func normalizeMafiaReplyButtonText(text string) string {
	text = strings.ReplaceAll(strings.TrimSpace(text), "\ufe0f", "")
	text = strings.TrimLeftFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	text = strings.TrimSpace(strings.TrimSuffix(text, "?"))
	return strings.ToLower(text)
}

func isKnownMafiaRoomReply(command string) bool {
	switch command {
	case "начать игру", "игроки", "живые игроки", "профили игроков", "пригласить игроков", "подключить группу",
		"настройки комнаты", "полный тест механики", "закрыть комнату", "покинуть комнату",
		"моя роль", "завещание", "моё завещание", "голосование", "выбрать жертву",
		"теневой канал", "читать теневой канал", "лечить игрока", "проверить игрока",
		"завершить ночь", "начать голосование", "завершить голосование", "скип фазы", "завершить партию",
		"открыть набор в группе", "автонабор из очереди", "разослать приглашение активным",
		"пригласить избранных", "найти игрока", "заявки игроков", "сменить режим набора":
		return true
	default:
		return false
	}
}

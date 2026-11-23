package main

import (
	tele "gopkg.in/telebot.v3"
)

const (
	mafiaBtnOpen            = "🎭 Мафия"
	mafiaBtnRegister        = "✅ Зарегистрироваться"
	mafiaBtnWhatIs          = "❓ Что такое Мафия?"
	mafiaBtnBack            = "⬅️ Назад"
	mafiaBtnCreateRoom      = "➕ Создать комнату"
	mafiaBtnJoinByCode      = "🚪 Войти по коду"
	mafiaBtnMyRooms         = "📋 Мои комнаты"
	mafiaBtnRules           = "📖 Правила"
	mafiaBtnRoleRules       = "📜 Правила"
	mafiaBtnQueueJoin       = "🎮 Я готов играть"
	mafiaBtnQueueLeave      = "🚫 Убрать меня из очереди"
	mafiaBtnInviteSettings  = "🔔 Настройки приглашений"
	mafiaBtnMyProfile       = "👤 Мой персонаж"
	mafiaBtnPlayerProfiles  = "👤 Профили игроков"
	mafiaBtnStartGame       = "▶️ Начать игру"
	mafiaBtnPlayers         = "👥 Игроки"
	mafiaBtnAlivePlayers    = "👥 Живые игроки"
	mafiaBtnInvitePlayers   = "➕ Пригласить игроков"
	mafiaBtnBindGroup       = "🔗 Подключить группу"
	mafiaBtnRoomSettings    = "⚙️ Настройки комнаты"
	mafiaBtnFullTest        = "🧪 Полный тест механики"
	mafiaBtnCloseRoom       = "🗑 Закрыть комнату"
	mafiaBtnLeaveRoom       = "🚪 Покинуть комнату"
	mafiaBtnMyRole          = "📜 Моя роль"
	mafiaBtnWill            = "📜 Завещание"
	mafiaBtnMyWill          = "📜 Моё завещание"
	mafiaBtnVoting          = "🗳 Голосование"
	mafiaBtnKill            = "🗡 Выбрать жертву"
	mafiaBtnShadow          = "💬 Теневой канал"
	mafiaBtnReadShadow      = "💬 Читать теневой канал"
	mafiaBtnHeal            = "💊 Лечить игрока"
	mafiaBtnCheck           = "🔎 Проверить игрока"
	mafiaBtnResolveNight    = "🌙 Завершить ночь"
	mafiaBtnStartVoting     = "🗳 Начать голосование"
	mafiaBtnResolveVoting   = "🗳 Завершить голосование"
	mafiaBtnSkipPhase       = "⏭ Скип фазы"
	mafiaBtnEndGame         = "⛔ Завершить партию"
	mafiaBtnRecruitOpen     = "📢 Открыть набор в группе"
	mafiaBtnQueueAutoFill   = "⚡ Автонабор из очереди"
	mafiaBtnInviteActive    = "🔔 Разослать приглашение активным"
	mafiaBtnInviteFavorites = "⭐ Пригласить избранных"
	mafiaBtnFindPlayer      = "🔎 Найти игрока"
	mafiaBtnJoinRequests    = "📋 Заявки игроков"
	mafiaBtnRecruitMode     = "🔁 Сменить режим набора"
	mafiaBtnMiniApp         = "Mini App"
)

func (a *App) mafiaMenuMarkup(c tele.Context, inline *tele.ReplyMarkup, userID int64) *tele.ReplyMarkup {
	if c == nil {
		return inline
	}
	chat := c.Chat()
	if chat == nil || chat.Type != tele.ChatPrivate {
		return inline
	}
	return a.mafiaReplyKeyboard(userID)
}

func (a *App) mafiaInviteMenuReplyMarkup(c tele.Context, inline *tele.ReplyMarkup, room MafiaRoom) *tele.ReplyMarkup {
	if c == nil {
		return inline
	}
	chat := c.Chat()
	if chat == nil || chat.Type != tele.ChatPrivate {
		return inline
	}
	return mafiaReplyKeyboardFromLabels([][]string{
		{mafiaBtnRecruitOpen},
		{mafiaBtnQueueAutoFill},
		{mafiaBtnInviteActive},
		{mafiaBtnInviteFavorites},
		{mafiaBtnFindPlayer, mafiaBtnJoinRequests},
		{mafiaBtnRecruitMode},
		{mafiaBtnBack},
	})
}

func (a *App) mafiaReplyKeyboard(userID int64) *tele.ReplyMarkup {
	if a == nil || a.store == nil || userID == 0 {
		return mafiaReplyKeyboardFromLabels(a.withMafiaMiniAppReplyRows([][]string{{mafiaBtnOpen}}))
	}
	user, ok := a.store.MafiaUser(userID)
	if !ok {
		return mafiaReplyKeyboardFromLabels(a.withMafiaMiniAppReplyRows([][]string{
			{mafiaBtnRegister},
			{mafiaBtnWhatIs, mafiaBtnBack},
		}))
	}
	if room, ok := a.activeMafiaRoomForUser(userID); ok {
		return a.mafiaRoomReplyKeyboard(room, userID)
	}
	if !mafiaProfileComplete(user) {
		return mafiaReplyKeyboardFromLabels(a.withMafiaMiniAppReplyRows([][]string{
			{mafiaBtnRegister},
			{mafiaBtnWhatIs, mafiaBtnBack},
		}))
	}
	rows := [][]string{
		{mafiaBtnCreateRoom, mafiaBtnJoinByCode},
	}
	if len(a.store.MafiaRoomsForUser(userID)) > 0 {
		rows = append(rows, []string{mafiaBtnMyRooms})
	}
	if mafiaReadyQueueHas(a.store, userID) {
		rows = append(rows, []string{mafiaBtnQueueLeave})
	} else {
		rows = append(rows, []string{mafiaBtnQueueJoin})
	}
	rows = append(rows,
		[]string{mafiaBtnMyProfile},
		[]string{mafiaBtnInviteSettings},
		[]string{mafiaBtnRules, mafiaBtnBack},
	)
	return mafiaReplyKeyboardFromLabels(a.withMafiaMiniAppReplyRows(rows))
}

func (a *App) mafiaRoomReplyKeyboard(room MafiaRoom, userID int64) *tele.ReplyMarkup {
	if !a.mafiaMiniAppEnabled() {
		return mafiaRoomReplyKeyboard(room, userID)
	}
	return mafiaReplyKeyboardPrependRow(mafiaRoomReplyKeyboard(room, userID), mafiaBtnMiniApp)
}

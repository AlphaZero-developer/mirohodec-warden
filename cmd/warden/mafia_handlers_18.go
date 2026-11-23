package main

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func mafiaApplyTimePreset(settings MafiaRoomSettings, preset string) MafiaRoomSettings {
	switch preset {
	case "fast":
		settings.DayDiscussionSeconds = 90
		settings.NightActionSeconds = 45
		settings.VotingSeconds = 30
	case "long":
		settings.DayDiscussionSeconds = 300
		settings.NightActionSeconds = 120
		settings.VotingSeconds = 90
	default:
		settings.DayDiscussionSeconds = 180
		settings.NightActionSeconds = 90
		settings.VotingSeconds = 60
	}
	return settings
}

func enabledMark(enabled bool) string {
	if enabled {
		return "✅"
	}
	return "❌"
}

func mafiaTieModeLabel(mode MafiaTieMode) string {
	if mode == MafiaTieRandomBetweenLeaders {
		return "случайный из лидеров"
	}
	return "никто не умирает"
}

func mafiaRevealPolicyLabel(policy MafiaRoleRevealPolicy) string {
	switch policy {
	case MafiaRoleRevealAllowClaims:
		return "🟢 Свободный"
	case MafiaRoleRevealBanSystemProof:
		return "🟡 Обычный"
	default:
		return "🔴 Строгий"
	}
}

func mafiaRevealPolicyDescription(policy MafiaRoleRevealPolicy) string {
	switch policy {
	case MafiaRoleRevealAllowClaims:
		return "Сейчас разрешено: заявлять свою роль и строить любые игровые версии.\nСейчас запрещено: критический слив союзников, пересылки и медиа во время партии."
	case MafiaRoleRevealBanSystemProof:
		return "Сейчас разрешено: игровые заявления и блеф про роль.\nСейчас запрещено: доказывать роль через Лайю, кнопки, ЛС, скрины, пересылки и теневой канал."
	default:
		return "Сейчас разрешено: подозревать и обвинять других игроков.\nСейчас запрещено: прямо раскрывать свою роль и доказывать роль через Лайю."
	}
}

func mafiaLeakActionLabel(action MafiaGameBreakingLeakAction) string {
	switch action {
	case MafiaLeakActionHostDecision:
		return "решает создатель комнаты"
	case MafiaLeakActionKillOffender:
		return "убить нарушителя"
	default:
		return "отмена партии"
	}
}

func mafiaNextLeakAction(action MafiaGameBreakingLeakAction) MafiaGameBreakingLeakAction {
	switch action {
	case MafiaLeakActionCancelGame:
		return MafiaLeakActionHostDecision
	case MafiaLeakActionHostDecision:
		return MafiaLeakActionKillOffender
	default:
		return MafiaLeakActionCancelGame
	}
}

func (a *App) renderMafiaRoomForUser(room MafiaRoom, userID int64) string {
	timer := mafiaPhaseTimerText(room, time.Now().UTC())
	if timer != "" {
		timer = "\n" + timer
	}
	if room.Status == MafiaStatusCancelled {
		return fmt.Sprintf("🎭 Комната: %s\n\nСтатус: %s%s\nПричина: критический слив\nИгроков: %d/%d\nГруппа: %s", room.ID, mafiaPhaseLabel(room.Status), timer, len(room.Players), normalizeMafiaSettings(room.Settings).MaxPlayers, mafiaGroupStatus(room))
	}
	return fmt.Sprintf("🎭 Комната: %s\n\nСтатус: %s%s\nИгроков: %d/%d\nГруппа: %s", room.ID, mafiaPhaseLabel(room.Status), timer, len(room.Players), normalizeMafiaSettings(room.Settings).MaxPlayers, mafiaGroupStatus(room))
}

func (a *App) renderMafiaGameStatus(room MafiaRoom, userID int64) string {
	status := "наблюдатель"
	extra := ""
	if idx := mafiaRoomPlayerIndex(room, userID); idx >= 0 {
		if room.Players[idx].Alive {
			status = "жив"
			if room.Status == MafiaStatusNight && room.Players[idx].Role == MafiaRoleCivilian {
				extra = "\n\nНочью у вашей роли нет активного действия.\nОжидайте утра."
			}
		} else {
			status = "мертв"
			extra = "\n\nВы больше не участвуете в действиях партии."
		}
	}
	timer := mafiaPhaseTimerText(room, time.Now().UTC())
	if timer != "" {
		timer = "\n" + timer
	}
	return fmt.Sprintf("🎭 Игра: %s\n\nФаза: %s%s\nДень: %d\nСтатус: %s%s", room.ID, mafiaPhaseLabel(room.Status), timer, room.DayNumber, status, extra)
}

func mafiaGroupStatus(room MafiaRoom) string {
	if room.GroupChatID == 0 {
		return "не подключена"
	}
	return "подключена"
}

func mafiaRulesText() string {
	return "📜 Правила Мафии\n\n" +
		"☀️ Днём живые игроки обсуждают партию и голосуют.\n\n" +
		"🌙 Ночью общий чат закрыт. Лайя ограничивает сообщения в группе, а действия проходят только в ЛС: мафия выбирает жертву, доктор лечит, детектив проверяет.\n\n" +
		"⚰️ Мёртвые игроки не пишут в общий чат, не голосуют и не делают ночные действия.\n\n" +
		"🎭 Роли и мета-информация:\n" +
		"⛔ Нельзя доказывать роль через Лайю: пересылать ЛС, скринить роль, писать про кнопки, теневой канал или системный текст.\n" +
		"⛔ В этой комнате нельзя напрямую раскрывать свою роль: «я мафия», «я доктор», «я детектив», «я мирный».\n" +
		"⛔ Нельзя сливать союзников и теневой канал.\n\n" +
		"✅ Можно подозревать и обвинять других: «думаю, он мафия», «проверьте Софию», «голосую против Виктора». Обычная игра и блеф разрешены.\n\n" +
		"💬 Теневой канал мафии существует только в ЛС с Лайей и никогда не появляется в общем чате."
}

func mafiaStatsText(user MafiaRegisteredUser) string {
	stats := user.Stats
	return fmt.Sprintf("📊 Статистика Мафии\nИгр: %d · Побед: %d · Смертей: %d\nПобед мафии: %d · Побед города: %d\nПобед детектива: %d · Побед доктора: %d\nРаскрытий роли: %d · Отмен по вине игрока: %d · Достижений: %d", stats.Games, stats.Wins, stats.Deaths, stats.MafiaWins, stats.CivilianWins, stats.DetectiveWins, stats.DoctorWins, stats.RoleReveals, stats.CancelledGamesCaused, len(stats.Achievements))
}

func mafiaPhaseLabel(status MafiaRoomStatus) string {
	switch status {
	case MafiaStatusLobby:
		return "🕰 Лобби"
	case MafiaStatusRoleDistribution:
		return "🎭 Раздача ролей"
	case MafiaStatusNight:
		return "🌙 Ночь"
	case MafiaStatusDayDiscussion:
		return "☀️ День"
	case MafiaStatusVoting:
		return "🗳 Голосование"
	case MafiaStatusFinished:
		return "Игра завершена"
	case MafiaStatusCancelled:
		return "⛔ Партия отменена"
	default:
		return "Неизвестно"
	}
}

func (a *App) sendMafiaGroup(room MafiaRoom, text string, markup *tele.ReplyMarkup) {
	if a == nil || a.bot == nil || room.GroupChatID == 0 || strings.TrimSpace(text) == "" {
		return
	}
	chat := &tele.Chat{ID: room.GroupChatID}
	if markup != nil {
		_, _ = a.bot.Send(chat, text, markup)
		return
	}
	_, _ = a.bot.Send(chat, text)
}

func (a *App) announceMafiaLobbyPlayerJoined(room MafiaRoom, userID int64) {
	if room.Status != MafiaStatusLobby || room.GroupChatID == 0 || userID <= 0 {
		return
	}
	a.syncMafiaGroupPlayerTitles(room)
	playerName := mafiaPlayerPublicName(room, userID)
	a.sendMafiaGroup(room, fmt.Sprintf("🎭 В лобби вошёл игрок: %s\n\nКомната: %s\nИгроков: %d/%d", playerName, room.ID, len(room.Players), normalizeMafiaSettings(room.Settings).MaxPlayers), nil)
}

package main

import (
	"fmt"
	"strings"
)

func mafiaMiniPhaseLabel(status MafiaRoomStatus) string {
	switch status {
	case MafiaStatusLobby:
		return "Лобби"
	case MafiaStatusRoleDistribution:
		return "Раздача ролей"
	case MafiaStatusNight:
		return "Ночь"
	case MafiaStatusDayDiscussion:
		return "День"
	case MafiaStatusVoting:
		return "Голосование"
	case MafiaStatusFinished:
		return "Игра завершена"
	case MafiaStatusCancelled:
		return "Партия отменена"
	default:
		return "Неизвестно"
	}
}

func mafiaMiniRoomSystemText(room MafiaRoom) string {
	switch room.Status {
	case MafiaStatusLobby:
		return "Лайя ждёт игроков в лобби. Код комнаты: " + room.ID + "."
	case MafiaStatusNight:
		return fmt.Sprintf("Наступила Ночь %d. Активные роли делают выбор.", room.DayNumber)
	case MafiaStatusDayDiscussion:
		return fmt.Sprintf("Наступил День %d. Город обсуждает, кому верить.", room.DayNumber)
	case MafiaStatusVoting:
		return "Лайя открыла голосование. Выберите цель во вкладке действий."
	case MafiaStatusFinished:
		return mafiaMiniWinnerText(room.Winner)
	case MafiaStatusCancelled:
		return "Партия отменена."
	default:
		return "Лайя синхронизирует состояние партии."
	}
}

func mafiaMiniWinnerText(winner string) string {
	switch winner {
	case "mafia":
		return "Игра завершена. Победила мафия."
	case "civilian":
		return "Игра завершена. Победил город."
	default:
		return "Игра завершена."
	}
}

func mafiaMiniHostCounts(room MafiaRoom) string {
	alive := mafiaAlivePlayersCount(room)
	return fmt.Sprintf("Host-сводка: живых %d из %d, голосов %d, ночных выборов %d.", alive, len(room.Players), len(room.Votes), mafiaMiniNightActionCount(room))
}

func mafiaMiniNightActionCount(room MafiaRoom) int {
	count := 0
	count += len(room.NightActions.MafiaVotes)
	if room.NightActions.DoctorTargetID != 0 {
		count++
	}
	count += len(room.NightActions.DetectiveChecks)
	return count
}

func mafiaMiniSubmittedText(submitted bool) string {
	if submitted {
		return "Выбор принят Лайей."
	}
	return ""
}

func mafiaMiniDetectiveStatus(room MafiaRoom, viewerID int64, submitted bool) string {
	if !submitted {
		return ""
	}
	targetID := room.NightActions.DetectiveChecks[idKey(viewerID)]
	if targetID == 0 {
		return "Выбор принят Лайей."
	}
	return mafiaMiniDetectiveResultText(room, targetID, room.NightActions.DetectiveResults[idKey(viewerID)])
}

func mafiaMiniDetectiveResultText(room MafiaRoom, targetID int64, checked bool) string {
	result := "не мафия"
	if checked {
		result = "мафия"
	}
	return fmt.Sprintf("Проверка Лайи: %s — %s.", mafiaPlayerName(room, targetID), result)
}

func mafiaMiniInitials(name string) string {
	words := strings.Fields(strings.TrimSpace(name))
	if len(words) == 0 {
		return "?"
	}
	var out []rune
	for _, word := range words {
		for _, r := range word {
			out = append(out, r)
			break
		}
		if len(out) == 2 {
			break
		}
	}
	return strings.ToUpper(string(out))
}

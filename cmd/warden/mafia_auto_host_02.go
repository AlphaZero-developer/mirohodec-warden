package main

import (
	"time"
)

func mafiaVotingReadyForLaya(room MafiaRoom) bool {
	if room.Status != MafiaStatusVoting {
		return false
	}
	alive := 0
	for _, player := range room.Players {
		if !player.Alive {
			continue
		}
		alive++
		if _, ok := room.Votes[idKey(player.TelegramID)]; !ok {
			return false
		}
	}
	return alive > 0
}

func (a *App) announceMafiaDayByLaya(room MafiaRoom, result MafiaNightResult) {
	a.syncMafiaGroupRestrictions(room)
	if room.Status == MafiaStatusFinished {
		a.sendMafiaGroup(room, mafiaPhaseMessage("ДЕНЬ", mafiaNightResultText(room, result)+"\n\n"+mafiaPhaseTimerText(room, time.Now().UTC())+"\n\n"+mafiaRosterText(room)), nil)
		a.notifyMafiaPlayerEliminated(room, result.KilledID, "night")
		a.announceMafiaWill(room, result.KilledID)
		a.announceMafiaFinished(room)
		return
	}
	text := a.mafiaNarration("day_started", map[string]interface{}{
		"room":              room.ID,
		"day":               room.DayNumber,
		"alivePlayersCount": mafiaAlivePlayersCount(room),
	}, "☀️ День начался.\n\nГород просыпается и начинает обсуждение. Лайя следит за временем.")
	a.sendMafiaGroup(room, mafiaPhaseMessage("ДЕНЬ", mafiaNightResultText(room, result)+"\n\n"+text+"\n\n"+mafiaPhaseTimerText(room, time.Now().UTC())+"\n\n"+mafiaRosterText(room)), nil)
	a.notifyMafiaPlayerEliminated(room, result.KilledID, "night")
	a.announceMafiaWill(room, result.KilledID)
}

func (a *App) announceMafiaVotingByLaya(room MafiaRoom) {
	a.syncMafiaGroupRestrictions(room)
	text := a.mafiaNarration("voting_started", map[string]interface{}{
		"room":              room.ID,
		"day":               room.DayNumber,
		"alivePlayersCount": mafiaAlivePlayersCount(room),
	}, "🗳 Лайя открывает голосование.\n\nВыберите, кого казнить:")
	body := text + "\n\n" + mafiaPhaseTimerText(room, time.Now().UTC()) + "\n\n" + mafiaRosterText(room)
	if botText := mafiaAIBotMessagesText(room, MafiaStatusVoting); botText != "" {
		body += "\n\n" + botText
	}
	body += "\n\n" + mafiaVotingProgressText(room, room.Votes)
	a.sendMafiaGroup(room, mafiaPhaseMessage("ГОЛОСОВАНИЕ", body), mafiaVotingMarkup(room, false))
}

func (a *App) announceMafiaVotingResolvedByLaya(room MafiaRoom, result MafiaVoteResult) {
	a.syncMafiaGroupRestrictions(room)
	a.sendMafiaGroup(room, mafiaPhaseMessage("ИТОГ ГОЛОСОВАНИЯ", mafiaVotingResultText(room, result)), nil)
	a.notifyMafiaPlayerEliminated(room, result.ExecutedID, "vote")
	a.announceMafiaWill(room, result.ExecutedID)
	if room.Status == MafiaStatusFinished {
		a.announceMafiaFinished(room)
		return
	}
	a.announceMafiaNight(room)
}

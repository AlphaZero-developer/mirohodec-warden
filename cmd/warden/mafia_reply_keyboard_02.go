package main

import (
	"strings"

	tele "gopkg.in/telebot.v3"
)

func mafiaRoomReplyKeyboard(room MafiaRoom, userID int64) *tele.ReplyMarkup {
	settings := normalizeMafiaSettings(room.Settings)
	idx := mafiaRoomPlayerIndex(room, userID)
	if idx >= 0 && !room.Players[idx].Alive {
		if room.OwnerID == userID {
			return mafiaOwnerEmergencyReplyKeyboard(room)
		}
		return mafiaDeadReplyKeyboard(room.Players[idx], settings)
	}
	switch room.Status {
	case MafiaStatusLobby:
		if room.OwnerID == userID {
			rows := [][]string{{mafiaBtnStartGame}}
			rows = append(rows,
				[]string{mafiaBtnPlayers, mafiaBtnPlayerProfiles},
				[]string{mafiaBtnInvitePlayers},
			)
			if room.GroupChatID == 0 {
				rows = append(rows, []string{mafiaBtnBindGroup})
			}
			rows = append(rows,
				[]string{mafiaBtnRoomSettings},
				[]string{mafiaBtnFullTest},
				[]string{mafiaBtnCloseRoom, mafiaBtnBack},
			)
			return mafiaReplyKeyboardFromLabels(rows)
		}
		return mafiaReplyKeyboardFromLabels([][]string{
			{mafiaBtnPlayers, mafiaBtnPlayerProfiles},
			{mafiaBtnRoleRules},
			{mafiaBtnLeaveRoom, mafiaBtnBack},
		})
	case MafiaStatusNight:
		if idx < 0 {
			return mafiaReplyKeyboardFromLabels([][]string{{mafiaBtnAlivePlayers, mafiaBtnPlayerProfiles}, {mafiaBtnRules, mafiaBtnBack}})
		}
		player := room.Players[idx]
		rows := [][]string{}
		switch player.Role {
		case MafiaRoleMafia:
			row := []string{mafiaBtnKill}
			if settings.MafiaChatMode != MafiaChatDisabled {
				row = append(row, mafiaBtnShadow)
			}
			rows = append(rows, row)
		case MafiaRoleDoctor:
			rows = append(rows, []string{mafiaBtnHeal})
		case MafiaRoleDetective:
			rows = append(rows, []string{mafiaBtnCheck})
		}
		rows = append(rows, []string{mafiaBtnMyRole})
		if settings.EnableWills {
			rows = append(rows, []string{mafiaBtnWill})
		}
		rows = append(rows, []string{mafiaBtnAlivePlayers, mafiaBtnPlayerProfiles})
		rows = append(rows, []string{mafiaBtnRules})
		rows = append(rows, []string{mafiaBtnSkipPhase})
		if room.OwnerID == userID {
			rows = append(rows, []string{mafiaBtnEndGame})
		}
		return mafiaReplyKeyboardFromLabels(rows)
	case MafiaStatusDayDiscussion:
		rows := [][]string{{mafiaBtnVoting, mafiaBtnMyRole}}
		if idx >= 0 && room.Players[idx].Role == MafiaRoleMafia && settings.MafiaChatMode == MafiaChatAlways {
			rows = append(rows, []string{mafiaBtnShadow})
		}
		if settings.EnableWills {
			rows = append(rows, []string{mafiaBtnWill})
		}
		rows = append(rows, []string{mafiaBtnAlivePlayers, mafiaBtnPlayerProfiles})
		rows = append(rows, []string{mafiaBtnRules})
		rows = append(rows, []string{mafiaBtnSkipPhase})
		if room.OwnerID == userID {
			rows = append(rows, []string{mafiaBtnEndGame})
		}
		return mafiaReplyKeyboardFromLabels(rows)
	case MafiaStatusVoting:
		rows := [][]string{{mafiaBtnVoting, mafiaBtnMyRole}}
		if settings.EnableWills {
			rows = append(rows, []string{mafiaBtnWill})
		}
		rows = append(rows, []string{mafiaBtnAlivePlayers, mafiaBtnPlayerProfiles})
		rows = append(rows, []string{mafiaBtnRules})
		rows = append(rows, []string{mafiaBtnSkipPhase})
		if room.OwnerID == userID {
			rows = append(rows, []string{mafiaBtnEndGame})
		}
		return mafiaReplyKeyboardFromLabels(rows)
	case MafiaStatusFinished:
		return mafiaReplyKeyboardFromLabels([][]string{{mafiaBtnMyProfile}, {mafiaBtnRules, mafiaBtnBack}})
	case MafiaStatusCancelled:
		return mafiaReplyKeyboardFromLabels([][]string{{mafiaBtnPlayers, mafiaBtnPlayerProfiles}, {mafiaBtnRules, mafiaBtnBack}})
	default:
		return mafiaReplyKeyboardFromLabels([][]string{{mafiaBtnOpen}})
	}
}

func mafiaOwnerEmergencyReplyKeyboard(room MafiaRoom) *tele.ReplyMarkup {
	return mafiaReplyKeyboardFromLabels([][]string{
		{mafiaBtnAlivePlayers, mafiaBtnPlayerProfiles},
		{mafiaBtnRules},
		{mafiaBtnEndGame},
		{mafiaBtnBack},
	})
}

func mafiaDeadReplyKeyboard(player MafiaPlayer, settings MafiaRoomSettings) *tele.ReplyMarkup {
	rows := [][]string{{mafiaBtnAlivePlayers, mafiaBtnPlayerProfiles}, {mafiaBtnRules}}
	if settings.EnableWills {
		rows = append(rows, []string{mafiaBtnMyWill})
	}
	if player.Role == MafiaRoleMafia && settings.DeadMafiaCanReadChat {
		rows = append(rows, []string{mafiaBtnReadShadow})
	}
	rows = append(rows, []string{mafiaBtnBack})
	return mafiaReplyKeyboardFromLabels(rows)
}

func mafiaReplyKeyboardFromLabels(labels [][]string) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{
		ResizeKeyboard: true,
		IsPersistent:   true,
		Placeholder:    "Выберите действие",
	}
	rows := make([]tele.Row, 0, len(labels))
	for _, labelRow := range labels {
		row := make(tele.Row, 0, len(labelRow))
		for _, label := range labelRow {
			if strings.TrimSpace(label) == "" {
				continue
			}
			row = append(row, markup.Text(label))
		}
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}
	markup.Reply(rows...)
	return markup
}

func (a *App) withMafiaMiniAppReplyRows(rows [][]string) [][]string {
	if !a.mafiaMiniAppEnabled() {
		return rows
	}
	out := make([][]string, 0, len(rows)+1)
	out = append(out, []string{mafiaBtnMiniApp})
	out = append(out, rows...)
	return out
}

func mafiaReplyKeyboardPrependRow(markup *tele.ReplyMarkup, labels ...string) *tele.ReplyMarkup {
	if markup == nil {
		return mafiaReplyKeyboardFromLabels([][]string{labels})
	}
	row := make([]tele.ReplyButton, 0, len(labels))
	for _, label := range labels {
		if strings.TrimSpace(label) == "" {
			continue
		}
		row = append(row, tele.ReplyButton{Text: label})
	}
	if len(row) == 0 {
		return markup
	}
	cp := *markup
	cp.ReplyKeyboard = append([][]tele.ReplyButton{row}, markup.ReplyKeyboard...)
	return &cp
}

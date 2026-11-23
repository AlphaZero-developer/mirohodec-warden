package main

import (
	"fmt"
	wardenweb "mirohodec-warden/internal/web"
	"sort"
	"strings"
	"time"
)

func (s webPanelService) mafiaMiniMessages(room MafiaRoom, viewerID int64, viewerIdx int, isHost bool) []wardenweb.MafiaMiniAppMessage {
	out := make([]wardenweb.MafiaMiniAppMessage, 0, len(room.MiniAppMessages)+len(room.MafiaPrivateChat)+len(room.AIBotMessages)+4)
	visible := mafiaMiniVisibleChannels(room, viewerIdx, isHost)
	add := func(msg wardenweb.MafiaMiniAppMessage) {
		msg.Channel = strings.ToUpper(msg.Channel)
		if visible[msg.Channel] {
			out = append(out, msg)
		}
	}
	add(wardenweb.MafiaMiniAppMessage{
		ID:         "system_room_" + room.ID,
		Channel:    mafiaMiniChannelSystem,
		SenderName: "Лайя",
		Text:       mafiaMiniRoomSystemText(room),
		System:     true,
		Important:  true,
		CreatedAt:  room.CreatedAt,
	})
	for _, msg := range room.MiniAppMessages {
		if !mafiaMiniCanSeePrivateMiniAppMessage(msg, viewerID, isHost) {
			continue
		}
		avatarURL, avatarText, avatarTone := s.mafiaMiniAvatar(msg.SenderID)
		add(wardenweb.MafiaMiniAppMessage{
			ID:                msg.ID,
			Channel:           strings.ToUpper(msg.Channel),
			SenderID:          msg.SenderID,
			SenderName:        msg.SenderName,
			SenderAvatarURL:   avatarURL,
			SenderAvatarText:  avatarText,
			SenderAvatarTone:  avatarTone,
			Text:              msg.Text,
			System:            msg.System,
			Important:         msg.Important,
			ReplyToID:         msg.ReplyToID,
			ReplyToSenderName: msg.ReplyToSenderName,
			ReplyToText:       msg.ReplyToText,
			CreatedAt:         msg.CreatedAt,
		})
	}
	for _, msg := range room.MafiaPrivateChat {
		avatarURL, avatarText, avatarTone := s.mafiaMiniAvatar(msg.SenderID)
		add(wardenweb.MafiaMiniAppMessage{
			ID:               msg.ID,
			Channel:          mafiaMiniChannelMafia,
			SenderID:         msg.SenderID,
			SenderName:       msg.SenderAlias,
			SenderAvatarURL:  avatarURL,
			SenderAvatarText: avatarText,
			SenderAvatarTone: avatarTone,
			Text:             msg.Text,
			CreatedAt:        msg.CreatedAt,
		})
	}
	for _, msg := range room.AIBotMessages {
		add(wardenweb.MafiaMiniAppMessage{
			ID:         fmt.Sprintf("ai_%d_%d_%d", msg.CreatedAt, msg.DayNumber, msg.PlayerID),
			Channel:    mafiaMiniChannelPublic,
			SenderID:   msg.PlayerID,
			SenderName: msg.PlayerName,
			Text:       msg.Text,
			CreatedAt:  msg.CreatedAt,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].CreatedAt == out[j].CreatedAt {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt < out[j].CreatedAt
	})
	if len(out) > 180 {
		out = out[len(out)-180:]
	}
	return out
}

func (s webPanelService) mafiaMiniPlayers(room MafiaRoom, viewerID int64, viewerIdx int, isHost bool) []wardenweb.MafiaMiniAppPlayer {
	out := make([]wardenweb.MafiaMiniAppPlayer, 0, len(room.Players))
	viewer := MafiaPlayer{}
	if viewerIdx >= 0 {
		viewer = room.Players[viewerIdx]
	}
	for _, player := range room.Players {
		avatarURL, avatarText, avatarTone := s.mafiaMiniAvatar(player.TelegramID)
		roleVisible := isHost || player.TelegramID == viewerID || (viewerIdx >= 0 && isMafiaTeam(viewer.Role) && isMafiaTeam(player.Role))
		row := wardenweb.MafiaMiniAppPlayer{
			ID:         player.TelegramID,
			Name:       mafiaRoomPlayerLabel(room, player),
			Username:   player.Username,
			Initials:   mafiaMiniInitials(mafiaRoomPlayerLabel(room, player)),
			AvatarURL:  avatarURL,
			AvatarText: avatarText,
			AvatarTone: avatarTone,
			Alive:      player.Alive,
			IsSelf:     player.TelegramID == viewerID,
			IsHost:     player.TelegramID == room.OwnerID,
			IsTestBot:  player.IsTestBot,
			Voted:      strings.TrimSpace(room.Votes[idKey(player.TelegramID)]) != "",
			ActionDone: player.NightActionDone || mafiaMiniPlayerActionSubmitted(room, player),
		}
		if roleVisible || mafiaMiniDetectiveKnowsRole(room, viewerID, player.TelegramID) {
			row.Role = string(player.Role)
			row.RoleLabel = mafiaMiniRoleLabel(player.Role)
			row.MafiaAlias = player.MafiaAlias
		}
		out = append(out, row)
	}
	return out
}

func (s webPanelService) mafiaMiniLog(room MafiaRoom, viewerID int64, viewerIdx int, isHost bool) []wardenweb.MafiaMiniAppLogEvent {
	if !isHost {
		return []wardenweb.MafiaMiniAppLogEvent{}
	}
	out := []wardenweb.MafiaMiniAppLogEvent{
		{ID: "created", Text: "Комната создана.", Kind: "system", CreatedAt: room.CreatedAt},
	}
	if room.Status != MafiaStatusLobby {
		out = append(out, wardenweb.MafiaMiniAppLogEvent{ID: "started", Text: "Игра началась.", Kind: "phase", Important: true, CreatedAt: room.PhaseStartedAt})
	}
	switch room.Status {
	case MafiaStatusNight:
		out = append(out, wardenweb.MafiaMiniAppLogEvent{ID: "phase_night", Text: fmt.Sprintf("Наступила Ночь %d.", room.DayNumber), Kind: "phase", Important: true, CreatedAt: room.PhaseStartedAt})
	case MafiaStatusDayDiscussion:
		out = append(out, wardenweb.MafiaMiniAppLogEvent{ID: "phase_day", Text: fmt.Sprintf("Наступил День %d.", room.DayNumber), Kind: "phase", Important: true, CreatedAt: room.PhaseStartedAt})
	case MafiaStatusVoting:
		out = append(out, wardenweb.MafiaMiniAppLogEvent{ID: "phase_vote", Text: "Лайя открыла голосование.", Kind: "vote", Important: true, CreatedAt: room.PhaseStartedAt})
	case MafiaStatusFinished:
		out = append(out, wardenweb.MafiaMiniAppLogEvent{ID: "finished", Text: mafiaMiniWinnerText(room.Winner), Kind: "finish", Important: true, CreatedAt: room.UpdatedAt})
	case MafiaStatusCancelled:
		out = append(out, wardenweb.MafiaMiniAppLogEvent{ID: "cancelled", Text: "Партия отменена.", Kind: "danger", Important: true, CreatedAt: room.CancelledAt})
	}
	if isHost {
		out = append(out, wardenweb.MafiaMiniAppLogEvent{ID: "host_counts", Text: mafiaMiniHostCounts(room), Kind: "host", CreatedAt: time.Now().UTC().Unix()})
	}
	for _, msg := range room.MiniAppMessages {
		if msg.System && mafiaMiniCanSeePrivateMiniAppMessage(msg, viewerID, isHost) && mafiaMiniCanSeeChannel(room, viewerIdx, isHost, strings.ToUpper(msg.Channel)) {
			out = append(out, wardenweb.MafiaMiniAppLogEvent{ID: "msg_" + msg.ID, Text: msg.Text, Kind: "system", Important: msg.Important, CreatedAt: msg.CreatedAt})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return out
}

func mafiaMiniCanSeePrivateMiniAppMessage(msg MafiaMiniAppMessage, viewerID int64, isHost bool) bool {
	return msg.RecipientID == 0 || msg.RecipientID == viewerID || isHost
}

func (s webPanelService) mafiaMiniAction(room MafiaRoom, viewerID int64, viewerIdx int, isHost bool) wardenweb.MafiaMiniAppAction {
	if viewerIdx < 0 {
		return wardenweb.MafiaMiniAppAction{Kind: "none", Title: "Сейчас действия нет", Description: "Лайя ведёт комнату. Действия появятся только у игроков партии.", Disabled: true, Targets: []wardenweb.MafiaMiniAppActionTarget{}}
	}
	player := room.Players[viewerIdx]
	if !player.Alive {
		return wardenweb.MafiaMiniAppAction{Kind: "none", Title: "Сейчас действия нет", Description: "Мёртвые не голосуют и не делают ночные действия.", Disabled: true, Targets: []wardenweb.MafiaMiniAppActionTarget{}}
	}
	switch room.Status {
	case MafiaStatusNight:
		switch player.Role {
		case MafiaRoleMafia:
			submitted := room.NightActions.MafiaVotes[idKey(viewerID)] != 0
			return wardenweb.MafiaMiniAppAction{Kind: "kill", Title: "Выберите жертву", Description: "Мафия делает ночной выбор. Лайя примет только разрешённую цель.", ConfirmLabel: "Подтвердить выбор", Submitted: submitted, Disabled: submitted, StatusText: mafiaMiniSubmittedText(submitted), Targets: mafiaMiniAliveTargets(room, viewerID, false, true)}
		case MafiaRoleDoctor:
			submitted := room.NightActions.DoctorTargetID != 0
			return wardenweb.MafiaMiniAppAction{Kind: "heal", Title: "Выберите, кого лечить", Description: "Доктор может защитить одного живого игрока этой ночью.", ConfirmLabel: "Подтвердить лечение", Submitted: submitted, Disabled: submitted, StatusText: mafiaMiniSubmittedText(submitted), Targets: mafiaMiniAliveTargets(room, 0, true, false)}
		case MafiaRoleDetective:
			_, submitted := room.NightActions.DetectiveChecks[idKey(viewerID)]
			return wardenweb.MafiaMiniAppAction{Kind: "check", Title: "Выберите игрока для проверки", Description: "Лайя вернёт результат только тебе.", ConfirmLabel: "Проверить", Submitted: submitted, Disabled: submitted, StatusText: mafiaMiniDetectiveStatus(room, viewerID, submitted), Targets: mafiaMiniAliveTargets(room, viewerID, true, false)}
		}
	case MafiaStatusVoting:
		_, voted := room.Votes[idKey(viewerID)]
		locked := voted && !normalizeMafiaSettings(room.Settings).AllowVoteChange
		targets := mafiaMiniAliveTargets(room, 0, true, false)
		targets = append(targets, wardenweb.MafiaMiniAppActionTarget{ID: 0, Key: "skip", Name: "Пропустить голос"})
		return wardenweb.MafiaMiniAppAction{Kind: "vote", Title: "Выберите, против кого голосовать", Description: "Голосование считает backend. Чужие скрытые роли не раскрываются.", ConfirmLabel: "Голосовать", Submitted: voted, Disabled: locked, StatusText: mafiaMiniSubmittedText(voted), Targets: targets}
	}
	return wardenweb.MafiaMiniAppAction{Kind: "none", Title: "Сейчас действия нет", Description: "Сейчас у твоей роли нет активного действия. Следи за обсуждением.", Disabled: true, Targets: []wardenweb.MafiaMiniAppActionTarget{}}
}

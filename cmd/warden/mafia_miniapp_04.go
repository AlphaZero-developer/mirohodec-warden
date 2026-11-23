package main

import (
	"fmt"
	wardenweb "mirohodec-warden/internal/web"
	"time"
)

func mafiaMiniRoomSummary(room MafiaRoom, viewerID int64) wardenweb.MafiaMiniAppRoomSummary {
	settings := normalizeMafiaSettings(room.Settings)
	return wardenweb.MafiaMiniAppRoomSummary{
		ID:          room.ID,
		Status:      string(room.Status),
		StatusLabel: mafiaMiniPhaseLabel(room.Status),
		DayNumber:   room.DayNumber,
		Players:     len(room.Players),
		MaxPlayers:  settings.MaxPlayers,
		IsHost:      room.OwnerID == viewerID && settings.EnableHostPowers,
		IsPlayer:    mafiaRoomPlayerIndex(room, viewerID) >= 0,
		CanStart:    room.OwnerID == viewerID && room.Status == MafiaStatusLobby && len(room.Players) >= settings.MinPlayers,
		Public:      settings.PublicLobby,
		UpdatedAt:   room.UpdatedAt,
	}
}

func mafiaMiniRoom(room MafiaRoom, settings MafiaRoomSettings, now time.Time, viewerID int64, viewerIdx int, hasOwnerAccess bool, isHost bool) wardenweb.MafiaMiniAppRoom {
	phaseEndsAt := int64(0)
	remaining := 0
	if duration := mafiaPhaseDuration(room); duration > 0 {
		startedAt := room.PhaseStartedAt
		if startedAt <= 0 {
			startedAt = room.UpdatedAt
		}
		if startedAt > 0 {
			phaseEndsAt = time.Unix(startedAt, 0).UTC().Add(duration).Unix()
			remaining = int(time.Until(time.Unix(phaseEndsAt, 0)).Seconds())
			remaining = int(time.Unix(phaseEndsAt, 0).Sub(now).Seconds())
			if remaining < 0 {
				remaining = 0
			}
		}
	}
	canSkipPhase := false
	if viewerIdx >= 0 {
		player := room.Players[viewerIdx]
		canSkipPhase = mafiaAutoHostStatus(room.Status) && player.Alive && !player.IsTestBot
	}
	phaseSkipVoted := room.PhaseSkipVotes != nil && room.PhaseSkipVotes[idKey(viewerID)]
	return wardenweb.MafiaMiniAppRoom{
		ID:                   room.ID,
		Title:                "Комната " + room.ID,
		Status:               string(room.Status),
		StatusLabel:          mafiaMiniPhaseLabel(room.Status),
		DayNumber:            room.DayNumber,
		PhaseStartedAt:       room.PhaseStartedAt,
		PhaseEndsAt:          phaseEndsAt,
		RemainingSeconds:     remaining,
		Players:              len(room.Players),
		MaxPlayers:           settings.MaxPlayers,
		AlivePlayers:         mafiaAlivePlayersCount(room),
		GroupChatID:          room.GroupChatID,
		Winner:               room.Winner,
		RequiresGroupBinding: room.Status == MafiaStatusLobby && room.GroupChatID == 0,
		Settings:             mafiaMiniSettings(settings, room.Status != MafiaStatusLobby),
		CanManage:            hasOwnerAccess && (room.Status == MafiaStatusLobby || isHost),
		CanStart:             hasOwnerAccess && room.Status == MafiaStatusLobby && len(room.Players) >= settings.MinPlayers,
		CanAdvance:           isHost && (room.Status == MafiaStatusNight || room.Status == MafiaStatusDayDiscussion || room.Status == MafiaStatusVoting),
		CanLeave:             viewerIdx >= 0 && room.OwnerID != viewerID && room.Status == MafiaStatusLobby,
		CanClose:             hasOwnerAccess,
		CanSkipPhase:         canSkipPhase,
		PhaseSkipVotes:       mafiaPhaseSkipVotesCount(room),
		PhaseSkipNeeded:      mafiaPhaseSkipNeeded(room),
		PhaseSkipVoted:       phaseSkipVoted,
	}
}

func (s webPanelService) mafiaMiniMe(room MafiaRoom, viewerID int64, viewerIdx int, isHost bool) wardenweb.MafiaMiniAppMe {
	name := fmt.Sprintf("user %d", viewerID)
	if user, ok := s.app.store.MafiaUser(viewerID); ok {
		name = mafiaUserCharacterName(user)
	}
	me := wardenweb.MafiaMiniAppMe{ID: viewerID, Name: name, IsHost: isHost, IsPlayer: viewerIdx >= 0}
	if viewerIdx >= 0 {
		player := room.Players[viewerIdx]
		me.Name = mafiaRoomPlayerLabel(room, player)
		me.Alive = player.Alive
		me.Role = string(player.Role)
		me.RoleLabel = mafiaMiniRoleLabel(player.Role)
		me.RoleDescription = mafiaMiniRoleDescription(player.Role)
		me.MafiaAlias = player.MafiaAlias
		me.Will = player.Will
		me.ActionSubmitted = mafiaMiniPlayerActionSubmitted(room, player)
	}
	if viewerIdx < 0 {
		me.Alive = true
		me.RoleLabel = "Создатель"
		me.RoleDescription = "Лайя ведёт комнату, а ты управляешь настройками и ходом партии."
	}
	return me
}

func (s webPanelService) mafiaMiniChannels(room MafiaRoom, viewerIdx int, isHost bool) []wardenweb.MafiaMiniAppChannel {
	player := MafiaPlayer{}
	if viewerIdx >= 0 {
		player = room.Players[viewerIdx]
	}
	publicWritable, publicReason := mafiaMiniCanWritePublic(room, player, viewerIdx, isHost)
	channels := []wardenweb.MafiaMiniAppChannel{
		{ID: mafiaMiniChannelPublic, Label: "Общий", Visible: true, Writable: publicWritable, Reason: publicReason},
		{ID: mafiaMiniChannelSystem, Label: "Лайя", Visible: true, Writable: false},
	}
	if isHost || mafiaMiniCanSeeMafiaChannel(room, player, viewerIdx) {
		mafiaWritable := viewerIdx >= 0 && player.Alive && isMafiaTeam(player.Role)
		channels = append(channels, wardenweb.MafiaMiniAppChannel{ID: mafiaMiniChannelMafia, Label: "Мафия", Visible: true, Writable: mafiaWritable})
	}
	if isHost || (viewerIdx >= 0 && !player.Alive) {
		channels = append(channels, wardenweb.MafiaMiniAppChannel{ID: mafiaMiniChannelDead, Label: "Мёртвые", Visible: true, Writable: isHost || (viewerIdx >= 0 && !player.Alive)})
	}
	if isHost {
		channels = append(channels, wardenweb.MafiaMiniAppChannel{ID: mafiaMiniChannelHost, Label: "Host", Visible: true, Writable: true})
	}
	return channels
}

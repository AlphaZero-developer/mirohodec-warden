package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *MafiaService) StartGame(roomID string, ownerID int64, now time.Time) (MafiaRoom, error) {
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, errMafiaRoomNotFound
	}
	if room.OwnerID != ownerID {
		return MafiaRoom{}, errMafiaNotRoomOwner
	}
	if room.Status != MafiaStatusLobby {
		return MafiaRoom{}, errMafiaRoomNotLobby
	}
	settings := normalizeMafiaSettings(room.Settings)
	if len(room.Players) < settings.MinPlayers {
		return MafiaRoom{}, fmt.Errorf("need at least %d players", settings.MinPlayers)
	}
	if room.GroupChatID == 0 {
		return MafiaRoom{}, errors.New("mafia group is not bound")
	}
	room.Settings = settings
	room.Status = MafiaStatusNight
	room.DayNumber = 1
	room.Votes = map[string]string{}
	room.PhaseSkipVotes = map[string]bool{}
	room.NightActions = MafiaNightActions{MafiaVotes: map[string]int64{}, DetectiveChecks: map[string]int64{}, DetectiveResults: map[string]bool{}}
	room.DetectiveKnownRoles = map[string]map[string]MafiaRole{}
	assignMafiaRoles(room.Players, s.rng)
	room.PlayersSnapshot = s.mafiaGamePlayerSnapshot(room.Players, settings)
	room.EloProcessed = false
	room.GameResult = MafiaGameResult{}
	room.PhaseStartedAt = now.UTC().Unix()
	room.UpdatedAt = now.UTC().Unix()
	return room, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) PrepareRematchLobby(roomID string, now time.Time) (MafiaRoom, error) {
	if s == nil || s.store == nil {
		return MafiaRoom{}, errMafiaRoomNotFound
	}
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, errMafiaRoomNotFound
	}
	if room.Status != MafiaStatusFinished {
		return MafiaRoom{}, errMafiaWrongPhase
	}
	room.Status = MafiaStatusLobby
	room.DayNumber = 0
	room.Winner = ""
	room.FinishedAt = 0
	room.EloProcessed = false
	room.GameResult = MafiaGameResult{}
	room.PlayersSnapshot = nil
	room.CancelReason = ""
	room.CancelledBy = 0
	room.CancelledAt = 0
	room.PendingLeakType = ""
	room.PendingLeakOffenderID = 0
	room.PendingLeakAt = 0
	room.Votes = map[string]string{}
	room.PhaseSkipVotes = map[string]bool{}
	room.NightActions = MafiaNightActions{MafiaVotes: map[string]int64{}, DetectiveChecks: map[string]int64{}, DetectiveResults: map[string]bool{}}
	room.DetectiveKnownRoles = map[string]map[string]MafiaRole{}
	room.MafiaPrivateChat = nil
	room.PhaseStartedAt = 0
	room.RecruitmentStatus = MafiaRecruitmentClosed
	room.JoinRequests = nil
	for i := range room.Players {
		room.Players[i].Role = ""
		room.Players[i].MafiaAlias = ""
		room.Players[i].Alive = true
		room.Players[i].WarnedForReveal = false
		room.Players[i].NightActionDone = false
		room.Players[i].Will = ""
	}
	room.UpdatedAt = now.UTC().Unix()
	return room, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) SendShadowMessage(roomID string, senderID int64, text string, now time.Time) (MafiaRoom, MafiaPrivateMessage, []int64, error) {
	text = strings.TrimSpace(text)
	if text == "" || commandName(text) != "" {
		return MafiaRoom{}, MafiaPrivateMessage{}, nil, errMafiaShadowUnavailable
	}
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, MafiaPrivateMessage{}, nil, errMafiaRoomNotFound
	}
	settings := normalizeMafiaSettings(room.Settings)
	if settings.MafiaChatMode == MafiaChatDisabled || (settings.MafiaChatMode == MafiaChatNightOnly && room.Status != MafiaStatusNight) {
		return MafiaRoom{}, MafiaPrivateMessage{}, nil, errMafiaShadowUnavailable
	}
	idx := mafiaRoomPlayerIndex(room, senderID)
	if idx < 0 {
		return MafiaRoom{}, MafiaPrivateMessage{}, nil, errMafiaPlayerNotFound
	}
	sender := room.Players[idx]
	if sender.Role != MafiaRoleMafia {
		return MafiaRoom{}, MafiaPrivateMessage{}, nil, errMafiaWrongRole
	}
	if !sender.Alive {
		return MafiaRoom{}, MafiaPrivateMessage{}, nil, errMafiaPlayerDead
	}
	msg := appendMafiaShadowMessage(&room, senderID, sender.MafiaAlias, text, now)
	recipients := mafiaShadowRecipients(room, settings)
	room.UpdatedAt = now.UTC().Unix()
	return room, msg, recipients, s.store.SaveMafiaRoom(room)
}

func appendMafiaShadowMessage(room *MafiaRoom, senderID int64, senderAlias string, text string, now time.Time) MafiaPrivateMessage {
	if room == nil {
		return MafiaPrivateMessage{}
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	senderAlias = strings.TrimSpace(senderAlias)
	if senderAlias == "" {
		senderAlias = "AI"
	}
	msg := MafiaPrivateMessage{
		ID:          fmt.Sprintf("shadow_%d_%d", now.UnixNano(), senderID),
		RoomID:      room.ID,
		SenderID:    senderID,
		SenderAlias: senderAlias,
		Text:        strings.TrimSpace(text),
		CreatedAt:   now.UTC().Unix(),
	}
	room.MafiaPrivateChat = append(room.MafiaPrivateChat, msg)
	if len(room.MafiaPrivateChat) > 200 {
		room.MafiaPrivateChat = room.MafiaPrivateChat[len(room.MafiaPrivateChat)-200:]
	}
	return msg
}

func mafiaShadowRecipients(room MafiaRoom, settings MafiaRoomSettings) []int64 {
	recipients := make([]int64, 0, 3)
	for _, player := range room.Players {
		if player.Role != MafiaRoleMafia {
			continue
		}
		if player.Alive || settings.DeadMafiaCanReadChat {
			recipients = append(recipients, player.TelegramID)
		}
	}
	return recipients
}

func (s *MafiaService) MafiaKill(roomID string, mafiaID, targetID int64, now time.Time) (MafiaRoom, error) {
	room, player, err := s.playerActionRoom(roomID, mafiaID, MafiaRoleMafia, MafiaStatusNight)
	if err != nil {
		return MafiaRoom{}, err
	}
	targetIdx := mafiaRoomPlayerIndex(room, targetID)
	if targetIdx < 0 {
		return MafiaRoom{}, errMafiaPlayerNotFound
	}
	if !room.Players[targetIdx].Alive || room.Players[targetIdx].Role == MafiaRoleMafia {
		return MafiaRoom{}, errMafiaWrongRole
	}
	if room.NightActions.MafiaVotes == nil {
		room.NightActions.MafiaVotes = map[string]int64{}
	}
	room.NightActions.MafiaVotes[idKey(player.TelegramID)] = targetID
	room.Players[mafiaRoomPlayerIndex(room, mafiaID)].NightActionDone = true
	room.UpdatedAt = now.UTC().Unix()
	return room, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) DoctorHeal(roomID string, doctorID, targetID int64, now time.Time) (MafiaRoom, error) {
	room, _, err := s.playerActionRoom(roomID, doctorID, MafiaRoleDoctor, MafiaStatusNight)
	if err != nil {
		return MafiaRoom{}, err
	}
	targetIdx := mafiaRoomPlayerIndex(room, targetID)
	if targetIdx < 0 {
		return MafiaRoom{}, errMafiaPlayerNotFound
	}
	if !room.Players[targetIdx].Alive {
		return MafiaRoom{}, errMafiaPlayerDead
	}
	room.NightActions.DoctorTargetID = targetID
	room.Players[mafiaRoomPlayerIndex(room, doctorID)].NightActionDone = true
	room.UpdatedAt = now.UTC().Unix()
	return room, s.store.SaveMafiaRoom(room)
}

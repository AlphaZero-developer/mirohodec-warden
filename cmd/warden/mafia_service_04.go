package main

import (
	"time"
)

func (s *MafiaService) EndGameByOwner(roomID string, ownerID int64, now time.Time) (MafiaRoom, error) {
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, errMafiaRoomNotFound
	}
	if room.OwnerID != ownerID {
		return MafiaRoom{}, errMafiaNotRoomOwner
	}
	if room.Status == MafiaStatusFinished || room.Status == MafiaStatusCancelled {
		return room, nil
	}
	room.Status = MafiaStatusCancelled
	room.CancelReason = MafiaRuleViolationNone
	room.CancelledBy = ownerID
	room.CancelledAt = now.UTC().Unix()
	room.UpdatedAt = now.UTC().Unix()
	return room, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) PunishRoleReveal(roomID string, playerID int64, now time.Time) (MafiaRoom, error) {
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, errMafiaRoomNotFound
	}
	idx := mafiaRoomPlayerIndex(room, playerID)
	if idx < 0 {
		return MafiaRoom{}, errMafiaPlayerNotFound
	}
	settings := normalizeMafiaSettings(room.Settings)
	if !settings.PunishRoleReveal {
		return room, nil
	}
	s.recordMafiaRoleReveal(playerID)
	switch settings.RevealPunishmentMode {
	case MafiaRevealWarning:
		if room.Players[idx].WarnedForReveal {
			room.Players[idx].Alive = false
		} else {
			room.Players[idx].WarnedForReveal = true
		}
	default:
		room.Players[idx].Alive = false
	}
	if winner := mafiaWinner(room); winner != "" {
		s.finishMafiaRoom(&room, winner)
	}
	room.UpdatedAt = now.UTC().Unix()
	return room, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) CancelGameForCriticalLeak(roomID string, offenderID int64, reason MafiaRuleViolationType, now time.Time) (MafiaRoom, error) {
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, errMafiaRoomNotFound
	}
	if room.Status == MafiaStatusFinished || room.Status == MafiaStatusCancelled {
		return room, nil
	}
	if mafiaRoomPlayerIndex(room, offenderID) < 0 {
		return MafiaRoom{}, errMafiaPlayerNotFound
	}
	if reason != MafiaRuleViolationAllyLeak {
		reason = MafiaRuleViolationGameBreakingLeak
	}
	room.Status = MafiaStatusCancelled
	room.CancelReason = reason
	room.CancelledBy = offenderID
	room.CancelledAt = now.UTC().Unix()
	room.PendingLeakType = ""
	room.PendingLeakOffenderID = 0
	room.PendingLeakAt = 0
	room.Winner = ""
	room.UpdatedAt = now.UTC().Unix()
	s.recordMafiaCancelledGameCaused(offenderID)
	_ = s.store.SaveMafiaCancelledGame(MafiaCancelledGameRecord{
		RoomID:      room.ID,
		CancelledAt: room.CancelledAt,
		Reason:      reason,
		OffenderID:  offenderID,
	})
	return room, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) MarkCriticalLeakPending(roomID string, offenderID int64, reason MafiaRuleViolationType, now time.Time) (MafiaRoom, error) {
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, errMafiaRoomNotFound
	}
	if room.Status == MafiaStatusFinished || room.Status == MafiaStatusCancelled {
		return room, nil
	}
	if mafiaRoomPlayerIndex(room, offenderID) < 0 {
		return MafiaRoom{}, errMafiaPlayerNotFound
	}
	room.PendingLeakType = reason
	room.PendingLeakOffenderID = offenderID
	room.PendingLeakAt = now.UTC().Unix()
	room.UpdatedAt = now.UTC().Unix()
	return room, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) ClearCriticalLeakPending(roomID string, ownerID int64, now time.Time) (MafiaRoom, error) {
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, errMafiaRoomNotFound
	}
	if room.OwnerID != ownerID {
		return MafiaRoom{}, errMafiaNotRoomOwner
	}
	room.PendingLeakType = ""
	room.PendingLeakOffenderID = 0
	room.PendingLeakAt = 0
	room.UpdatedAt = now.UTC().Unix()
	return room, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) KillOffenderForCriticalLeak(roomID string, ownerID int64, now time.Time) (MafiaRoom, error) {
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, errMafiaRoomNotFound
	}
	if room.OwnerID != ownerID {
		return MafiaRoom{}, errMafiaNotRoomOwner
	}
	offenderID := room.PendingLeakOffenderID
	if offenderID == 0 {
		return room, nil
	}
	idx := mafiaRoomPlayerIndex(room, offenderID)
	if idx < 0 {
		return MafiaRoom{}, errMafiaPlayerNotFound
	}
	room.Players[idx].Alive = false
	room.PendingLeakType = ""
	room.PendingLeakOffenderID = 0
	room.PendingLeakAt = 0
	if winner := mafiaWinner(room); winner != "" {
		s.finishMafiaRoom(&room, winner)
	}
	room.UpdatedAt = now.UTC().Unix()
	return room, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) RecreateCancelledRoom(roomID string, ownerID int64, now time.Time) (MafiaRoom, error) {
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, errMafiaRoomNotFound
	}
	if room.OwnerID != ownerID {
		return MafiaRoom{}, errMafiaNotRoomOwner
	}
	if room.Status != MafiaStatusCancelled {
		return MafiaRoom{}, errMafiaWrongPhase
	}
	next := MafiaRoom{
		ID:                s.newRoomID(),
		OwnerID:           room.OwnerID,
		GroupChatID:       room.GroupChatID,
		Status:            MafiaStatusLobby,
		Settings:          normalizeMafiaSettings(room.Settings),
		CreatedAt:         now.UTC().Unix(),
		UpdatedAt:         now.UTC().Unix(),
		RecruitmentStatus: MafiaRecruitmentClosed,
		RecruitmentMode:   room.RecruitmentMode,
	}
	for _, player := range room.Players {
		if player.TelegramID == room.CancelledBy && player.TelegramID != room.OwnerID {
			continue
		}
		player.Role = ""
		player.MafiaAlias = ""
		player.Alive = true
		player.WarnedForReveal = false
		player.NightActionDone = false
		player.Will = ""
		player.JoinedAt = now.UTC().Unix()
		next.Players = append(next.Players, player)
	}
	if len(next.Players) == 0 {
		if owner, ok := s.store.MafiaUser(ownerID); ok {
			next.Players = append(next.Players, mafiaPlayerFromUser(owner, now))
		}
	}
	return next, s.store.SaveMafiaRoom(next)
}

package main

import (
	"time"

	tele "gopkg.in/telebot.v3"
)

func (s *MafiaService) SubmitJoinRequest(roomID string, user *tele.User, now time.Time) (MafiaRoom, bool, error) {
	if user == nil || user.ID == 0 {
		return MafiaRoom{}, false, errMafiaUserNotRegistered
	}
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, false, errMafiaRoomNotFound
	}
	if room.Status != MafiaStatusLobby {
		return MafiaRoom{}, false, errMafiaRoomNotLobby
	}
	if room.RecruitmentStatus != MafiaRecruitmentOpen {
		return MafiaRoom{}, false, errMafiaRecruitmentClosed
	}
	if s.store.MafiaHostBlocked(room.OwnerID, user.ID) {
		return MafiaRoom{}, false, errMafiaHostBlocked
	}
	registered, ok := s.store.MafiaUser(user.ID)
	if !ok {
		return MafiaRoom{}, false, errMafiaUserNotRegistered
	}
	if mafiaRoomPlayerIndex(room, user.ID) >= 0 {
		return room, false, nil
	}
	if len(room.Players) >= normalizeMafiaSettings(room.Settings).MaxPlayers {
		return MafiaRoom{}, false, errMafiaRoomFull
	}
	if room.RecruitmentMode == MafiaRecruitmentAutoAccept {
		room.Players = append(room.Players, mafiaPlayerFromUser(registered, now))
		room.UpdatedAt = now.UTC().Unix()
		return room, true, s.store.SaveMafiaRoom(room)
	}
	if mafiaJoinRequestIndex(room, user.ID, MafiaJoinRequestPending) >= 0 {
		return MafiaRoom{}, false, errMafiaRequestExists
	}
	room.JoinRequests = append(room.JoinRequests, MafiaJoinRequest{
		TelegramID:    registered.TelegramID,
		CharacterName: mafiaUserCharacterName(registered),
		Username:      registered.Username,
		RequestedAt:   now.UTC().Unix(),
		Status:        MafiaJoinRequestPending,
	})
	room.UpdatedAt = now.UTC().Unix()
	return room, false, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) AcceptJoinRequest(roomID string, ownerID, userID int64, now time.Time) (MafiaRoom, error) {
	room, err := s.ownerLobbyRoom(roomID, ownerID)
	if err != nil {
		return MafiaRoom{}, err
	}
	if len(room.Players) >= normalizeMafiaSettings(room.Settings).MaxPlayers {
		return MafiaRoom{}, errMafiaRoomFull
	}
	if s.store.MafiaHostBlocked(room.OwnerID, userID) {
		return MafiaRoom{}, errMafiaHostBlocked
	}
	idx := mafiaJoinRequestIndex(room, userID, MafiaJoinRequestPending)
	if idx < 0 {
		return MafiaRoom{}, errMafiaPlayerNotFound
	}
	rec, ok := s.store.MafiaUser(userID)
	if !ok {
		return MafiaRoom{}, errMafiaUserNotRegistered
	}
	if mafiaRoomPlayerIndex(room, userID) < 0 {
		room.Players = append(room.Players, mafiaPlayerFromUser(rec, now))
	}
	room.JoinRequests[idx].Status = MafiaJoinRequestAccepted
	room.UpdatedAt = now.UTC().Unix()
	return room, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) DeclineJoinRequest(roomID string, ownerID, userID int64, now time.Time) (MafiaRoom, error) {
	room, err := s.ownerLobbyRoom(roomID, ownerID)
	if err != nil {
		return MafiaRoom{}, err
	}
	idx := mafiaJoinRequestIndex(room, userID, MafiaJoinRequestPending)
	if idx < 0 {
		return MafiaRoom{}, errMafiaPlayerNotFound
	}
	room.JoinRequests[idx].Status = MafiaJoinRequestDeclined
	room.UpdatedAt = now.UTC().Unix()
	return room, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) AcceptFirstJoinRequests(roomID string, ownerID int64, now time.Time) (MafiaRoom, []int64, error) {
	room, err := s.ownerLobbyRoom(roomID, ownerID)
	if err != nil {
		return MafiaRoom{}, nil, err
	}
	settings := normalizeMafiaSettings(room.Settings)
	var accepted []int64
	for i := range room.JoinRequests {
		if len(room.Players) >= settings.MaxPlayers {
			break
		}
		if room.JoinRequests[i].Status != MafiaJoinRequestPending {
			continue
		}
		rec, ok := s.store.MafiaUser(room.JoinRequests[i].TelegramID)
		if !ok {
			room.JoinRequests[i].Status = MafiaJoinRequestDeclined
			continue
		}
		if mafiaRoomPlayerIndex(room, rec.TelegramID) < 0 {
			if s.store.MafiaHostBlocked(room.OwnerID, rec.TelegramID) {
				room.JoinRequests[i].Status = MafiaJoinRequestDeclined
				continue
			}
			room.Players = append(room.Players, mafiaPlayerFromUser(rec, now))
			accepted = append(accepted, rec.TelegramID)
		}
		room.JoinRequests[i].Status = MafiaJoinRequestAccepted
	}
	room.UpdatedAt = now.UTC().Unix()
	return room, accepted, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) DeclineAllJoinRequests(roomID string, ownerID int64, now time.Time) (MafiaRoom, error) {
	room, err := s.ownerLobbyRoom(roomID, ownerID)
	if err != nil {
		return MafiaRoom{}, err
	}
	for i := range room.JoinRequests {
		if room.JoinRequests[i].Status == MafiaJoinRequestPending {
			room.JoinRequests[i].Status = MafiaJoinRequestDeclined
		}
	}
	room.UpdatedAt = now.UTC().Unix()
	return room, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) JoinReadyQueue(user *tele.User, now time.Time) (MafiaReadyQueueEntry, error) {
	if user == nil || user.ID == 0 {
		return MafiaReadyQueueEntry{}, errMafiaUserNotRegistered
	}
	rec, ok := s.store.MafiaUser(user.ID)
	if !ok {
		return MafiaReadyQueueEntry{}, errMafiaUserNotRegistered
	}
	queue := s.store.MafiaReadyQueue()
	for _, entry := range queue {
		if entry.TelegramID == rec.TelegramID {
			return entry, nil
		}
	}
	entry := MafiaReadyQueueEntry{
		TelegramID:    rec.TelegramID,
		CharacterName: mafiaUserCharacterName(rec),
		Username:      rec.Username,
		QueuedAt:      now.UTC().Unix(),
	}
	queue = append(queue, entry)
	return entry, s.store.SaveMafiaReadyQueue(queue)
}

func (s *MafiaService) LeaveReadyQueue(userID int64) error {
	queue := s.store.MafiaReadyQueue()
	out := queue[:0]
	for _, entry := range queue {
		if entry.TelegramID != userID {
			out = append(out, entry)
		}
	}
	return s.store.SaveMafiaReadyQueue(out)
}

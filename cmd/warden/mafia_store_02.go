package main

import (
	"sort"
)

func (s *Store) SaveMafiaRoom(room MafiaRoom) error {
	room.ID = normalizeMafiaRoomID(room.ID)
	if room.ID == "" {
		return errMafiaRoomNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureMafiaLocked()
	s.state.Mafia.Rooms[room.ID] = room
	return s.saveLocked()
}

func (s *Store) MafiaRoom(roomID string) (MafiaRoom, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	room, ok := s.state.Mafia.Rooms[normalizeMafiaRoomID(roomID)]
	if ok {
		room = normalizeMafiaRoomRecruitment(room)
	}
	return room, ok
}

func (s *Store) MafiaRooms() []MafiaRoom {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]MafiaRoom, 0, len(s.state.Mafia.Rooms))
	for _, room := range s.state.Mafia.Rooms {
		out = append(out, normalizeMafiaRoomRecruitment(room))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	return out
}

func (s *Store) MafiaRoomsForUser(userID int64) []MafiaRoom {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]MafiaRoom, 0, 4)
	for _, room := range s.state.Mafia.Rooms {
		room = normalizeMafiaRoomRecruitment(room)
		if room.OwnerID == userID || mafiaRoomPlayerIndex(room, userID) >= 0 {
			out = append(out, room)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	return out
}

func (s *Store) MafiaRoomByGroup(chatID int64) (MafiaRoom, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, room := range s.state.Mafia.Rooms {
		room = normalizeMafiaRoomRecruitment(room)
		if room.GroupChatID == chatID && room.Status != MafiaStatusFinished && room.Status != MafiaStatusCancelled {
			return room, true
		}
	}
	return MafiaRoom{}, false
}

func (s *Store) DeleteMafiaRoom(roomID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.state.Mafia.Rooms, normalizeMafiaRoomID(roomID))
	return s.saveLocked()
}

func (s *Store) ensureMafiaLocked() {
	if s.state.Mafia.RegisteredUsers == nil {
		s.state.Mafia.RegisteredUsers = make(map[string]MafiaRegisteredUser)
	}
	if s.state.Mafia.Rooms == nil {
		s.state.Mafia.Rooms = make(map[string]MafiaRoom)
	}
	if s.state.Mafia.ReadyQueue == nil {
		s.state.Mafia.ReadyQueue = []MafiaReadyQueueEntry{}
	}
	if s.state.Mafia.CancelledGames == nil {
		s.state.Mafia.CancelledGames = []MafiaCancelledGameRecord{}
	}
	if s.state.Mafia.HostBlacklist == nil {
		s.state.Mafia.HostBlacklist = []MafiaHostBlacklistEntry{}
	}
	if s.state.Mafia.PlayerRatings == nil {
		s.state.Mafia.PlayerRatings = make(map[string]MafiaPlayerRating)
	}
	if s.state.Mafia.EloHistory == nil {
		s.state.Mafia.EloHistory = []MafiaEloHistoryEntry{}
	}
	if s.state.Mafia.GameResults == nil {
		s.state.Mafia.GameResults = []MafiaGameResult{}
	}
}

func uniqueInt64s(values []int64) []int64 {
	seen := map[int64]struct{}{}
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

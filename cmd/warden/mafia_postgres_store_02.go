package main

import (
	"encoding/json"
	"sort"
	"time"
)

func (s *PostgresStore) SaveMafiaRoom(room MafiaRoom) error {
	room.ID = normalizeMafiaRoomID(room.ID)
	if room.ID == "" {
		return errMafiaRoomNotFound
	}
	s.mafiaMu.Lock()
	defer s.mafiaMu.Unlock()
	state := s.loadMafiaStateLocked()
	state.Rooms[room.ID] = room
	return s.saveMafiaStateLocked(state)
}

func (s *PostgresStore) MafiaRoom(roomID string) (MafiaRoom, bool) {
	state := s.loadMafiaStateLocked()
	room, ok := state.Rooms[normalizeMafiaRoomID(roomID)]
	if ok {
		room = normalizeMafiaRoomRecruitment(room)
	}
	return room, ok
}

func (s *PostgresStore) MafiaRooms() []MafiaRoom {
	state := s.loadMafiaStateLocked()
	out := make([]MafiaRoom, 0, len(state.Rooms))
	for _, room := range state.Rooms {
		out = append(out, normalizeMafiaRoomRecruitment(room))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	return out
}

func (s *PostgresStore) MafiaRoomsForUser(userID int64) []MafiaRoom {
	state := s.loadMafiaStateLocked()
	out := make([]MafiaRoom, 0, 4)
	for _, room := range state.Rooms {
		room = normalizeMafiaRoomRecruitment(room)
		if room.OwnerID == userID || mafiaRoomPlayerIndex(room, userID) >= 0 {
			out = append(out, room)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	return out
}

func (s *PostgresStore) MafiaRoomByGroup(chatID int64) (MafiaRoom, bool) {
	state := s.loadMafiaStateLocked()
	for _, room := range state.Rooms {
		room = normalizeMafiaRoomRecruitment(room)
		if room.GroupChatID == chatID && room.Status != MafiaStatusFinished && room.Status != MafiaStatusCancelled {
			return room, true
		}
	}
	return MafiaRoom{}, false
}

func (s *PostgresStore) DeleteMafiaRoom(roomID string) error {
	s.mafiaMu.Lock()
	defer s.mafiaMu.Unlock()
	state := s.loadMafiaStateLocked()
	delete(state.Rooms, normalizeMafiaRoomID(roomID))
	return s.saveMafiaStateLocked(state)
}

func (s *PostgresStore) loadMafiaStateLocked() MafiaState {
	state := MafiaState{
		RegisteredUsers: make(map[string]MafiaRegisteredUser),
		Rooms:           make(map[string]MafiaRoom),
	}
	if s == nil || s.db == nil {
		return state
	}
	var raw []byte
	if err := s.db.QueryRow(`SELECT value FROM bot_settings WHERE key=$1`, mafiaStateSettingKey).Scan(&raw); err != nil {
		return state
	}
	_ = json.Unmarshal(raw, &state)
	if state.RegisteredUsers == nil {
		state.RegisteredUsers = make(map[string]MafiaRegisteredUser)
	}
	if state.Rooms == nil {
		state.Rooms = make(map[string]MafiaRoom)
	}
	if state.ReadyQueue == nil {
		state.ReadyQueue = []MafiaReadyQueueEntry{}
	}
	if state.CancelledGames == nil {
		state.CancelledGames = []MafiaCancelledGameRecord{}
	}
	if state.HostBlacklist == nil {
		state.HostBlacklist = []MafiaHostBlacklistEntry{}
	}
	if state.PlayerRatings == nil {
		state.PlayerRatings = map[string]MafiaPlayerRating{}
	}
	if state.EloHistory == nil {
		state.EloHistory = []MafiaEloHistoryEntry{}
	}
	if state.GameResults == nil {
		state.GameResults = []MafiaGameResult{}
	}
	return state
}

func (s *PostgresStore) saveMafiaStateLocked(state MafiaState) error {
	if state.RegisteredUsers == nil {
		state.RegisteredUsers = make(map[string]MafiaRegisteredUser)
	}
	if state.Rooms == nil {
		state.Rooms = make(map[string]MafiaRoom)
	}
	if state.ReadyQueue == nil {
		state.ReadyQueue = []MafiaReadyQueueEntry{}
	}
	if state.CancelledGames == nil {
		state.CancelledGames = []MafiaCancelledGameRecord{}
	}
	if state.HostBlacklist == nil {
		state.HostBlacklist = []MafiaHostBlacklistEntry{}
	}
	if state.PlayerRatings == nil {
		state.PlayerRatings = map[string]MafiaPlayerRating{}
	}
	if state.EloHistory == nil {
		state.EloHistory = []MafiaEloHistoryEntry{}
	}
	if state.GameResults == nil {
		state.GameResults = []MafiaGameResult{}
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO bot_settings(key,value,updated_unix)
		VALUES($1,$2::jsonb,$3)
		ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value, updated_unix=EXCLUDED.updated_unix`,
		mafiaStateSettingKey, string(raw), time.Now().UTC().Unix())
	return err
}

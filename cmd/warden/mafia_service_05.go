package main

import (
	"strings"
	"time"
)

func (s *MafiaService) SaveWill(roomID string, userID int64, text string, now time.Time) (MafiaRoom, error) {
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, errMafiaRoomNotFound
	}
	settings := normalizeMafiaSettings(room.Settings)
	if !settings.EnableWills {
		return MafiaRoom{}, errMafiaWillsDisabled
	}
	idx := mafiaRoomPlayerIndex(room, userID)
	if idx < 0 {
		return MafiaRoom{}, errMafiaPlayerNotFound
	}
	text = strings.TrimSpace(text)
	if text == "" {
		room.Players[idx].Will = ""
	} else {
		if len([]rune(text)) > 700 {
			text = string([]rune(text)[:700])
		}
		room.Players[idx].Will = text
	}
	room.Settings = settings
	room.UpdatedAt = now.UTC().Unix()
	return room, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) UpdateLobbySettings(roomID string, ownerID int64, now time.Time, update func(MafiaRoomSettings) MafiaRoomSettings) (MafiaRoom, error) {
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, errMafiaRoomNotFound
	}
	if room.OwnerID != ownerID {
		return MafiaRoom{}, errMafiaNotRoomOwner
	}
	if room.Status != MafiaStatusLobby {
		return MafiaRoom{}, errMafiaSettingsLocked
	}
	room.Settings = normalizeMafiaSettings(update(normalizeMafiaSettings(room.Settings)))
	if room.Settings.MaxPlayers < len(room.Players) {
		room.Settings.MaxPlayers = len(room.Players)
	}
	if room.Settings.MinPlayers > room.Settings.MaxPlayers {
		room.Settings.MinPlayers = room.Settings.MaxPlayers
	}
	room.UpdatedAt = now.UTC().Unix()
	return room, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) AddTestBots(roomID string, ownerID int64, count int, now time.Time) (MafiaRoom, []MafiaPlayer, error) {
	room, err := s.ownerLobbyRoom(roomID, ownerID)
	if err != nil {
		return MafiaRoom{}, nil, err
	}
	settings := normalizeMafiaSettings(room.Settings)
	if count <= 0 {
		count = 1
	}
	free := settings.MaxPlayers - len(room.Players)
	if free <= 0 {
		return MafiaRoom{}, nil, errMafiaRoomFull
	}
	if count > free {
		count = free
	}
	added := make([]MafiaPlayer, 0, count)
	for i := 0; i < count; i++ {
		player := mafiaNewTestBotPlayer(room, now)
		room.Players = append(room.Players, player)
		added = append(added, player)
	}
	room.Settings = settings
	room.UpdatedAt = now.UTC().Unix()
	return room, added, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) FillTestBotsToMinimum(roomID string, ownerID int64, now time.Time) (MafiaRoom, []MafiaPlayer, error) {
	room, err := s.ownerLobbyRoom(roomID, ownerID)
	if err != nil {
		return MafiaRoom{}, nil, err
	}
	settings := normalizeMafiaSettings(room.Settings)
	needed := settings.MinPlayers - len(room.Players)
	if needed <= 0 {
		return room, nil, nil
	}
	return s.AddTestBots(room.ID, ownerID, needed, now)
}

func (s *MafiaService) RemoveTestBots(roomID string, ownerID int64, now time.Time) (MafiaRoom, int, error) {
	room, err := s.ownerLobbyRoom(roomID, ownerID)
	if err != nil {
		return MafiaRoom{}, 0, err
	}
	players := room.Players[:0]
	removed := 0
	for _, player := range room.Players {
		if player.IsTestBot {
			removed++
			continue
		}
		players = append(players, player)
	}
	room.Players = players
	room.Settings = normalizeMafiaSettings(room.Settings)
	room.UpdatedAt = now.UTC().Unix()
	return room, removed, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) OpenRecruitment(roomID string, ownerID int64, mode MafiaRecruitmentMode, now time.Time) (MafiaRoom, error) {
	room, err := s.ownerLobbyRoom(roomID, ownerID)
	if err != nil {
		return MafiaRoom{}, err
	}
	if len(room.Players) >= normalizeMafiaSettings(room.Settings).MaxPlayers {
		return MafiaRoom{}, errMafiaRoomFull
	}
	if mode == "" {
		mode = room.RecruitmentMode
	}
	if mode == "" {
		mode = MafiaRecruitmentRequests
	}
	room.RecruitmentStatus = MafiaRecruitmentOpen
	room.RecruitmentMode = mode
	room.UpdatedAt = now.UTC().Unix()
	return room, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) CloseRecruitment(roomID string, ownerID int64, now time.Time) (MafiaRoom, error) {
	room, err := s.ownerLobbyRoom(roomID, ownerID)
	if err != nil {
		return MafiaRoom{}, err
	}
	room.RecruitmentStatus = MafiaRecruitmentClosed
	room.UpdatedAt = now.UTC().Unix()
	return room, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) SetRecruitmentMode(roomID string, ownerID int64, mode MafiaRecruitmentMode, now time.Time) (MafiaRoom, error) {
	room, err := s.ownerLobbyRoom(roomID, ownerID)
	if err != nil {
		return MafiaRoom{}, err
	}
	if mode != MafiaRecruitmentAutoAccept {
		mode = MafiaRecruitmentRequests
	}
	room.RecruitmentMode = mode
	room.UpdatedAt = now.UTC().Unix()
	return room, s.store.SaveMafiaRoom(room)
}

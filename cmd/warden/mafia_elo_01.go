package main

import (
	"sort"
	"time"
)

const mafiaDefaultElo = 1000

func (s *MafiaService) mafiaGamePlayerSnapshot(players []MafiaPlayer, settings MafiaRoomSettings) []MafiaGamePlayerSnapshot {
	out := make([]MafiaGamePlayerSnapshot, 0, len(players))
	for _, player := range players {
		rating := mafiaDefaultPlayerRating(player.TelegramID)
		if s != nil && s.store != nil && !player.IsTestBot {
			if stored, ok := s.store.MafiaPlayerRating(player.TelegramID); ok {
				rating = mafiaNormalizePlayerRating(stored)
			} else if rec, ok := s.store.MafiaUser(player.TelegramID); ok {
				rating = mafiaRatingFromStats(player.TelegramID, rec.Stats)
			}
		}
		side := mafiaRatingSideForRole(player.Role)
		out = append(out, MafiaGamePlayerSnapshot{
			UserID:             player.TelegramID,
			Username:           player.Username,
			DisplayName:        player.DisplayName,
			Role:               player.Role,
			Side:               side,
			IsTestBot:          player.IsTestBot,
			StartingOverallElo: rating.OverallElo,
			StartingRoleElo:    mafiaRoleElo(rating, side),
		})
	}
	return out
}

func (s *Store) MafiaPlayerRating(userID int64) (MafiaPlayerRating, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.state.Mafia.PlayerRatings == nil {
		return mafiaDefaultPlayerRating(userID), false
	}
	rating, ok := s.state.Mafia.PlayerRatings[idKey(userID)]
	if ok {
		return mafiaNormalizePlayerRating(rating), true
	}
	if rec, ok := s.state.Mafia.RegisteredUsers[idKey(userID)]; ok {
		return mafiaRatingFromStats(userID, rec.Stats), true
	}
	return mafiaDefaultPlayerRating(userID), false
}

func (s *Store) MafiaPlayerRatings() []MafiaPlayerRating {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]MafiaPlayerRating, 0, len(s.state.Mafia.RegisteredUsers))
	seen := map[string]bool{}
	for key, rating := range s.state.Mafia.PlayerRatings {
		out = append(out, mafiaNormalizePlayerRating(rating))
		seen[key] = true
	}
	for key, rec := range s.state.Mafia.RegisteredUsers {
		if seen[key] {
			continue
		}
		out = append(out, mafiaRatingFromStats(rec.TelegramID, rec.Stats))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].OverallElo == out[j].OverallElo {
			return out[i].RankedWins > out[j].RankedWins
		}
		return out[i].OverallElo > out[j].OverallElo
	})
	return out
}

func (s *Store) MafiaGameResultsForUser(userID int64) []MafiaGameResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return mafiaGameResultsForUser(s.state.Mafia.GameResults, userID)
}

func (s *Store) ApplyMafiaGameResult(roomID string, winner string, finishedAt time.Time) (MafiaRoom, MafiaGameResult, []MafiaEloHistoryEntry, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureMafiaLocked()
	roomID = normalizeMafiaRoomID(roomID)
	room, ok := s.state.Mafia.Rooms[roomID]
	if !ok {
		return MafiaRoom{}, MafiaGameResult{}, nil, false, errMafiaRoomNotFound
	}
	room, result, history, processed, err := mafiaApplyGameResultToState(&s.state.Mafia, room, winner, finishedAt)
	if err != nil {
		return MafiaRoom{}, MafiaGameResult{}, nil, false, err
	}
	s.state.Mafia.Rooms[room.ID] = room
	if err := s.saveLocked(); err != nil {
		return MafiaRoom{}, MafiaGameResult{}, nil, false, err
	}
	return room, result, history, processed, nil
}

func (s *PostgresStore) MafiaPlayerRating(userID int64) (MafiaPlayerRating, bool) {
	state := s.loadMafiaStateLocked()
	if rating, ok := state.PlayerRatings[idKey(userID)]; ok {
		return mafiaNormalizePlayerRating(rating), true
	}
	if rec, ok := state.RegisteredUsers[idKey(userID)]; ok {
		return mafiaRatingFromStats(userID, rec.Stats), true
	}
	return mafiaDefaultPlayerRating(userID), false
}

func (s *PostgresStore) MafiaPlayerRatings() []MafiaPlayerRating {
	state := s.loadMafiaStateLocked()
	out := make([]MafiaPlayerRating, 0, len(state.RegisteredUsers))
	seen := map[string]bool{}
	for key, rating := range state.PlayerRatings {
		out = append(out, mafiaNormalizePlayerRating(rating))
		seen[key] = true
	}
	for key, rec := range state.RegisteredUsers {
		if seen[key] {
			continue
		}
		out = append(out, mafiaRatingFromStats(rec.TelegramID, rec.Stats))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].OverallElo == out[j].OverallElo {
			return out[i].RankedWins > out[j].RankedWins
		}
		return out[i].OverallElo > out[j].OverallElo
	})
	return out
}

func (s *PostgresStore) MafiaGameResultsForUser(userID int64) []MafiaGameResult {
	state := s.loadMafiaStateLocked()
	return mafiaGameResultsForUser(state.GameResults, userID)
}

func (s *PostgresStore) ApplyMafiaGameResult(roomID string, winner string, finishedAt time.Time) (MafiaRoom, MafiaGameResult, []MafiaEloHistoryEntry, bool, error) {
	s.mafiaMu.Lock()
	defer s.mafiaMu.Unlock()
	state := s.loadMafiaStateLocked()
	roomID = normalizeMafiaRoomID(roomID)
	room, ok := state.Rooms[roomID]
	if !ok {
		return MafiaRoom{}, MafiaGameResult{}, nil, false, errMafiaRoomNotFound
	}
	room, result, history, processed, err := mafiaApplyGameResultToState(&state, room, winner, finishedAt)
	if err != nil {
		return MafiaRoom{}, MafiaGameResult{}, nil, false, err
	}
	state.Rooms[room.ID] = room
	if err := s.saveMafiaStateLocked(state); err != nil {
		return MafiaRoom{}, MafiaGameResult{}, nil, false, err
	}
	return room, result, history, processed, nil
}

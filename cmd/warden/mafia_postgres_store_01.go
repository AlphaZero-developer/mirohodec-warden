package main

import (
	"sort"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

const mafiaStateSettingKey = "mafia_state"

func (s *PostgresStore) RegisterMafiaUser(user *tele.User, now time.Time) (MafiaRegisteredUser, error) {
	if user == nil || user.ID == 0 || user.IsBot {
		return MafiaRegisteredUser{}, errMafiaUserNotRegistered
	}
	s.mafiaMu.Lock()
	defer s.mafiaMu.Unlock()
	state := s.loadMafiaStateLocked()
	key := idKey(user.ID)
	rec := state.RegisteredUsers[key]
	next := mafiaUserFromTelegram(user, now)
	if rec.RegisteredAt != 0 {
		next.RegisteredAt = rec.RegisteredAt
		next.Profile = rec.Profile
		next.Stats = rec.Stats
		next.Settings = normalizeMafiaUserSettings(rec.Settings)
		next.FavoriteMafiaUserIDs = append([]int64(nil), rec.FavoriteMafiaUserIDs...)
	}
	next.Settings = normalizeMafiaUserSettings(next.Settings)
	state.RegisteredUsers[key] = next
	if err := s.saveMafiaStateLocked(state); err != nil {
		return MafiaRegisteredUser{}, err
	}
	return next, nil
}

func (s *PostgresStore) SetMafiaProfile(userID int64, profile MafiaProfile) error {
	s.mafiaMu.Lock()
	defer s.mafiaMu.Unlock()
	state := s.loadMafiaStateLocked()
	key := idKey(userID)
	rec, ok := state.RegisteredUsers[key]
	if !ok {
		return errMafiaUserNotRegistered
	}
	now := time.Now().UTC().Unix()
	profile.CharacterName = strings.TrimSpace(profile.CharacterName)
	profile.Backstory = strings.TrimSpace(profile.Backstory)
	profile.AvatarURL = mafiaMiniCleanAvatarURL(profile.AvatarURL)
	profile.AvatarText = mafiaMiniCleanAvatarText(profile.AvatarText)
	profile.AvatarTone = mafiaMiniCleanAvatarTone(profile.AvatarTone)
	if profile.CreatedAt == 0 {
		profile.CreatedAt = rec.Profile.CreatedAt
	}
	if profile.CreatedAt == 0 {
		profile.CreatedAt = now
	}
	profile.UpdatedAt = now
	rec.Profile = profile
	state.RegisteredUsers[key] = rec
	return s.saveMafiaStateLocked(state)
}

func (s *PostgresStore) MafiaUser(userID int64) (MafiaRegisteredUser, bool) {
	state := s.loadMafiaStateLocked()
	rec, ok := state.RegisteredUsers[idKey(userID)]
	if ok {
		rec.Settings = normalizeMafiaUserSettings(rec.Settings)
	}
	return rec, ok
}

func (s *PostgresStore) MafiaUsers() []MafiaRegisteredUser {
	state := s.loadMafiaStateLocked()
	out := make([]MafiaRegisteredUser, 0, len(state.RegisteredUsers))
	for _, rec := range state.RegisteredUsers {
		rec.Settings = normalizeMafiaUserSettings(rec.Settings)
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RegisteredAt > out[j].RegisteredAt })
	return out
}

func (s *PostgresStore) SetMafiaUserSettings(userID int64, settings MafiaUserSettings) error {
	s.mafiaMu.Lock()
	defer s.mafiaMu.Unlock()
	state := s.loadMafiaStateLocked()
	key := idKey(userID)
	rec, ok := state.RegisteredUsers[key]
	if !ok {
		return errMafiaUserNotRegistered
	}
	rec.Settings = normalizeMafiaUserSettings(settings)
	state.RegisteredUsers[key] = rec
	return s.saveMafiaStateLocked(state)
}

func (s *PostgresStore) SetMafiaUserFavorites(userID int64, favorites []int64) error {
	s.mafiaMu.Lock()
	defer s.mafiaMu.Unlock()
	state := s.loadMafiaStateLocked()
	key := idKey(userID)
	rec, ok := state.RegisteredUsers[key]
	if !ok {
		return errMafiaUserNotRegistered
	}
	rec.FavoriteMafiaUserIDs = uniqueInt64s(favorites)
	state.RegisteredUsers[key] = rec
	return s.saveMafiaStateLocked(state)
}

func (s *PostgresStore) SaveMafiaCancelledGame(record MafiaCancelledGameRecord) error {
	if strings.TrimSpace(record.RoomID) == "" {
		return errMafiaRoomNotFound
	}
	s.mafiaMu.Lock()
	defer s.mafiaMu.Unlock()
	state := s.loadMafiaStateLocked()
	record.RoomID = normalizeMafiaRoomID(record.RoomID)
	state.CancelledGames = append(state.CancelledGames, record)
	if len(state.CancelledGames) > 500 {
		state.CancelledGames = state.CancelledGames[len(state.CancelledGames)-500:]
	}
	return s.saveMafiaStateLocked(state)
}

func (s *PostgresStore) AddMafiaHostBlacklist(hostID, blockedUserID int64, reason string, now time.Time) error {
	if hostID == 0 || blockedUserID == 0 {
		return errMafiaUserNotRegistered
	}
	s.mafiaMu.Lock()
	defer s.mafiaMu.Unlock()
	state := s.loadMafiaStateLocked()
	for i, entry := range state.HostBlacklist {
		if entry.HostID == hostID && entry.BlockedUserID == blockedUserID {
			entry.Reason = strings.TrimSpace(reason)
			entry.CreatedAt = now.UTC().Unix()
			state.HostBlacklist[i] = entry
			return s.saveMafiaStateLocked(state)
		}
	}
	state.HostBlacklist = append(state.HostBlacklist, MafiaHostBlacklistEntry{
		HostID:        hostID,
		BlockedUserID: blockedUserID,
		Reason:        strings.TrimSpace(reason),
		CreatedAt:     now.UTC().Unix(),
	})
	return s.saveMafiaStateLocked(state)
}

func (s *PostgresStore) MafiaHostBlocked(hostID, userID int64) bool {
	state := s.loadMafiaStateLocked()
	for _, entry := range state.HostBlacklist {
		if entry.HostID == hostID && entry.BlockedUserID == userID {
			return true
		}
	}
	return false
}

func (s *PostgresStore) MafiaReadyQueue() []MafiaReadyQueueEntry {
	state := s.loadMafiaStateLocked()
	out := append([]MafiaReadyQueueEntry(nil), state.ReadyQueue...)
	sort.Slice(out, func(i, j int) bool { return out[i].QueuedAt < out[j].QueuedAt })
	return out
}

func (s *PostgresStore) SaveMafiaReadyQueue(queue []MafiaReadyQueueEntry) error {
	s.mafiaMu.Lock()
	defer s.mafiaMu.Unlock()
	state := s.loadMafiaStateLocked()
	state.ReadyQueue = append([]MafiaReadyQueueEntry(nil), queue...)
	return s.saveMafiaStateLocked(state)
}

func (s *PostgresStore) SetMafiaUserStats(userID int64, stats MafiaStats) error {
	s.mafiaMu.Lock()
	defer s.mafiaMu.Unlock()
	state := s.loadMafiaStateLocked()
	rec, ok := state.RegisteredUsers[idKey(userID)]
	if !ok {
		return errMafiaUserNotRegistered
	}
	rec.Stats = stats
	state.RegisteredUsers[idKey(userID)] = rec
	return s.saveMafiaStateLocked(state)
}

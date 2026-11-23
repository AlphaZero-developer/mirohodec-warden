package main

import (
	"sort"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (s *Store) RegisterMafiaUser(user *tele.User, now time.Time) (MafiaRegisteredUser, error) {
	if user == nil || user.ID == 0 || user.IsBot {
		return MafiaRegisteredUser{}, errMafiaUserNotRegistered
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureMafiaLocked()
	key := idKey(user.ID)
	rec := s.state.Mafia.RegisteredUsers[key]
	next := mafiaUserFromTelegram(user, now)
	if rec.RegisteredAt != 0 {
		next.RegisteredAt = rec.RegisteredAt
		next.Profile = rec.Profile
		next.Stats = rec.Stats
		next.Settings = normalizeMafiaUserSettings(rec.Settings)
		next.FavoriteMafiaUserIDs = append([]int64(nil), rec.FavoriteMafiaUserIDs...)
	}
	next.Settings = normalizeMafiaUserSettings(next.Settings)
	s.state.Mafia.RegisteredUsers[key] = next
	return next, s.saveLocked()
}

func (s *Store) SetMafiaProfile(userID int64, profile MafiaProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureMafiaLocked()
	key := idKey(userID)
	rec, ok := s.state.Mafia.RegisteredUsers[key]
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
	s.state.Mafia.RegisteredUsers[key] = rec
	return s.saveLocked()
}

func (s *Store) MafiaUser(userID int64) (MafiaRegisteredUser, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.state.Mafia.RegisteredUsers[idKey(userID)]
	if ok {
		rec.Settings = normalizeMafiaUserSettings(rec.Settings)
	}
	return rec, ok
}

func (s *Store) MafiaUsers() []MafiaRegisteredUser {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]MafiaRegisteredUser, 0, len(s.state.Mafia.RegisteredUsers))
	for _, rec := range s.state.Mafia.RegisteredUsers {
		rec.Settings = normalizeMafiaUserSettings(rec.Settings)
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].RegisteredAt > out[j].RegisteredAt
	})
	return out
}

func (s *Store) SetMafiaUserSettings(userID int64, settings MafiaUserSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureMafiaLocked()
	key := idKey(userID)
	rec, ok := s.state.Mafia.RegisteredUsers[key]
	if !ok {
		return errMafiaUserNotRegistered
	}
	rec.Settings = normalizeMafiaUserSettings(settings)
	s.state.Mafia.RegisteredUsers[key] = rec
	return s.saveLocked()
}

func (s *Store) SetMafiaUserFavorites(userID int64, favorites []int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureMafiaLocked()
	key := idKey(userID)
	rec, ok := s.state.Mafia.RegisteredUsers[key]
	if !ok {
		return errMafiaUserNotRegistered
	}
	rec.FavoriteMafiaUserIDs = uniqueInt64s(favorites)
	s.state.Mafia.RegisteredUsers[key] = rec
	return s.saveLocked()
}

func (s *Store) SaveMafiaCancelledGame(record MafiaCancelledGameRecord) error {
	if strings.TrimSpace(record.RoomID) == "" {
		return errMafiaRoomNotFound
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureMafiaLocked()
	record.RoomID = normalizeMafiaRoomID(record.RoomID)
	s.state.Mafia.CancelledGames = append(s.state.Mafia.CancelledGames, record)
	if len(s.state.Mafia.CancelledGames) > 500 {
		s.state.Mafia.CancelledGames = s.state.Mafia.CancelledGames[len(s.state.Mafia.CancelledGames)-500:]
	}
	return s.saveLocked()
}

func (s *Store) AddMafiaHostBlacklist(hostID, blockedUserID int64, reason string, now time.Time) error {
	if hostID == 0 || blockedUserID == 0 {
		return errMafiaUserNotRegistered
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureMafiaLocked()
	for i, entry := range s.state.Mafia.HostBlacklist {
		if entry.HostID == hostID && entry.BlockedUserID == blockedUserID {
			entry.Reason = strings.TrimSpace(reason)
			entry.CreatedAt = now.UTC().Unix()
			s.state.Mafia.HostBlacklist[i] = entry
			return s.saveLocked()
		}
	}
	s.state.Mafia.HostBlacklist = append(s.state.Mafia.HostBlacklist, MafiaHostBlacklistEntry{
		HostID:        hostID,
		BlockedUserID: blockedUserID,
		Reason:        strings.TrimSpace(reason),
		CreatedAt:     now.UTC().Unix(),
	})
	return s.saveLocked()
}

func (s *Store) MafiaHostBlocked(hostID, userID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, entry := range s.state.Mafia.HostBlacklist {
		if entry.HostID == hostID && entry.BlockedUserID == userID {
			return true
		}
	}
	return false
}

func (s *Store) MafiaReadyQueue() []MafiaReadyQueueEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := append([]MafiaReadyQueueEntry(nil), s.state.Mafia.ReadyQueue...)
	sort.Slice(out, func(i, j int) bool { return out[i].QueuedAt < out[j].QueuedAt })
	return out
}

func (s *Store) SaveMafiaReadyQueue(queue []MafiaReadyQueueEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureMafiaLocked()
	s.state.Mafia.ReadyQueue = append([]MafiaReadyQueueEntry(nil), queue...)
	return s.saveLocked()
}

func (s *Store) SetMafiaUserStats(userID int64, stats MafiaStats) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureMafiaLocked()
	rec, ok := s.state.Mafia.RegisteredUsers[idKey(userID)]
	if !ok {
		return errMafiaUserNotRegistered
	}
	rec.Stats = stats
	s.state.Mafia.RegisteredUsers[idKey(userID)] = rec
	return s.saveLocked()
}

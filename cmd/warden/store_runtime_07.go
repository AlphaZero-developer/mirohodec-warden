package main

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (s *Store) RandomActiveMember(chatID int64, activeWindow time.Duration, now time.Time, excludeID int64) (MemberRecord, bool) {
	if activeWindow <= 0 {
		activeWindow = 7 * 24 * time.Hour
	}
	cutoff := now.Add(-activeWindow).Unix()
	members := s.Members(chatID)
	weighted := make([]MemberRecord, 0, len(members))
	for _, rec := range members {
		if rec.ID == 0 || rec.ID == excludeID || rec.IsBot || rec.LastMessage < cutoff || rec.Messages <= 0 {
			continue
		}
		weight := int(messagesToday(rec)) + 1
		if weight < 1 {
			weight = 1
		}
		if weight > 10 {
			weight = 10
		}
		for i := 0; i < weight; i++ {
			weighted = append(weighted, rec)
		}
	}
	if len(weighted) == 0 && excludeID != 0 {
		return s.RandomActiveMember(chatID, activeWindow, now, 0)
	}
	if len(weighted) == 0 {
		return MemberRecord{}, false
	}
	return weighted[rand.Intn(len(weighted))], true
}

func (s *Store) RegisterOwner(user *tele.User, chat *tele.Chat) error {
	if user == nil || chat == nil {
		return errors.New("missing auth user or chat")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	exists := false
	for _, id := range s.state.Bot.OwnerIDs {
		if id == user.ID {
			exists = true
			break
		}
	}
	if !exists {
		s.state.Bot.OwnerIDs = append(s.state.Bot.OwnerIDs, user.ID)
	}
	s.state.Bot.LogChatID = chat.ID
	s.state.Bot.AuthenticatedAt = time.Now().UTC().Unix()
	return s.saveLocked()
}

func (s *Store) IsOwner(userID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, id := range s.state.Bot.OwnerIDs {
		if id == userID {
			return true
		}
	}
	return false
}

func (s *Store) HasOwner() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.state.Bot.OwnerIDs) > 0
}

func (s *Store) LogChatID() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.Bot.LogChatID
}

func normalizeOwnerReportKind(kind string) string {
	return strings.ToLower(strings.TrimSpace(kind))
}

func ownerReportStateKey(kind string, chatID, userID int64) string {
	kind = normalizeOwnerReportKind(kind)
	if kind == "" || chatID == 0 || userID == 0 {
		return ""
	}
	return fmt.Sprintf("%s:%d:%d", kind, chatID, userID)
}

func (s *Store) OwnerReportLastSent(kind string, chatID, userID int64) time.Time {
	key := ownerReportStateKey(kind, chatID, userID)
	if key == "" {
		return time.Time{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.state.Bot.OwnerReports[key]
	if !ok || rec.LastSentUnix <= 0 {
		return time.Time{}
	}
	return time.Unix(rec.LastSentUnix, 0).UTC()
}

func (s *Store) SetOwnerReportLastSent(kind string, chatID, userID int64, now time.Time) error {
	kind = normalizeOwnerReportKind(kind)
	key := ownerReportStateKey(kind, chatID, userID)
	if key == "" {
		return errors.New("missing owner report state key")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.Bot.OwnerReports == nil {
		s.state.Bot.OwnerReports = make(map[string]OwnerReportState)
	}
	s.state.Bot.OwnerReports[key] = OwnerReportState{
		Kind:         kind,
		ChatID:       chatID,
		UserID:       userID,
		LastSentUnix: now.UTC().Unix(),
	}
	return s.saveLocked()
}

func (s *Store) LayaRuntimeState() LayaRuntimeState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.Bot.Laya
}

func (s *Store) SetLayaRuntimeState(state LayaRuntimeState) error {
	state = normalizeLayaRuntimeState(state)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Bot.Laya = state
	return s.saveLocked()
}

func (s *Store) LayaUserProfile(userID int64) (LayaUserProfile, bool) {
	if userID == 0 {
		return LayaUserProfile{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.state.Bot.LayaProfiles == nil {
		return LayaUserProfile{}, false
	}
	profile, ok := s.state.Bot.LayaProfiles[idKey(userID)]
	if !ok || profile.UserID == 0 {
		return LayaUserProfile{}, false
	}
	return normalizeLayaUserProfile(profile), true
}

func (s *Store) LayaUserProfiles() []LayaUserProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]LayaUserProfile, 0, len(s.state.Bot.LayaProfiles))
	for _, profile := range s.state.Bot.LayaProfiles {
		profile = normalizeLayaUserProfile(profile)
		if profile.UserID != 0 {
			out = append(out, profile)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DisplayName == out[j].DisplayName {
			return out[i].UserID < out[j].UserID
		}
		return out[i].DisplayName < out[j].DisplayName
	})
	return out
}

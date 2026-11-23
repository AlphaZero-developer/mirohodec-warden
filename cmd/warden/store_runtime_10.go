package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func (s *Store) RehabEscapeChance(chatID, userID int64) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ch := s.state.Chats[idKey(chatID)]
	if ch == nil {
		return defaultRehabEscapeChance
	}
	rec := ch.Members[idKey(userID)]
	return rehabEscapeChanceFromPenalty(rec.RehabEscapePenalty)
}

func (s *Store) RecordRehabEscapeSuccess(chatID, userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := s.ensureChatLocked(chatID)
	key := idKey(userID)
	member := ch.Members[key]
	if member.ID == 0 {
		member.ID = userID
		member.DisplayName = fmt.Sprintf("user %d", userID)
	}
	member.RehabEscapePenalty++
	ch.Members[key] = member
	if err := s.saveLocked(); err != nil {
		s.logError("store save rehab escape chance failed", err)
	}
}

func (s *Store) SetMute(chatID int64, rec ModerationRecord) {
	s.setRecord(chatID, rec, func(ch *ChatState) map[string]ModerationRecord { return ch.Muted })
}

func (s *Store) RemoveMute(chatID, userID int64) (ModerationRecord, bool) {
	return s.removeRecord(chatID, userID, func(ch *ChatState) map[string]ModerationRecord { return ch.Muted })
}

func (s *Store) ActiveMute(chatID, userID int64, now time.Time) (ModerationRecord, bool) {
	return s.activeRecord(chatID, userID, now, func(ch *ChatState) map[string]ModerationRecord { return ch.Muted })
}

func (s *Store) SetForwardRestriction(chatID int64, rec ModerationRecord) {
	s.setRecord(chatID, rec, func(ch *ChatState) map[string]ModerationRecord { return ch.ForwardBan })
}

func (s *Store) RemoveForwardRestriction(chatID, userID int64) (ModerationRecord, bool) {
	return s.removeRecord(chatID, userID, func(ch *ChatState) map[string]ModerationRecord { return ch.ForwardBan })
}

func (s *Store) ActiveForwardRestriction(chatID, userID int64, now time.Time) (ModerationRecord, bool) {
	return s.activeRecord(chatID, userID, now, func(ch *ChatState) map[string]ModerationRecord { return ch.ForwardBan })
}

func (s *Store) SetBan(chatID int64, rec ModerationRecord) {
	s.setRecord(chatID, rec, func(ch *ChatState) map[string]ModerationRecord { return ch.Banned })
}

func (s *Store) RemoveBan(chatID, userID int64) (ModerationRecord, bool) {
	return s.removeRecord(chatID, userID, func(ch *ChatState) map[string]ModerationRecord { return ch.Banned })
}

func (s *Store) Ban(chatID, userID int64) (ModerationRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ch := s.state.Chats[idKey(chatID)]
	if ch == nil {
		return ModerationRecord{}, false
	}
	rec, ok := ch.Banned[idKey(userID)]
	return rec, ok
}

func (s *Store) ModerationRecords(chatID int64, kind string, activeOnly bool, now time.Time) []ModerationRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ch := s.state.Chats[idKey(chatID)]
	if ch == nil {
		return nil
	}
	var source map[string]ModerationRecord
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "jail", "jailed", "rehab":
		source = ch.Jailed
	case "mute", "muted":
		source = ch.Muted
	case "forward", "forwards", "forwardban", "forward_ban":
		source = ch.ForwardBan
	case "ban", "banned", "banforever":
		source = ch.Banned
	default:
		return nil
	}
	out := make([]ModerationRecord, 0, len(source))
	for _, rec := range source {
		if activeOnly && kind != "ban" && !isActive(rec.UntilUnix, now) {
			continue
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SinceUnix > out[j].SinceUnix })
	return out
}

func (s *Store) setRecord(chatID int64, rec ModerationRecord, pick func(*ChatState) map[string]ModerationRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := s.ensureChatLocked(chatID)
	pick(ch)[idKey(rec.UserID)] = rec
	if err := s.saveLocked(); err != nil {
		s.logError("store save record failed", err)
	}
}

func (s *Store) removeRecord(chatID, userID int64, pick func(*ChatState) map[string]ModerationRecord) (ModerationRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := s.ensureChatLocked(chatID)
	rec, ok := pick(ch)[idKey(userID)]
	if ok {
		delete(pick(ch), idKey(userID))
		if err := s.saveLocked(); err != nil {
			s.logError("store remove record failed", err)
		}
	}
	return rec, ok
}

func (s *Store) activeRecord(chatID, userID int64, now time.Time, pick func(*ChatState) map[string]ModerationRecord) (ModerationRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ch := s.state.Chats[idKey(chatID)]
	if ch == nil {
		return ModerationRecord{}, false
	}
	rec, ok := pick(ch)[idKey(userID)]
	if !ok || !isActive(rec.UntilUnix, now) {
		return ModerationRecord{}, false
	}
	return rec, true
}

func (s *Store) CleanupExpired(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	removed := 0
	for _, ch := range s.state.Chats {
		for id, rec := range ch.Jailed {
			if !isActive(rec.UntilUnix, now) {
				delete(ch.Jailed, id)
				s.rememberLastRehabDurationLocked(ch, rec.UserID, rec, now.UTC())
				removed++
			}
		}
		for id, rec := range ch.Muted {
			if !isActive(rec.UntilUnix, now) {
				delete(ch.Muted, id)
				removed++
			}
		}
		for id, rec := range ch.ForwardBan {
			if !isActive(rec.UntilUnix, now) {
				delete(ch.ForwardBan, id)
				removed++
			}
		}
	}
	if removed > 0 {
		if err := s.saveLocked(); err != nil {
			s.logError("store cleanup failed", err)
		}
	}
	return removed
}

package main

import (
	"sort"
	"strconv"
	"time"
)

func normalizeJailRecord(rec ModerationRecord, now time.Time) ModerationRecord {
	if rec.SinceUnix == 0 {
		rec.SinceUnix = now.Unix()
	}
	if rec.StartedAtUnix == 0 {
		rec.StartedAtUnix = rec.SinceUnix
	}
	rec.RehabWard = normalizeRehabWard(rec.RehabWard)
	if rec.RehabWard == "" {
		rec.RehabWard = rehabWardQuiet
	}
	return rec
}

func (s *Store) rememberLastRehabDurationLocked(ch *ChatState, userID int64, rec ModerationRecord, endedAt time.Time) {
	if ch == nil || userID == 0 {
		return
	}
	duration := rehabTreatmentDuration(rec, endedAt)
	if duration <= 0 {
		return
	}
	if ch.Members == nil {
		ch.Members = make(map[string]MemberRecord)
	}
	key := idKey(userID)
	member := ch.Members[key]
	if member.ID == 0 {
		member.ID = userID
		member.Username = rec.Username
		member.DisplayName = rec.DisplayName
	}
	member.LastRehabDuration = int64(duration.Round(time.Second) / time.Second)
	ch.Members[key] = member
}

func (s *Store) AddWarning(chatID int64, rec WarningRecord) (WarningRecord, error) {
	now := time.Now().UTC()
	if rec.CreatedUnix == 0 {
		rec.CreatedUnix = now.Unix()
	}
	if rec.ID == "" {
		rec.ID = "warn_" + strconv.FormatInt(now.UnixNano(), 36)
	}
	rec.Active = true

	s.mu.Lock()
	defer s.mu.Unlock()

	ch := s.ensureChatLocked(chatID)
	base := rec.ID
	for i := 1; ; i++ {
		if _, exists := ch.Warnings[rec.ID]; !exists {
			break
		}
		rec.ID = base + "-" + strconv.Itoa(i)
	}
	ch.Warnings[rec.ID] = rec
	if err := s.saveLocked(); err != nil {
		return WarningRecord{}, err
	}
	return rec, nil
}

func (s *Store) Warnings(chatID, userID int64, includeInactive bool, limit int, now time.Time) []WarningRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ch := s.state.Chats[idKey(chatID)]
	if ch == nil {
		return nil
	}
	out := make([]WarningRecord, 0, len(ch.Warnings))
	for _, rec := range ch.Warnings {
		if userID != 0 && rec.UserID != userID {
			continue
		}
		if !includeInactive && !warningActive(rec, now) {
			continue
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedUnix > out[j].CreatedUnix })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (s *Store) ActiveWarningCount(chatID, userID int64, now time.Time) int {
	return len(s.Warnings(chatID, userID, false, 0, now))
}

func (s *Store) DeactivateWarning(chatID int64, warningID string) (WarningRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := s.ensureChatLocked(chatID)
	rec, ok := ch.Warnings[warningID]
	if !ok {
		return WarningRecord{}, false, nil
	}
	rec.Active = false
	ch.Warnings[warningID] = rec
	if err := s.saveLocked(); err != nil {
		return WarningRecord{}, false, err
	}
	return rec, true, nil
}

func (s *Store) ClearWarnings(chatID, userID int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := s.ensureChatLocked(chatID)
	changed := 0
	for id, rec := range ch.Warnings {
		if rec.UserID == userID && rec.Active {
			rec.Active = false
			ch.Warnings[id] = rec
			changed++
		}
	}
	if changed > 0 {
		if err := s.saveLocked(); err != nil {
			return 0, err
		}
	}
	return changed, nil
}

func (s *Store) Karma(chatID, userID int64) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ch := s.state.Chats[idKey(chatID)]
	if ch == nil {
		return 0
	}
	return ch.Karma[idKey(userID)].Score
}

func (s *Store) TopKarma(chatID int64, limit int) []KarmaTotal {
	if limit <= 0 {
		limit = 10
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	ch := s.state.Chats[idKey(chatID)]
	if ch == nil {
		return nil
	}
	out := make([]KarmaTotal, 0, len(ch.Karma))
	for _, rec := range ch.Karma {
		if rec.UserID != 0 {
			out = append(out, rec)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].UpdatedUnix > out[j].UpdatedUnix
		}
		return out[i].Score > out[j].Score
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

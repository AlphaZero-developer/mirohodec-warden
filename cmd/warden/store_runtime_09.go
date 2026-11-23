package main

import (
	"errors"
	"sort"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (s *Store) SetRehabReleaseExamRuntimeStateForChat(chatID int64, state RehabReleaseExamRuntimeState) error {
	if chatID == 0 {
		return errors.New("chat_id is required")
	}
	state = normalizeRehabReleaseExamRuntimeState(state)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.Bot.RehabReleaseExams == nil {
		s.state.Bot.RehabReleaseExams = make(map[string]RehabReleaseExamRuntimeState)
	}
	s.state.Bot.RehabReleaseExams[idKey(chatID)] = state
	return s.saveLocked()
}

func (s *Store) ReplyCipherRuntimeState() ReplyCipherRuntimeState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return normalizeReplyCipherRuntimeState(s.state.Bot.ReplyCipher)
}

func (s *Store) SetReplyCipherRuntimeState(state ReplyCipherRuntimeState) error {
	state = normalizeReplyCipherRuntimeState(state)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Bot.ReplyCipher = state
	return s.saveLocked()
}

func (s *Store) SetOwnerSleep(user *tele.User, aliases []string, now time.Time) (OwnerSleepState, error) {
	if user == nil || user.ID == 0 {
		return OwnerSleepState{}, errors.New("missing owner user")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state.Bot.Sleep == nil {
		s.state.Bot.Sleep = make(map[string]OwnerSleepState)
	}
	key := idKey(user.ID)
	prev := s.state.Bot.Sleep[key]
	if prev.SinceUnix == 0 {
		prev.SinceUnix = now.UTC().Unix()
	}
	prev.UserID = user.ID
	prev.Username = user.Username
	prev.DisplayName = displayNameFromUser(user)
	prev.Aliases = append([]string(nil), aliases...)
	s.state.Bot.Sleep[key] = prev
	return prev, s.saveLocked()
}

func (s *Store) ClearOwnerSleep(userID int64) (OwnerSleepState, bool, error) {
	if userID == 0 {
		return OwnerSleepState{}, false, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state.Bot.Sleep == nil {
		return OwnerSleepState{}, false, nil
	}
	key := idKey(userID)
	state, ok := s.state.Bot.Sleep[key]
	if !ok {
		return OwnerSleepState{}, false, nil
	}
	delete(s.state.Bot.Sleep, key)
	return state, true, s.saveLocked()
}

func (s *Store) OwnerSleep(userID int64) (OwnerSleepState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.state.Bot.Sleep[idKey(userID)]
	return copyOwnerSleepState(state), ok
}

func (s *Store) ActiveOwnerSleeps() []OwnerSleepState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]OwnerSleepState, 0, len(s.state.Bot.Sleep))
	for _, state := range s.state.Bot.Sleep {
		out = append(out, copyOwnerSleepState(state))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].SinceUnix < out[j].SinceUnix
	})
	return out
}

func (s *Store) AddOwnerSleepCall(userID int64, call OwnerSleepCall) (OwnerSleepState, bool, error) {
	if userID == 0 {
		return OwnerSleepState{}, false, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state.Bot.Sleep == nil {
		return OwnerSleepState{}, false, nil
	}
	key := idKey(userID)
	state, ok := s.state.Bot.Sleep[key]
	if !ok {
		return OwnerSleepState{}, false, nil
	}
	state.Calls = append(state.Calls, call)
	if len(state.Calls) > 100 {
		state.Calls = append([]OwnerSleepCall(nil), state.Calls[len(state.Calls)-100:]...)
	}
	s.state.Bot.Sleep[key] = state
	return copyOwnerSleepState(state), true, s.saveLocked()
}

func copyOwnerSleepState(state OwnerSleepState) OwnerSleepState {
	state.Aliases = append([]string(nil), state.Aliases...)
	state.Calls = append([]OwnerSleepCall(nil), state.Calls...)
	return state
}

type StoreCounts struct {
	KnownUsers   int
	ActiveJailed int
	ActiveMuted  int
	ForeverBans  int
}

func (s *Store) Counts(now time.Time) StoreCounts {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var counts StoreCounts
	for _, ch := range s.state.Chats {
		for _, rec := range ch.Members {
			if rec.ID != 0 && !rec.IsBot {
				counts.KnownUsers++
			}
		}
		for _, rec := range ch.Jailed {
			if isActive(rec.UntilUnix, now) {
				counts.ActiveJailed++
			}
		}
		for _, rec := range ch.Muted {
			if isActive(rec.UntilUnix, now) {
				counts.ActiveMuted++
			}
		}
		counts.ForeverBans += len(ch.Banned)
	}
	return counts
}

func (s *Store) SetJail(chatID int64, rec ModerationRecord) {
	rec = normalizeJailRecord(rec, time.Now().UTC())
	s.setRecord(chatID, rec, func(ch *ChatState) map[string]ModerationRecord { return ch.Jailed })
}

func (s *Store) RemoveJail(chatID, userID int64) (ModerationRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := s.ensureChatLocked(chatID)
	rec, ok := ch.Jailed[idKey(userID)]
	if ok {
		delete(ch.Jailed, idKey(userID))
		s.rememberLastRehabDurationLocked(ch, userID, rec, time.Now().UTC())
		if err := s.saveLocked(); err != nil {
			s.logError("store remove record failed", err)
		}
	}
	return rec, ok
}

func (s *Store) ActiveJail(chatID, userID int64, now time.Time) (ModerationRecord, bool) {
	return s.activeRecord(chatID, userID, now, func(ch *ChatState) map[string]ModerationRecord { return ch.Jailed })
}

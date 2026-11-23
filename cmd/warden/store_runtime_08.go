package main

import (
	"errors"
)

func (s *Store) SetLayaUserProfile(profile LayaUserProfile) error {
	profile = normalizeLayaUserProfile(profile)
	if profile.UserID == 0 {
		return errors.New("missing laya profile user id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.Bot.LayaProfiles == nil {
		s.state.Bot.LayaProfiles = make(map[string]LayaUserProfile)
	}
	s.state.Bot.LayaProfiles[idKey(profile.UserID)] = profile
	return s.saveLocked()
}

func (s *Store) ScriptwriterExemption(chatID, userID int64) (ScriptwriterExemption, bool) {
	if chatID == 0 || userID == 0 {
		return ScriptwriterExemption{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.state.Bot.Scriptwriters[scriptwriterExemptionKey(chatID, userID)]
	if !ok || rec.ChatID == 0 || rec.UserID == 0 {
		return ScriptwriterExemption{}, false
	}
	return normalizeScriptwriterExemption(rec), true
}

func (s *Store) SetScriptwriterExemption(rec ScriptwriterExemption) error {
	rec = normalizeScriptwriterExemption(rec)
	if rec.ChatID == 0 || rec.UserID == 0 {
		return errors.New("missing scriptwriter exemption chat or user id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.Bot.Scriptwriters == nil {
		s.state.Bot.Scriptwriters = make(map[string]ScriptwriterExemption)
	}
	s.state.Bot.Scriptwriters[scriptwriterExemptionKey(rec.ChatID, rec.UserID)] = rec
	return s.saveLocked()
}

func (s *Store) RemoveScriptwriterExemption(chatID, userID int64) (ScriptwriterExemption, bool, error) {
	if chatID == 0 || userID == 0 {
		return ScriptwriterExemption{}, false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := scriptwriterExemptionKey(chatID, userID)
	rec, ok := s.state.Bot.Scriptwriters[key]
	if ok {
		delete(s.state.Bot.Scriptwriters, key)
		if len(s.state.Bot.Scriptwriters) == 0 {
			s.state.Bot.Scriptwriters = nil
		}
	}
	if err := s.saveLocked(); err != nil {
		return ScriptwriterExemption{}, false, err
	}
	return normalizeScriptwriterExemption(rec), ok, nil
}

func (s *Store) LayaIgnoredUser(chatID, userID int64) (LayaIgnoredUser, bool) {
	if chatID == 0 || userID == 0 {
		return LayaIgnoredUser{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.state.Bot.LayaIgnored[layaIgnoredUserKey(chatID, userID)]
	if !ok || rec.ChatID == 0 || rec.UserID == 0 {
		return LayaIgnoredUser{}, false
	}
	return normalizeLayaIgnoredUser(rec), true
}

func (s *Store) SetLayaIgnoredUser(rec LayaIgnoredUser) error {
	rec = normalizeLayaIgnoredUser(rec)
	if rec.ChatID == 0 || rec.UserID == 0 {
		return errors.New("missing laya ignored user chat or user id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.Bot.LayaIgnored == nil {
		s.state.Bot.LayaIgnored = make(map[string]LayaIgnoredUser)
	}
	s.state.Bot.LayaIgnored[layaIgnoredUserKey(rec.ChatID, rec.UserID)] = rec
	return s.saveLocked()
}

func (s *Store) RemoveLayaIgnoredUser(chatID, userID int64) (LayaIgnoredUser, bool, error) {
	if chatID == 0 || userID == 0 {
		return LayaIgnoredUser{}, false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := layaIgnoredUserKey(chatID, userID)
	rec, ok := s.state.Bot.LayaIgnored[key]
	if ok {
		delete(s.state.Bot.LayaIgnored, key)
		if len(s.state.Bot.LayaIgnored) == 0 {
			s.state.Bot.LayaIgnored = nil
		}
	}
	if err := s.saveLocked(); err != nil {
		return LayaIgnoredUser{}, false, err
	}
	return normalizeLayaIgnoredUser(rec), ok, nil
}

func (s *Store) ContentRuntimeState() ContentRuntimeState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.Bot.Content
}

func (s *Store) SetContentRuntimeState(state ContentRuntimeState) error {
	state = normalizeContentRuntimeState(state)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Bot.Content = state
	return s.saveLocked()
}

func (s *Store) DiscussionRuntimeState() DiscussionRuntimeState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.Bot.Discussion
}

func (s *Store) SetDiscussionRuntimeState(state DiscussionRuntimeState) error {
	state = normalizeDiscussionRuntimeState(state)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Bot.Discussion = state
	return s.saveLocked()
}

func (s *Store) RehabBrainwashRuntimeState() RehabBrainwashRuntimeState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.Bot.RehabBrainwash
}

func (s *Store) SetRehabBrainwashRuntimeState(state RehabBrainwashRuntimeState) error {
	state = normalizeRehabBrainwashRuntimeState(state)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Bot.RehabBrainwash = state
	return s.saveLocked()
}

func (s *Store) RehabReleaseExamRuntimeState() RehabReleaseExamRuntimeState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state.Bot.RehabReleaseExam
}

func (s *Store) SetRehabReleaseExamRuntimeState(state RehabReleaseExamRuntimeState) error {
	state = normalizeRehabReleaseExamRuntimeState(state)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Bot.RehabReleaseExam = state
	return s.saveLocked()
}

func (s *Store) RehabReleaseExamRuntimeStateForChat(chatID int64) (RehabReleaseExamRuntimeState, bool) {
	if chatID == 0 {
		return RehabReleaseExamRuntimeState{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.state.Bot.RehabReleaseExams == nil {
		return RehabReleaseExamRuntimeState{}, false
	}
	state, ok := s.state.Bot.RehabReleaseExams[idKey(chatID)]
	if !ok || state.UpdatedUnix == 0 {
		return RehabReleaseExamRuntimeState{}, false
	}
	return normalizeRehabReleaseExamRuntimeState(state), true
}

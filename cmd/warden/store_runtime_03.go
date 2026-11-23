package main

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
)

func (s *Store) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

func (s *Store) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reloadLocked()
}

func (s *Store) ReloadIfChanged() (bool, error) {
	stat, err := os.Stat(s.path)
	if err != nil {
		return false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !stat.ModTime().After(s.modTime) {
		return false, nil
	}
	if err := s.reloadLocked(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) reloadLocked() error {
	oldManagedChats := copyManagedChats(s.state.ManagedChats)
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		s.state = State{
			Chats:        make(map[string]*ChatState),
			ManagedChats: make(map[string]ChatSettings),
			Mafia: MafiaState{
				RegisteredUsers: make(map[string]MafiaRegisteredUser),
				Rooms:           make(map[string]MafiaRoom),
			},
		}
		if s.keepPersistedRehabTopicsLocked(oldManagedChats) {
			if err := s.saveLocked(); err != nil {
				return err
			}
		}
		if stat, err := os.Stat(s.path); err == nil {
			s.modTime = stat.ModTime()
		}
		return nil
	}

	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return err
	}
	if state.Chats == nil {
		state.Chats = make(map[string]*ChatState)
	}
	if state.ManagedChats == nil {
		state.ManagedChats = make(map[string]ChatSettings)
	}
	s.state = state
	keptTopics := s.keepPersistedRehabTopicsLocked(oldManagedChats)
	if s.migrateStateLocked(time.Now().UTC()) {
		keptTopics = true
	}
	if keptTopics {
		if err := s.saveLocked(); err != nil {
			return err
		}
	}
	if stat, err := os.Stat(s.path); err == nil {
		s.modTime = stat.ModTime()
	}
	return nil
}

func copyManagedChats(in map[string]ChatSettings) map[string]ChatSettings {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]ChatSettings, len(in))
	for key, settings := range in {
		out[key] = settings
	}
	return out
}

func (s *Store) keepPersistedRehabTopicsLocked(oldManagedChats map[string]ChatSettings) bool {
	if len(oldManagedChats) == 0 {
		return false
	}
	if s.state.ManagedChats == nil {
		s.state.ManagedChats = make(map[string]ChatSettings)
	}
	changed := false
	for key, old := range oldManagedChats {
		topicID := old.PersistedRehabTopicID
		if topicID == 0 {
			topicID = old.RehabTopicID
		}
		if topicID == 0 {
			continue
		}
		current := s.state.ManagedChats[key]
		if current.ChatID == 0 {
			current = old
		}
		if current.RehabTopicID == 0 && current.PersistedRehabTopicID == 0 {
			current.RehabTopicID = topicID
			current.PersistedRehabTopicID = topicID
			s.state.ManagedChats[key] = current
			changed = true
			if s.log != nil {
				s.log.Info("rehab topic kept from state", zap.Int64("chat", current.ChatID), zap.Int("topic", topicID))
			}
		}
	}
	return changed
}

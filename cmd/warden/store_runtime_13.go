package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func NewStore(path string, logger *AsyncLogger) (*Store, error) {
	s := &Store{
		path: path,
		log:  logger,
		state: State{
			Chats:        make(map[string]*ChatState),
			ManagedChats: make(map[string]ChatSettings),
			Mafia: MafiaState{
				RegisteredUsers: make(map[string]MafiaRegisteredUser),
				Rooms:           make(map[string]MafiaRoom),
			},
		},
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := s.saveLocked(); err != nil {
			return nil, err
		}
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		if stat, err := os.Stat(path); err == nil {
			s.modTime = stat.ModTime()
		}
		return s, nil
	}
	if err := json.Unmarshal(raw, &s.state); err != nil {
		return nil, err
	}
	if s.state.Chats == nil {
		s.state.Chats = make(map[string]*ChatState)
	}
	if s.state.ManagedChats == nil {
		s.state.ManagedChats = make(map[string]ChatSettings)
	}
	if s.migrateStateLocked(time.Now().UTC()) {
		if err := s.saveLocked(); err != nil {
			return nil, err
		}
	}
	if stat, err := os.Stat(path); err == nil {
		s.modTime = stat.ModTime()
	}
	return s, nil
}

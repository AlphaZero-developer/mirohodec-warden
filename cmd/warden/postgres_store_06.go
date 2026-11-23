package main

import (
	"encoding/json"
	"errors"
)

func (s *PostgresStore) DiscussionRuntimeState() DiscussionRuntimeState {
	var raw []byte
	err := s.db.QueryRow(`SELECT value FROM bot_settings WHERE key='discussion_runtime'`).Scan(&raw)
	if err != nil {
		return DiscussionRuntimeState{}
	}
	var state DiscussionRuntimeState
	if err := json.Unmarshal(raw, &state); err != nil {
		s.logError("postgres discussion runtime state decode failed", err)
		return DiscussionRuntimeState{}
	}
	return normalizeDiscussionRuntimeState(state)
}

func (s *PostgresStore) SetDiscussionRuntimeState(state DiscussionRuntimeState) error {
	state = normalizeDiscussionRuntimeState(state)
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO bot_settings(key,value,updated_unix)
		VALUES('discussion_runtime',$1::jsonb,$2)
		ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value, updated_unix=EXCLUDED.updated_unix`,
		string(raw), state.UpdatedUnix)
	if err != nil {
		s.logError("postgres save discussion runtime state failed", err)
	}
	return err
}

func (s *PostgresStore) RehabBrainwashRuntimeState() RehabBrainwashRuntimeState {
	var raw []byte
	err := s.db.QueryRow(`SELECT value FROM bot_settings WHERE key='rehab_brainwash'`).Scan(&raw)
	if err != nil {
		return RehabBrainwashRuntimeState{}
	}
	var state RehabBrainwashRuntimeState
	if err := json.Unmarshal(raw, &state); err != nil {
		s.logError("postgres rehab brainwash runtime state decode failed", err)
		return RehabBrainwashRuntimeState{}
	}
	return normalizeRehabBrainwashRuntimeState(state)
}

func (s *PostgresStore) SetRehabBrainwashRuntimeState(state RehabBrainwashRuntimeState) error {
	state = normalizeRehabBrainwashRuntimeState(state)
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO bot_settings(key,value,updated_unix)
		VALUES('rehab_brainwash',$1::jsonb,$2)
		ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value, updated_unix=EXCLUDED.updated_unix`,
		string(raw), state.UpdatedUnix)
	if err != nil {
		s.logError("postgres save rehab brainwash runtime state failed", err)
	}
	return err
}

func (s *PostgresStore) RehabReleaseExamRuntimeState() RehabReleaseExamRuntimeState {
	var raw []byte
	err := s.db.QueryRow(`SELECT value FROM bot_settings WHERE key='rehab_release_exam'`).Scan(&raw)
	if err != nil {
		return RehabReleaseExamRuntimeState{}
	}
	var state RehabReleaseExamRuntimeState
	if err := json.Unmarshal(raw, &state); err != nil {
		s.logError("postgres rehab release exam runtime state decode failed", err)
		return RehabReleaseExamRuntimeState{}
	}
	return normalizeRehabReleaseExamRuntimeState(state)
}

func (s *PostgresStore) SetRehabReleaseExamRuntimeState(state RehabReleaseExamRuntimeState) error {
	state = normalizeRehabReleaseExamRuntimeState(state)
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO bot_settings(key,value,updated_unix)
		VALUES('rehab_release_exam',$1::jsonb,$2)
		ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value, updated_unix=EXCLUDED.updated_unix`,
		string(raw), state.UpdatedUnix)
	if err != nil {
		s.logError("postgres save rehab release exam runtime state failed", err)
	}
	return err
}

func (s *PostgresStore) RehabReleaseExamRuntimeStateForChat(chatID int64) (RehabReleaseExamRuntimeState, bool) {
	if chatID == 0 {
		return RehabReleaseExamRuntimeState{}, false
	}
	records := s.rehabReleaseExamRuntimeStatesByChat()
	state, ok := records[idKey(chatID)]
	if !ok || state.UpdatedUnix == 0 {
		return RehabReleaseExamRuntimeState{}, false
	}
	return normalizeRehabReleaseExamRuntimeState(state), true
}

func (s *PostgresStore) SetRehabReleaseExamRuntimeStateForChat(chatID int64, state RehabReleaseExamRuntimeState) error {
	if chatID == 0 {
		return errors.New("chat_id is required")
	}
	state = normalizeRehabReleaseExamRuntimeState(state)
	records := s.rehabReleaseExamRuntimeStatesByChat()
	records[idKey(chatID)] = state
	raw, err := json.Marshal(records)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO bot_settings(key,value,updated_unix)
		VALUES('rehab_release_exam_by_chat',$1::jsonb,$2)
		ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value, updated_unix=EXCLUDED.updated_unix`,
		string(raw), state.UpdatedUnix)
	if err != nil {
		s.logError("postgres save rehab release exam by chat failed", err)
	}
	return err
}

func (s *PostgresStore) rehabReleaseExamRuntimeStatesByChat() map[string]RehabReleaseExamRuntimeState {
	var raw []byte
	err := s.db.QueryRow(`SELECT value FROM bot_settings WHERE key='rehab_release_exam_by_chat'`).Scan(&raw)
	if err != nil {
		return make(map[string]RehabReleaseExamRuntimeState)
	}
	var records map[string]RehabReleaseExamRuntimeState
	if err := json.Unmarshal(raw, &records); err != nil {
		s.logError("postgres rehab release exam by chat decode failed", err)
		return make(map[string]RehabReleaseExamRuntimeState)
	}
	if records == nil {
		records = make(map[string]RehabReleaseExamRuntimeState)
	}
	for key, state := range records {
		records[key] = normalizeRehabReleaseExamRuntimeState(state)
	}
	return records
}

func (s *PostgresStore) ReplyCipherRuntimeState() ReplyCipherRuntimeState {
	var raw []byte
	err := s.db.QueryRow(`SELECT value FROM bot_settings WHERE key='reply_cipher_runtime'`).Scan(&raw)
	if err != nil {
		return ReplyCipherRuntimeState{}
	}
	var state ReplyCipherRuntimeState
	if err := json.Unmarshal(raw, &state); err != nil {
		s.logError("postgres reply cipher runtime state decode failed", err)
		return ReplyCipherRuntimeState{}
	}
	return normalizeReplyCipherRuntimeState(state)
}

func (s *PostgresStore) SetReplyCipherRuntimeState(state ReplyCipherRuntimeState) error {
	state = normalizeReplyCipherRuntimeState(state)
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO bot_settings(key,value,updated_unix)
		VALUES('reply_cipher_runtime',$1::jsonb,$2)
		ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value, updated_unix=EXCLUDED.updated_unix`,
		string(raw), state.UpdatedUnix)
	if err != nil {
		s.logError("postgres save reply cipher runtime state failed", err)
	}
	return err
}

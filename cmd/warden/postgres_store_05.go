package main

import (
	"encoding/json"
	"errors"
	"time"
)

func (s *PostgresStore) ScriptwriterExemption(chatID, userID int64) (ScriptwriterExemption, bool) {
	if chatID == 0 || userID == 0 {
		return ScriptwriterExemption{}, false
	}
	records := s.scriptwriterExemptions()
	rec, ok := records[scriptwriterExemptionKey(chatID, userID)]
	if !ok || rec.ChatID == 0 || rec.UserID == 0 {
		return ScriptwriterExemption{}, false
	}
	return normalizeScriptwriterExemption(rec), true
}

func (s *PostgresStore) SetScriptwriterExemption(rec ScriptwriterExemption) error {
	rec = normalizeScriptwriterExemption(rec)
	if rec.ChatID == 0 || rec.UserID == 0 {
		return errors.New("missing scriptwriter exemption chat or user id")
	}
	records := s.scriptwriterExemptions()
	records[scriptwriterExemptionKey(rec.ChatID, rec.UserID)] = rec
	return s.saveScriptwriterExemptions(records, rec.AddedUnix)
}

func (s *PostgresStore) RemoveScriptwriterExemption(chatID, userID int64) (ScriptwriterExemption, bool, error) {
	if chatID == 0 || userID == 0 {
		return ScriptwriterExemption{}, false, nil
	}
	records := s.scriptwriterExemptions()
	key := scriptwriterExemptionKey(chatID, userID)
	rec, ok := records[key]
	if ok {
		delete(records, key)
	}
	if err := s.saveScriptwriterExemptions(records, time.Now().UTC().Unix()); err != nil {
		return ScriptwriterExemption{}, false, err
	}
	return normalizeScriptwriterExemption(rec), ok, nil
}

func (s *PostgresStore) scriptwriterExemptions() map[string]ScriptwriterExemption {
	var raw []byte
	err := s.db.QueryRow(`SELECT value FROM bot_settings WHERE key='scriptwriter_exemptions'`).Scan(&raw)
	if err != nil {
		return make(map[string]ScriptwriterExemption)
	}
	var records map[string]ScriptwriterExemption
	if err := json.Unmarshal(raw, &records); err != nil {
		s.logError("postgres scriptwriter exemptions decode failed", err)
		return make(map[string]ScriptwriterExemption)
	}
	if records == nil {
		records = make(map[string]ScriptwriterExemption)
	}
	for key, rec := range records {
		records[key] = normalizeScriptwriterExemption(rec)
	}
	return records
}

func (s *PostgresStore) saveScriptwriterExemptions(records map[string]ScriptwriterExemption, updatedUnix int64) error {
	if records == nil {
		records = make(map[string]ScriptwriterExemption)
	}
	raw, err := json.Marshal(records)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO bot_settings(key,value,updated_unix)
		VALUES('scriptwriter_exemptions',$1::jsonb,$2)
		ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value, updated_unix=EXCLUDED.updated_unix`,
		string(raw), updatedUnix)
	if err != nil {
		s.logError("postgres save scriptwriter exemptions failed", err)
	}
	return err
}

func (s *PostgresStore) LayaIgnoredUser(chatID, userID int64) (LayaIgnoredUser, bool) {
	if chatID == 0 || userID == 0 {
		return LayaIgnoredUser{}, false
	}
	records := s.layaIgnoredUsers()
	rec, ok := records[layaIgnoredUserKey(chatID, userID)]
	if !ok || rec.ChatID == 0 || rec.UserID == 0 {
		return LayaIgnoredUser{}, false
	}
	return normalizeLayaIgnoredUser(rec), true
}

func (s *PostgresStore) SetLayaIgnoredUser(rec LayaIgnoredUser) error {
	rec = normalizeLayaIgnoredUser(rec)
	if rec.ChatID == 0 || rec.UserID == 0 {
		return errors.New("missing laya ignored user chat or user id")
	}
	records := s.layaIgnoredUsers()
	records[layaIgnoredUserKey(rec.ChatID, rec.UserID)] = rec
	return s.saveLayaIgnoredUsers(records, rec.AddedUnix)
}

func (s *PostgresStore) RemoveLayaIgnoredUser(chatID, userID int64) (LayaIgnoredUser, bool, error) {
	if chatID == 0 || userID == 0 {
		return LayaIgnoredUser{}, false, nil
	}
	records := s.layaIgnoredUsers()
	key := layaIgnoredUserKey(chatID, userID)
	rec, ok := records[key]
	if ok {
		delete(records, key)
	}
	if err := s.saveLayaIgnoredUsers(records, time.Now().UTC().Unix()); err != nil {
		return LayaIgnoredUser{}, false, err
	}
	return normalizeLayaIgnoredUser(rec), ok, nil
}

func (s *PostgresStore) layaIgnoredUsers() map[string]LayaIgnoredUser {
	var raw []byte
	err := s.db.QueryRow(`SELECT value FROM bot_settings WHERE key='laya_ignored_users'`).Scan(&raw)
	if err != nil {
		return make(map[string]LayaIgnoredUser)
	}
	var records map[string]LayaIgnoredUser
	if err := json.Unmarshal(raw, &records); err != nil {
		s.logError("postgres laya ignored users decode failed", err)
		return make(map[string]LayaIgnoredUser)
	}
	if records == nil {
		records = make(map[string]LayaIgnoredUser)
	}
	for key, rec := range records {
		records[key] = normalizeLayaIgnoredUser(rec)
	}
	return records
}

func (s *PostgresStore) saveLayaIgnoredUsers(records map[string]LayaIgnoredUser, updatedUnix int64) error {
	if records == nil {
		records = make(map[string]LayaIgnoredUser)
	}
	raw, err := json.Marshal(records)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO bot_settings(key,value,updated_unix)
		VALUES('laya_ignored_users',$1::jsonb,$2)
		ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value, updated_unix=EXCLUDED.updated_unix`,
		string(raw), updatedUnix)
	if err != nil {
		s.logError("postgres save laya ignored users failed", err)
	}
	return err
}

func (s *PostgresStore) ContentRuntimeState() ContentRuntimeState {
	var raw []byte
	err := s.db.QueryRow(`SELECT value FROM bot_settings WHERE key='content_runtime'`).Scan(&raw)
	if err != nil {
		return ContentRuntimeState{}
	}
	var state ContentRuntimeState
	if err := json.Unmarshal(raw, &state); err != nil {
		s.logError("postgres content runtime state decode failed", err)
		return ContentRuntimeState{}
	}
	return normalizeContentRuntimeState(state)
}

func (s *PostgresStore) SetContentRuntimeState(state ContentRuntimeState) error {
	state = normalizeContentRuntimeState(state)
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO bot_settings(key,value,updated_unix)
		VALUES('content_runtime',$1::jsonb,$2)
		ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value, updated_unix=EXCLUDED.updated_unix`,
		string(raw), state.UpdatedUnix)
	if err != nil {
		s.logError("postgres save content runtime state failed", err)
	}
	return err
}

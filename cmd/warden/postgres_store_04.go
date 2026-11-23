package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"time"
)

func (s *PostgresStore) LogChatID() int64 {
	var id int64
	err := s.db.QueryRow(`SELECT log_chat_id FROM owners WHERE log_chat_id<>0 ORDER BY authenticated_at DESC LIMIT 1`).Scan(&id)
	if err != nil {
		return 0
	}
	return id
}

func (s *PostgresStore) OwnerReportLastSent(kind string, chatID, userID int64) time.Time {
	key := ownerReportStateKey(kind, chatID, userID)
	if key == "" {
		return time.Time{}
	}
	records, err := s.ownerReportStates()
	if err != nil {
		s.logError("postgres owner report state lookup failed", err)
		return time.Time{}
	}
	rec, ok := records[key]
	if !ok || rec.LastSentUnix <= 0 {
		return time.Time{}
	}
	return time.Unix(rec.LastSentUnix, 0).UTC()
}

func (s *PostgresStore) SetOwnerReportLastSent(kind string, chatID, userID int64, now time.Time) error {
	kind = normalizeOwnerReportKind(kind)
	key := ownerReportStateKey(kind, chatID, userID)
	if key == "" {
		return errors.New("missing owner report state key")
	}
	records, err := s.ownerReportStates()
	if err != nil {
		return err
	}
	records[key] = OwnerReportState{
		Kind:         kind,
		ChatID:       chatID,
		UserID:       userID,
		LastSentUnix: now.UTC().Unix(),
	}
	raw, err := json.Marshal(records)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO bot_settings(key,value,updated_unix)
		VALUES('owner_reports',$1::jsonb,$2)
		ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value, updated_unix=EXCLUDED.updated_unix`,
		string(raw), now.UTC().Unix())
	if err != nil {
		s.logError("postgres save owner report state failed", err)
	}
	return err
}

func (s *PostgresStore) ownerReportStates() (map[string]OwnerReportState, error) {
	var raw []byte
	err := s.db.QueryRow(`SELECT value FROM bot_settings WHERE key='owner_reports'`).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return map[string]OwnerReportState{}, nil
	}
	if err != nil {
		return nil, err
	}
	var records map[string]OwnerReportState
	if err := json.Unmarshal(raw, &records); err != nil {
		return nil, err
	}
	if records == nil {
		records = map[string]OwnerReportState{}
	}
	return records, nil
}

func (s *PostgresStore) LayaRuntimeState() LayaRuntimeState {
	var raw []byte
	err := s.db.QueryRow(`SELECT value FROM bot_settings WHERE key='laya_runtime'`).Scan(&raw)
	if err != nil {
		return LayaRuntimeState{}
	}
	var state LayaRuntimeState
	if err := json.Unmarshal(raw, &state); err != nil {
		s.logError("postgres laya runtime state decode failed", err)
		return LayaRuntimeState{}
	}
	return normalizeLayaRuntimeState(state)
}

func (s *PostgresStore) SetLayaRuntimeState(state LayaRuntimeState) error {
	state = normalizeLayaRuntimeState(state)
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO bot_settings(key,value,updated_unix)
		VALUES('laya_runtime',$1::jsonb,$2)
		ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value, updated_unix=EXCLUDED.updated_unix`,
		string(raw), state.UpdatedUnix)
	if err != nil {
		s.logError("postgres save laya runtime state failed", err)
	}
	return err
}

func (s *PostgresStore) LayaUserProfile(userID int64) (LayaUserProfile, bool) {
	if userID == 0 {
		return LayaUserProfile{}, false
	}
	profiles := s.layaUserProfiles()
	profile, ok := profiles[idKey(userID)]
	if !ok || profile.UserID == 0 {
		return LayaUserProfile{}, false
	}
	return normalizeLayaUserProfile(profile), true
}

func (s *PostgresStore) LayaUserProfiles() []LayaUserProfile {
	profiles := s.layaUserProfiles()
	out := make([]LayaUserProfile, 0, len(profiles))
	for _, profile := range profiles {
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

func (s *PostgresStore) SetLayaUserProfile(profile LayaUserProfile) error {
	profile = normalizeLayaUserProfile(profile)
	if profile.UserID == 0 {
		return errors.New("missing laya profile user id")
	}
	profiles := s.layaUserProfiles()
	profiles[idKey(profile.UserID)] = profile
	raw, err := json.Marshal(profiles)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO bot_settings(key,value,updated_unix)
		VALUES('laya_profiles',$1::jsonb,$2)
		ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value, updated_unix=EXCLUDED.updated_unix`,
		string(raw), profile.UpdatedUnix)
	if err != nil {
		s.logError("postgres save laya user profile failed", err)
	}
	return err
}

func (s *PostgresStore) layaUserProfiles() map[string]LayaUserProfile {
	var raw []byte
	err := s.db.QueryRow(`SELECT value FROM bot_settings WHERE key='laya_profiles'`).Scan(&raw)
	if err != nil {
		return make(map[string]LayaUserProfile)
	}
	var profiles map[string]LayaUserProfile
	if err := json.Unmarshal(raw, &profiles); err != nil {
		s.logError("postgres laya profiles decode failed", err)
		return make(map[string]LayaUserProfile)
	}
	if profiles == nil {
		profiles = make(map[string]LayaUserProfile)
	}
	for key, profile := range profiles {
		profiles[key] = normalizeLayaUserProfile(profile)
	}
	return profiles
}

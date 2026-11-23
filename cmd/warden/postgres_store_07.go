package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/lib/pq"
	tele "gopkg.in/telebot.v3"
)

func (s *PostgresStore) SetOwnerSleep(user *tele.User, aliases []string, now time.Time) (OwnerSleepState, error) {
	if user == nil || user.ID == 0 {
		return OwnerSleepState{}, errors.New("missing owner user")
	}
	state := OwnerSleepState{
		UserID:      user.ID,
		Username:    user.Username,
		DisplayName: displayNameFromUser(user),
		Aliases:     append([]string(nil), aliases...),
		SinceUnix:   now.UTC().Unix(),
	}
	var prevSince int64
	var prevCalls []byte
	err := s.db.QueryRow(`SELECT since_unix, calls FROM owner_sleep WHERE user_id=$1`, user.ID).Scan(&prevSince, &prevCalls)
	if err == nil {
		if prevSince != 0 {
			state.SinceUnix = prevSince
		}
		_ = json.Unmarshal(prevCalls, &state.Calls)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return OwnerSleepState{}, err
	}
	calls, err := json.Marshal(state.Calls)
	if err != nil {
		return OwnerSleepState{}, err
	}
	_, err = s.db.Exec(`INSERT INTO owner_sleep(user_id, username, display_name, aliases, since_unix, calls)
		VALUES($1,$2,$3,$4,$5,$6::jsonb)
		ON CONFLICT(user_id) DO UPDATE SET username=EXCLUDED.username, display_name=EXCLUDED.display_name, aliases=EXCLUDED.aliases, since_unix=owner_sleep.since_unix, calls=owner_sleep.calls`,
		state.UserID, state.Username, state.DisplayName, pq.Array(state.Aliases), state.SinceUnix, string(calls))
	return copyOwnerSleepState(state), err
}

func (s *PostgresStore) ClearOwnerSleep(userID int64) (OwnerSleepState, bool, error) {
	var state OwnerSleepState
	var calls []byte
	err := s.db.QueryRow(`DELETE FROM owner_sleep WHERE user_id=$1 RETURNING user_id, username, display_name, aliases, since_unix, calls`, userID).
		Scan(&state.UserID, &state.Username, &state.DisplayName, pq.Array(&state.Aliases), &state.SinceUnix, &calls)
	if errors.Is(err, sql.ErrNoRows) {
		return OwnerSleepState{}, false, nil
	}
	if err != nil {
		return OwnerSleepState{}, false, err
	}
	_ = json.Unmarshal(calls, &state.Calls)
	return state, true, nil
}

func (s *PostgresStore) OwnerSleep(userID int64) (OwnerSleepState, bool) {
	state, ok, err := s.ownerSleepByID(userID)
	if err != nil {
		s.logError("postgres owner sleep lookup failed", err)
		return OwnerSleepState{}, false
	}
	return state, ok
}

func (s *PostgresStore) ActiveOwnerSleeps() []OwnerSleepState {
	rows, err := s.db.Query(`SELECT user_id, username, display_name, aliases, since_unix, calls FROM owner_sleep ORDER BY since_unix`)
	if err != nil {
		s.logError("postgres owner sleep list failed", err)
		return nil
	}
	defer rows.Close()
	var out []OwnerSleepState
	for rows.Next() {
		state, ok := scanOwnerSleep(rows)
		if ok {
			out = append(out, state)
		}
	}
	return out
}

func (s *PostgresStore) AddOwnerSleepCall(userID int64, call OwnerSleepCall) (OwnerSleepState, bool, error) {
	state, ok, err := s.ownerSleepByID(userID)
	if err != nil || !ok {
		return state, ok, err
	}
	state.Calls = append(state.Calls, call)
	if len(state.Calls) > 100 {
		state.Calls = append([]OwnerSleepCall(nil), state.Calls[len(state.Calls)-100:]...)
	}
	calls, err := json.Marshal(state.Calls)
	if err != nil {
		return OwnerSleepState{}, false, err
	}
	_, err = s.db.Exec(`UPDATE owner_sleep SET calls=$2::jsonb WHERE user_id=$1`, userID, string(calls))
	return state, true, err
}

func (s *PostgresStore) ownerSleepByID(userID int64) (OwnerSleepState, bool, error) {
	row := s.db.QueryRow(`SELECT user_id, username, display_name, aliases, since_unix, calls FROM owner_sleep WHERE user_id=$1`, userID)
	state, ok, err := scanOwnerSleepRow(row)
	return state, ok, err
}

func (s *PostgresStore) Counts(now time.Time) StoreCounts {
	var c StoreCounts
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM members WHERE is_bot=false`).Scan(&c.KnownUsers)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM moderation_records WHERE kind='jail' AND (until_unix=0 OR until_unix>$1)`, now.Unix()).Scan(&c.ActiveJailed)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM moderation_records WHERE kind='mute' AND (until_unix=0 OR until_unix>$1)`, now.Unix()).Scan(&c.ActiveMuted)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM moderation_records WHERE kind='ban'`).Scan(&c.ForeverBans)
	return c
}

func (s *PostgresStore) SetJail(chatID int64, rec ModerationRecord) {
	rec = normalizeJailRecord(rec, time.Now().UTC())
	s.setModerationRecord(chatID, "jail", rec)
}

func (s *PostgresStore) SetMute(chatID int64, rec ModerationRecord) {
	s.setModerationRecord(chatID, "mute", rec)
}

func (s *PostgresStore) SetForwardRestriction(chatID int64, rec ModerationRecord) {
	s.setModerationRecord(chatID, "forward", rec)
}

func (s *PostgresStore) SetBan(chatID int64, rec ModerationRecord) {
	s.setModerationRecord(chatID, "ban", rec)
}

func (s *PostgresStore) RemoveJail(chatID, userID int64) (ModerationRecord, bool) {
	return s.removeModerationRecord(chatID, userID, "jail")
}

func (s *PostgresStore) RemoveMute(chatID, userID int64) (ModerationRecord, bool) {
	return s.removeModerationRecord(chatID, userID, "mute")
}

func (s *PostgresStore) RemoveForwardRestriction(chatID, userID int64) (ModerationRecord, bool) {
	return s.removeModerationRecord(chatID, userID, "forward")
}

func (s *PostgresStore) RemoveBan(chatID, userID int64) (ModerationRecord, bool) {
	return s.removeModerationRecord(chatID, userID, "ban")
}

func (s *PostgresStore) ActiveJail(chatID, userID int64, now time.Time) (ModerationRecord, bool) {
	rec, ok := s.moderationRecord(chatID, userID, "jail")
	return rec, ok && isActive(rec.UntilUnix, now)
}

func (s *PostgresStore) RehabEscapeChance(chatID, userID int64) int {
	var penalty int
	err := s.db.QueryRow(`SELECT rehab_escape_penalty FROM members WHERE chat_id=$1 AND user_id=$2`, chatID, userID).Scan(&penalty)
	if err != nil {
		return defaultRehabEscapeChance
	}
	return rehabEscapeChanceFromPenalty(penalty)
}

func (s *PostgresStore) RecordRehabEscapeSuccess(chatID, userID int64) {
	_, err := s.db.Exec(`INSERT INTO members(chat_id,user_id,rehab_escape_penalty)
		VALUES($1,$2,1)
		ON CONFLICT(chat_id,user_id) DO UPDATE SET rehab_escape_penalty=members.rehab_escape_penalty+1`, chatID, userID)
	if err != nil {
		s.logError("postgres save rehab escape chance failed", err)
	}
}

func (s *PostgresStore) ActiveMute(chatID, userID int64, now time.Time) (ModerationRecord, bool) {
	rec, ok := s.moderationRecord(chatID, userID, "mute")
	return rec, ok && isActive(rec.UntilUnix, now)
}

func (s *PostgresStore) ActiveForwardRestriction(chatID, userID int64, now time.Time) (ModerationRecord, bool) {
	rec, ok := s.moderationRecord(chatID, userID, "forward")
	return rec, ok && isActive(rec.UntilUnix, now)
}

func (s *PostgresStore) Ban(chatID, userID int64) (ModerationRecord, bool) {
	return s.moderationRecord(chatID, userID, "ban")
}

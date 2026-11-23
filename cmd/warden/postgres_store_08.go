package main

import (
	"database/sql"
	"errors"
	"strconv"
	"time"
)

func (s *PostgresStore) setModerationRecord(chatID int64, kind string, rec ModerationRecord) {
	if kind == "jail" {
		rec = normalizeJailRecord(rec, time.Now().UTC())
	}
	_, err := s.db.Exec(`INSERT INTO moderation_records(chat_id,user_id,kind,username,display_name,reason,rehab_ward,by_admin_id,by_admin_name,since_unix,started_at,until_unix)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT(chat_id,user_id,kind) DO UPDATE SET username=EXCLUDED.username, display_name=EXCLUDED.display_name, reason=EXCLUDED.reason, rehab_ward=EXCLUDED.rehab_ward, by_admin_id=EXCLUDED.by_admin_id, by_admin_name=EXCLUDED.by_admin_name, since_unix=EXCLUDED.since_unix, started_at=EXCLUDED.started_at, until_unix=EXCLUDED.until_unix`,
		chatID, rec.UserID, kind, rec.Username, rec.DisplayName, rec.Reason, rec.RehabWard, rec.ByAdminID, rec.ByAdminName, rec.SinceUnix, rec.StartedAtUnix, rec.UntilUnix)
	if err != nil {
		s.logError("postgres save moderation record failed", err)
	}
}

func (s *PostgresStore) moderationRecord(chatID, userID int64, kind string) (ModerationRecord, bool) {
	var rec ModerationRecord
	err := s.db.QueryRow(`SELECT user_id, username, display_name, reason, rehab_ward, by_admin_id, by_admin_name, since_unix, started_at, until_unix
		FROM moderation_records WHERE chat_id=$1 AND user_id=$2 AND kind=$3`, chatID, userID, kind).Scan(&rec.UserID, &rec.Username, &rec.DisplayName, &rec.Reason, &rec.RehabWard, &rec.ByAdminID, &rec.ByAdminName, &rec.SinceUnix, &rec.StartedAtUnix, &rec.UntilUnix)
	if err != nil {
		return ModerationRecord{}, false
	}
	if kind == "jail" {
		rec = normalizeJailRecord(rec, time.Now().UTC())
	}
	return rec, true
}

func (s *PostgresStore) removeModerationRecord(chatID, userID int64, kind string) (ModerationRecord, bool) {
	rec, ok := s.moderationRecord(chatID, userID, kind)
	if !ok {
		return ModerationRecord{}, false
	}
	if kind == "jail" {
		s.rememberLastRehabDuration(chatID, userID, rec, time.Now().UTC())
	}
	_, _ = s.db.Exec(`DELETE FROM moderation_records WHERE chat_id=$1 AND user_id=$2 AND kind=$3`, chatID, userID, kind)
	return rec, true
}

func (s *PostgresStore) ModerationRecords(chatID int64, kind string, activeOnly bool, now time.Time) []ModerationRecord {
	kind = moderationKind(kind)
	if kind == "" {
		return nil
	}
	query := `SELECT user_id, username, display_name, reason, rehab_ward, by_admin_id, by_admin_name, since_unix, started_at, until_unix FROM moderation_records WHERE chat_id=$1 AND kind=$2`
	args := []interface{}{chatID, kind}
	if activeOnly && kind != "ban" {
		query += ` AND (until_unix=0 OR until_unix>$3)`
		args = append(args, now.Unix())
	}
	query += ` ORDER BY since_unix DESC`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []ModerationRecord
	for rows.Next() {
		var rec ModerationRecord
		if err := rows.Scan(&rec.UserID, &rec.Username, &rec.DisplayName, &rec.Reason, &rec.RehabWard, &rec.ByAdminID, &rec.ByAdminName, &rec.SinceUnix, &rec.StartedAtUnix, &rec.UntilUnix); err == nil {
			if kind == "jail" {
				rec = normalizeJailRecord(rec, time.Now().UTC())
			}
			out = append(out, rec)
		}
	}
	return out
}

func (s *PostgresStore) CleanupExpired(now time.Time) int {
	for _, rec := range s.expiredJailRecords(now) {
		s.rememberLastRehabDuration(rec.chatID, rec.UserID, rec.ModerationRecord, now.UTC())
	}
	res, err := s.db.Exec(`DELETE FROM moderation_records WHERE kind IN ('jail','mute','forward') AND until_unix<>0 AND until_unix<=$1`, now.Unix())
	if err != nil {
		s.logError("postgres cleanup failed", err)
		return 0
	}
	n, _ := res.RowsAffected()
	return int(n)
}

type postgresJailRecord struct {
	ModerationRecord
	chatID int64
}

func (s *PostgresStore) expiredJailRecords(now time.Time) []postgresJailRecord {
	rows, err := s.db.Query(`SELECT chat_id,user_id,username,display_name,reason,rehab_ward,by_admin_id,by_admin_name,since_unix,started_at,until_unix
		FROM moderation_records WHERE kind='jail' AND until_unix<>0 AND until_unix<=$1`, now.Unix())
	if err != nil {
		s.logError("postgres expired jail lookup failed", err)
		return nil
	}
	defer rows.Close()
	var out []postgresJailRecord
	for rows.Next() {
		var rec postgresJailRecord
		if err := rows.Scan(&rec.chatID, &rec.UserID, &rec.Username, &rec.DisplayName, &rec.Reason, &rec.RehabWard, &rec.ByAdminID, &rec.ByAdminName, &rec.SinceUnix, &rec.StartedAtUnix, &rec.UntilUnix); err == nil {
			rec.ModerationRecord = normalizeJailRecord(rec.ModerationRecord, time.Now().UTC())
			out = append(out, rec)
		}
	}
	return out
}

func (s *PostgresStore) rememberLastRehabDuration(chatID, userID int64, rec ModerationRecord, endedAt time.Time) {
	duration := rehabTreatmentDuration(rec, endedAt)
	if duration <= 0 {
		return
	}
	_, err := s.db.Exec(`UPDATE members SET last_rehab_duration=$3 WHERE chat_id=$1 AND user_id=$2`, chatID, userID, int64(duration.Round(time.Second)/time.Second))
	if err != nil {
		s.logError("postgres save last rehab duration failed", err)
	}
}

func (s *PostgresStore) AddWarning(chatID int64, rec WarningRecord) (WarningRecord, error) {
	now := time.Now().UTC()
	if rec.ID == "" {
		rec.ID = "warn_" + strconv.FormatInt(now.UnixNano(), 36)
	}
	if rec.CreatedUnix == 0 {
		rec.CreatedUnix = now.Unix()
	}
	rec.Active = true
	_, err := s.db.Exec(`INSERT INTO warnings(id,chat_id,user_id,username,display_name,reason,by_admin_id,by_admin_name,created_unix,expires_unix,active)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, rec.ID, chatID, rec.UserID, rec.Username, rec.DisplayName, rec.Reason, rec.ByAdminID, rec.ByAdminName, rec.CreatedUnix, rec.ExpiresUnix, rec.Active)
	return rec, err
}

func (s *PostgresStore) Warnings(chatID, userID int64, includeInactive bool, limit int, now time.Time) []WarningRecord {
	query := `SELECT id,user_id,username,display_name,reason,by_admin_id,by_admin_name,created_unix,expires_unix,active FROM warnings WHERE chat_id=$1`
	args := []interface{}{chatID}
	if userID != 0 {
		query += ` AND user_id=$` + strconv.Itoa(len(args)+1)
		args = append(args, userID)
	}
	if !includeInactive {
		query += ` AND active=true AND (expires_unix=0 OR expires_unix>$` + strconv.Itoa(len(args)+1) + `)`
		args = append(args, now.Unix())
	}
	query += ` ORDER BY created_unix DESC`
	if limit > 0 {
		query += ` LIMIT ` + strconv.Itoa(limit)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []WarningRecord
	for rows.Next() {
		var rec WarningRecord
		if err := rows.Scan(&rec.ID, &rec.UserID, &rec.Username, &rec.DisplayName, &rec.Reason, &rec.ByAdminID, &rec.ByAdminName, &rec.CreatedUnix, &rec.ExpiresUnix, &rec.Active); err == nil {
			out = append(out, rec)
		}
	}
	return out
}

func (s *PostgresStore) ActiveWarningCount(chatID, userID int64, now time.Time) int {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM warnings WHERE chat_id=$1 AND user_id=$2 AND active=true AND (expires_unix=0 OR expires_unix>$3)`, chatID, userID, now.Unix()).Scan(&count)
	return count
}

func (s *PostgresStore) DeactivateWarning(chatID int64, warningID string) (WarningRecord, bool, error) {
	var rec WarningRecord
	err := s.db.QueryRow(`UPDATE warnings SET active=false WHERE chat_id=$1 AND id=$2 RETURNING id,user_id,username,display_name,reason,by_admin_id,by_admin_name,created_unix,expires_unix,active`,
		chatID, warningID).Scan(&rec.ID, &rec.UserID, &rec.Username, &rec.DisplayName, &rec.Reason, &rec.ByAdminID, &rec.ByAdminName, &rec.CreatedUnix, &rec.ExpiresUnix, &rec.Active)
	if errors.Is(err, sql.ErrNoRows) {
		return WarningRecord{}, false, nil
	}
	return rec, err == nil, err
}

func (s *PostgresStore) ClearWarnings(chatID, userID int64) (int, error) {
	res, err := s.db.Exec(`UPDATE warnings SET active=false WHERE chat_id=$1 AND user_id=$2 AND active=true`, chatID, userID)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

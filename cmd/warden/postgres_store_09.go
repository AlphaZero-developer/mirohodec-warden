package main

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	tele "gopkg.in/telebot.v3"
)

func (s *PostgresStore) Karma(chatID, userID int64) int64 {
	var score int64
	_ = s.db.QueryRow(`SELECT score FROM karma_totals WHERE chat_id=$1 AND user_id=$2`, chatID, userID).Scan(&score)
	return score
}

func (s *PostgresStore) TopKarma(chatID int64, limit int) []KarmaTotal {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.Query(`SELECT user_id, display_name, username, score, updated_unix FROM karma_totals WHERE chat_id=$1 ORDER BY score DESC, updated_unix DESC LIMIT $2`, chatID, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []KarmaTotal
	for rows.Next() {
		var rec KarmaTotal
		if err := rows.Scan(&rec.UserID, &rec.DisplayName, &rec.Username, &rec.Score, &rec.UpdatedUnix); err == nil {
			out = append(out, rec)
		}
	}
	return out
}

func (s *PostgresStore) AddKarma(chatID, userID int64, delta int64, reason string, byUser *tele.User, messageID int, now time.Time) (int64, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	member, _ := s.Member(chatID, userID)
	byID := int64(0)
	byName := ""
	if byUser != nil {
		byID = byUser.ID
		byName = displayNameFromUser(byUser)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`INSERT INTO karma_totals(chat_id,user_id,display_name,username,score,updated_unix)
		VALUES($1,$2,$3,$4,$5,$6)
		ON CONFLICT(chat_id,user_id) DO UPDATE SET display_name=EXCLUDED.display_name, username=EXCLUDED.username, score=karma_totals.score + EXCLUDED.score, updated_unix=EXCLUDED.updated_unix`,
		chatID, userID, member.DisplayName, member.Username, delta, now.Unix())
	if err != nil {
		return 0, err
	}
	_, err = tx.Exec(`INSERT INTO karma_events(chat_id,user_id,delta,reason,by_user_id,by_user_name,message_id,day,created_unix)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, chatID, userID, delta, reason, byID, byName, messageID, now.Format("2006-01-02"), now.Unix())
	if err != nil {
		return 0, err
	}
	var score int64
	if err := tx.QueryRow(`SELECT score FROM karma_totals WHERE chat_id=$1 AND user_id=$2`, chatID, userID).Scan(&score); err != nil {
		return 0, err
	}
	return score, tx.Commit()
}

func (s *PostgresStore) MaybeAwardActivityKarma(chatID, userID int64, messagesPerPoint, dailyLimit int, now time.Time) (int64, bool) {
	if messagesPerPoint <= 0 || dailyLimit <= 0 {
		return 0, false
	}
	member, ok := s.Member(chatID, userID)
	if !ok {
		return 0, false
	}
	allowed := int(member.MessagesToday) / messagesPerPoint
	if allowed > dailyLimit {
		allowed = dailyLimit
	}
	if allowed <= 0 {
		return 0, false
	}
	day := now.Format("2006-01-02")
	var earned int
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(delta),0) FROM karma_events WHERE chat_id=$1 AND user_id=$2 AND reason='activity' AND day=$3`, chatID, userID, day).Scan(&earned)
	if earned >= allowed {
		return 0, false
	}
	score, err := s.AddKarma(chatID, userID, 1, "activity", nil, 0, now)
	return score, err == nil
}

func (s *PostgresStore) LastManualKarma(chatID, targetID, giverID int64, now time.Time) time.Time {
	var ts int64
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(created_unix),0) FROM karma_events WHERE chat_id=$1 AND user_id=$2 AND by_user_id=$3 AND reason='manual'`, chatID, targetID, giverID).Scan(&ts)
	if ts == 0 {
		return time.Time{}
	}
	return time.Unix(ts, 0).UTC()
}

func (p *PostgresPhotoStore) Add(photo *tele.Photo, tags []string, caption string, admin *tele.User) (PhotoRecord, error) {
	if photo == nil || photo.FileID == "" {
		return PhotoRecord{}, errors.New("photo has no Telegram file_id")
	}
	now := time.Now().UTC()
	rec := PhotoRecord{ID: strconv.FormatInt(now.UnixNano(), 36), FileID: photo.FileID, UniqueID: photo.UniqueID, Caption: strings.TrimSpace(caption), Tags: normalizeTags(tags), AddedAtUnix: now.Unix()}
	if admin != nil {
		rec.AddedByID = admin.ID
		rec.AddedByName = displayNameFromUser(admin)
	}
	return rec, p.base.insertPhoto(rec)
}

func (s *PostgresStore) insertPhoto(rec PhotoRecord) error {
	if rec.ID == "" {
		rec.ID = strconv.FormatInt(time.Now().UTC().UnixNano(), 36)
	}
	_, err := s.db.Exec(`INSERT INTO photos(id,file_id,unique_id,caption,tags,added_by_id,added_by_name,added_at_unix,uses,last_used_unix)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(id) DO NOTHING`,
		rec.ID, rec.FileID, rec.UniqueID, rec.Caption, pq.Array(normalizeTags(rec.Tags)), rec.AddedByID, rec.AddedByName, rec.AddedAtUnix, rec.Uses, rec.LastUsedUnix)
	return err
}

func (p *PostgresPhotoStore) Delete(id string) (PhotoRecord, bool, error) {
	rec, ok := p.base.photoByID(id)
	if !ok {
		return PhotoRecord{}, false, nil
	}
	_, err := p.base.db.Exec(`DELETE FROM photos WHERE id=$1`, id)
	return rec, err == nil, err
}

func (p *PostgresPhotoStore) List(tag string) []PhotoRecord {
	tag = normalizeTag(tag)
	query := `SELECT id,file_id,unique_id,caption,tags,added_by_id,added_by_name,added_at_unix,uses,last_used_unix FROM photos`
	args := []interface{}{}
	if tag != "" {
		query += ` WHERE $1=ANY(tags)`
		args = append(args, tag)
	}
	query += ` ORDER BY added_at_unix DESC`
	rows, err := p.base.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []PhotoRecord
	for rows.Next() {
		if rec, ok := scanPhoto(rows); ok {
			out = append(out, rec)
		}
	}
	return out
}

func (p *PostgresPhotoStore) Random(tag string) (PhotoRecord, bool) {
	records := p.List(tag)
	if len(records) == 0 {
		return PhotoRecord{}, false
	}
	rec := records[p.base.rng.Intn(len(records))]
	rec.Uses++
	rec.LastUsedUnix = time.Now().UTC().Unix()
	_, _ = p.base.db.Exec(`UPDATE photos SET uses=$2,last_used_unix=$3 WHERE id=$1`, rec.ID, rec.Uses, rec.LastUsedUnix)
	return rec, true
}

func (p *PostgresPhotoStore) Count() int {
	var count int
	_ = p.base.db.QueryRow(`SELECT COUNT(*) FROM photos`).Scan(&count)
	return count
}

func (s *PostgresStore) photoByID(id string) (PhotoRecord, bool) {
	row := s.db.QueryRow(`SELECT id,file_id,unique_id,caption,tags,added_by_id,added_by_name,added_at_unix,uses,last_used_unix FROM photos WHERE id=$1`, id)
	return scanPhoto(row)
}

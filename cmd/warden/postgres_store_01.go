package main

import (
	"database/sql"
	"encoding/json"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/lib/pq"
)

type PostgresStore struct {
	db        *sql.DB
	log       *AsyncLogger
	rng       *rand.Rand
	statePath string
	photoPath string
	quotePath string
	faqPath   string
	mafiaMu   sync.Mutex
}

type PostgresPhotoStore struct{ base *PostgresStore }

type PostgresQuoteStore struct{ base *PostgresStore }

type PostgresFAQStore struct{ base *PostgresStore }

func NewPostgresStore(cfg Config, logger *AsyncLogger) (*PostgresStore, error) {
	db, err := sql.Open("postgres", cfg.Storage.Postgres.DSN)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	s := &PostgresStore{
		db:        db,
		log:       logger,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
		statePath: cfg.Storage.Path,
		photoPath: cfg.Photos.Path,
		quotePath: cfg.Quotes.Path,
		faqPath:   cfg.FAQ.Path,
	}
	if cfg.Storage.AutoMigrate {
		if err := s.Migrate(); err != nil {
			db.Close()
			return nil, err
		}
	}
	if cfg.Storage.ImportJSONOnEmpty {
		if err := s.ImportJSONIfEmpty(); err != nil {
			db.Close()
			return nil, err
		}
	}
	return s, nil
}

func (s *PostgresStore) Migrate() error {
	return applyEmbeddedMigrations(s.db)
}

func (s *PostgresStore) ImportJSONIfEmpty() error {
	if !s.databaseEmpty() {
		return nil
	}
	state, err := loadLegacyState(s.statePath)
	if err != nil {
		return err
	}
	photos, err := loadLegacyPhotos(s.photoPath)
	if err != nil {
		return err
	}
	quotes, err := loadLegacyQuotes(s.quotePath)
	if err != nil {
		return err
	}
	faqs, err := loadLegacyFAQs(s.faqPath)
	if err != nil {
		return err
	}
	for _, ownerID := range state.Bot.OwnerIDs {
		_, _ = s.db.Exec(`INSERT INTO owners(user_id, log_chat_id, authenticated_at)
			VALUES($1,$2,$3) ON CONFLICT(user_id) DO UPDATE SET log_chat_id=EXCLUDED.log_chat_id, authenticated_at=EXCLUDED.authenticated_at`,
			ownerID, state.Bot.LogChatID, state.Bot.AuthenticatedAt)
	}
	for _, sleep := range state.Bot.Sleep {
		calls, _ := json.Marshal(sleep.Calls)
		_, _ = s.db.Exec(`INSERT INTO owner_sleep(user_id, username, display_name, aliases, since_unix, calls)
			VALUES($1,$2,$3,$4,$5,$6::jsonb)
			ON CONFLICT(user_id) DO UPDATE SET username=EXCLUDED.username, display_name=EXCLUDED.display_name, aliases=EXCLUDED.aliases, since_unix=EXCLUDED.since_unix, calls=EXCLUDED.calls`,
			sleep.UserID, sleep.Username, sleep.DisplayName, pq.Array(sleep.Aliases), sleep.SinceUnix, string(calls))
	}
	if !emptyLayaRuntimeState(state.Bot.Laya) {
		_ = s.SetLayaRuntimeState(state.Bot.Laya)
	}
	if !emptyContentRuntimeState(state.Bot.Content) {
		_ = s.SetContentRuntimeState(state.Bot.Content)
	}
	for _, rec := range state.Bot.OwnerReports {
		if rec.Kind != "" && rec.ChatID != 0 && rec.UserID != 0 && rec.LastSentUnix > 0 {
			_ = s.SetOwnerReportLastSent(rec.Kind, rec.ChatID, rec.UserID, time.Unix(rec.LastSentUnix, 0).UTC())
		}
	}
	for _, profile := range state.Bot.LayaProfiles {
		_ = s.SetLayaUserProfile(profile)
	}
	for _, rec := range state.Bot.Scriptwriters {
		_ = s.SetScriptwriterExemption(rec)
	}
	for _, rec := range state.Bot.LayaIgnored {
		_ = s.SetLayaIgnoredUser(rec)
	}
	if state.Bot.Discussion.UpdatedUnix != 0 {
		_ = s.SetDiscussionRuntimeState(state.Bot.Discussion)
	}
	if state.Bot.RehabBrainwash.UpdatedUnix != 0 {
		_ = s.SetRehabBrainwashRuntimeState(state.Bot.RehabBrainwash)
	}
	if state.Bot.RehabReleaseExam.UpdatedUnix != 0 {
		_ = s.SetRehabReleaseExamRuntimeState(state.Bot.RehabReleaseExam)
	}
	for chatKey, exam := range state.Bot.RehabReleaseExams {
		chatID, err := strconv.ParseInt(chatKey, 10, 64)
		if err != nil || chatID == 0 || exam.UpdatedUnix == 0 {
			continue
		}
		_ = s.SetRehabReleaseExamRuntimeStateForChat(chatID, exam)
	}
	if !emptyReplyCipherRuntimeState(state.Bot.ReplyCipher) {
		_ = s.SetReplyCipherRuntimeState(state.Bot.ReplyCipher)
	}
	if len(state.Mafia.RegisteredUsers) > 0 || len(state.Mafia.Rooms) > 0 {
		_ = s.saveMafiaStateLocked(state.Mafia)
	}
	for chatKey, ch := range state.Chats {
		chatID, err := strconv.ParseInt(chatKey, 10, 64)
		if err != nil {
			continue
		}
		for _, member := range ch.Members {
			s.upsertMemberRecord(chatID, member)
		}
		for _, rec := range ch.Jailed {
			rec = normalizeJailRecord(rec, time.Now().UTC())
			s.setModerationRecord(chatID, "jail", rec)
		}
		for _, rec := range ch.Muted {
			s.setModerationRecord(chatID, "mute", rec)
		}
		for _, rec := range ch.Banned {
			s.setModerationRecord(chatID, "ban", rec)
		}
		for _, rec := range ch.Warnings {
			_, _ = s.AddWarning(chatID, rec)
		}
		for _, rec := range ch.Karma {
			_, _ = s.db.Exec(`INSERT INTO karma_totals(chat_id,user_id,display_name,username,score,updated_unix)
				VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(chat_id,user_id) DO UPDATE SET display_name=EXCLUDED.display_name, username=EXCLUDED.username, score=EXCLUDED.score, updated_unix=EXCLUDED.updated_unix`,
				chatID, rec.UserID, rec.DisplayName, rec.Username, rec.Score, rec.UpdatedUnix)
		}
	}
	for _, settings := range state.ManagedChats {
		settings, _ = normalizeChatSettings(settings, ChatSettings{})
		_ = s.UpsertChatSettings(settings)
	}
	for _, reply := range state.DiscussionReplies {
		if reply.PostedUnix == 0 {
			reply.PostedUnix = time.Now().UTC().Unix()
		}
		_, _ = s.MarkDiscussionReply(reply.ChatID, reply.TopicID, time.Unix(reply.PostedUnix, 0).UTC())
	}
	for _, rec := range photos.Photos {
		s.insertPhoto(rec)
	}
	for _, rec := range quotes.Quotes {
		s.insertQuote(rec)
	}
	for _, rec := range faqs.FAQs {
		s.insertFAQ(rec)
	}
	if s.log != nil {
		s.log.Info("legacy JSON imported into PostgreSQL")
	}
	return nil
}

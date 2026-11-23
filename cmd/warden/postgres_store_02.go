package main

import (
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (s *PostgresStore) databaseEmpty() bool {
	for _, table := range []string{"managed_chats", "discussion_replies", "members", "moderation_records", "photos", "quotes", "faqs"} {
		var count int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count > 0 {
			return false
		}
	}
	return true
}

func (s *PostgresStore) Flush() error { return nil }

func (s *PostgresStore) Reload() error { return nil }

func (s *PostgresStore) ReloadIfChanged() (bool, error) { return false, nil }

func (p *PostgresPhotoStore) Flush() error { return nil }

func (p *PostgresPhotoStore) Reload() error { return nil }

func (p *PostgresPhotoStore) ReloadIfChanged() (bool, error) { return false, nil }

func (q *PostgresQuoteStore) Flush() error { return nil }

func (q *PostgresQuoteStore) Reload() error { return nil }

func (q *PostgresQuoteStore) ReloadIfChanged() (bool, error) { return false, nil }

func (f *PostgresFAQStore) Flush() error { return nil }

func (f *PostgresFAQStore) Reload() error { return nil }

func (f *PostgresFAQStore) ReloadIfChanged() (bool, error) { return false, nil }

func (s *PostgresStore) UpsertMember(chatID int64, u *tele.User) {
	s.TrackMember(chatID, u, 0, 0, false)
}

func (s *PostgresStore) TrackMember(chatID int64, u *tele.User, threadID, messageID int, countMessage bool) {
	if u == nil || u.ID == 0 {
		return
	}
	nowTime := time.Now().UTC()
	now := nowTime.Unix()
	day := nowTime.Format("2006-01-02")
	next := memberFromUser(u, now)
	next.LastThreadID = threadID

	old, exists := s.Member(chatID, u.ID)
	if exists {
		next.FirstSeen = old.FirstSeen
		next.Messages = old.Messages
		next.MessagesToday = old.MessagesToday
		next.ActivityDay = old.ActivityDay
		next.LastMessage = old.LastMessage
		next.LastMessageID = old.LastMessageID
		next.LastThreadID = old.LastThreadID
		next.LastRehabDuration = old.LastRehabDuration
		next.RehabEscapePenalty = old.RehabEscapePenalty
		next.Achievements = append([]string(nil), old.Achievements...)
		next.ShopTitle = old.ShopTitle
		next.ShopTitleUntilUnix = old.ShopTitleUntilUnix
		next.LayaTone = old.LayaTone
		next.LayaCharmUntilUnix = old.LayaCharmUntilUnix
		next.QuoteBoostUntilUnix = old.QuoteBoostUntilUnix
	}
	if next.FirstSeen == 0 {
		next.FirstSeen = now
	}
	if countMessage {
		next.Messages++
		if next.ActivityDay != day {
			next.ActivityDay = day
			next.MessagesToday = 0
		}
		next.MessagesToday++
		next.LastMessage = now
		next.LastMessageID = messageID
		next.LastThreadID = threadID
	}
	s.upsertMemberRecord(chatID, next)
}

func (s *PostgresStore) upsertMemberRecord(chatID int64, rec MemberRecord) {
	_, err := s.db.Exec(`INSERT INTO members(chat_id,user_id,username,first_name,last_name,display_name,is_bot,first_seen,last_seen,last_message,last_message_id,last_thread_id,messages,messages_today,activity_day,last_rehab_duration,rehab_escape_penalty,achievements,shop_title,shop_title_until,laya_tone,laya_charm_until,quote_boost_until)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)
		ON CONFLICT(chat_id,user_id) DO UPDATE SET username=EXCLUDED.username, first_name=EXCLUDED.first_name, last_name=EXCLUDED.last_name, display_name=EXCLUDED.display_name, is_bot=EXCLUDED.is_bot, first_seen=EXCLUDED.first_seen, last_seen=EXCLUDED.last_seen, last_message=EXCLUDED.last_message, last_message_id=EXCLUDED.last_message_id, last_thread_id=EXCLUDED.last_thread_id, messages=EXCLUDED.messages, messages_today=EXCLUDED.messages_today, activity_day=EXCLUDED.activity_day, last_rehab_duration=EXCLUDED.last_rehab_duration, rehab_escape_penalty=EXCLUDED.rehab_escape_penalty, achievements=EXCLUDED.achievements, shop_title=EXCLUDED.shop_title, shop_title_until=EXCLUDED.shop_title_until, laya_tone=EXCLUDED.laya_tone, laya_charm_until=EXCLUDED.laya_charm_until, quote_boost_until=EXCLUDED.quote_boost_until`,
		chatID, rec.ID, rec.Username, rec.FirstName, rec.LastName, rec.DisplayName, rec.IsBot, rec.FirstSeen, rec.LastSeen, rec.LastMessage, rec.LastMessageID, rec.LastThreadID, rec.Messages, rec.MessagesToday, rec.ActivityDay, rec.LastRehabDuration, rec.RehabEscapePenalty, pq.Array(normalizeAchievements(rec.Achievements)), rec.ShopTitle, rec.ShopTitleUntilUnix, normalizeLayaTone(rec.LayaTone), rec.LayaCharmUntilUnix, rec.QuoteBoostUntilUnix)
	if err != nil {
		s.logError("postgres save member failed", err)
	}
}

func (s *PostgresStore) Member(chatID, userID int64) (MemberRecord, bool) {
	var rec MemberRecord
	err := s.db.QueryRow(`SELECT user_id, username, first_name, last_name, display_name, is_bot, first_seen, last_seen, last_message, last_message_id, last_thread_id, messages, messages_today, activity_day, last_rehab_duration, rehab_escape_penalty, achievements, shop_title, shop_title_until, laya_tone, laya_charm_until, quote_boost_until
		FROM members WHERE chat_id=$1 AND user_id=$2`, chatID, userID).Scan(&rec.ID, &rec.Username, &rec.FirstName, &rec.LastName, &rec.DisplayName, &rec.IsBot, &rec.FirstSeen, &rec.LastSeen, &rec.LastMessage, &rec.LastMessageID, &rec.LastThreadID, &rec.Messages, &rec.MessagesToday, &rec.ActivityDay, &rec.LastRehabDuration, &rec.RehabEscapePenalty, pq.Array(&rec.Achievements), &rec.ShopTitle, &rec.ShopTitleUntilUnix, &rec.LayaTone, &rec.LayaCharmUntilUnix, &rec.QuoteBoostUntilUnix)
	if errors.Is(err, sql.ErrNoRows) {
		return MemberRecord{}, false
	}
	if err != nil {
		s.logError("postgres member lookup failed", err)
		return MemberRecord{}, false
	}
	return rec, true
}

func (s *PostgresStore) UserByUsername(chatID int64, username string) (targetUser, bool) {
	username = normalizeUsername(username)
	if username == "" {
		return targetUser{}, false
	}
	var target targetUser
	err := s.db.QueryRow(`SELECT user_id, username, display_name, is_bot FROM members WHERE chat_id=$1 AND lower(username)=$2 LIMIT 1`, chatID, username).Scan(&target.ID, &target.Username, &target.Name, &target.IsBot)
	if err == nil {
		return target, true
	}
	err = s.db.QueryRow(`SELECT user_id, username, display_name FROM moderation_records WHERE chat_id=$1 AND lower(username)=$2 LIMIT 1`, chatID, username).Scan(&target.ID, &target.Username, &target.Name)
	if err == nil {
		return target, true
	}
	return targetUser{}, false
}

func (s *PostgresStore) Members(chatID int64) []MemberRecord {
	rows, err := s.db.Query(`SELECT user_id, username, first_name, last_name, display_name, is_bot, first_seen, last_seen, last_message, last_message_id, last_thread_id, messages, messages_today, activity_day, last_rehab_duration, rehab_escape_penalty, achievements, shop_title, shop_title_until, laya_tone, laya_charm_until, quote_boost_until
		FROM members WHERE chat_id=$1 AND is_bot=false ORDER BY display_name`, chatID)
	if err != nil {
		s.logError("postgres members list failed", err)
		return nil
	}
	defer rows.Close()
	var out []MemberRecord
	for rows.Next() {
		var rec MemberRecord
		if err := rows.Scan(&rec.ID, &rec.Username, &rec.FirstName, &rec.LastName, &rec.DisplayName, &rec.IsBot, &rec.FirstSeen, &rec.LastSeen, &rec.LastMessage, &rec.LastMessageID, &rec.LastThreadID, &rec.Messages, &rec.MessagesToday, &rec.ActivityDay, &rec.LastRehabDuration, &rec.RehabEscapePenalty, pq.Array(&rec.Achievements), &rec.ShopTitle, &rec.ShopTitleUntilUnix, &rec.LayaTone, &rec.LayaCharmUntilUnix, &rec.QuoteBoostUntilUnix); err == nil {
			out = append(out, rec)
		}
	}
	return out
}

func (s *PostgresStore) ChatIDs() []int64 {
	rows, err := s.db.Query(`SELECT chat_id FROM managed_chats UNION SELECT DISTINCT chat_id FROM members UNION SELECT DISTINCT chat_id FROM moderation_records ORDER BY chat_id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil && id != 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

func (s *PostgresStore) UpsertChatSettings(settings ChatSettings) error {
	if settings.ChatID == 0 {
		return errors.New("chat_id is required")
	}
	requestedTopicID := settings.RehabTopicID
	old, hadOld := s.ChatSettings(settings.ChatID)
	settings, _ = normalizeChatSettings(settings, old)
	if hadOld && settings.Title == "" {
		settings.Title = old.Title
	}
	settings.UpdatedUnix = time.Now().UTC().Unix()
	if hadOld && requestedTopicID == 0 && (old.RehabTopicID > 0 || old.PersistedRehabTopicID > 0) && settings.RehabTopicID > 0 && s.log != nil {
		s.log.Info("rehab topic kept from state", zap.Int64("chat", settings.ChatID), zap.Int("topic", settings.RehabTopicID))
	} else if settings.RehabTopicID > 0 && (!hadOld || old.RehabTopicID != settings.RehabTopicID || old.PersistedRehabTopicID != settings.PersistedRehabTopicID) && s.log != nil {
		s.log.Info("rehab topic persisted", zap.Int64("chat", settings.ChatID), zap.Int("topic", settings.RehabTopicID))
	}
	_, err := s.db.Exec(`INSERT INTO managed_chats(chat_id,title,enabled,rehab_topic_id,persisted_rehab_topic_id,spam_no_rehab_action,debug_commands,updated_unix,ai_enabled_mode,spam_enabled_mode,warn_enabled_mode,karma_enabled_mode,faq_enabled_mode,quotes_enabled_mode,photos_enabled_mode,warn_expire_after,warn_mute_threshold,warn_mute_duration,warn_rehab_threshold,warn_rehab_duration,karma_messages_per_point,karma_daily_limit,karma_manual_cooldown)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)
		ON CONFLICT(chat_id) DO UPDATE SET title=EXCLUDED.title, enabled=EXCLUDED.enabled, rehab_topic_id=EXCLUDED.rehab_topic_id, persisted_rehab_topic_id=EXCLUDED.persisted_rehab_topic_id, spam_no_rehab_action=EXCLUDED.spam_no_rehab_action, debug_commands=EXCLUDED.debug_commands, updated_unix=EXCLUDED.updated_unix, ai_enabled_mode=EXCLUDED.ai_enabled_mode, spam_enabled_mode=EXCLUDED.spam_enabled_mode, warn_enabled_mode=EXCLUDED.warn_enabled_mode, karma_enabled_mode=EXCLUDED.karma_enabled_mode, faq_enabled_mode=EXCLUDED.faq_enabled_mode, quotes_enabled_mode=EXCLUDED.quotes_enabled_mode, photos_enabled_mode=EXCLUDED.photos_enabled_mode, warn_expire_after=EXCLUDED.warn_expire_after, warn_mute_threshold=EXCLUDED.warn_mute_threshold, warn_mute_duration=EXCLUDED.warn_mute_duration, warn_rehab_threshold=EXCLUDED.warn_rehab_threshold, warn_rehab_duration=EXCLUDED.warn_rehab_duration, karma_messages_per_point=EXCLUDED.karma_messages_per_point, karma_daily_limit=EXCLUDED.karma_daily_limit, karma_manual_cooldown=EXCLUDED.karma_manual_cooldown`,
		settings.ChatID, settings.Title, settings.Enabled, settings.RehabTopicID, settings.PersistedRehabTopicID, settings.SpamNoRehabAction, settings.DebugCommands, settings.UpdatedUnix, settings.AIEnabledMode, settings.SpamEnabledMode, settings.WarnEnabledMode, settings.KarmaEnabledMode, settings.FAQEnabledMode, settings.QuotesEnabledMode, settings.PhotosEnabledMode, settings.WarnExpireAfter, settings.WarnMuteThreshold, settings.WarnMuteDuration, settings.WarnRehabThreshold, settings.WarnRehabDuration, settings.KarmaMessagesPerPoint, settings.KarmaDailyLimit, settings.KarmaManualCooldown)
	return err
}

package main

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (s *PostgresStore) ChatSettings(chatID int64) (ChatSettings, bool) {
	var settings ChatSettings
	err := s.db.QueryRow(`SELECT chat_id,title,enabled,rehab_topic_id,persisted_rehab_topic_id,spam_no_rehab_action,debug_commands,updated_unix,ai_enabled_mode,spam_enabled_mode,warn_enabled_mode,karma_enabled_mode,faq_enabled_mode,quotes_enabled_mode,photos_enabled_mode,warn_expire_after,warn_mute_threshold,warn_mute_duration,warn_rehab_threshold,warn_rehab_duration,karma_messages_per_point,karma_daily_limit,karma_manual_cooldown FROM managed_chats WHERE chat_id=$1`, chatID).
		Scan(&settings.ChatID, &settings.Title, &settings.Enabled, &settings.RehabTopicID, &settings.PersistedRehabTopicID, &settings.SpamNoRehabAction, &settings.DebugCommands, &settings.UpdatedUnix, &settings.AIEnabledMode, &settings.SpamEnabledMode, &settings.WarnEnabledMode, &settings.KarmaEnabledMode, &settings.FAQEnabledMode, &settings.QuotesEnabledMode, &settings.PhotosEnabledMode, &settings.WarnExpireAfter, &settings.WarnMuteThreshold, &settings.WarnMuteDuration, &settings.WarnRehabThreshold, &settings.WarnRehabDuration, &settings.KarmaMessagesPerPoint, &settings.KarmaDailyLimit, &settings.KarmaManualCooldown)
	if err != nil {
		return ChatSettings{}, false
	}
	return settings, true
}

func (s *PostgresStore) ManagedChats(includeDisabled bool) []ChatSettings {
	query := `SELECT chat_id,title,enabled,rehab_topic_id,persisted_rehab_topic_id,spam_no_rehab_action,debug_commands,updated_unix,ai_enabled_mode,spam_enabled_mode,warn_enabled_mode,karma_enabled_mode,faq_enabled_mode,quotes_enabled_mode,photos_enabled_mode,warn_expire_after,warn_mute_threshold,warn_mute_duration,warn_rehab_threshold,warn_rehab_duration,karma_messages_per_point,karma_daily_limit,karma_manual_cooldown FROM managed_chats`
	if !includeDisabled {
		query += ` WHERE enabled=true`
	}
	query += ` ORDER BY title, chat_id`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []ChatSettings
	for rows.Next() {
		var settings ChatSettings
		if err := rows.Scan(&settings.ChatID, &settings.Title, &settings.Enabled, &settings.RehabTopicID, &settings.PersistedRehabTopicID, &settings.SpamNoRehabAction, &settings.DebugCommands, &settings.UpdatedUnix, &settings.AIEnabledMode, &settings.SpamEnabledMode, &settings.WarnEnabledMode, &settings.KarmaEnabledMode, &settings.FAQEnabledMode, &settings.QuotesEnabledMode, &settings.PhotosEnabledMode, &settings.WarnExpireAfter, &settings.WarnMuteThreshold, &settings.WarnMuteDuration, &settings.WarnRehabThreshold, &settings.WarnRehabDuration, &settings.KarmaMessagesPerPoint, &settings.KarmaDailyLimit, &settings.KarmaManualCooldown); err == nil {
			out = append(out, settings)
		}
	}
	return out
}

func (s *PostgresStore) DeleteChatSettings(chatID int64) error {
	_, err := s.db.Exec(`DELETE FROM managed_chats WHERE chat_id=$1`, chatID)
	return err
}

func (s *PostgresStore) DiscussionReplyPosted(chatID int64, topicID int) bool {
	if topicID == 0 {
		return false
	}
	var exists bool
	err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM discussion_replies WHERE chat_id=$1 AND topic_id=$2)`, chatID, topicID).Scan(&exists)
	if err != nil {
		s.logError("postgres discussion reply lookup failed", err)
		return false
	}
	return exists
}

func (s *PostgresStore) MarkDiscussionReply(chatID int64, topicID int, now time.Time) (bool, error) {
	if topicID == 0 {
		return false, nil
	}
	var inserted bool
	err := s.db.QueryRow(`INSERT INTO discussion_replies(chat_id, topic_id, posted_unix)
		VALUES($1,$2,$3) ON CONFLICT(chat_id, topic_id) DO NOTHING RETURNING TRUE`,
		chatID, topicID, now.UTC().Unix()).Scan(&inserted)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return inserted, nil
}

func (s *PostgresStore) Activity(chatID, userID int64) (MemberRecord, bool) {
	return s.Member(chatID, userID)
}

func (s *PostgresStore) TopActive(chatID int64, limit int) []MemberRecord {
	if limit <= 0 {
		limit = 10
	}
	members := s.Members(chatID)
	sort.Slice(members, func(i, j int) bool {
		if members[i].Messages == members[j].Messages {
			return members[i].LastMessage > members[j].LastMessage
		}
		return members[i].Messages > members[j].Messages
	})
	if len(members) > limit {
		members = members[:limit]
	}
	return members
}

func (s *PostgresStore) RandomActiveMember(chatID int64, activeWindow time.Duration, now time.Time, excludeID int64) (MemberRecord, bool) {
	if activeWindow <= 0 {
		activeWindow = 7 * 24 * time.Hour
	}
	cutoff := now.Add(-activeWindow).Unix()
	var weighted []MemberRecord
	for _, rec := range s.Members(chatID) {
		if rec.ID == 0 || rec.ID == excludeID || rec.IsBot || rec.LastMessage < cutoff || rec.Messages <= 0 {
			continue
		}
		weight := int(messagesToday(rec)) + 1
		if weight < 1 {
			weight = 1
		}
		if weight > 10 {
			weight = 10
		}
		for i := 0; i < weight; i++ {
			weighted = append(weighted, rec)
		}
	}
	if len(weighted) == 0 && excludeID != 0 {
		return s.RandomActiveMember(chatID, activeWindow, now, 0)
	}
	if len(weighted) == 0 {
		return MemberRecord{}, false
	}
	return weighted[s.rng.Intn(len(weighted))], true
}

func (s *PostgresStore) UnlockAchievement(chatID, userID int64, achievement string, now time.Time) (bool, error) {
	achievement = normalizeAchievement(achievement)
	if achievement == "" || userID == 0 {
		return false, nil
	}
	member, _ := s.Member(chatID, userID)
	if member.ID == 0 {
		member.ID = userID
		member.DisplayName = fmt.Sprintf("user %d", userID)
		if now.IsZero() {
			now = time.Now().UTC()
		}
		member.FirstSeen = now.Unix()
		member.LastSeen = now.Unix()
	}
	if memberHasAchievement(member, achievement) {
		return false, nil
	}
	member.Achievements = append(normalizeAchievements(member.Achievements), achievement)
	s.upsertMemberRecord(chatID, member)
	return true, nil
}

func (s *PostgresStore) SetMemberRewards(chatID, userID int64, rewards MemberRewards) error {
	if userID == 0 {
		return nil
	}
	member, _ := s.Member(chatID, userID)
	if member.ID == 0 {
		member.ID = userID
		member.DisplayName = fmt.Sprintf("user %d", userID)
		now := time.Now().UTC().Unix()
		member.FirstSeen = now
		member.LastSeen = now
	}
	member = applyMemberRewards(member, rewards)
	s.upsertMemberRecord(chatID, member)
	return nil
}

func (s *PostgresStore) RegisterOwner(user *tele.User, chat *tele.Chat) error {
	if user == nil || chat == nil {
		return errors.New("missing auth user or chat")
	}
	_, err := s.db.Exec(`INSERT INTO owners(user_id, log_chat_id, authenticated_at) VALUES($1,$2,$3)
		ON CONFLICT(user_id) DO UPDATE SET log_chat_id=EXCLUDED.log_chat_id, authenticated_at=EXCLUDED.authenticated_at`,
		user.ID, chat.ID, time.Now().UTC().Unix())
	return err
}

func (s *PostgresStore) IsOwner(userID int64) bool {
	var exists bool
	err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM owners WHERE user_id=$1)`, userID).Scan(&exists)
	return err == nil && exists
}

func (s *PostgresStore) HasOwner() bool {
	var exists bool
	err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM owners)`).Scan(&exists)
	return err == nil && exists
}

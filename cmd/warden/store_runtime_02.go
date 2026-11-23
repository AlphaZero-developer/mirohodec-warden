package main

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

type ReplyCipherTarget struct {
	UserID      int64  `json:"user_id"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Cipher      string `json:"cipher,omitempty"`
	AddedUnix   int64  `json:"added_unix,omitempty"`
	AddedByID   int64  `json:"added_by_id,omitempty"`
	AddedByName string `json:"added_by_name,omitempty"`
}

type OwnerSleepState struct {
	UserID      int64            `json:"user_id"`
	Username    string           `json:"username,omitempty"`
	DisplayName string           `json:"display_name,omitempty"`
	Aliases     []string         `json:"aliases,omitempty"`
	SinceUnix   int64            `json:"since_unix"`
	Calls       []OwnerSleepCall `json:"calls,omitempty"`
}

type OwnerSleepCall struct {
	AtUnix       int64  `json:"at_unix"`
	ChatID       int64  `json:"chat_id"`
	ChatTitle    string `json:"chat_title,omitempty"`
	ThreadID     int    `json:"thread_id,omitempty"`
	MessageID    int    `json:"message_id,omitempty"`
	FromID       int64  `json:"from_id"`
	FromUsername string `json:"from_username,omitempty"`
	FromName     string `json:"from_name,omitempty"`
	Text         string `json:"text,omitempty"`
	Matched      string `json:"matched,omitempty"`
}

type ChatState struct {
	Members     map[string]MemberRecord     `json:"members"`
	Jailed      map[string]ModerationRecord `json:"jailed"`
	Muted       map[string]ModerationRecord `json:"muted"`
	ForwardBan  map[string]ModerationRecord `json:"forward_ban,omitempty"`
	Banned      map[string]ModerationRecord `json:"banned"`
	Warnings    map[string]WarningRecord    `json:"warnings,omitempty"`
	Karma       map[string]KarmaTotal       `json:"karma,omitempty"`
	KarmaEvents []KarmaEvent                `json:"karma_events,omitempty"`
}

type MemberRecord struct {
	ID                  int64    `json:"id"`
	Username            string   `json:"username,omitempty"`
	FirstName           string   `json:"first_name,omitempty"`
	LastName            string   `json:"last_name,omitempty"`
	DisplayName         string   `json:"display_name"`
	IsBot               bool     `json:"is_bot"`
	FirstSeen           int64    `json:"first_seen"`
	LastSeen            int64    `json:"last_seen"`
	LastMessage         int64    `json:"last_message"`
	LastMessageID       int      `json:"last_message_id,omitempty"`
	LastThreadID        int      `json:"last_thread_id"`
	Messages            int64    `json:"messages"`
	MessagesToday       int64    `json:"messages_today"`
	ActivityDay         string   `json:"activity_day,omitempty"`
	LastRehabDuration   int64    `json:"last_rehab_duration,omitempty"`
	RehabEscapePenalty  int      `json:"rehab_escape_penalty,omitempty"`
	Achievements        []string `json:"achievements,omitempty"`
	ShopTitle           string   `json:"shop_title,omitempty"`
	ShopTitleUntilUnix  int64    `json:"shop_title_until,omitempty"`
	LayaTone            string   `json:"laya_tone,omitempty"`
	LayaCharmUntilUnix  int64    `json:"laya_charm_until,omitempty"`
	QuoteBoostUntilUnix int64    `json:"quote_boost_until,omitempty"`
}

type MemberRewards struct {
	Achievements        []string
	ShopTitle           string
	ShopTitleUntilUnix  int64
	LayaTone            string
	LayaCharmUntilUnix  int64
	QuoteBoostUntilUnix int64
}

type ModerationRecord struct {
	UserID        int64  `json:"user_id"`
	Username      string `json:"username,omitempty"`
	DisplayName   string `json:"display_name"`
	Reason        string `json:"reason,omitempty"`
	RehabWard     string `json:"rehab_ward,omitempty"`
	ByAdminID     int64  `json:"by_admin_id"`
	ByAdminName   string `json:"by_admin_name"`
	SinceUnix     int64  `json:"since_unix"`
	StartedAtUnix int64  `json:"started_at,omitempty"`
	UntilUnix     int64  `json:"until_unix"`
}

type WarningRecord struct {
	ID          string `json:"id"`
	UserID      int64  `json:"user_id"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"display_name"`
	Reason      string `json:"reason,omitempty"`
	ByAdminID   int64  `json:"by_admin_id"`
	ByAdminName string `json:"by_admin_name"`
	CreatedUnix int64  `json:"created_unix"`
	ExpiresUnix int64  `json:"expires_unix"`
	Active      bool   `json:"active"`
}

type KarmaTotal struct {
	UserID      int64  `json:"user_id"`
	DisplayName string `json:"display_name,omitempty"`
	Username    string `json:"username,omitempty"`
	Score       int64  `json:"score"`
	UpdatedUnix int64  `json:"updated_unix"`
}

type KarmaEvent struct {
	ID          string `json:"id"`
	UserID      int64  `json:"user_id"`
	Delta       int64  `json:"delta"`
	Reason      string `json:"reason"`
	ByUserID    int64  `json:"by_user_id,omitempty"`
	ByUserName  string `json:"by_user_name,omitempty"`
	MessageID   int    `json:"message_id,omitempty"`
	Day         string `json:"day,omitempty"`
	CreatedUnix int64  `json:"created_unix"`
}

type ChatSettings struct {
	ChatID                int64  `json:"chat_id"`
	Title                 string `json:"title,omitempty"`
	Enabled               bool   `json:"enabled"`
	RehabTopicID          int    `json:"rehab_topic_id"`
	PersistedRehabTopicID int    `json:"persisted_rehab_topic_id,omitempty"`
	SpamNoRehabAction     string `json:"spam_no_rehab_action,omitempty"`
	DebugCommands         bool   `json:"debug_commands"`
	UpdatedUnix           int64  `json:"updated_unix"`
	AIEnabledMode         string `json:"ai_enabled_mode,omitempty"`
	SpamEnabledMode       string `json:"spam_enabled_mode,omitempty"`
	WarnEnabledMode       string `json:"warn_enabled_mode,omitempty"`
	KarmaEnabledMode      string `json:"karma_enabled_mode,omitempty"`
	FAQEnabledMode        string `json:"faq_enabled_mode,omitempty"`
	QuotesEnabledMode     string `json:"quotes_enabled_mode,omitempty"`
	PhotosEnabledMode     string `json:"photos_enabled_mode,omitempty"`
	WarnExpireAfter       string `json:"warn_expire_after,omitempty"`
	WarnMuteThreshold     int    `json:"warn_mute_threshold,omitempty"`
	WarnMuteDuration      string `json:"warn_mute_duration,omitempty"`
	WarnRehabThreshold    int    `json:"warn_rehab_threshold,omitempty"`
	WarnRehabDuration     string `json:"warn_rehab_duration,omitempty"`
	KarmaMessagesPerPoint int    `json:"karma_messages_per_point,omitempty"`
	KarmaDailyLimit       int    `json:"karma_daily_limit,omitempty"`
	KarmaManualCooldown   string `json:"karma_manual_cooldown,omitempty"`
}

type DiscussionReplyRecord struct {
	ChatID     int64 `json:"chat_id"`
	TopicID    int   `json:"topic_id"`
	PostedUnix int64 `json:"posted_unix,omitempty"`
}

type Store struct {
	mu               sync.RWMutex
	path             string
	log              *AsyncLogger
	state            State
	modTime          time.Time
	lastActivitySave time.Time
}

func (s *Store) saveLocked() error {
	raw, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return err
	}
	if stat, err := os.Stat(s.path); err == nil {
		s.modTime = stat.ModTime()
	}
	return nil
}

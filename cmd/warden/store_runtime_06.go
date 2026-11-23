package main

import (
	"errors"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

func (s *Store) UpsertChatSettings(settings ChatSettings) error {
	if settings.ChatID == 0 {
		return errors.New("chat_id is required")
	}
	requestedTopicID := settings.RehabTopicID
	settings.Title = strings.TrimSpace(settings.Title)
	settings.UpdatedUnix = time.Now().UTC().Unix()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state.ManagedChats == nil {
		s.state.ManagedChats = make(map[string]ChatSettings)
	}
	old, ok := s.state.ManagedChats[idKey(settings.ChatID)]
	if ok {
		if settings.Title == "" {
			settings.Title = old.Title
		}
	}
	settings, _ = normalizeChatSettings(settings, old)
	if ok && settings.RehabTopicID == old.RehabTopicID && settings.PersistedRehabTopicID == old.PersistedRehabTopicID {

	} else if requestedTopicID == 0 && ok && (old.RehabTopicID > 0 || old.PersistedRehabTopicID > 0) && settings.RehabTopicID > 0 && s.log != nil {
		s.log.Info("rehab topic kept from state", zap.Int64("chat", settings.ChatID), zap.Int("topic", settings.RehabTopicID))
	} else if settings.RehabTopicID > 0 && s.log != nil {
		s.log.Info("rehab topic persisted", zap.Int64("chat", settings.ChatID), zap.Int("topic", settings.RehabTopicID))
	}
	s.state.ManagedChats[idKey(settings.ChatID)] = settings
	return s.saveLocked()
}

func normalizeChatSettings(settings, old ChatSettings) (ChatSettings, bool) {
	changed := false
	if settings.SpamNoRehabAction == "" {
		settings.SpamNoRehabAction = spamNoRehabMuteOnDelete
		changed = true
	}
	if old.RehabTopicID > 0 && settings.RehabTopicID == 0 {
		settings.RehabTopicID = old.RehabTopicID
		changed = true
	}
	if old.PersistedRehabTopicID > 0 && settings.PersistedRehabTopicID == 0 {
		settings.PersistedRehabTopicID = old.PersistedRehabTopicID
		changed = true
	}
	if settings.RehabTopicID == 0 && settings.PersistedRehabTopicID > 0 {
		settings.RehabTopicID = settings.PersistedRehabTopicID
		changed = true
	}
	if settings.RehabTopicID > 0 && settings.PersistedRehabTopicID != settings.RehabTopicID {
		settings.PersistedRehabTopicID = settings.RehabTopicID
		changed = true
	}
	settings.AIEnabledMode = normalizeLayaRuntimeMode(settings.AIEnabledMode, "on", "off")
	settings.SpamEnabledMode = normalizeLayaRuntimeMode(settings.SpamEnabledMode, "on", "off")
	settings.WarnEnabledMode = normalizeLayaRuntimeMode(settings.WarnEnabledMode, "on", "off")
	settings.KarmaEnabledMode = normalizeLayaRuntimeMode(settings.KarmaEnabledMode, "on", "off")
	settings.FAQEnabledMode = normalizeLayaRuntimeMode(settings.FAQEnabledMode, "on", "off")
	settings.QuotesEnabledMode = normalizeLayaRuntimeMode(settings.QuotesEnabledMode, "on", "off")
	settings.PhotosEnabledMode = normalizeLayaRuntimeMode(settings.PhotosEnabledMode, "on", "off")
	settings.WarnExpireAfter = strings.TrimSpace(settings.WarnExpireAfter)
	settings.WarnMuteDuration = strings.TrimSpace(settings.WarnMuteDuration)
	settings.WarnRehabDuration = strings.TrimSpace(settings.WarnRehabDuration)
	settings.KarmaManualCooldown = strings.TrimSpace(settings.KarmaManualCooldown)
	if settings.WarnMuteThreshold < 0 {
		settings.WarnMuteThreshold = 0
		changed = true
	}
	if settings.WarnRehabThreshold < 0 {
		settings.WarnRehabThreshold = 0
		changed = true
	}
	if settings.KarmaMessagesPerPoint < 0 {
		settings.KarmaMessagesPerPoint = 0
		changed = true
	}
	if settings.KarmaDailyLimit < 0 {
		settings.KarmaDailyLimit = 0
		changed = true
	}
	return settings, changed
}

func (s *Store) ChatSettings(chatID int64) (ChatSettings, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.state.ManagedChats == nil {
		return ChatSettings{}, false
	}
	settings, ok := s.state.ManagedChats[idKey(chatID)]
	return settings, ok
}

func (s *Store) ManagedChats(includeDisabled bool) []ChatSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]ChatSettings, 0, len(s.state.ManagedChats))
	for _, settings := range s.state.ManagedChats {
		if includeDisabled || settings.Enabled {
			out = append(out, settings)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Title == out[j].Title {
			return out[i].ChatID < out[j].ChatID
		}
		return out[i].Title < out[j].Title
	})
	return out
}

func (s *Store) DeleteChatSettings(chatID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.state.ManagedChats, idKey(chatID))
	return s.saveLocked()
}

func (s *Store) DiscussionReplyPosted(chatID int64, topicID int) bool {
	if topicID == 0 {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return discussionReplyPosted(s.state.DiscussionReplies, chatID, topicID)
}

func (s *Store) MarkDiscussionReply(chatID int64, topicID int, now time.Time) (bool, error) {
	if topicID == 0 {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if discussionReplyPosted(s.state.DiscussionReplies, chatID, topicID) {
		return false, nil
	}
	s.state.DiscussionReplies = append(s.state.DiscussionReplies, DiscussionReplyRecord{
		ChatID:     chatID,
		TopicID:    topicID,
		PostedUnix: now.UTC().Unix(),
	})
	if err := s.saveLocked(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) Activity(chatID, userID int64) (MemberRecord, bool) {
	return s.Member(chatID, userID)
}

func (s *Store) TopActive(chatID int64, limit int) []MemberRecord {
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

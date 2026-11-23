package main

import (
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (s *Store) migrateStateLocked(now time.Time) bool {
	changed := false
	if s.state.Chats == nil {
		s.state.Chats = make(map[string]*ChatState)
		changed = true
	}
	if s.state.ManagedChats == nil {
		s.state.ManagedChats = make(map[string]ChatSettings)
		changed = true
	}
	if s.state.Mafia.RegisteredUsers == nil {
		s.state.Mafia.RegisteredUsers = make(map[string]MafiaRegisteredUser)
		changed = true
	}
	if s.state.Mafia.Rooms == nil {
		s.state.Mafia.Rooms = make(map[string]MafiaRoom)
		changed = true
	}
	if normalizeDiscussionReplies(&s.state.DiscussionReplies) {
		changed = true
	}
	for key, ch := range s.state.Chats {
		if ch == nil {
			ch = &ChatState{}
			s.state.Chats[key] = ch
			changed = true
		}
		if ch.Members == nil {
			ch.Members = make(map[string]MemberRecord)
			changed = true
		}
		if ch.Jailed == nil {
			ch.Jailed = make(map[string]ModerationRecord)
			changed = true
		}
		if ch.Muted == nil {
			ch.Muted = make(map[string]ModerationRecord)
			changed = true
		}
		if ch.ForwardBan == nil {
			ch.ForwardBan = make(map[string]ModerationRecord)
			changed = true
		}
		if ch.Banned == nil {
			ch.Banned = make(map[string]ModerationRecord)
			changed = true
		}
		if ch.Warnings == nil {
			ch.Warnings = make(map[string]WarningRecord)
			changed = true
		}
		if ch.Karma == nil {
			ch.Karma = make(map[string]KarmaTotal)
			changed = true
		}
		for recKey, rec := range ch.Jailed {
			if rec.StartedAtUnix == 0 {
				if rec.SinceUnix != 0 {
					rec.StartedAtUnix = rec.SinceUnix
				} else {
					rec.StartedAtUnix = now.Unix()
				}
				ch.Jailed[recKey] = rec
				changed = true
			}
		}
	}
	for key, settings := range s.state.ManagedChats {
		restoredFromState := settings.RehabTopicID == 0 && settings.PersistedRehabTopicID > 0
		persistedKnownTopic := settings.RehabTopicID > 0 && settings.PersistedRehabTopicID == 0
		next, normalized := normalizeChatSettings(settings, ChatSettings{})
		if normalized {
			s.state.ManagedChats[key] = next
			changed = true
			if restoredFromState && s.log != nil {
				s.log.Info("rehab topic restored", zap.Int64("chat", next.ChatID), zap.Int("topic", next.RehabTopicID))
			} else if persistedKnownTopic && s.log != nil {
				s.log.Info("rehab topic persisted", zap.Int64("chat", next.ChatID), zap.Int("topic", next.RehabTopicID))
			}
		}
	}
	return changed
}

func (s *Store) ensureChatLocked(chatID int64) *ChatState {
	if s.state.Chats == nil {
		s.state.Chats = make(map[string]*ChatState)
	}
	key := idKey(chatID)
	ch, ok := s.state.Chats[key]
	if !ok {
		ch = &ChatState{}
		s.state.Chats[key] = ch
	}
	if ch.Members == nil {
		ch.Members = make(map[string]MemberRecord)
	}
	if ch.Jailed == nil {
		ch.Jailed = make(map[string]ModerationRecord)
	}
	if ch.Muted == nil {
		ch.Muted = make(map[string]ModerationRecord)
	}
	if ch.ForwardBan == nil {
		ch.ForwardBan = make(map[string]ModerationRecord)
	}
	if ch.Banned == nil {
		ch.Banned = make(map[string]ModerationRecord)
	}
	if ch.Warnings == nil {
		ch.Warnings = make(map[string]WarningRecord)
	}
	if ch.Karma == nil {
		ch.Karma = make(map[string]KarmaTotal)
	}
	return ch
}

func (s *Store) UpsertMember(chatID int64, u *tele.User) {
	s.TrackMember(chatID, u, 0, 0, false)
}

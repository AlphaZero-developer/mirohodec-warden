package main

import (
	"sort"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (s *Store) TrackMember(chatID int64, u *tele.User, threadID, messageID int, countMessage bool) {
	if u == nil || u.ID == 0 {
		return
	}

	nowTime := time.Now().UTC()
	now := nowTime.Unix()
	activityDay := nowTime.Format("2006-01-02")
	next := memberFromUser(u, now)
	next.LastThreadID = threadID

	s.mu.Lock()
	defer s.mu.Unlock()

	ch := s.ensureChatLocked(chatID)
	old, exists := ch.Members[idKey(u.ID)]
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
		if next.ActivityDay != activityDay {
			next.ActivityDay = activityDay
			next.MessagesToday = 0
		}
		next.MessagesToday++
		next.LastMessage = now
		next.LastMessageID = messageID
		next.LastThreadID = threadID
	}

	profileChanged := !exists ||
		old.Username != next.Username ||
		old.FirstName != next.FirstName ||
		old.LastName != next.LastName ||
		old.DisplayName != next.DisplayName ||
		old.IsBot != next.IsBot
	seenFresh := exists && now-old.LastSeen < int64((5*time.Minute).Seconds())

	if exists &&
		!countMessage &&
		!profileChanged &&
		seenFresh {
		return
	}
	ch.Members[idKey(u.ID)] = next

	shouldSave := !exists || profileChanged || !seenFresh
	if countMessage && nowTime.Sub(s.lastActivitySave) >= 15*time.Second {
		shouldSave = true
	}
	if shouldSave {
		s.lastActivitySave = nowTime
		if err := s.saveLocked(); err != nil {
			s.logError("store save member failed", err)
		}
	}
}

func (s *Store) Member(chatID, userID int64) (MemberRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ch := s.state.Chats[idKey(chatID)]
	if ch == nil {
		return MemberRecord{}, false
	}
	rec, ok := ch.Members[idKey(userID)]
	return rec, ok
}

func (s *Store) UserByUsername(chatID int64, username string) (targetUser, bool) {
	username = normalizeUsername(username)
	if username == "" {
		return targetUser{}, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	ch := s.state.Chats[idKey(chatID)]
	if ch == nil {
		return targetUser{}, false
	}

	for _, rec := range ch.Members {
		if normalizeUsername(rec.Username) == username {
			return targetUser{
				ID:       rec.ID,
				Username: rec.Username,
				Name:     rec.DisplayName,
				IsBot:    rec.IsBot,
			}, true
		}
	}

	for _, records := range []map[string]ModerationRecord{ch.Jailed, ch.Muted, ch.Banned} {
		for _, rec := range records {
			if normalizeUsername(rec.Username) == username {
				name := rec.DisplayName
				if strings.TrimSpace(name) == "" {
					name = "@" + rec.Username
				}
				return targetUser{
					ID:       rec.UserID,
					Username: rec.Username,
					Name:     name,
				}, true
			}
		}
	}

	return targetUser{}, false
}

func (s *Store) Members(chatID int64) []MemberRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ch := s.state.Chats[idKey(chatID)]
	if ch == nil {
		return nil
	}

	members := make([]MemberRecord, 0, len(ch.Members))
	for _, rec := range ch.Members {
		if rec.ID != 0 && !rec.IsBot {
			members = append(members, rec)
		}
	}
	sort.Slice(members, func(i, j int) bool {
		return members[i].DisplayName < members[j].DisplayName
	})
	return members
}

func (s *Store) ChatIDs() []int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[int64]struct{}, len(s.state.Chats)+len(s.state.ManagedChats))
	ids := make([]int64, 0, len(s.state.Chats)+len(s.state.ManagedChats))
	for _, settings := range s.state.ManagedChats {
		if settings.ChatID != 0 {
			seen[settings.ChatID] = struct{}{}
		}
	}
	for key := range s.state.Chats {
		id, err := strconv.ParseInt(key, 10, 64)
		if err == nil && id != 0 {
			seen[id] = struct{}{}
		}
	}
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

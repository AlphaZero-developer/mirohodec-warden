package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (s *Store) AddKarma(chatID, userID int64, delta int64, reason string, byUser *tele.User, messageID int, now time.Time) (int64, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := s.ensureChatLocked(chatID)
	key := idKey(userID)
	total := ch.Karma[key]
	total.UserID = userID
	if member, ok := ch.Members[key]; ok {
		total.DisplayName = member.DisplayName
		total.Username = member.Username
	}
	total.Score += delta
	total.UpdatedUnix = now.Unix()
	ch.Karma[key] = total

	event := KarmaEvent{
		ID:          "karma_" + strconv.FormatInt(now.UnixNano(), 36),
		UserID:      userID,
		Delta:       delta,
		Reason:      strings.TrimSpace(reason),
		MessageID:   messageID,
		Day:         now.Format("2006-01-02"),
		CreatedUnix: now.Unix(),
	}
	if byUser != nil {
		event.ByUserID = byUser.ID
		event.ByUserName = displayNameFromUser(byUser)
	}
	ch.KarmaEvents = append(ch.KarmaEvents, event)
	if len(ch.KarmaEvents) > 2000 {
		ch.KarmaEvents = ch.KarmaEvents[len(ch.KarmaEvents)-2000:]
	}
	if err := s.saveLocked(); err != nil {
		return 0, err
	}
	return total.Score, nil
}

func (s *Store) MaybeAwardActivityKarma(chatID, userID int64, messagesPerPoint, dailyLimit int, now time.Time) (int64, bool) {
	if messagesPerPoint <= 0 || dailyLimit <= 0 {
		return 0, false
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ch := s.ensureChatLocked(chatID)
	member, ok := ch.Members[idKey(userID)]
	if !ok || member.MessagesToday <= 0 {
		return 0, false
	}
	day := now.Format("2006-01-02")
	earnedToday := 0
	for _, event := range ch.KarmaEvents {
		if event.UserID == userID && event.Reason == "activity" && event.Day == day {
			earnedToday += int(event.Delta)
		}
	}
	allowedByMessages := int(member.MessagesToday) / messagesPerPoint
	if allowedByMessages > dailyLimit {
		allowedByMessages = dailyLimit
	}
	if earnedToday >= allowedByMessages {
		return 0, false
	}

	key := idKey(userID)
	total := ch.Karma[key]
	total.UserID = userID
	total.DisplayName = member.DisplayName
	total.Username = member.Username
	total.Score++
	total.UpdatedUnix = now.Unix()
	ch.Karma[key] = total
	ch.KarmaEvents = append(ch.KarmaEvents, KarmaEvent{
		ID:          "karma_" + strconv.FormatInt(now.UnixNano(), 36),
		UserID:      userID,
		Delta:       1,
		Reason:      "activity",
		Day:         day,
		CreatedUnix: now.Unix(),
	})
	if len(ch.KarmaEvents) > 2000 {
		ch.KarmaEvents = ch.KarmaEvents[len(ch.KarmaEvents)-2000:]
	}
	if err := s.saveLocked(); err != nil {
		s.logError("store save activity karma failed", err)
		return 0, false
	}
	return total.Score, true
}

func (s *Store) LastManualKarma(chatID, targetID, giverID int64, now time.Time) time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ch := s.state.Chats[idKey(chatID)]
	if ch == nil {
		return time.Time{}
	}
	var last time.Time
	for _, event := range ch.KarmaEvents {
		if event.UserID == targetID && event.ByUserID == giverID && event.Reason == "manual" && event.CreatedUnix > last.Unix() {
			last = time.Unix(event.CreatedUnix, 0).UTC()
		}
	}
	return last
}

func (s *Store) logError(msg string, err error) {
	if s.log != nil {
		s.log.Error(msg, zap.Error(err))
	}
}

func warningActive(rec WarningRecord, now time.Time) bool {
	if !rec.Active {
		return false
	}
	if rec.ExpiresUnix == 0 {
		return true
	}
	return rec.ExpiresUnix > now.Unix()
}

func discussionReplyPosted(records []DiscussionReplyRecord, chatID int64, topicID int) bool {
	for _, rec := range records {
		if rec.ChatID == chatID && rec.TopicID == topicID {
			return true
		}
	}
	return false
}

func normalizeDiscussionReplies(records *[]DiscussionReplyRecord) bool {
	if records == nil || len(*records) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(*records))
	out := (*records)[:0]
	changed := false
	for _, rec := range *records {
		if rec.ChatID == 0 || rec.TopicID == 0 {
			changed = true
			continue
		}
		key := fmt.Sprintf("%d:%d", rec.ChatID, rec.TopicID)
		if _, ok := seen[key]; ok {
			changed = true
			continue
		}
		seen[key] = struct{}{}
		out = append(out, rec)
	}
	if len(out) != len(*records) {
		changed = true
	}
	*records = out
	return changed
}

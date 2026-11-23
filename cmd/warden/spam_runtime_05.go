package main

import (
	"fmt"
	"strings"
	"time"
)

func countRepeatedSpamEvents(events []spamEvent, cutoff time.Time, kind, signature string) int {
	if signature == "" {
		return 0
	}
	count := 0
	for _, event := range events {
		if event.At.Before(cutoff) || event.Kind != kind || event.Signature != signature {
			continue
		}
		if event.Count > 0 {
			count += event.Count
		} else {
			count++
		}
	}
	return count
}

func maxRepeatedCurrentSpamEvents(all, current []spamEvent, cutoff time.Time, kind string, minSignatureLen int) int {
	maxCount := 0
	seen := make(map[string]struct{})
	for _, event := range current {
		if event.Kind != kind || event.Signature == "" {
			continue
		}
		if minSignatureLen > 1 && len([]rune(event.Signature)) < minSignatureLen {
			continue
		}
		if _, ok := seen[event.Signature]; ok {
			continue
		}
		seen[event.Signature] = struct{}{}
		count := countRepeatedSpamEvents(all, cutoff, kind, event.Signature)
		if count > maxCount {
			maxCount = count
		}
	}
	return maxCount
}

func hasRecentAlbumEvent(events []spamEvent, albumID string, cutoff time.Time) bool {
	for _, event := range events {
		if event.At.Before(cutoff) {
			continue
		}
		if event.Kind == spamKindMedia && event.AlbumID == albumID {
			return true
		}
	}
	return false
}

func (a *App) appendSpamLogLocked(bucket *spamBucket, entry spamLogEntry, limit int) {
	if limit <= 0 {
		limit = 25
	}
	bucket.Logs = append(bucket.Logs, entry)
	if len(bucket.Logs) > limit {
		bucket.Logs = bucket.Logs[len(bucket.Logs)-limit:]
	}
	a.spamLogs = append(a.spamLogs, entry)
	if len(a.spamLogs) > limit {
		a.spamLogs = a.spamLogs[len(a.spamLogs)-limit:]
	}
}

func (a *App) spamLogRecords(chatID, userID int64, limit int) []spamLogEntry {
	if limit <= 0 {
		limit = 25
	}
	a.spamMu.Lock()
	defer a.spamMu.Unlock()

	var source []spamLogEntry
	if userID != 0 {
		if bucket := a.spamBuckets[fmt.Sprintf("%d:%d", chatID, userID)]; bucket != nil {
			source = bucket.Logs
		}
	} else {
		source = a.spamLogs
	}

	records := make([]spamLogEntry, 0, minInt(limit, len(source)))
	for i := len(source) - 1; i >= 0 && len(records) < limit; i-- {
		if source[i].ChatID != chatID {
			continue
		}
		records = append(records, source[i])
	}
	return records
}

func (a *App) spamStatus(chatID, userID int64) spamStatusSnapshot {
	now := time.Now().UTC()
	limits := spamLimitsFromConfig(a.config().Spam)
	a.spamMu.Lock()
	defer a.spamMu.Unlock()

	bucket := a.spamBuckets[fmt.Sprintf("%d:%d", chatID, userID)]
	if bucket == nil {
		return spamStatusSnapshot{}
	}

	status := spamStatusSnapshot{
		Text:     countSpamEvents(bucket.Events, now.Add(-limits.TextWindow), spamKindText),
		Media:    countSpamEvents(bucket.Events, now.Add(-limits.MediaWindow), spamKindMedia),
		Stickers: countSpamEvents(bucket.Events, now.Add(-limits.StickerWindow), spamKindSticker),
		Links:    countSpamEvents(bucket.Events, now.Add(-limits.LinkWindow), spamKindLink),
		Mentions: countSpamEvents(bucket.Events, now.Add(-limits.MentionWindow), spamKindMention),
		Forwards: countSpamEvents(bucket.Events, now.Add(-limits.ForwardWindow), spamKindForward),
		Repeated: maxRepeatedSpamEvents(bucket.Events, now.Add(-limits.RepeatWindow)),
	}
	for _, strike := range bucket.Strikes {
		if !strike.At.Before(now.Add(-limits.StrikeWindow)) {
			status.Strikes++
		}
	}
	if len(bucket.Logs) > 0 {
		last := bucket.Logs[len(bucket.Logs)-1]
		status.LastAction = last.Action
		status.LastReason = last.Reason
		status.LastAt = last.At
	}
	return status
}

func maxRepeatedSpamEvents(events []spamEvent, cutoff time.Time) int {
	counts := make(map[string]int)
	maxCount := 0
	for _, event := range events {
		if event.At.Before(cutoff) || event.Signature == "" {
			continue
		}
		key := event.Kind + ":" + event.Signature
		if event.Count > 0 {
			counts[key] += event.Count
		} else {
			counts[key]++
		}
		if counts[key] > maxCount {
			maxCount = counts[key]
		}
	}
	return maxCount
}

func (a *App) clearSpamState(chatID, userID int64) {
	a.spamMu.Lock()
	defer a.spamMu.Unlock()
	delete(a.spamBuckets, fmt.Sprintf("%d:%d", chatID, userID))
}

func (a *App) isBotMute(rec ModerationRecord) bool {
	if a.bot.Me != nil && rec.ByAdminID == a.bot.Me.ID {
		return true
	}
	reason := strings.ToLower(rec.Reason)
	return strings.Contains(reason, "spam") || strings.Contains(reason, "спам")
}

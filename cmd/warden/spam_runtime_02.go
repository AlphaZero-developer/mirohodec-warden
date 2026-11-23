package main

import (
	"fmt"
	"mirohodec-warden/internal/domain/antispam"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) spamGuard(c tele.Context) bool {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || sender.IsBot || chat.Type == tele.ChatPrivate {
		return false
	}
	cfg := a.activeSpamConfigForChat(chat.ID)
	if !cfg.Enabled {
		return false
	}
	if a.isSpamExempt(chat.ID, sender.ID) {
		return false
	}

	now := time.Now().UTC()
	limits := spamLimitsFromConfig(cfg)
	events := spamEventsFromMessage(msg, now)
	if len(events) == 0 {
		return false
	}

	key := fmt.Sprintf("%d:%d", chat.ID, sender.ID)
	var entry spamLogEntry
	var shouldMute bool
	var shouldDelete bool
	action := "log"
	reason := ""
	reasonKind := ""
	repeatCount := 0

	a.spamMu.Lock()
	bucket := a.spamBuckets[key]
	if bucket == nil {
		bucket = &spamBucket{}
		a.spamBuckets[key] = bucket
	}

	filteredEvents := events[:0]
	for _, event := range events {
		if event.Count <= 0 {
			event.Count = 1
		}
		if event.Kind == spamKindMedia && event.AlbumID != "" && hasRecentAlbumEvent(bucket.Events, event.AlbumID, now.Add(-limits.MediaWindow)) {
			continue
		}
		filteredEvents = append(filteredEvents, event)
	}
	if len(filteredEvents) == 0 {
		a.spamMu.Unlock()
		return false
	}

	bucket.Events = pruneSpamEvents(bucket.Events, now.Add(-limits.maxWindow()))
	bucket.Strikes = pruneSpamStrikes(bucket.Strikes, now.Add(-limits.StrikeWindow))
	bucket.Events = append(bucket.Events, filteredEvents...)

	textCount := countSpamEvents(bucket.Events, now.Add(-limits.TextWindow), spamKindText)
	mediaCount := countSpamEvents(bucket.Events, now.Add(-limits.MediaWindow), spamKindMedia)
	stickerCount := countSpamEvents(bucket.Events, now.Add(-limits.StickerWindow), spamKindSticker)
	linkCount := countSpamEvents(bucket.Events, now.Add(-limits.LinkWindow), spamKindLink)
	mentionCount := countSpamEvents(bucket.Events, now.Add(-limits.MentionWindow), spamKindMention)
	forwardCount := countSpamEvents(bucket.Events, now.Add(-limits.ForwardWindow), spamKindForward)
	messageLinkCount := countSpamEvents(filteredEvents, time.Time{}, spamKindLink)
	messageMentionCount := countSpamEvents(filteredEvents, time.Time{}, spamKindMention)

	repeatedLinks := maxRepeatedCurrentSpamEvents(bucket.Events, filteredEvents, now.Add(-limits.RepeatWindow), spamKindLink, 1)
	repeatedStickers := maxRepeatedCurrentSpamEvents(bucket.Events, filteredEvents, now.Add(-limits.RepeatWindow), spamKindSticker, 1)
	repeatedText := maxRepeatedCurrentSpamEvents(bucket.Events, filteredEvents, now.Add(-limits.RepeatWindow), spamKindText, limits.MinRepeatedTextLen)

	rateMultiplier := a.spamRateMultiplier(chat.ID, sender.ID, now)
	maxTextMessages := applySpamRateMultiplier(limits.MaxTextMessages, rateMultiplier)
	maxMedia := applySpamRateMultiplier(limits.MaxMedia, rateMultiplier)
	maxStickers := applySpamRateMultiplier(limits.MaxStickers, rateMultiplier)

	highConfidence := false
	hardRateLimit := false
	switch {
	case limits.MaxLinksPerMessage > 0 && messageLinkCount > limits.MaxLinksPerMessage:
		reason = a.t("spam_reason_links_per_message")
		reasonKind = spamKindLink
		repeatCount = messageLinkCount
		highConfidence = true
	case limits.MaxMentionsPerMsg > 0 && messageMentionCount > limits.MaxMentionsPerMsg:
		reason = a.t("spam_reason_mentions_per_message")
		reasonKind = spamKindMention
		repeatCount = messageMentionCount
		highConfidence = true
	case limits.MaxRepeatedLinks > 0 && repeatedLinks > limits.MaxRepeatedLinks:
		reason = a.t("spam_reason_repeated_links")
		reasonKind = spamKindLink
		repeatCount = repeatedLinks
		highConfidence = true
	case limits.MaxRepeatedStickers > 0 && repeatedStickers > limits.MaxRepeatedStickers:
		reason = a.t("spam_reason_repeated_stickers")
		reasonKind = spamKindSticker
		repeatCount = repeatedStickers
		highConfidence = true
	case limits.MaxRepeatedText > 0 && repeatedText > limits.MaxRepeatedText:
		reason = a.t("spam_reason_repeated_text")
		reasonKind = spamKindText
		repeatCount = repeatedText
		highConfidence = true
	case maxMedia > 0 && mediaCount > maxMedia:
		reason = a.t("spam_reason_media")
		reasonKind = spamKindMedia
		hardRateLimit = mediaCount > maxMedia*2
	case limits.MaxLinks > 0 && linkCount > limits.MaxLinks:
		reason = a.t("spam_reason_links")
		reasonKind = spamKindLink
		highConfidence = true
	case limits.MaxMentions > 0 && mentionCount > limits.MaxMentions:
		reason = a.t("spam_reason_mentions")
		reasonKind = spamKindMention
		highConfidence = true
	case limits.MaxForwards > 0 && forwardCount > limits.MaxForwards:
		reason = a.t("spam_reason_forwards")
		reasonKind = spamKindForward
		highConfidence = true
	case maxStickers > 0 && stickerCount > maxStickers:
		reason = a.t("spam_reason_stickers")
		reasonKind = spamKindSticker
		hardRateLimit = stickerCount > maxStickers*2
	case maxTextMessages > 0 && textCount > maxTextMessages:
		reason = a.t("spam_reason_messages")
		reasonKind = spamKindText
		hardRateLimit = textCount > maxTextMessages*2
	}

	if reason == "" {
		a.spamMu.Unlock()
		return false
	}

	confidence := spamConfidenceLow
	if highConfidence {
		confidence = spamConfidenceHigh
	}
	bucket.Strikes = append(bucket.Strikes, spamStrike{At: now, Reason: reason, Confidence: confidence})
	strikeCount := len(bucket.Strikes)
	highStrikeCount := countSpamStrikes(bucket.Strikes, spamConfidenceHigh)
	if antispam.ShouldDelete(cfg.DeleteExcess, highConfidence, hardRateLimit, strikeCount) {
		shouldDelete = true
		action = "delete"
	}
	muteCooldown := limits.ActionCooldown > 0 && !bucket.LastMute.IsZero() && now.Sub(bucket.LastMute) < limits.ActionCooldown
	shouldMuteForHigh := highConfidence && highStrikeCount >= limits.MuteAfterStrikes
	shouldMuteForLow := !highConfidence && hardRateLimit && strikeCount >= limits.MuteAfterStrikes+2
	if cfg.AutoMute && (shouldMuteForHigh || shouldMuteForLow) && !muteCooldown {
		shouldMute = true
		shouldDelete = cfg.DeleteExcess
		action = "mute"
		bucket.LastMute = now
	}
	settings := a.effectiveChatSettings(chat)
	if shouldMuteNoRehabSpam(cfg, settings, settings.RehabTopicID, shouldDelete, shouldMute, muteCooldown) {
		shouldMute = true
		action = "mute"
		bucket.LastMute = now
	}
	if action != "log" {
		bucket.LastAction = now
	}

	counts := fmt.Sprintf("text=%d/%d media=%d/%d stickers=%d/%d links=%d mentions=%d forwards=%d repeat=%d strikes=%d high=%d", textCount, maxTextMessages, mediaCount, maxMedia, stickerCount, maxStickers, linkCount, mentionCount, forwardCount, repeatCount, strikeCount, highStrikeCount)
	entry = spamLogEntry{
		At:         now,
		ChatID:     chat.ID,
		UserID:     sender.ID,
		UserName:   displayNameFromUser(sender),
		Kind:       reasonKind,
		Confidence: confidence,
		Reason:     reason,
		Action:     action,
		Counts:     counts,
	}
	a.appendSpamLogLocked(bucket, entry, limits.LogLimit)
	a.spamMu.Unlock()

	return a.applySpamModeration(c, chat, sender, msg, cfg, entry, shouldDelete, shouldMute, reason, now)
}

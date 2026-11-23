package main

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) applySpamModeration(c tele.Context, chat *tele.Chat, sender *tele.User, msg *tele.Message, cfg SpamConfig, entry spamLogEntry, shouldDelete, shouldMute bool, reason string, now time.Time) bool {
	if a.config().Security.ProtectAdmins {
		if err := a.ensureModeratable(chat, sender.ID); err != nil {
			a.log.Warn("spam action skipped for protected user", zap.Int64("chat", chat.ID), zap.Int64("user", sender.ID), zap.Error(err))
			return false
		}
	}

	a.log.Warn("spam filter triggered",
		zap.Int64("chat", entry.ChatID),
		zap.Int64("user", entry.UserID),
		zap.String("kind", entry.Kind),
		zap.String("confidence", entry.Confidence),
		zap.String("action", entry.Action),
		zap.String("reason", entry.Reason),
		zap.String("counts", entry.Counts),
	)

	if shouldDelete {
		if err := c.Delete(); err != nil {
			a.log.Warn("delete spam message failed", zap.Int64("chat", chat.ID), zap.Int64("user", sender.ID), zap.Error(err))
		}
		a.metrics.IncDeletedSpamMessage()
		a.metrics.IncModeration("spam_delete")
	}

	if shouldMute {
		a.muteSpamUser(chat, sender, msg, cfg, reason, now)
	}
	if shouldDelete || shouldMute {
		a.maybeSendLayaReaction(chat, msg, "spam")
	}

	return shouldDelete
}

func shouldMuteNoRehabSpam(cfg SpamConfig, settings ChatSettings, rehabTopicID int, shouldDelete, alreadyMute, muteCooldown bool) bool {
	return cfg.AutoMute &&
		shouldDelete &&
		!alreadyMute &&
		rehabTopicID == 0 &&
		settings.SpamNoRehabAction == spamNoRehabMuteOnDelete &&
		!muteCooldown
}

func spamLimitsFromConfig(cfg SpamConfig) spamLimits {
	return spamLimits{
		TextWindow:          mustDuration(cfg.TextWindow, 8*time.Second),
		MaxTextMessages:     cfg.MaxTextMessages,
		MediaWindow:         mustDuration(cfg.MediaWindow, 30*time.Second),
		MaxMedia:            cfg.MaxMedia,
		StickerWindow:       mustDuration(cfg.StickerWindow, 10*time.Second),
		MaxStickers:         cfg.MaxStickers,
		RepeatWindow:        mustDuration(cfg.RepeatWindow, 20*time.Second),
		MaxRepeatedText:     cfg.MaxRepeatedText,
		MaxRepeatedStickers: cfg.MaxRepeatedStickers,
		MinRepeatedTextLen:  cfg.MinRepeatedTextLen,
		LinkWindow:          mustDuration(cfg.LinkWindow, 30*time.Second),
		MaxLinks:            cfg.MaxLinks,
		MaxRepeatedLinks:    cfg.MaxRepeatedLinks,
		MaxLinksPerMessage:  cfg.MaxLinksPerMessage,
		MentionWindow:       mustDuration(cfg.MentionWindow, 30*time.Second),
		MaxMentions:         cfg.MaxMentions,
		MaxMentionsPerMsg:   cfg.MaxMentionsPerMsg,
		ForwardWindow:       mustDuration(cfg.ForwardWindow, 45*time.Second),
		MaxForwards:         cfg.MaxForwards,
		ActionCooldown:      mustDuration(cfg.ActionCooldown, 20*time.Second),
		StrikeWindow:        mustDuration(cfg.StrikeWindow, 2*time.Minute),
		MuteAfterStrikes:    cfg.MuteAfterStrikes,
		LogLimit:            cfg.LogLimit,
	}
}

func (l spamLimits) maxWindow() time.Duration {
	maxWindow := l.TextWindow
	for _, d := range []time.Duration{l.MediaWindow, l.StickerWindow, l.LinkWindow, l.MentionWindow, l.ForwardWindow, l.RepeatWindow, l.StrikeWindow} {
		if d > maxWindow {
			maxWindow = d
		}
	}
	return maxWindow
}

func (a *App) spamRateMultiplier(chatID, userID int64, now time.Time) int {
	rec, ok := a.store.Member(chatID, userID)
	if !ok {
		return 1
	}
	if rec.Messages >= 30 && rec.LastMessage >= now.Add(-14*24*time.Hour).Unix() {
		return 2
	}
	return 1
}

func applySpamRateMultiplier(limit, multiplier int) int {
	if limit <= 0 {
		return limit
	}
	if multiplier < 1 {
		multiplier = 1
	}
	return limit * multiplier
}

func spamEventsFromMessage(msg *tele.Message, now time.Time) []spamEvent {
	if msg == nil {
		return nil
	}

	events := make([]spamEvent, 0, 4)
	if isForwardedMessage(msg) {
		events = append(events, spamEvent{At: now, Kind: spamKindForward, Signature: forwardedSpamSignature(msg), Count: 1})
	}
	for _, signature := range spamLinkSignatures(msg) {
		events = append(events, spamEvent{At: now, Kind: spamKindLink, Signature: signature, Count: 1})
	}
	for _, signature := range spamMentionSignatures(msg) {
		events = append(events, spamEvent{At: now, Kind: spamKindMention, Signature: signature, Count: 1})
	}

	base := spamEvent{At: now, Kind: spamKindOther, Count: 1}
	switch {
	case msg.Sticker != nil:
		base.Kind = spamKindSticker
		base.Signature = msg.Sticker.UniqueID
		if base.Signature == "" {
			base.Signature = msg.Sticker.FileID
		}
		if base.Signature == "" {
			base.Signature = strings.TrimSpace(msg.Sticker.SetName + ":" + msg.Sticker.Emoji)
		}
	case isMediaMessage(msg):
		base.Kind = spamKindMedia
		base.AlbumID = msg.AlbumID
		base.Signature = base.Kind
	case strings.TrimSpace(msg.Text) != "" || strings.TrimSpace(msg.Caption) != "":
		base.Kind = spamKindText
		base.Signature = normalizeSpamText(msg.Text + " " + msg.Caption)
	case isOtherSpamCandidate(msg):
		base.Kind = spamKindText
		base.Signature = spamKindOther
	default:
		return events
	}
	events = append(events, base)
	return events
}

func isSpamCandidate(msg *tele.Message) bool {
	return len(spamEventsFromMessage(msg, time.Now().UTC())) > 0
}

func isForwardedMessage(msg *tele.Message) bool {
	return msg.OriginalSender != nil ||
		msg.OriginalChat != nil ||
		msg.OriginalMessageID != 0 ||
		msg.OriginalSignature != "" ||
		msg.OriginalSenderName != "" ||
		msg.OriginalUnixtime != 0 ||
		msg.Origin != nil
}

func forwardedSpamSignature(msg *tele.Message) string {
	switch {
	case msg.Origin != nil && msg.Origin.Chat != nil:
		return fmt.Sprintf("chat:%d", msg.Origin.Chat.ID)
	case msg.Origin != nil && msg.Origin.SenderChat != nil:
		return fmt.Sprintf("senderchat:%d", msg.Origin.SenderChat.ID)
	case msg.Origin != nil && msg.Origin.Sender != nil:
		return fmt.Sprintf("user:%d", msg.Origin.Sender.ID)
	case msg.OriginalChat != nil:
		return fmt.Sprintf("chat:%d", msg.OriginalChat.ID)
	case msg.OriginalSender != nil:
		return fmt.Sprintf("user:%d", msg.OriginalSender.ID)
	case msg.OriginalSenderName != "":
		return "name:" + normalizeSpamText(msg.OriginalSenderName)
	default:
		return spamKindForward
	}
}

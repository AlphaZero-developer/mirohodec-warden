package main

import (
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) maybeHandleDiscussionAutoReply(c tele.Context) bool {
	if a == nil || a.bot == nil || a.store == nil || c == nil {
		return false
	}
	cfg := a.activeDiscussionConfig()
	if !cfg.Enabled {
		return false
	}
	text := strings.TrimSpace(cfg.Text)
	if text == "" {
		return false
	}

	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if !isDiscussionAutoReplyTargetChat(chat, cfg) {
		return false
	}
	if !a.isDiscussionAutoReplyPost(chat, msg, sender) {
		return false
	}
	markerID := discussionAutoReplyMarkerID(msg)
	if markerID == 0 {
		return false
	}

	a.discussionMu.Lock()
	defer a.discussionMu.Unlock()
	if a.store.DiscussionReplyPosted(chat.ID, markerID) {
		return false
	}

	sendOpts := discussionAutoReplySendOptions(msg)
	sendOpts.ReplyMarkup = discussionAutoReplyButton(cfg.ChatButtonText, cfg.ChatButtonURL)
	sent, err := a.bot.Send(chat, text, sendOpts)
	if err != nil {
		if a.log != nil {
			a.log.Warn("discussion auto-reply send failed",
				zap.Int64("chat", chat.ID),
				zap.Int("marker", markerID),
				zap.Error(err),
			)
		}
		return false
	}
	if _, err := a.store.MarkDiscussionReply(chat.ID, markerID, time.Now().UTC()); err != nil && a.log != nil {
		a.log.Warn("discussion auto-reply mark failed",
			zap.Int64("chat", chat.ID),
			zap.Int("marker", markerID),
			zap.Error(err),
		)
	}
	for _, followUp := range discussionAutoReplyFollowUps(cfg) {
		opts := discussionAutoReplySendOptions(msg)
		if followUp.ReplyMarkup != nil {
			opts.ReplyMarkup = followUp.ReplyMarkup
		}
		if _, err := a.bot.Send(chat, followUp.Text, opts); err != nil && a.log != nil {
			a.log.Warn("discussion auto-reply follow-up send failed",
				zap.Int64("chat", chat.ID),
				zap.Int("marker", markerID),
				zap.Error(err),
			)
		}
	}
	if cfg.Pin && sent != nil {
		if err := a.bot.Pin(sent); err != nil && a.log != nil {
			a.log.Warn("discussion auto-reply pin failed",
				zap.Int64("chat", chat.ID),
				zap.Int("marker", markerID),
				zap.Error(err),
			)
		}
	}
	if a.log != nil {
		a.log.Info("discussion auto-reply sent",
			zap.Int64("chat", chat.ID),
			zap.Int("marker", markerID),
			zap.Bool("pinned", cfg.Pin),
		)
	}
	if a.metrics != nil {
		a.metrics.IncModeration("discussion_auto_reply")
	}
	return true
}

func isDiscussionAutoReplyTargetChat(chat *tele.Chat, cfg DiscussionAutoReplyConfig) bool {
	if chat == nil {
		return false
	}
	return cfg.ChatID == 0 || chat.ID == cfg.ChatID
}

func discussionAutoReplyMarkerID(msg *tele.Message) int {
	if msg == nil {
		return 0
	}
	if msg.ThreadID != 0 {
		return msg.ThreadID
	}
	return msg.ID
}

func discussionAutoReplySendOptions(msg *tele.Message) *tele.SendOptions {
	opts := &tele.SendOptions{}
	if msg == nil {
		return opts
	}
	if msg.ThreadID != 0 {
		opts.ThreadID = msg.ThreadID
		return opts
	}
	opts.ReplyTo = msg
	return opts
}

type discussionAutoReplyMessage struct {
	Text        string
	ReplyMarkup *tele.ReplyMarkup
}

func discussionAutoReplyFollowUps(cfg DiscussionAutoReplyConfig) []discussionAutoReplyMessage {
	text := strings.TrimSpace(cfg.FollowUpText)
	if text == "" {
		return nil
	}
	return []discussionAutoReplyMessage{{
		Text:        text,
		ReplyMarkup: discussionAutoReplyButton(cfg.FollowUpButtonText, cfg.FollowUpButtonURL),
	}}
}

func discussionAutoReplyButton(text, rawURL string) *tele.ReplyMarkup {
	text = strings.TrimSpace(text)
	rawURL = strings.TrimSpace(rawURL)
	if text == "" || !isHTTPURL(rawURL) {
		return nil
	}
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.URL(text, rawURL)))
	return markup
}

func isHTTPURL(rawURL string) bool {
	rawURL = strings.ToLower(strings.TrimSpace(rawURL))
	return strings.HasPrefix(rawURL, "https://") || strings.HasPrefix(rawURL, "http://")
}

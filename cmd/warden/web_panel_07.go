package main

import (
	"context"
	"errors"
	"fmt"
	wardenweb "mirohodec-warden/internal/web"
	"strconv"
	"strings"
	"time"
)

func webAILogChat(entry aiWebLogEntry) string {
	if strings.TrimSpace(entry.ChatTitle) != "" && entry.ChatID != 0 {
		return fmt.Sprintf("%s (%d)", strings.TrimSpace(entry.ChatTitle), entry.ChatID)
	}
	if strings.TrimSpace(entry.ChatTitle) != "" {
		return strings.TrimSpace(entry.ChatTitle)
	}
	if entry.ChatID != 0 {
		return strconv.FormatInt(entry.ChatID, 10)
	}
	return "-"
}

func webFormatTime(ts time.Time) string {
	if ts.IsZero() {
		return "-"
	}
	return ts.UTC().Format(time.RFC3339)
}

func webFormatDuration(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	return d.Truncate(time.Millisecond).String()
}

func (s webPanelService) Discussion(ctx context.Context) (wardenweb.DiscussionPage, error) {
	if s.app == nil {
		return wardenweb.DiscussionPage{}, errors.New("app is not ready")
	}
	base := s.app.config().Discussion
	cfg := s.app.activeDiscussionConfig()
	runtime := DiscussionRuntimeState{}
	if s.app.store != nil {
		runtime = normalizeDiscussionRuntimeState(s.app.store.DiscussionRuntimeState())
	}
	rows := []wardenweb.GroupSetting{
		{Section: "Scope", Name: "enabled", Value: webBool(cfg.Enabled), Note: "Turns comment auto-replies on or off."},
		{Section: "Scope", Name: "comment chat id", Value: s.webChatIDLabel(cfg.ChatID, "any managed group"), Note: "Group/topic where Laya writes the comment replies."},
		{Section: "Scope", Name: "watched source id", Value: s.webChatIDLabel(cfg.SourceChatID, "any source channel/group"), Note: "Channel/group whose forwarded posts Laya watches. Set this to lock replies to one source."},
		{Section: "Scope", Name: "managed chat gate", Value: "enabled groups only", Note: "Middleware calls this feature only for managed chats that are enabled in the panel/store."},
		{Section: "Message", Name: "chat text", Value: textOrDash(cfg.Text), Note: "First message in the channel-post topic."},
		{Section: "Button", Name: "chat button text", Value: textOrDash(cfg.ChatButtonText), Note: "Inline button label under the first message."},
		{Section: "Button", Name: "chat button url", Value: textOrDash(cfg.ChatButtonURL), Note: "Only http/https URLs are sent to Telegram."},
		{Section: "Message", Name: "follow-up text", Value: textOrDash(cfg.FollowUpText), Note: "Second message after the chat link."},
		{Section: "Button", Name: "follow-up button text", Value: textOrDash(cfg.FollowUpButtonText), Note: "Inline button label under the follow-up."},
		{Section: "Button", Name: "follow-up button url", Value: textOrDash(cfg.FollowUpButtonURL), Note: "Only http/https URLs are sent to Telegram."},
		{Section: "Telegram", Name: "pin first message", Value: webBool(cfg.Pin), Note: "Pins the first chat-link message when Telegram allows it."},
		{Section: "Checks", Name: "strict checks", Value: webBool(cfg.StrictChecks), Note: "When enabled, only real Telegram automatic forwards are accepted; admin/manual fallbacks are disabled."},
		{Section: "Checks", Name: "marker", Value: "topic id or message id", Note: "Forum topics use message_thread_id; normal linked comments use the forwarded message_id."},
		{Section: "Checks", Name: "root post only", Value: "yes", Note: "Replies and commands are ignored, so normal comments do not trigger the link."},
		{Section: "Checks", Name: "source lock", Value: discussionSourceLockText(cfg), Note: "If watched source id is set, the forwarded source must match exactly."},
		{Section: "Checks", Name: "fallback sources", Value: discussionFallbackText(cfg), Note: "Strict mode disables admin/manual fallbacks."},
		{Section: "Checks", Name: "send limit", Value: "once per topic", Note: "Already answered topics are stored and skipped."},
	}
	updated := ""
	if runtime.UpdatedUnix != 0 {
		updated = webFormatUnix(runtime.UpdatedUnix)
	}
	return wardenweb.DiscussionPage{
		Rows:    rows,
		Active:  webDiscussionSettings(cfg),
		Config:  webDiscussionSettings(base),
		Runtime: webDiscussionSettings(discussionConfigFromRuntime(runtime)),
		Updated: updated,
	}, nil
}

func (s webPanelService) UpdateDiscussion(ctx context.Context, input wardenweb.DiscussionInput) error {
	if s.app == nil || s.app.store == nil {
		return errors.New("app store is not ready")
	}
	cfg := DiscussionAutoReplyConfig{
		Enabled:            input.Enabled,
		ChatID:             input.ChatID,
		SourceChatID:       input.SourceChatID,
		StrictChecks:       input.StrictChecks,
		Text:               input.Text,
		ChatButtonText:     input.ChatButtonText,
		ChatButtonURL:      input.ChatButtonURL,
		FollowUpText:       input.FollowUpText,
		FollowUpButtonText: input.FollowUpButtonText,
		FollowUpButtonURL:  input.FollowUpButtonURL,
		Pin:                input.Pin,
	}
	if strings.TrimSpace(cfg.Text) == "" {
		return errors.New("текст первого сообщения обязателен")
	}
	state := discussionRuntimeStateFromConfig(cfg, time.Now().UTC())
	return s.app.store.SetDiscussionRuntimeState(state)
}

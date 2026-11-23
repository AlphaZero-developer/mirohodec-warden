package main

import (
	"errors"
	"fmt"
	wardenweb "mirohodec-warden/internal/web"
	"strconv"
	"strings"
	"time"
)

func validateContentDurationInput(name, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if _, err := parseDuration(value); err != nil {
		return fmt.Errorf("%s has invalid duration: %w", name, err)
	}
	return nil
}

func webIntList(values []int) string {
	if len(values) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, ", ")
}

func webStringList(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ", ")
}

func brainwashLeaderSetting(cfg RehabBrainwashConfig) string {
	cfg = normalizeRehabBrainwashConfig(cfg)
	if cfg.LeaderUserID == 0 {
		return cfg.LeaderName
	}
	return fmt.Sprintf("%s (%d)", cfg.LeaderName, cfg.LeaderUserID)
}

func webDiscussionChatID(chatID int64) string {
	if chatID == 0 {
		return "any managed group"
	}
	return strconv.FormatInt(chatID, 10)
}

func (s webPanelService) webChatIDLabel(chatID int64, zeroLabel string) string {
	if chatID == 0 {
		return zeroLabel
	}
	label := strconv.FormatInt(chatID, 10)
	if s.app != nil && s.app.store != nil {
		if settings, ok := s.app.store.ChatSettings(chatID); ok && strings.TrimSpace(settings.Title) != "" {
			label += " (" + strings.TrimSpace(settings.Title) + ")"
		}
	}
	return label
}

func discussionButtonLabel(text, rawURL string) string {
	text = strings.TrimSpace(text)
	rawURL = strings.TrimSpace(rawURL)
	if text == "" && rawURL == "" {
		return "-"
	}
	if text == "" {
		return rawURL
	}
	if rawURL == "" {
		return text
	}
	return text + " -> " + rawURL
}

func webDiscussionSettings(cfg DiscussionAutoReplyConfig) wardenweb.DiscussionSettings {
	return wardenweb.DiscussionSettings{
		Enabled:            cfg.Enabled,
		ChatID:             cfg.ChatID,
		SourceChatID:       cfg.SourceChatID,
		StrictChecks:       cfg.StrictChecks,
		Text:               cfg.Text,
		ChatButtonText:     cfg.ChatButtonText,
		ChatButtonURL:      cfg.ChatButtonURL,
		FollowUpText:       cfg.FollowUpText,
		FollowUpButtonText: cfg.FollowUpButtonText,
		FollowUpButtonURL:  cfg.FollowUpButtonURL,
		Pin:                cfg.Pin,
	}
}

func discussionSourceLockText(cfg DiscussionAutoReplyConfig) string {
	if cfg.SourceChatID == 0 {
		return "off"
	}
	return "exact id " + strconv.FormatInt(cfg.SourceChatID, 10)
}

func discussionFallbackText(cfg DiscussionAutoReplyConfig) string {
	if cfg.StrictChecks {
		return "automatic forward only"
	}
	return "channel source or configured admin"
}

func (s webPanelService) syncMenus() {
	if s.app != nil && s.app.bot != nil {
		go s.app.syncCommandMenus()
	}
}

func moderationRow(kind string, rec ModerationRecord) wardenweb.ModerationRow {
	return wardenweb.ModerationRow{
		Kind:        kind,
		UserID:      rec.UserID,
		DisplayName: rec.DisplayName,
		Username:    rec.Username,
		Reason:      rec.Reason,
		Since:       webFormatUnix(rec.SinceUnix),
		Until:       webReleaseLine(rec.UntilUnix),
		By:          rec.ByAdminName,
	}
}

func sleepChatLabel(call OwnerSleepCall) string {
	title := strings.TrimSpace(call.ChatTitle)
	if title == "" && call.ChatID != 0 {
		title = strconv.FormatInt(call.ChatID, 10)
	}
	if title == "" {
		return "-"
	}
	return title
}

func sleepUserLabel(call OwnerSleepCall) string {
	name := strings.TrimSpace(call.FromName)
	if name == "" && call.FromUsername != "" {
		name = "@" + call.FromUsername
	}
	if name == "" && call.FromID != 0 {
		name = strconv.FormatInt(call.FromID, 10)
	}
	if name == "" {
		return "-"
	}
	return name
}

func parseWebDuration(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	d, permanent, ok, err := parseDurationToken(value)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, errors.New("invalid duration")
	}
	if permanent || d == 0 {
		return 0, nil
	}
	return time.Now().UTC().Add(d).Unix(), nil
}

func webReleaseLine(untilUnix int64) string {
	if untilUnix == 0 {
		return "вручную"
	}
	return webFormatUnix(untilUnix)
}

func webFormatUnix(ts int64) string {
	if ts == 0 {
		return "-"
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

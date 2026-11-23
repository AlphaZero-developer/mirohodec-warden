package main

import (
	"context"
	"errors"
	"fmt"
	wardenweb "mirohodec-warden/internal/web"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (s webPanelService) SetGroupRehabTopic(ctx context.Context, chatID int64, topicID int) error {
	if chatID == 0 || topicID == 0 {
		return errors.New("chat_id and topic_id are required")
	}
	settings, ok := s.app.store.ChatSettings(chatID)
	if !ok {
		settings = ChatSettings{ChatID: chatID, Enabled: true, SpamNoRehabAction: spamNoRehabMuteOnDelete}
	}
	settings.RehabTopicID = topicID
	if settings.SpamNoRehabAction == "" {
		settings.SpamNoRehabAction = spamNoRehabMuteOnDelete
	}
	if err := s.app.store.UpsertChatSettings(settings); err != nil {
		return err
	}
	s.syncMenus()
	return nil
}

func (s webPanelService) Rehab(ctx context.Context, chatID int64) (wardenweb.RehabPage, error) {
	chatID = s.resolveChatID(chatID)
	chat := &tele.Chat{ID: chatID, Type: tele.ChatSuperGroup}
	page := wardenweb.RehabPage{
		ChatID:       chatID,
		RehabTopicID: s.app.rehabTopicID(chat),
		Brainwash:    s.webRehabBrainwashSettings(),
		Exam:         s.webRehabReleaseExamSettingsForChat(chatID),
	}
	for _, member := range s.app.store.TopActive(chatID, 200) {
		page.Users = append(page.Users, wardenweb.UserChoice{ID: member.ID, Label: member.DisplayName})
	}
	for _, rec := range s.app.store.ModerationRecords(chatID, "jail", true, time.Now().UTC()) {
		page.Rows = append(page.Rows, moderationRow("rehab", rec))
	}
	return page, nil
}

func (s webPanelService) AddRehab(ctx context.Context, input wardenweb.RehabInput) error {
	if input.ChatID == 0 || input.UserID == 0 {
		return errors.New("chat_id and user_id are required")
	}
	chat := &tele.Chat{ID: input.ChatID, Type: tele.ChatSuperGroup}
	if s.app.rehabTopicID(chat) == 0 {
		return errors.New("topic for psychiatric center is not configured")
	}
	untilUnix, err := parseWebDuration(input.Duration)
	if err != nil {
		return err
	}
	member, _ := s.app.store.Member(input.ChatID, input.UserID)
	name := strings.TrimSpace(member.DisplayName)
	if name == "" {
		name = fmt.Sprintf("user %d", input.UserID)
	}
	rec := ModerationRecord{
		UserID:      input.UserID,
		Username:    member.Username,
		DisplayName: name,
		Reason:      strings.TrimSpace(input.Reason),
		ByAdminName: "web panel",
		SinceUnix:   time.Now().UTC().Unix(),
		UntilUnix:   untilUnix,
	}
	rec.RehabWard = s.app.chooseRehabWard(input.ChatID, targetUser{ID: input.UserID, Username: member.Username, Name: name}, rec.Reason, "", time.Now().UTC())
	s.app.store.SetJail(input.ChatID, rec)
	s.app.clearRehabEscapeCooldown(input.ChatID, rec.UserID)
	s.app.showRehabEscapeCommand(input.ChatID, rec.UserID)
	if s.app.bot != nil {
		s.app.sendRehabNotice(chat, rec)
		s.app.sendOwnerLog("Rehab", logField{"User", fmt.Sprintf("%s (%d)", name, input.UserID)}, logField{"Chat", strconv.FormatInt(input.ChatID, 10)}, logField{"Release", webReleaseLine(untilUnix)}, logField{"Reason", textOrDash(rec.Reason)}, logField{"By", "web panel"})
	}
	return nil
}

func (s webPanelService) ReleaseRehab(ctx context.Context, chatID, userID int64) error {
	if chatID == 0 || userID == 0 {
		return errors.New("chat_id and user_id are required")
	}
	now := time.Now().UTC()
	if rec, ok := s.app.jailRecord(chatID, userID, now); ok && s.app.maybeStartRehabReleaseExam(&tele.Chat{ID: chatID, Type: tele.ChatSuperGroup}, rec, "web", nil) {
		return nil
	}
	rec, released := s.app.store.RemoveJail(chatID, userID)
	if released {
		s.app.hideRehabEscapeCommand(chatID, userID)
		s.app.clearRehabEscapeCooldown(chatID, userID)
	}
	duration := s.app.rehabReleaseDurationText(rec, released, now)
	if s.app.bot != nil {
		if released {
			s.app.logRehabRelease(chatID, rec, duration, "web")
			s.app.sendRehabReleaseNotice(&tele.Chat{ID: chatID, Type: tele.ChatSuperGroup}, rec, duration, 0)
			s.app.schedulePostRehabCheck(chatID, rec, now)
		}
		s.app.sendOwnerLog("Rehab released", logField{"User ID", strconv.FormatInt(userID, 10)}, logField{"Chat", strconv.FormatInt(chatID, 10)}, logField{"Duration", duration}, logField{"By", "web panel"})
	}
	return nil
}

func (s webPanelService) webRehabBrainwashSettings() wardenweb.RehabBrainwashSettings {
	cfg := s.app.activeRehabBrainwashConfig()
	updated := ""
	if cfg.UpdatedUnix != 0 {
		updated = webFormatUnix(cfg.UpdatedUnix)
	}
	return wardenweb.RehabBrainwashSettings{
		Enabled:      cfg.Enabled,
		AIEnabled:    cfg.AIEnabled,
		LeaderUserID: cfg.LeaderUserID,
		LeaderName:   cfg.LeaderName,
		AIPrompt:     cfg.AIPrompt,
		Messages:     cfg.Messages,
		Updated:      updated,
	}
}

func (s webPanelService) UpdateRehabBrainwash(ctx context.Context, input wardenweb.RehabBrainwashInput) error {
	if s.app == nil || s.app.store == nil {
		return errors.New("app store is not ready")
	}
	cfg := RehabBrainwashConfig{
		Enabled:      input.Enabled,
		AIEnabled:    input.AIEnabled,
		LeaderUserID: input.LeaderUserID,
		LeaderName:   input.LeaderName,
		AIPrompt:     input.AIPrompt,
		Messages:     input.Messages,
	}
	state := rehabBrainwashRuntimeStateFromConfig(cfg, input.UpdatedByID, input.UpdatedByName, time.Now().UTC())
	return s.app.store.SetRehabBrainwashRuntimeState(state)
}

func (s webPanelService) webRehabReleaseExamSettings() wardenweb.RehabReleaseExamSettings {
	return s.webRehabReleaseExamSettingsForChat(0)
}

func (s webPanelService) webRehabReleaseExamSettingsForChat(chatID int64) wardenweb.RehabReleaseExamSettings {
	cfg := s.app.activeRehabReleaseExamConfigForChat(chatID)
	scope := "global"
	if chatID != 0 && s.app != nil && s.app.store != nil {
		if _, ok := s.app.store.RehabReleaseExamRuntimeStateForChat(chatID); ok {
			scope = "group"
		}
	}
	updated := ""
	if cfg.UpdatedUnix != 0 {
		updated = webFormatUnix(cfg.UpdatedUnix)
	}
	return wardenweb.RehabReleaseExamSettings{
		Enabled:      cfg.Enabled,
		Questions:    cfg.Questions,
		AnswerWindow: cfg.AnswerWindow,
		FailDuration: cfg.FailDuration,
		Scope:        scope,
		Updated:      updated,
	}
}

func (s webPanelService) UpdateRehabReleaseExam(ctx context.Context, input wardenweb.RehabReleaseExamInput) error {
	if s.app == nil || s.app.store == nil {
		return errors.New("app store is not ready")
	}
	cfg := RehabReleaseExamConfig{
		Enabled:      input.Enabled,
		Questions:    input.Questions,
		AnswerWindow: input.AnswerWindow,
		FailDuration: input.FailDuration,
	}
	state := rehabReleaseExamRuntimeStateFromConfig(cfg, input.UpdatedByID, input.UpdatedByName, time.Now().UTC())
	if strings.EqualFold(strings.TrimSpace(input.Scope), "group") {
		if input.ChatID == 0 {
			return errors.New("chat_id is required for group exam settings")
		}
		return s.app.store.SetRehabReleaseExamRuntimeStateForChat(input.ChatID, state)
	}
	return s.app.store.SetRehabReleaseExamRuntimeState(state)
}

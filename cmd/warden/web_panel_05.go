package main

import (
	"context"
	"errors"
	"fmt"
	wardenweb "mirohodec-warden/internal/web"
	"sort"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (s webPanelService) UpdateLayaControl(ctx context.Context, input wardenweb.LayaControlInput) error {
	if s.app == nil || s.app.store == nil {
		return errors.New("app store is not ready")
	}
	relationships, err := parseLayaRelationshipsText(input.Relationships)
	if err != nil {
		return err
	}
	state := LayaRuntimeState{
		SystemPrompt:         input.SystemPrompt,
		ExtraPrompt:          input.ExtraPrompt,
		FallbackReply:        input.FallbackReply,
		BaseURL:              input.BaseURL,
		Model:                input.Model,
		Timeout:              input.Timeout,
		SoftTimeout:          input.SoftTimeout,
		FasMuteDuration:      input.FasMute,
		MaxTokens:            input.MaxTokens,
		MaxReplyChars:        input.MaxReplyChars,
		LogConversationsMode: input.LogConversations,
		FatherUserID:         input.FatherUserID,
		MotherUserID:         input.MotherUserID,
		BrotherUserID:        input.BrotherUserID,
		CreatorUserID:        input.CreatorUserID,
		FatherPrompt:         input.FatherPrompt,
		MotherPrompt:         input.MotherPrompt,
		BrotherPrompt:        input.BrotherPrompt,
		CreatorPrompt:        input.CreatorPrompt,
		Relationships:        relationships,
		MoodMode:             input.MoodMode,
		MoodPrompt:           input.MoodPrompt,
		DirectInsultMode:     input.DirectInsultMode,
		DirectInsultPrompt:   input.DirectInsultPrompt,
		DirectInsultFallback: input.DirectInsultFallback,
		SafetyCooldownMode:   input.SafetyCooldownMode,
		SafetyPauseReply:     input.SafetyPauseReply,
		TagReactionRules:     input.TagReactionRules,
		RetryAttempts:        input.RetryAttempts,
		RetryBackoff:         input.RetryBackoff,
	}
	state = layaRuntimeUpdated(state, input.UpdatedByID, input.UpdatedByName, time.Now().UTC())
	if err := s.app.store.SetLayaRuntimeState(state); err != nil {
		return err
	}
	s.app.clearAICaches()
	return nil
}

func (s webPanelService) SendLayaMessage(ctx context.Context, input wardenweb.LayaManualMessageInput) error {
	if s.app == nil || s.app.bot == nil {
		return errors.New("bot is not ready")
	}
	if input.ChatID == 0 {
		return errors.New("chat_id is required")
	}
	text := strings.TrimSpace(input.Text)
	if text == "" {
		return errors.New("text is required")
	}
	chat := &tele.Chat{ID: input.ChatID, Type: tele.ChatSuperGroup}
	opts := &tele.SendOptions{}
	if input.ThreadID != 0 {
		opts.ThreadID = input.ThreadID
	}
	if input.ReplyToMessageID != 0 {
		opts.ReplyTo = &tele.Message{ID: input.ReplyToMessageID, Chat: chat, ThreadID: input.ThreadID}
	}
	_, err := s.app.bot.Send(chat, text, opts)
	if err != nil {
		return err
	}
	s.app.sendOwnerLog("Laya manual reply",
		logField{"Chat", strconv.FormatInt(input.ChatID, 10)},
		logField{"Topic", strconv.Itoa(input.ThreadID)},
		logField{"Message", shortText(text, 700)},
	)
	return nil
}

func (s webPanelService) LayaUsers(ctx context.Context, chatID int64) (wardenweb.LayaUsersPage, error) {
	if s.app == nil || s.app.store == nil {
		return wardenweb.LayaUsersPage{}, errors.New("app store is not ready")
	}
	chatID = s.resolveChatID(chatID)
	page := wardenweb.LayaUsersPage{ChatID: chatID}
	seen := make(map[int64]struct{})
	for _, member := range s.app.store.Members(chatID) {
		if member.ID == 0 || member.IsBot {
			continue
		}
		page.Rows = append(page.Rows, s.layaUserRow(chatID, member.ID, member.Username, member.DisplayName))
		seen[member.ID] = struct{}{}
	}
	for _, profile := range s.app.store.LayaUserProfiles() {
		if _, ok := seen[profile.UserID]; ok {
			continue
		}
		page.Rows = append(page.Rows, s.layaUserRow(chatID, profile.UserID, profile.Username, profile.DisplayName))
	}
	sort.Slice(page.Rows, func(i, j int) bool {
		if page.Rows[i].DisplayName == page.Rows[j].DisplayName {
			return page.Rows[i].ID < page.Rows[j].ID
		}
		return page.Rows[i].DisplayName < page.Rows[j].DisplayName
	})
	return page, nil
}

func (s webPanelService) layaUserRow(chatID, userID int64, username, displayName string) wardenweb.LayaUserRow {
	profile, _ := s.app.store.LayaUserProfile(userID)
	if strings.TrimSpace(profile.Username) != "" {
		username = profile.Username
	}
	if strings.TrimSpace(profile.DisplayName) != "" {
		displayName = profile.DisplayName
	}
	if strings.TrimSpace(displayName) == "" {
		displayName = fmt.Sprintf("user %d", userID)
	}
	return wardenweb.LayaUserRow{
		ID:          userID,
		DisplayName: displayName,
		Username:    username,
		Gender:      profile.Gender,
		Address:     profile.Address,
		TagsText:    strings.Join(profile.Tags, ","),
		Note:        profile.Note,
		Updated:     webFormatUnix(profile.UpdatedUnix),
	}
}

func (s webPanelService) UpdateLayaUser(ctx context.Context, input wardenweb.LayaUserInput) error {
	if s.app == nil || s.app.store == nil {
		return errors.New("app store is not ready")
	}
	if input.UserID == 0 {
		return errors.New("user_id is required")
	}
	username := strings.TrimPrefix(strings.TrimSpace(input.Username), "@")
	displayName := strings.TrimSpace(input.DisplayName)
	if input.ChatID != 0 {
		if member, ok := s.app.store.Member(input.ChatID, input.UserID); ok {
			if username == "" {
				username = member.Username
			}
			if displayName == "" {
				displayName = member.DisplayName
			}
		}
	}
	profile := LayaUserProfile{
		UserID:        input.UserID,
		Username:      username,
		DisplayName:   displayName,
		Gender:        input.Gender,
		Address:       strings.TrimSpace(input.Address),
		Tags:          splitPatternList(input.Tags),
		Note:          input.Note,
		UpdatedUnix:   time.Now().UTC().Unix(),
		UpdatedChatID: input.ChatID,
	}
	if err := s.app.store.SetLayaUserProfile(profile); err != nil {
		return err
	}
	s.app.clearAIConversation(&tele.Chat{ID: input.ChatID, Type: tele.ChatSuperGroup}, &tele.User{ID: input.UserID})
	s.app.clearAICaches()
	return nil
}

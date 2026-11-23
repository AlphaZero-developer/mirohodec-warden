package main

import (
	"context"
	"errors"
	wardenweb "mirohodec-warden/internal/web"
	"strconv"
	"strings"
)

func (s webPanelService) Karma(ctx context.Context, chatID int64) ([]wardenweb.KarmaRow, error) {
	chatID = s.resolveChatID(chatID)
	rows := s.app.store.TopKarma(chatID, 100)
	out := make([]wardenweb.KarmaRow, 0, len(rows))
	for i, rec := range rows {
		out = append(out, wardenweb.KarmaRow{
			Rank:        i + 1,
			UserID:      rec.UserID,
			DisplayName: rec.DisplayName,
			Username:    rec.Username,
			Score:       rec.Score,
			Updated:     webFormatUnix(rec.UpdatedUnix),
		})
	}
	return out, nil
}

func (s webPanelService) OwnerSleep(ctx context.Context) (wardenweb.OwnerSleepPage, error) {
	if s.app == nil || s.app.store == nil {
		return wardenweb.OwnerSleepPage{}, errors.New("app store is not ready")
	}
	var page wardenweb.OwnerSleepPage
	for _, state := range s.app.store.ActiveOwnerSleeps() {
		row := wardenweb.OwnerSleepRow{
			UserID:      state.UserID,
			DisplayName: state.DisplayName,
			Username:    state.Username,
			AliasesText: strings.Join(state.Aliases, ", "),
			Since:       webFormatUnix(state.SinceUnix),
			CallCount:   len(state.Calls),
		}
		for _, call := range state.Calls {
			row.Calls = append(row.Calls, wardenweb.OwnerSleepCallRow{
				At:        webFormatUnix(call.AtUnix),
				ChatID:    call.ChatID,
				Chat:      sleepChatLabel(call),
				FromID:    call.FromID,
				From:      sleepUserLabel(call),
				Matched:   call.Matched,
				Text:      call.Text,
				ThreadID:  call.ThreadID,
				MessageID: call.MessageID,
			})
		}
		page.Rows = append(page.Rows, row)
	}
	return page, nil
}

func (s webPanelService) ClearOwnerSleep(ctx context.Context, userID int64) error {
	if s.app == nil || s.app.store == nil {
		return errors.New("app store is not ready")
	}
	if userID == 0 {
		return errors.New("user_id is required")
	}
	_, ok, err := s.app.store.ClearOwnerSleep(userID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("sleep state not found")
	}
	return nil
}

func (s webPanelService) LayaControl(ctx context.Context) (wardenweb.LayaControlPage, error) {
	if s.app == nil || s.app.store == nil {
		return wardenweb.LayaControlPage{}, errors.New("app store is not ready")
	}
	cfg := s.app.config().AI
	layaCfg := normalizeLayaConfig(s.app.config().Laya)
	state := normalizeLayaRuntimeState(s.app.store.LayaRuntimeState())
	active := s.app.applyLayaRuntimeConfig(cfg)
	activeLaya := s.app.activeLayaConfig()
	cache := s.app.aiCacheSnapshot()

	updatedBy := strings.TrimSpace(state.UpdatedByName)
	if updatedBy == "" && state.UpdatedByID != 0 {
		updatedBy = strconv.FormatInt(state.UpdatedByID, 10)
	}
	return wardenweb.LayaControlPage{
		ConfigSystemPrompt:      cfg.SystemPrompt,
		ActiveSystemPrompt:      active.SystemPrompt,
		RuntimeSystemPrompt:     state.SystemPrompt,
		ExtraPrompt:             state.ExtraPrompt,
		FallbackReply:           state.FallbackReply,
		ConfigBaseURL:           cfg.BaseURL,
		ConfigModel:             cfg.Model,
		ConfigTimeout:           cfg.Timeout,
		ConfigSoftTimeout:       cfg.SoftTimeout,
		ConfigFasMute:           cfg.FasMuteDuration,
		ConfigMaxTokens:         cfg.MaxTokens,
		ConfigMaxReplyChars:     cfg.MaxReplyChars,
		ConfigLogConversations:  cfg.LogConversations,
		ActiveBaseURL:           active.BaseURL,
		ActiveModel:             active.Model,
		ActiveTimeout:           active.Timeout,
		ActiveSoftTimeout:       active.SoftTimeout,
		ActiveFasMute:           active.FasMuteDuration,
		ActiveMaxTokens:         active.MaxTokens,
		ActiveMaxReplyChars:     active.MaxReplyChars,
		ActiveLogConversations:  active.LogConversations,
		RuntimeBaseURL:          state.BaseURL,
		RuntimeModel:            state.Model,
		RuntimeTimeout:          state.Timeout,
		RuntimeSoftTimeout:      state.SoftTimeout,
		RuntimeFasMute:          state.FasMuteDuration,
		RuntimeMaxTokens:        state.MaxTokens,
		RuntimeMaxReplyChars:    state.MaxReplyChars,
		RuntimeLogConversations: state.LogConversationsMode,
		ConfigFatherUserID:      layaCfg.FatherUserID,
		ConfigMotherUserID:      layaCfg.MotherUserID,
		ConfigBrotherUserID:     layaCfg.BrotherUserID,
		ConfigCreatorUserID:     layaCfg.CreatorUserID,
		ActiveFatherUserID:      activeLaya.FatherUserID,
		ActiveMotherUserID:      activeLaya.MotherUserID,
		ActiveBrotherUserID:     activeLaya.BrotherUserID,
		ActiveCreatorUserID:     activeLaya.CreatorUserID,
		RuntimeFatherUserID:     state.FatherUserID,
		RuntimeMotherUserID:     state.MotherUserID,
		RuntimeBrotherUserID:    state.BrotherUserID,
		RuntimeCreatorUserID:    state.CreatorUserID,
		ConfigFatherPrompt:      layaCfg.FatherPrompt,
		ConfigMotherPrompt:      layaCfg.MotherPrompt,
		ConfigBrotherPrompt:     layaCfg.BrotherPrompt,
		ConfigCreatorPrompt:     layaCfg.CreatorPrompt,
		ActiveFatherPrompt:      activeLaya.FatherPrompt,
		ActiveMotherPrompt:      activeLaya.MotherPrompt,
		ActiveBrotherPrompt:     activeLaya.BrotherPrompt,
		ActiveCreatorPrompt:     activeLaya.CreatorPrompt,
		RuntimeFatherPrompt:     state.FatherPrompt,
		RuntimeMotherPrompt:     state.MotherPrompt,
		RuntimeBrotherPrompt:    state.BrotherPrompt,
		RuntimeCreatorPrompt:    state.CreatorPrompt,
		RuntimeRelationships:    layaRelationshipsText(state.Relationships),
		ActiveRelationships:     webLayaRelationshipRows(s.app.activeLayaRelationships()),
		MoodMode:                state.MoodMode,
		MoodPrompt:              state.MoodPrompt,
		DirectInsultMode:        state.DirectInsultMode,
		DirectInsultPrompt:      state.DirectInsultPrompt,
		DirectInsultFallback:    state.DirectInsultFallback,
		SafetyCooldownMode:      state.SafetyCooldownMode,
		SafetyPauseReply:        state.SafetyPauseReply,
		TagReactionRules:        state.TagReactionRules,
		RetryAttempts:           state.RetryAttempts,
		RetryBackoff:            state.RetryBackoff,
		Updated:                 webFormatUnix(state.UpdatedUnix),
		UpdatedBy:               updatedBy,
		Cache: wardenweb.AICacheStats{
			SystemPrompts: cache.SystemPrompts,
		},
	}, nil
}

package main

import (
	"strings"
	"time"
)

func normalizeDiscussionRuntimeState(state DiscussionRuntimeState) DiscussionRuntimeState {
	state.Text = strings.TrimSpace(state.Text)
	state.ChatButtonText = strings.TrimSpace(state.ChatButtonText)
	state.ChatButtonURL = strings.TrimSpace(state.ChatButtonURL)
	state.FollowUpText = strings.TrimSpace(state.FollowUpText)
	state.FollowUpButtonText = strings.TrimSpace(state.FollowUpButtonText)
	state.FollowUpButtonURL = strings.TrimSpace(state.FollowUpButtonURL)
	return state
}

func discussionRuntimeStateFromConfig(cfg DiscussionAutoReplyConfig, now time.Time) DiscussionRuntimeState {
	return normalizeDiscussionRuntimeState(DiscussionRuntimeState{
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
		UpdatedUnix:        now.UTC().Unix(),
	})
}

func discussionConfigFromRuntime(state DiscussionRuntimeState) DiscussionAutoReplyConfig {
	state = normalizeDiscussionRuntimeState(state)
	return DiscussionAutoReplyConfig{
		Enabled:            state.Enabled,
		ChatID:             state.ChatID,
		SourceChatID:       state.SourceChatID,
		StrictChecks:       state.StrictChecks,
		Text:               state.Text,
		ChatButtonText:     state.ChatButtonText,
		ChatButtonURL:      state.ChatButtonURL,
		FollowUpText:       state.FollowUpText,
		FollowUpButtonText: state.FollowUpButtonText,
		FollowUpButtonURL:  state.FollowUpButtonURL,
		Pin:                state.Pin,
	}
}

func (a *App) activeDiscussionConfig() DiscussionAutoReplyConfig {
	if a == nil {
		return DiscussionAutoReplyConfig{}
	}
	cfg := a.config().Discussion
	if a.store != nil {
		state := normalizeDiscussionRuntimeState(a.store.DiscussionRuntimeState())
		if state.UpdatedUnix != 0 {
			cfg = discussionConfigFromRuntime(state)
		}
	}
	if strings.TrimSpace(cfg.Text) == "" {
		cfg.Text = defaultDiscussionAutoReplyText
	}
	return cfg
}

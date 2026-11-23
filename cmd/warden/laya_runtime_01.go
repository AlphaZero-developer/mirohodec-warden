package main

import (
	"strings"
	"time"
)

func normalizeLayaRuntimeState(state LayaRuntimeState) LayaRuntimeState {
	state.SystemPrompt = strings.TrimSpace(state.SystemPrompt)
	state.ExtraPrompt = strings.TrimSpace(state.ExtraPrompt)
	state.FallbackReply = strings.TrimSpace(state.FallbackReply)
	state.BaseURL = strings.TrimSpace(state.BaseURL)
	state.Model = strings.TrimSpace(state.Model)
	state.Timeout = strings.TrimSpace(state.Timeout)
	state.SoftTimeout = strings.TrimSpace(state.SoftTimeout)
	state.FasMuteDuration = strings.TrimSpace(state.FasMuteDuration)
	state.LogConversationsMode = strings.ToLower(strings.TrimSpace(state.LogConversationsMode))
	if state.LogConversationsMode != "on" && state.LogConversationsMode != "off" {
		state.LogConversationsMode = ""
	}
	if state.MaxTokens < 0 {
		state.MaxTokens = 0
	}
	if state.MaxReplyChars < 0 {
		state.MaxReplyChars = 0
	}
	state.FatherPrompt = strings.TrimSpace(state.FatherPrompt)
	state.MotherPrompt = strings.TrimSpace(state.MotherPrompt)
	state.BrotherPrompt = strings.TrimSpace(state.BrotherPrompt)
	state.CreatorPrompt = strings.TrimSpace(state.CreatorPrompt)
	state.Relationships = normalizeLayaRelationships(state.Relationships)
	state.MoodMode = normalizeLayaRuntimeMode(state.MoodMode, "random", "off", "custom")
	state.MoodPrompt = strings.TrimSpace(state.MoodPrompt)
	state.DirectInsultMode = normalizeLayaRuntimeMode(state.DirectInsultMode, "ai", "local", "off")
	state.DirectInsultPrompt = strings.TrimSpace(state.DirectInsultPrompt)
	state.DirectInsultFallback = strings.TrimSpace(state.DirectInsultFallback)
	state.SafetyCooldownMode = normalizeLayaRuntimeMode(state.SafetyCooldownMode, "on", "off")
	state.SafetyPauseReply = strings.TrimSpace(state.SafetyPauseReply)
	state.TagReactionRules = strings.TrimSpace(state.TagReactionRules)
	if state.RetryAttempts < 0 {
		state.RetryAttempts = 0
	}
	if state.RetryAttempts > 5 {
		state.RetryAttempts = 5
	}
	state.RetryBackoff = strings.TrimSpace(state.RetryBackoff)
	state.UpdatedByName = strings.TrimSpace(state.UpdatedByName)
	return state
}

func emptyLayaRuntimeState(state LayaRuntimeState) bool {
	state = normalizeLayaRuntimeState(state)
	return state.SystemPrompt == "" && state.ExtraPrompt == "" && state.FallbackReply == "" &&
		state.BaseURL == "" && state.Model == "" && state.Timeout == "" && state.SoftTimeout == "" &&
		state.FasMuteDuration == "" && state.MaxTokens == 0 && state.MaxReplyChars == 0 && state.LogConversationsMode == "" &&
		state.FatherUserID == 0 && state.MotherUserID == 0 && state.BrotherUserID == 0 && state.CreatorUserID == 0 &&
		state.FatherPrompt == "" && state.MotherPrompt == "" && state.BrotherPrompt == "" && state.CreatorPrompt == "" &&
		len(state.Relationships) == 0 &&
		state.MoodMode == "" && state.MoodPrompt == "" && state.DirectInsultMode == "" && state.DirectInsultPrompt == "" &&
		state.DirectInsultFallback == "" && state.SafetyCooldownMode == "" && state.SafetyPauseReply == "" &&
		state.TagReactionRules == "" && state.RetryAttempts == 0 && state.RetryBackoff == "" &&
		state.UpdatedUnix == 0 && state.UpdatedByID == 0 && state.UpdatedByName == ""
}

func normalizeLayaRuntimeMode(value string, allowed ...string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || value == "inherit" || value == "default" {
		return ""
	}
	for _, item := range allowed {
		if value == item {
			return value
		}
	}
	return ""
}

func normalizeLayaRelationships(in []LayaRelationship) []LayaRelationship {
	if len(in) == 0 {
		return nil
	}
	out := make([]LayaRelationship, 0, len(in))
	positions := make(map[int64]int, len(in))
	for _, rec := range in {
		rec.Role = strings.TrimSpace(rec.Role)
		rec.Prompt = strings.TrimSpace(rec.Prompt)
		if rec.UserID == 0 || (rec.Role == "" && rec.Prompt == "") {
			continue
		}
		if rec.Role == "" {
			rec.Role = "особая связь"
		}
		if idx, ok := positions[rec.UserID]; ok {
			out[idx] = rec
			continue
		}
		positions[rec.UserID] = len(out)
		out = append(out, rec)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (a *App) applyLayaRuntimeConfig(cfg AIConfig) AIConfig {
	if a == nil || a.store == nil {
		return cfg
	}
	state := normalizeLayaRuntimeState(a.store.LayaRuntimeState())
	if emptyLayaRuntimeState(state) {
		return cfg
	}

	if state.SystemPrompt != "" {
		base := strings.TrimSpace(a.config().AI.SystemPrompt)
		current := strings.TrimSpace(cfg.SystemPrompt)
		switch {
		case base != "" && current == base:
			cfg.SystemPrompt = state.SystemPrompt
		case base != "" && strings.HasPrefix(current, base+"\n\n"):
			cfg.SystemPrompt = strings.TrimSpace(state.SystemPrompt + strings.TrimPrefix(current, base))
		case current == "":
			cfg.SystemPrompt = state.SystemPrompt
		default:
			cfg.SystemPrompt = strings.TrimSpace(state.SystemPrompt + "\n\n" + current)
		}
	}
	if state.ExtraPrompt != "" {
		cfg.SystemPrompt = strings.TrimSpace(cfg.SystemPrompt + "\n\n" + state.ExtraPrompt)
	}
	if state.FallbackReply != "" {
		cfg.FallbackReply = state.FallbackReply
	}
	if state.BaseURL != "" {
		cfg.BaseURL = state.BaseURL
	}
	if state.Model != "" {
		cfg.Model = state.Model
	}
	if state.Timeout != "" {
		cfg.Timeout = state.Timeout
	}
	if state.SoftTimeout != "" {
		cfg.SoftTimeout = state.SoftTimeout
	}
	if state.FasMuteDuration != "" {
		cfg.FasMuteDuration = state.FasMuteDuration
	}
	if state.MaxTokens > 0 {
		cfg.MaxTokens = state.MaxTokens
	}
	if state.MaxReplyChars > 0 {
		cfg.MaxReplyChars = state.MaxReplyChars
		if cfg.MaxReplyChars > telegramMaxTextChars {
			cfg.MaxReplyChars = telegramMaxTextChars
		}
	}
	switch state.LogConversationsMode {
	case "on":
		cfg.LogConversations = true
	case "off":
		cfg.LogConversations = false
	}
	return cfg
}

func layaRuntimeUpdated(state LayaRuntimeState, byID int64, byName string, now time.Time) LayaRuntimeState {
	state = normalizeLayaRuntimeState(state)
	state.UpdatedUnix = now.UTC().Unix()
	state.UpdatedByID = byID
	state.UpdatedByName = strings.TrimSpace(byName)
	return state
}

func (a *App) activeLayaRuntimeState() LayaRuntimeState {
	if a == nil || a.store == nil {
		return LayaRuntimeState{}
	}
	return normalizeLayaRuntimeState(a.store.LayaRuntimeState())
}

func (a *App) activeLayaTagReactionRules() string {
	return a.activeLayaRuntimeState().TagReactionRules
}

func (a *App) layaRuntimeRetryAttempts() int {
	state := a.activeLayaRuntimeState()
	if state.RetryAttempts > 0 {
		return state.RetryAttempts
	}
	return 1
}

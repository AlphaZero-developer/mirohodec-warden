package main

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

const (
	maxRehabBrainwashAIAttempts      = 4
	maxRehabBrainwashNoRepeatMemory  = 32
	maxRehabBrainwashForbiddenPrompt = 10
)

type RehabBrainwashConfig struct {
	Enabled       bool
	AIEnabled     bool
	LeaderUserID  int64
	LeaderName    string
	AIPrompt      string
	Messages      string
	UpdatedUnix   int64
	UpdatedByID   int64
	UpdatedByName string
}

func normalizeRehabBrainwashRuntimeState(state RehabBrainwashRuntimeState) RehabBrainwashRuntimeState {
	state.LeaderName = strings.TrimSpace(state.LeaderName)
	state.AIPrompt = strings.TrimSpace(state.AIPrompt)
	state.Messages = strings.TrimSpace(state.Messages)
	state.UpdatedByName = strings.TrimSpace(state.UpdatedByName)
	return state
}

func rehabBrainwashConfigFromRehab(cfg RehabConfig) RehabBrainwashConfig {
	return normalizeRehabBrainwashConfig(RehabBrainwashConfig{
		Enabled:      cfg.BrainwashEnabled,
		AIEnabled:    cfg.BrainwashAIEnabled,
		LeaderUserID: cfg.BrainwashLeaderUserID,
		LeaderName:   cfg.BrainwashLeaderName,
		AIPrompt:     cfg.BrainwashAIPrompt,
		Messages:     cfg.BrainwashMessages,
	})
}

func normalizeRehabBrainwashConfig(cfg RehabBrainwashConfig) RehabBrainwashConfig {
	cfg.LeaderName = strings.TrimSpace(cfg.LeaderName)
	if cfg.LeaderName == "" {
		cfg.LeaderName = defaultRehabBrainwashLeader
	}
	cfg.AIPrompt = strings.TrimSpace(cfg.AIPrompt)
	if cfg.AIPrompt == "" {
		cfg.AIPrompt = defaultRehabBrainwashAIPrompt
	}
	cfg.Messages = strings.TrimSpace(cfg.Messages)
	if cfg.Messages == "" {
		cfg.Messages = defaultRehabBrainwashMessages
	}
	cfg.UpdatedByName = strings.TrimSpace(cfg.UpdatedByName)
	return cfg
}

func rehabBrainwashRuntimeStateFromConfig(cfg RehabBrainwashConfig, byID int64, byName string, now time.Time) RehabBrainwashRuntimeState {
	cfg = normalizeRehabBrainwashConfig(cfg)
	return normalizeRehabBrainwashRuntimeState(RehabBrainwashRuntimeState{
		Enabled:       cfg.Enabled,
		AIEnabled:     cfg.AIEnabled,
		LeaderUserID:  cfg.LeaderUserID,
		LeaderName:    cfg.LeaderName,
		AIPrompt:      cfg.AIPrompt,
		Messages:      cfg.Messages,
		UpdatedUnix:   now.UTC().Unix(),
		UpdatedByID:   byID,
		UpdatedByName: byName,
	})
}

func (a *App) activeRehabBrainwashConfig() RehabBrainwashConfig {
	if a == nil {
		return normalizeRehabBrainwashConfig(RehabBrainwashConfig{})
	}
	cfg := rehabBrainwashConfigFromRehab(a.config().Rehab)
	if a.store == nil {
		return cfg
	}
	state := normalizeRehabBrainwashRuntimeState(a.store.RehabBrainwashRuntimeState())
	if state.UpdatedUnix == 0 {
		return cfg
	}
	return normalizeRehabBrainwashConfig(RehabBrainwashConfig{
		Enabled:       state.Enabled,
		AIEnabled:     state.AIEnabled,
		LeaderUserID:  state.LeaderUserID,
		LeaderName:    state.LeaderName,
		AIPrompt:      state.AIPrompt,
		Messages:      state.Messages,
		UpdatedUnix:   state.UpdatedUnix,
		UpdatedByID:   state.UpdatedByID,
		UpdatedByName: state.UpdatedByName,
	})
}

func (a *App) rehabBrainwashReply(jail ModerationRecord) (string, bool) {
	cfg := a.activeRehabBrainwashConfig()
	if !cfg.Enabled {
		return "", false
	}
	recent := a.rehabBrainwashRecentReplies(cfg, jail)
	if cfg.AIEnabled {
		if reply := a.rehabBrainwashAIReply(cfg, jail, recent); reply != "" {
			a.rememberRehabBrainwashReply(cfg, jail, reply)
			return formatAIReplyTextForEncoding(a.config().AI, reply), true
		}
	}
	reply := rehabBrainwashFallbackReply(cfg, jail, recent, nil)
	if reply == "" {
		return "", false
	}
	a.rememberRehabBrainwashReply(cfg, jail, reply)
	return trimRunes(reply, 1000), true
}

func (a *App) rehabBrainwashAIReply(cfg RehabBrainwashConfig, jail ModerationRecord, recent []string) string {
	if a == nil {
		return ""
	}
	ai := a.currentAI()
	aiCfg := a.config().AI
	if ai == nil || !aiCfg.Enabled {
		return ""
	}
	cfg = normalizeRehabBrainwashConfig(cfg)
	aiCfg.SystemPrompt = renderRehabBrainwashMessage(cfg.AIPrompt, cfg, jail)
	aiCfg.MaxReplyChars = 1000
	if aiCfg.MaxTokens <= 0 || aiCfg.MaxTokens > 180 {
		aiCfg.MaxTokens = 180
	}
	lines := rehabBrainwashMessageLines(cfg.Messages)
	for i, line := range lines {
		lines[i] = renderRehabBrainwashMessage(line, cfg, jail)
	}
	rejected := make([]string, 0, maxRehabBrainwashAIAttempts)
	for attempt := 0; attempt < maxRehabBrainwashAIAttempts; attempt++ {
		attemptCfg := aiCfg
		attemptCfg.SystemPrompt = appendPrompt(aiCfg.SystemPrompt, rehabBrainwashNoRepeatPrompt(recent, rejected))
		attemptCfg.SystemPrompt = appendPrompt(attemptCfg.SystemPrompt, layaAntiLoopDiversityPrompt(attempt))
		payload := map[string]interface{}{
			"task": "rehab_brainwash_phrase",
			"leader": map[string]interface{}{
				"id":   cfg.LeaderUserID,
				"name": cfg.LeaderName,
			},
			"user": map[string]interface{}{
				"id":   jail.UserID,
				"name": strings.TrimSpace(jail.DisplayName),
			},
			"attempt":              attempt + 1,
			"fallback_examples":    rehabBrainwashExampleLines(lines, recent, attempt),
			"forbidden_recent":     rehabBrainwashShortList(append(append([]string(nil), recent...), rejected...), maxRehabBrainwashForbiddenPrompt),
			"hard_requirements":    []string{"one short phrase", "do not copy examples", "do not repeat or lightly rewrite forbidden_recent", "change the image, first words, rhythm, and ending"},
			"preferred_response":   `{"reply":"..."}`,
			"max_reply_characters": 280,
		}
		raw, _ := json.Marshal(payload)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		reply, err := ai.Reply(ctx, attemptCfg, string(raw))
		cancel()
		if err != nil {
			return ""
		}
		reply = cleanRehabBrainwashAIReply(reply)
		if reply == "" {
			continue
		}
		if !rehabBrainwashReplyRepeated(reply, recent, rejected) {
			return reply
		}
		rejected = append(rejected, reply)
	}
	return ""
}

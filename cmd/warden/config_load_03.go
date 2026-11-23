package main

import (
	"errors"
	"time"
)

func applyAIDefaults(cfg *Config) error {
	if cfg.AI.BaseURL == "" {
		cfg.AI.BaseURL = defaultAIBaseURL
	}
	if cfg.AI.Model == "" {
		cfg.AI.Model = defaultAIModel
	}
	if cfg.AI.Timeout == "" {
		cfg.AI.Timeout = defaultAITimeout
	}
	if cfg.AI.SoftTimeout == "" {
		cfg.AI.SoftTimeout = defaultAISoftTimeout
	}
	if cfg.AI.MaxTokens <= 0 {
		cfg.AI.MaxTokens = defaultAIMaxTokens
	}
	if cfg.AI.MaxReplyChars <= 0 {
		cfg.AI.MaxReplyChars = defaultAIMaxReplyChars
	}
	if cfg.AI.MaxReplyChars > telegramMaxTextChars {
		cfg.AI.MaxReplyChars = telegramMaxTextChars
	}
	if cfg.AI.FasMuteDuration == "" {
		cfg.AI.FasMuteDuration = defaultAIFasMute
	}
	cfg.AI.ReplyEncoding = normalizeAIReplyEncoding(cfg.AI.ReplyEncoding)
	cfg.AI.ReplyCipher = normalizeAIReplyCipherOrDefault(cfg.AI.ReplyCipher)
	cfg.AI.ReplyCipherUserIDs = uniqueInt64s(cfg.AI.ReplyCipherUserIDs)
	cfg.AI.ReplyCipherUsernames = normalizeReplyCipherUsernames(cfg.AI.ReplyCipherUsernames)
	if cfg.AI.SystemPrompt == "" {
		cfg.AI.SystemPrompt = defaultAISystemPrompt
	}
	if cfg.AI.FallbackReply == "" {
		cfg.AI.FallbackReply = defaultAIFallbackReply
	}
	if _, err := parseDuration(cfg.AI.Timeout); err != nil {
		return errors.New("ai.timeout is invalid: " + err.Error())
	}
	if _, err := parseDuration(cfg.AI.SoftTimeout); err != nil {
		return errors.New("ai.soft_timeout is invalid: " + err.Error())
	}
	if d, permanent, ok, err := parseDurationToken(cfg.AI.FasMuteDuration); err != nil || !ok {
		if err != nil {
			return errors.New("ai.fas_mute_duration is invalid: " + err.Error())
		}
		return errors.New("ai.fas_mute_duration is invalid")
	} else if !permanent && d > 0 && d < 30*time.Second {
		return errors.New("ai.fas_mute_duration must be 30s or more")
	}
	return nil
}

func applyRuntimeDefaults(cfg *Config) {
	if cfg.Photos.Cooldown == "" {
		cfg.Photos.Cooldown = "30s"
	}
	if cfg.Photos.AutoReplyChance < 0 {
		cfg.Photos.AutoReplyChance = 0
	}
	if cfg.Photos.AutoReplyChance > 1 {
		cfg.Photos.AutoReplyChance = 1
	}
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if len(cfg.Logging.OutputPaths) == 0 {
		cfg.Logging.OutputPaths = []string{"stdout"}
	}
	if cfg.Logging.QueueSize <= 0 {
		cfg.Logging.QueueSize = 2048
	}
	if cfg.Metrics.Listen == "" {
		cfg.Metrics.Listen = ":2112"
	}
	if cfg.Metrics.Path == "" {
		cfg.Metrics.Path = "/metrics"
	}
}

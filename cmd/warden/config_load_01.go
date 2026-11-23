package main

import (
	"errors"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func LoadConfig(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal([]byte(os.ExpandEnv(string(raw))), &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Telegram.Token == "" {
		cfg.Telegram.Token = os.Getenv("BOT_TOKEN")
	}
	if cfg.Telegram.Token == "" {
		return Config{}, errors.New("telegram token is empty: set telegram.token or BOT_TOKEN")
	}
	applyWebEnvOverrides(&cfg.Web)
	if err := applyTransportDefaults(&cfg); err != nil {
		return Config{}, err
	}
	applyRehabDefaults(&cfg)
	applyCommunityDefaults(&cfg)
	applySpamDefaults(&cfg)
	applyContentDefaults(&cfg)
	if err := applyAIDefaults(&cfg); err != nil {
		return Config{}, err
	}
	applyRuntimeDefaults(&cfg)
	return cfg, nil
}

func applyTransportDefaults(cfg *Config) error {
	if cfg.Telegram.PollerTimeout == "" {
		cfg.Telegram.PollerTimeout = "10s"
	}
	if cfg.Telegram.Mode == "" {
		cfg.Telegram.Mode = "polling"
	}
	if !strings.EqualFold(cfg.Telegram.Mode, "polling") && !strings.EqualFold(cfg.Telegram.Mode, "webhook") {
		return errors.New("telegram.mode must be polling or webhook")
	}
	if cfg.Telegram.Webhook.Listen == "" {
		cfg.Telegram.Webhook.Listen = ":8080"
	}
	if cfg.Telegram.Webhook.MaxConnections <= 0 {
		cfg.Telegram.Webhook.MaxConnections = 40
	}
	if strings.EqualFold(cfg.Telegram.Mode, "webhook") && cfg.Telegram.Webhook.PublicURL == "" {
		return errors.New("telegram.webhook.public_url is required in webhook mode")
	}
	if cfg.Storage.Path == "" {
		cfg.Storage.Path = "data/state.json"
	}
	if cfg.Storage.Driver == "" {
		cfg.Storage.Driver = "json"
	}
	cfg.Storage.Driver = strings.ToLower(strings.TrimSpace(cfg.Storage.Driver))
	if cfg.Storage.Driver != "json" && cfg.Storage.Driver != "postgres" {
		return errors.New("storage.driver must be json or postgres")
	}
	if cfg.Storage.Driver == "postgres" && strings.TrimSpace(cfg.Storage.Postgres.DSN) == "" {
		return errors.New("storage.postgres.dsn is required when storage.driver is postgres")
	}
	return nil
}

func applyRehabDefaults(cfg *Config) {
	if cfg.Rehab.Timezone == "" {
		cfg.Rehab.Timezone = "Europe/Moscow"
	}
	if len(cfg.Rehab.ReminderTimes) == 0 {
		cfg.Rehab.ReminderTimes = []string{"09:00", "14:00", "20:00"}
	}
	if cfg.Rehab.ReminderWindow == "" {
		cfg.Rehab.ReminderWindow = "15m"
	}
	if cfg.Rehab.PrivateMessage == "" {
		cfg.Rehab.PrivateMessage = "Привет, когда ты уже выздоровеешь?"
	}
	if cfg.Rehab.HarshReply == "" {
		cfg.Rehab.HarshReply = "В психиатрический центр тебе. Личку открыть не судьба, значит слушай меня здесь."
	}
	if strings.TrimSpace(cfg.Rehab.BrainwashLeaderName) == "" {
		cfg.Rehab.BrainwashLeaderName = defaultRehabBrainwashLeader
	}
	if strings.TrimSpace(cfg.Rehab.BrainwashAIPrompt) == "" {
		cfg.Rehab.BrainwashAIPrompt = defaultRehabBrainwashAIPrompt
	}
	if strings.TrimSpace(cfg.Rehab.BrainwashMessages) == "" {
		cfg.Rehab.BrainwashMessages = defaultRehabBrainwashMessages
	}
	if cfg.Rehab.TickerMinInterval == "" {
		cfg.Rehab.TickerMinInterval = defaultRehabTickerMin
	}
	if cfg.Rehab.TickerMaxInterval == "" {
		cfg.Rehab.TickerMaxInterval = defaultRehabTickerMax
	}
	if cfg.Rehab.PostReleaseCheckMinDelay == "" {
		cfg.Rehab.PostReleaseCheckMinDelay = "3m"
	}
	if cfg.Rehab.PostReleaseCheckMaxDelay == "" {
		cfg.Rehab.PostReleaseCheckMaxDelay = "5h"
	}
	if cfg.Rehab.PostReleaseCheckAnswerWindow == "" {
		cfg.Rehab.PostReleaseCheckAnswerWindow = "2h"
	}
	if cfg.Rehab.PostReleaseCheckRejailDuration == "" {
		cfg.Rehab.PostReleaseCheckRejailDuration = "24h"
	}
	if cfg.Rehab.ReleaseExamAnswerWindow == "" {
		cfg.Rehab.ReleaseExamAnswerWindow = "10m"
	}
	if cfg.Rehab.ReleaseExamFailDuration == "" {
		cfg.Rehab.ReleaseExamFailDuration = "24h"
	}
	if strings.TrimSpace(cfg.Rehab.ReleaseExamQuestions) == "" {
		cfg.Rehab.ReleaseExamQuestions = defaultRehabReleaseExamQuestions
	}
}

func applyCommunityDefaults(cfg *Config) {
	if cfg.Laya.FatherUserID == 0 {
		cfg.Laya.FatherUserID = defaultLayaFatherUserID
	}
	if cfg.Laya.BrotherUserID == 0 {
		cfg.Laya.BrotherUserID = defaultLayaBrotherUserID
	}
	if cfg.Laya.CreatorUserID == 0 {
		cfg.Laya.CreatorUserID = defaultLayaCreatorUserID
	}
	cfg.Laya = normalizeLayaConfig(cfg.Laya)
	if cfg.Moderation.WarningCooldown == "" {
		cfg.Moderation.WarningCooldown = "2m"
	}
	if cfg.Moderation.CleanupInterval == "" {
		cfg.Moderation.CleanupInterval = "1m"
	}
	if cfg.Moderation.ReloadInterval == "" {
		cfg.Moderation.ReloadInterval = "30s"
	}
	if !cfg.Warn.Enabled {
		cfg.Warn.Enabled = true
	}
	if cfg.Warn.ExpireAfter == "" {
		cfg.Warn.ExpireAfter = "30d"
	}
	if cfg.Warn.MuteThreshold <= 0 {
		cfg.Warn.MuteThreshold = 3
	}
	if cfg.Warn.MuteDuration == "" {
		cfg.Warn.MuteDuration = "1h"
	}
	if cfg.Warn.RehabThreshold <= 0 {
		cfg.Warn.RehabThreshold = 5
	}
	if cfg.Warn.RehabDuration == "" {
		cfg.Warn.RehabDuration = "24h"
	}
	if !cfg.Karma.Enabled {
		cfg.Karma.Enabled = true
	}
	if cfg.Karma.ActivityMessagesPerPoint <= 0 {
		cfg.Karma.ActivityMessagesPerPoint = 10
	}
	if cfg.Karma.ActivityDailyLimit <= 0 {
		cfg.Karma.ActivityDailyLimit = 5
	}
	if cfg.Karma.ManualCooldown == "" {
		cfg.Karma.ManualCooldown = "12h"
	}
	if cfg.Web.Listen == "" {
		cfg.Web.Listen = ":8090"
	}
	if cfg.Web.SessionTTL == 0 {
		cfg.Web.SessionTTL = 24 * time.Hour
	}
	if len(cfg.Security.PublicCommands) == 0 {
		cfg.Security.PublicCommands = []string{"profile", "профиль", "passport", "паспорт", "activity", "topactive", "karma", "topkarma", "shop", "buy", "event", "week", "ward", "палата", "menu", "selfmute", "selfjail", "selfrehab", "selfrelease", "selfban", "register", "mafia", "mafia_bind", "mafia_chat", "mafia_status", "mafia_leave", "mafia_join", "mafia_find", "will", "quote", "photo", "faq", "job", "hire", "donate", "idea", "bug", "reportbug", "appeal", "clearhistory"}
	}
	cfg.Security.PublicCommands = appendMissingPublicCommands(cfg.Security.PublicCommands, "passport", "паспорт", "shop", "buy", "event", "week", "ward", "палата", "menu", "selfmute", "selfjail", "selfrehab", "selfrelease", "selfban", "job", "hire", "donate", "clearhistory", "register", "mafia", "mafia_bind", "mafia_chat", "mafia_status", "mafia_leave", "mafia_join", "mafia_find", "will")
}

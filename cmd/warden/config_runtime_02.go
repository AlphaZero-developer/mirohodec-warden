package main

import (
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func (cfg *WebConfig) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Enabled              bool   `yaml:"enabled"`
		Listen               string `yaml:"listen"`
		PublicURL            string `yaml:"public_url"`
		MafiaMiniAppURL      string `yaml:"mafia_mini_app_url"`
		MafiaMiniAppDir      string `yaml:"mafia_mini_app_dir"`
		AdminToken           string `yaml:"admin_token"`
		SessionTTL           string `yaml:"session_ttl"`
		BehindProxy          bool   `yaml:"behind_proxy"`
		CookieSecure         bool   `yaml:"cookie_secure"`
		SendPanelLinkToOwner bool   `yaml:"send_panel_link_to_owner"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	*cfg = WebConfig{
		Enabled:              raw.Enabled,
		Listen:               raw.Listen,
		PublicURL:            raw.PublicURL,
		MafiaMiniAppURL:      raw.MafiaMiniAppURL,
		MafiaMiniAppDir:      raw.MafiaMiniAppDir,
		AdminToken:           raw.AdminToken,
		BehindProxy:          raw.BehindProxy,
		CookieSecure:         raw.CookieSecure,
		SendPanelLinkToOwner: raw.SendPanelLinkToOwner,
	}
	if strings.TrimSpace(raw.SessionTTL) != "" {
		d, permanent, ok, err := parseDurationToken(raw.SessionTTL)
		if err != nil {
			return err
		}
		if permanent {
			cfg.SessionTTL = -1
		} else if ok {
			cfg.SessionTTL = d
		} else {
			d, err := parseDuration(raw.SessionTTL)
			if err != nil {
				return err
			}
			cfg.SessionTTL = d
		}
	}
	return nil
}

type SpamConfig struct {
	Enabled             bool    `yaml:"enabled"`
	Window              string  `yaml:"window"`
	MaxMessages         int     `yaml:"max_messages"`
	TextWindow          string  `yaml:"text_window"`
	MaxTextMessages     int     `yaml:"max_text_messages"`
	MediaWindow         string  `yaml:"media_window"`
	MaxMedia            int     `yaml:"max_media"`
	StickerWindow       string  `yaml:"sticker_window"`
	MaxStickers         int     `yaml:"max_stickers"`
	RepeatWindow        string  `yaml:"repeat_window"`
	MaxRepeatedText     int     `yaml:"max_repeated_text"`
	MaxRepeatedStickers int     `yaml:"max_repeated_stickers"`
	MinRepeatedTextLen  int     `yaml:"min_repeated_text_length"`
	LinkWindow          string  `yaml:"link_window"`
	MaxLinks            int     `yaml:"max_links"`
	MaxRepeatedLinks    int     `yaml:"max_repeated_links"`
	MaxLinksPerMessage  int     `yaml:"max_links_per_message"`
	MentionWindow       string  `yaml:"mention_window"`
	MaxMentions         int     `yaml:"max_mentions"`
	MaxMentionsPerMsg   int     `yaml:"max_mentions_per_message"`
	ForwardWindow       string  `yaml:"forward_window"`
	MaxForwards         int     `yaml:"max_forwards"`
	DeleteExcess        bool    `yaml:"delete_excess"`
	AutoMute            bool    `yaml:"auto_mute"`
	MuteDuration        string  `yaml:"mute_duration"`
	ActionCooldown      string  `yaml:"action_cooldown"`
	MuteAfterStrikes    int     `yaml:"mute_after_strikes"`
	StrikeWindow        string  `yaml:"strike_window"`
	LogLimit            int     `yaml:"log_limit"`
	Notify              bool    `yaml:"notify"`
	ExemptUserIDs       []int64 `yaml:"exempt_user_ids"`
}

type AllConfig struct {
	Enabled            bool   `yaml:"enabled"`
	AdminsOnly         bool   `yaml:"admins_only"`
	ChunkSize          int    `yaml:"chunk_size"`
	DelayBetweenChunks string `yaml:"delay_between_chunks"`
}

type IdeaConfig struct {
	Enabled      bool   `yaml:"enabled"`
	UserCooldown string `yaml:"user_cooldown"`
	ChatCooldown string `yaml:"chat_cooldown"`
	MaxLength    int    `yaml:"max_length"`
}

type DiscussionAutoReplyConfig struct {
	Enabled            bool   `yaml:"enabled"`
	ChatID             int64  `yaml:"chat_id"`
	SourceChatID       int64  `yaml:"source_chat_id"`
	StrictChecks       bool   `yaml:"strict_checks"`
	Text               string `yaml:"text"`
	ChatButtonText     string `yaml:"chat_button_text"`
	ChatButtonURL      string `yaml:"chat_button_url"`
	FollowUpText       string `yaml:"follow_up_text"`
	FollowUpButtonText string `yaml:"follow_up_button_text"`
	FollowUpButtonURL  string `yaml:"follow_up_button_url"`
	Pin                bool   `yaml:"pin"`
}

func applyWebEnvOverrides(cfg *WebConfig) {
	if cfg == nil {
		return
	}
	if value := strings.TrimSpace(os.Getenv("WEB_PUBLIC_URL")); value != "" {
		cfg.PublicURL = value
	}
	if value := strings.TrimSpace(os.Getenv("WEB_MAFIA_MINI_APP_URL")); value != "" {
		cfg.MafiaMiniAppURL = value
	}
	if value := strings.TrimSpace(os.Getenv("MAFIA_MINI_APP_URL")); value != "" {
		cfg.MafiaMiniAppURL = value
	}
	if value := strings.TrimSpace(os.Getenv("WEB_MAFIA_MINI_APP_DIR")); value != "" {
		cfg.MafiaMiniAppDir = value
	}
}

func appendMissingPublicCommands(commands []string, required ...string) []string {
	seen := make(map[string]struct{}, len(commands)+len(required))
	for _, cmd := range commands {
		seen[strings.ToLower(strings.TrimPrefix(strings.TrimSpace(cmd), "/"))] = struct{}{}
	}
	for _, cmd := range required {
		key := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(cmd), "/"))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		commands = append(commands, key)
		seen[key] = struct{}{}
	}
	return commands
}

func mustDuration(value string, fallback time.Duration) time.Duration {
	d, err := parseDuration(value)
	if err != nil {
		return fallback
	}
	return d
}

func (a *App) config() Config {
	return a.cfg.Load().(Config)
}

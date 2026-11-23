package main

import (
	"strings"
)

func applySpamDefaults(cfg *Config) {
	if cfg.Spam.TextWindow == "" {
		if cfg.Spam.Window != "" {
			cfg.Spam.TextWindow = cfg.Spam.Window
		} else {
			cfg.Spam.TextWindow = "8s"
		}
	}
	if cfg.Spam.MaxTextMessages <= 0 {
		if cfg.Spam.MaxMessages > 0 {
			cfg.Spam.MaxTextMessages = cfg.Spam.MaxMessages
		} else {
			cfg.Spam.MaxTextMessages = 12
		}
	}
	if cfg.Spam.MediaWindow == "" {
		cfg.Spam.MediaWindow = "30s"
	}
	if cfg.Spam.MaxMedia <= 0 {
		cfg.Spam.MaxMedia = 12
	}
	if cfg.Spam.StickerWindow == "" {
		cfg.Spam.StickerWindow = "10s"
	}
	if cfg.Spam.MaxStickers <= 0 {
		cfg.Spam.MaxStickers = 10
	}
	if cfg.Spam.RepeatWindow == "" {
		cfg.Spam.RepeatWindow = "20s"
	}
	if cfg.Spam.MaxRepeatedText <= 0 {
		cfg.Spam.MaxRepeatedText = 4
	}
	if cfg.Spam.MaxRepeatedStickers <= 0 {
		cfg.Spam.MaxRepeatedStickers = 4
	}
	if cfg.Spam.MinRepeatedTextLen <= 0 {
		cfg.Spam.MinRepeatedTextLen = 8
	}
	if cfg.Spam.LinkWindow == "" {
		cfg.Spam.LinkWindow = "30s"
	}
	if cfg.Spam.MaxLinks <= 0 {
		cfg.Spam.MaxLinks = 6
	}
	if cfg.Spam.MaxRepeatedLinks <= 0 {
		cfg.Spam.MaxRepeatedLinks = 3
	}
	if cfg.Spam.MaxLinksPerMessage <= 0 {
		cfg.Spam.MaxLinksPerMessage = 5
	}
	if cfg.Spam.MentionWindow == "" {
		cfg.Spam.MentionWindow = "30s"
	}
	if cfg.Spam.MaxMentions <= 0 {
		cfg.Spam.MaxMentions = 12
	}
	if cfg.Spam.MaxMentionsPerMsg <= 0 {
		cfg.Spam.MaxMentionsPerMsg = 8
	}
	if cfg.Spam.ForwardWindow == "" {
		cfg.Spam.ForwardWindow = "45s"
	}
	if cfg.Spam.MaxForwards <= 0 {
		cfg.Spam.MaxForwards = 5
	}
	if cfg.Spam.MuteDuration == "" {
		cfg.Spam.MuteDuration = "10m"
	}
	if cfg.Spam.ActionCooldown == "" {
		cfg.Spam.ActionCooldown = "20s"
	}
	if cfg.Spam.MuteAfterStrikes <= 0 {
		cfg.Spam.MuteAfterStrikes = 3
	}
	if cfg.Spam.StrikeWindow == "" {
		cfg.Spam.StrikeWindow = "2m"
	}
	if cfg.Spam.LogLimit <= 0 {
		cfg.Spam.LogLimit = 25
	}
	if cfg.Spam.Enabled && !cfg.Spam.DeleteExcess && !cfg.Spam.AutoMute {
		cfg.Spam.DeleteExcess = true
	}
}

func applyContentDefaults(cfg *Config) {
	if cfg.All.ChunkSize <= 0 {
		cfg.All.ChunkSize = 25
	}
	if cfg.All.DelayBetweenChunks == "" {
		cfg.All.DelayBetweenChunks = "1200ms"
	}
	if cfg.I18n.Locale == "" {
		cfg.I18n.Locale = "ru"
	}
	if cfg.Photos.Path == "" {
		cfg.Photos.Path = "data/photos.json"
	}
	if cfg.Photos.LocalDir == "" {
		cfg.Photos.LocalDir = "data/photos"
	}
	if len(cfg.Photos.LocalExtensions) == 0 {
		cfg.Photos.LocalExtensions = []string{".jpg", ".jpeg", ".png", ".webp"}
	}
	if cfg.Quotes.Path == "" {
		cfg.Quotes.Path = "data/quotes.json"
	}
	if cfg.Quotes.AutoReplyChance < 0 {
		cfg.Quotes.AutoReplyChance = 0
	}
	if cfg.Quotes.AutoReplyChance > 1 {
		cfg.Quotes.AutoReplyChance = 1
	}
	if cfg.Quotes.AutoReplyCooldown == "" {
		cfg.Quotes.AutoReplyCooldown = "30m"
	}
	if cfg.Quotes.ActiveUserWindow == "" {
		cfg.Quotes.ActiveUserWindow = "7d"
	}
	if cfg.Quotes.PhotoChance < 0 {
		cfg.Quotes.PhotoChance = 0
	}
	if cfg.Quotes.PhotoChance > 1 {
		cfg.Quotes.PhotoChance = 1
	}
	if cfg.FAQ.Path == "" {
		cfg.FAQ.Path = "data/faq.json"
	}
	if cfg.FAQ.EntryCooldown == "" {
		cfg.FAQ.EntryCooldown = "10m"
	}
	if cfg.FAQ.UserCooldown == "" {
		cfg.FAQ.UserCooldown = "1m"
	}
	if cfg.Ideas.UserCooldown == "" {
		cfg.Ideas.UserCooldown = "15m"
	}
	if cfg.Ideas.ChatCooldown == "" {
		cfg.Ideas.ChatCooldown = "30s"
	}
	if cfg.Ideas.MaxLength <= 0 {
		cfg.Ideas.MaxLength = 1500
	}
	if strings.TrimSpace(cfg.Discussion.Text) == "" {
		cfg.Discussion.Text = defaultDiscussionAutoReplyText
	}
}

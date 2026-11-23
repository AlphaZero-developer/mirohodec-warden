package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type I18nConfig struct {
	Locale string `yaml:"locale"`
}

var translations = loadTranslations()

func tr(locale, key string, args ...interface{}) string {
	locale = strings.ToLower(strings.TrimSpace(locale))
	if strings.HasPrefix(locale, "ru") {
		locale = "ru"
	}
	if strings.HasPrefix(locale, "en") {
		locale = "en"
	}
	msg := key
	if byKey, ok := translations[locale]; ok {
		if value, ok := byKey[key]; ok {
			msg = value
		}
	}
	if msg == key {
		if value, ok := translations["en"][key]; ok {
			msg = value
		}
	}
	if len(args) == 0 {
		return decorateTranslation(key, msg)
	}
	return decorateTranslation(key, fmt.Sprintf(msg, args...))
}

func decorateTranslation(key, msg string) string {
	prefix := translationSticker(key)
	msg = strings.TrimSpace(msg)
	if prefix == "" || msg == "" || strings.HasPrefix(msg, prefix) || startsWithVisualSticker(msg) {
		return msg
	}
	return prefix + " " + msg
}

func startsWithVisualSticker(msg string) bool {
	if msg == "" {
		return false
	}
	first, _ := utf8.DecodeRuneInString(msg)
	return first >= 0x1F000 || first == '✅' || first == '❌' || first == '⚠' || first == '⛔' || first == 'ℹ'
}

func translationSticker(key string) string {
	switch key {
	case "quote_author", "quote_list_line", "photo_list_line", "faq_list_line",
		"reason_line", "until_line",
		"forever", "never", "no_data",
		"duration_minute_one", "duration_minute_few", "duration_minute_many",
		"duration_hour_one", "duration_hour_few", "duration_hour_many":
		return ""
	}

	switch {
	case strings.HasPrefix(key, "usage_"):
		return "🧾"
	case strings.HasPrefix(key, "profile_"):
		return "👤"
	case strings.HasPrefix(key, "activity_") || strings.HasPrefix(key, "top_active_"):
		return "📊"
	case strings.HasPrefix(key, "status_"):
		return "📋"
	case strings.HasPrefix(key, "spam"):
		return "🧹"
	case strings.HasPrefix(key, "photo_"):
		return "🖼️"
	case strings.HasPrefix(key, "quote_"):
		return "💬"
	case strings.HasPrefix(key, "faq_"):
		return "📚"
	case strings.HasPrefix(key, "report_") || strings.HasPrefix(key, "idea_") || strings.HasPrefix(key, "bug_") || strings.HasPrefix(key, "appeal_"):
		return "📨"
	case strings.HasPrefix(key, "telegram_") || strings.HasPrefix(key, "err_") || strings.HasPrefix(key, "target_"):
		return "⚠️"
	case strings.HasPrefix(key, "laya_mood_"):
		return "🎭"
	case strings.HasPrefix(key, "laya_reaction_spam_"):
		return "🧹"
	case strings.HasPrefix(key, "laya_reaction_rehab_"):
		return "🏥"
	case strings.HasPrefix(key, "laya_reaction_idea_"):
		return "💡"
	case strings.HasPrefix(key, "laya_reaction_appeal_"):
		return "⚖️"
	}

	switch key {
	case "admin_help", "debug_help":
		return "📜"
	case "no_chat", "chat_id", "topic_id":
		return "🧭"
	case "no_sender", "myid_user", "myid_chat", "myid_topic", "myid_hint":
		return "🪪"
	case "no_message":
		return "✉️"
	case "no_admin":
		return "🚫"
	case "managed_chat_required":
		return "🏠"
	case "auth_done", "auth_private_only", "auth_disabled", "auth_invalid", "auth_save_failed":
		return "🔐"
	case "jail_done", "jail_notice", "jailed_warning", "rehab_done", "rehab_release_notice", "rehab_ticker_message":
		return "🏥"
	case "mute_done":
		return "🔇"
	case "unmute_done":
		return "🔊"
	case "ban_done":
		return "⛔"
	case "unban_done":
		return "✅"
	case "all_disabled", "all_default_prefix", "all_send_failed", "unknown_members":
		return "📣"
	case "reload_done", "reload_failed":
		return "🔄"
	case "job_sent":
		return "💼"
	case "donate_sent":
		return "💸"
	case "duration_required", "duration_too_short":
		return "⚠️"
	default:
		return ""
	}
}

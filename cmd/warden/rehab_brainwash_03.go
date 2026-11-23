package main

import (
	"strconv"
	"strings"
)

func rehabBrainwashMemoryKey(cfg RehabBrainwashConfig, jail ModerationRecord) string {
	cfg = normalizeRehabBrainwashConfig(cfg)
	userKey := strconv.FormatInt(jail.UserID, 10)
	if jail.UserID == 0 {
		userKey = aiReplySignature(jail.DisplayName)
	}
	leaderKey := strconv.FormatInt(cfg.LeaderUserID, 10)
	if cfg.LeaderUserID == 0 {
		leaderKey = aiReplySignature(cfg.LeaderName)
	}
	if userKey == "" || leaderKey == "" {
		return ""
	}
	return userKey + ":" + leaderKey
}

func rehabBrainwashMessageLines(messages string) []string {
	messages = strings.TrimSpace(messages)
	if messages == "" {
		messages = defaultRehabBrainwashMessages
	}
	rawLines := strings.Split(messages, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func renderRehabBrainwashMessage(template string, cfg RehabBrainwashConfig, jail ModerationRecord) string {
	cfg = normalizeRehabBrainwashConfig(cfg)
	userName := strings.TrimSpace(jail.DisplayName)
	if userName == "" {
		userName = "пациент"
	}
	leaderID := ""
	if cfg.LeaderUserID != 0 {
		leaderID = strconv.FormatInt(cfg.LeaderUserID, 10)
	}
	userID := ""
	if jail.UserID != 0 {
		userID = strconv.FormatInt(jail.UserID, 10)
	}
	replacements := map[string]string{
		"{leader}":      cfg.LeaderName,
		"{leader_name}": cfg.LeaderName,
		"{leader_id}":   leaderID,
		"{user}":        userName,
		"{name}":        userName,
		"{user_id}":     userID,
	}
	out := template
	for from, to := range replacements {
		out = strings.ReplaceAll(out, from, to)
	}
	return out
}

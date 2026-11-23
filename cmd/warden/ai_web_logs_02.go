package main

import (
	"errors"
	"fmt"
	"strings"
)

func aiWebLogMessages(messages []AIMessage) []aiWebLogMessage {
	if len(messages) == 0 {
		return nil
	}
	out := make([]aiWebLogMessage, 0, len(messages))
	for _, msg := range messages {
		role := strings.TrimSpace(msg.Role)
		content := strings.TrimSpace(msg.Content)
		if role == "" || content == "" {
			continue
		}
		out = append(out, aiWebLogMessage{
			Role:    role,
			Content: content,
			Chars:   runeLen(content),
		})
	}
	return out
}

func aiWebLogError(err error) string {
	if err == nil {
		return ""
	}
	var apiErr aiAPIError
	if errors.As(err, &apiErr) {
		if strings.TrimSpace(apiErr.Message) == "" {
			return fmt.Sprintf("OpenRouter status %d", apiErr.StatusCode)
		}
		return fmt.Sprintf("OpenRouter status %d: %s", apiErr.StatusCode, strings.TrimSpace(apiErr.Message))
	}
	return strings.TrimSpace(err.Error())
}

func aiWebLogLevel(status string, err error) string {
	status = strings.ToLower(strings.TrimSpace(status))
	if err == nil {
		if strings.Contains(status, "fallback") {
			return "warn"
		}
		return "info"
	}
	if errors.Is(err, errAISoftTimeout) || errors.Is(err, errAIEmptyReply) {
		return "warn"
	}
	return "error"
}

func runeLen(value string) int {
	return len([]rune(value))
}

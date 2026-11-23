package main

import (
	"context"
	"errors"
	"fmt"
	wardenweb "mirohodec-warden/internal/web"
	"strconv"
	"strings"
	"time"
)

func parseLayaRelationshipsText(value string) ([]LayaRelationship, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	lines := strings.Split(value, "\n")
	out := make([]LayaRelationship, 0, len(lines))
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 2 {
			return nil, fmt.Errorf("laya relationship line %d: use user_id | role | optional prompt", i+1)
		}
		userID, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		if err != nil || userID == 0 {
			return nil, fmt.Errorf("laya relationship line %d: user_id must be numeric", i+1)
		}
		rec := LayaRelationship{
			UserID: userID,
			Role:   strings.TrimSpace(parts[1]),
		}
		if len(parts) > 2 {
			rec.Prompt = strings.TrimSpace(parts[2])
		}
		out = append(out, rec)
	}
	return normalizeLayaRelationships(out), nil
}

func layaRelationshipsText(rels []LayaRelationship) string {
	rels = normalizeLayaRelationships(rels)
	if len(rels) == 0 {
		return ""
	}
	lines := make([]string, 0, len(rels))
	for _, rec := range rels {
		parts := []string{strconv.FormatInt(rec.UserID, 10), rec.Role}
		if strings.TrimSpace(rec.Prompt) != "" {
			parts = append(parts, rec.Prompt)
		}
		lines = append(lines, strings.Join(parts, " | "))
	}
	return strings.Join(lines, "\n")
}

func webLayaRelationshipRows(rels []LayaRelationship) []wardenweb.LayaRelationshipRow {
	rels = normalizeLayaRelationships(rels)
	out := make([]wardenweb.LayaRelationshipRow, 0, len(rels))
	for _, rec := range rels {
		out = append(out, wardenweb.LayaRelationshipRow{
			UserID: rec.UserID,
			Role:   rec.Role,
			Prompt: rec.Prompt,
		})
	}
	return out
}

func (s webPanelService) ClearAICache(ctx context.Context) error {
	if s.app == nil {
		return errors.New("app is not ready")
	}
	s.app.clearAICaches()
	return nil
}

func (s webPanelService) AILogs(ctx context.Context) (wardenweb.AILogPage, error) {
	if s.app == nil {
		return wardenweb.AILogPage{}, errors.New("app is not ready")
	}
	entries := s.app.aiWebLogSnapshot(maxAIWebLogEntries)
	page := wardenweb.AILogPage{
		Rows:       make([]wardenweb.AILogRow, 0, len(entries)),
		MaxEntries: maxAIWebLogEntries,
	}
	var totalElapsed time.Duration
	for _, entry := range entries {
		page.Rows = append(page.Rows, webAILogRow(entry))
		page.Total++
		totalElapsed += entry.Elapsed
		switch entry.Level {
		case "warn":
			page.Warn++
		case "error":
			page.Error++
		default:
			page.Info++
		}
		if entry.OK {
			page.OK++
		} else {
			page.Errors++
		}
	}
	if page.Total > 0 {
		page.AverageTime = (totalElapsed / time.Duration(page.Total)).Truncate(time.Millisecond).String()
		page.LastEntry = page.Rows[0].At
	}
	return page, nil
}

func (s webPanelService) ClearAILogs(ctx context.Context) error {
	if s.app == nil {
		return errors.New("app is not ready")
	}
	s.app.clearAIWebLogs()
	return nil
}

func webAILogRow(entry aiWebLogEntry) wardenweb.AILogRow {
	return wardenweb.AILogRow{
		ID:            entry.ID,
		At:            webFormatTime(entry.At),
		Source:        entry.Source,
		Stage:         entry.Stage,
		Level:         webAILogLevel(entry.Level),
		Status:        entry.Status,
		Error:         entry.Error,
		OK:            entry.OK,
		Elapsed:       webFormatDuration(entry.Elapsed),
		Model:         entry.Model,
		BaseURL:       entry.BaseURL,
		MaxTokens:     entry.MaxTokens,
		MaxReplyChars: entry.MaxReplyChars,
		UserID:        entry.UserID,
		User:          webAILogUser(entry),
		Username:      entry.Username,
		ChatID:        entry.ChatID,
		Chat:          webAILogChat(entry),
		ChatType:      entry.ChatType,
		ThreadID:      entry.ThreadID,
		MessageID:     entry.MessageID,
		UserInput:     entry.UserInput,
		SystemPrompt:  entry.SystemPrompt,
		Messages:      webAILogMessages(entry.Messages),
		RawReply:      entry.RawReply,
		FinalReply:    entry.FinalReply,
		RequestChars:  entry.RequestChars,
		ResponseChars: entry.ResponseChars,
	}
}

func webAILogMessages(messages []aiWebLogMessage) []wardenweb.AILogMessage {
	if len(messages) == 0 {
		return nil
	}
	out := make([]wardenweb.AILogMessage, 0, len(messages))
	for _, msg := range messages {
		out = append(out, wardenweb.AILogMessage{
			Role:    msg.Role,
			Content: msg.Content,
			Chars:   msg.Chars,
		})
	}
	return out
}

func webAILogLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "warn":
		return "warn"
	case "error":
		return "error"
	default:
		return "info"
	}
}

func webAILogUser(entry aiWebLogEntry) string {
	if strings.TrimSpace(entry.UserName) != "" && entry.UserID != 0 {
		return fmt.Sprintf("%s (%d)", strings.TrimSpace(entry.UserName), entry.UserID)
	}
	if strings.TrimSpace(entry.UserName) != "" {
		return strings.TrimSpace(entry.UserName)
	}
	if entry.UserID != 0 {
		return strconv.FormatInt(entry.UserID, 10)
	}
	return "-"
}

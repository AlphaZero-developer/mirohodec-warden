package main

import (
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

const maxAIWebLogEntries = 120

type aiWebLogMeta struct {
	Source    string
	Stage     string
	ChatID    int64
	ChatTitle string
	ChatType  string
	ThreadID  int
	MessageID int
}

type aiWebLogMessage struct {
	Role    string
	Content string
	Chars   int
}

type aiWebLogEntry struct {
	ID            uint64
	At            time.Time
	Source        string
	Stage         string
	Level         string
	Status        string
	Error         string
	OK            bool
	Elapsed       time.Duration
	Model         string
	BaseURL       string
	MaxTokens     int
	MaxReplyChars int
	UserID        int64
	UserName      string
	Username      string
	ChatID        int64
	ChatTitle     string
	ChatType      string
	ThreadID      int
	MessageID     int
	UserInput     string
	SystemPrompt  string
	Messages      []aiWebLogMessage
	RawReply      string
	FinalReply    string
	RequestChars  int
	ResponseChars int
}

func aiWebLogMetaFromTelegram(chat *tele.Chat, msg *tele.Message) aiWebLogMeta {
	meta := aiWebLogMeta{Source: "Telegram"}
	if chat != nil {
		meta.ChatID = chat.ID
		meta.ChatTitle = strings.TrimSpace(chat.Title)
		meta.ChatType = strings.TrimSpace(string(chat.Type))
		if chat.Type == tele.ChatPrivate {
			meta.Source = "Telegram private"
		} else if chat.Type != "" {
			meta.Source = "Telegram chat"
		}
	}
	if msg != nil {
		meta.ThreadID = msg.ThreadID
		meta.MessageID = msg.ID
	}
	return meta
}

func (m aiWebLogMeta) withStage(stage string) aiWebLogMeta {
	m.Stage = strings.TrimSpace(stage)
	return m
}

func (m aiWebLogMeta) normalized() aiWebLogMeta {
	m.Source = strings.TrimSpace(m.Source)
	if m.Source == "" {
		m.Source = "AI"
	}
	m.Stage = strings.TrimSpace(m.Stage)
	if m.Stage == "" {
		m.Stage = "single request"
	}
	m.ChatTitle = strings.TrimSpace(m.ChatTitle)
	m.ChatType = strings.TrimSpace(m.ChatType)
	return m
}

func (a *App) recordAIWebLog(meta aiWebLogMeta, cfg AIConfig, sender *tele.User, messages []AIMessage, userInput, rawReply, finalReply, status string, elapsed time.Duration, err error) {
	if a == nil {
		return
	}
	meta = meta.normalized()
	entry := aiWebLogEntry{
		At:            time.Now().UTC(),
		Source:        meta.Source,
		Stage:         meta.Stage,
		Level:         aiWebLogLevel(status, err),
		Status:        strings.TrimSpace(status),
		Error:         aiWebLogError(err),
		OK:            err == nil,
		Elapsed:       elapsed,
		Model:         strings.TrimSpace(cfg.Model),
		BaseURL:       strings.TrimSpace(cfg.BaseURL),
		MaxTokens:     cfg.MaxTokens,
		MaxReplyChars: cfg.MaxReplyChars,
		ChatID:        meta.ChatID,
		ChatTitle:     meta.ChatTitle,
		ChatType:      meta.ChatType,
		ThreadID:      meta.ThreadID,
		MessageID:     meta.MessageID,
		UserInput:     strings.TrimSpace(userInput),
		SystemPrompt:  strings.TrimSpace(cfg.SystemPrompt),
		RawReply:      strings.TrimSpace(rawReply),
		FinalReply:    strings.TrimSpace(finalReply),
	}
	if sender != nil {
		entry.UserID = sender.ID
		entry.UserName = displayNameFromUser(sender)
		entry.Username = strings.TrimPrefix(strings.TrimSpace(sender.Username), "@")
	}
	entry.Messages = aiWebLogMessages(messages)
	entry.RequestChars = runeLen(entry.SystemPrompt)
	for _, msg := range entry.Messages {
		entry.RequestChars += msg.Chars
	}
	if entry.RawReply != "" {
		entry.ResponseChars = runeLen(entry.RawReply)
	} else {
		entry.ResponseChars = runeLen(entry.FinalReply)
	}
	a.appendAIWebLog(entry)
}

func (a *App) appendAIWebLog(entry aiWebLogEntry) {
	a.aiWebLogMu.Lock()
	defer a.aiWebLogMu.Unlock()
	a.aiWebLogNext++
	entry.ID = a.aiWebLogNext
	a.aiWebLogs = append(a.aiWebLogs, entry)
	if len(a.aiWebLogs) > maxAIWebLogEntries {
		a.aiWebLogs = append([]aiWebLogEntry(nil), a.aiWebLogs[len(a.aiWebLogs)-maxAIWebLogEntries:]...)
	}
}

func (a *App) aiWebLogSnapshot(limit int) []aiWebLogEntry {
	if a == nil {
		return nil
	}
	a.aiWebLogMu.Lock()
	defer a.aiWebLogMu.Unlock()
	if limit <= 0 || limit > len(a.aiWebLogs) {
		limit = len(a.aiWebLogs)
	}
	out := make([]aiWebLogEntry, 0, limit)
	for i := len(a.aiWebLogs) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, cloneAIWebLogEntry(a.aiWebLogs[i]))
	}
	return out
}

func (a *App) clearAIWebLogs() {
	if a == nil {
		return
	}
	a.aiWebLogMu.Lock()
	defer a.aiWebLogMu.Unlock()
	a.aiWebLogs = nil
}

func cloneAIWebLogEntry(entry aiWebLogEntry) aiWebLogEntry {
	if len(entry.Messages) > 0 {
		entry.Messages = append([]aiWebLogMessage(nil), entry.Messages...)
	}
	return entry
}

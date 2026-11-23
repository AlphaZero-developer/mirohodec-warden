package main

import (
	crand "crypto/rand"
	"math/big"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func layaChoiceOptionsByWord(rest string) []string {
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return nil
	}

	var raw []string
	var current []string
	flush := func() {
		if len(current) == 0 {
			return
		}
		raw = append(raw, strings.Join(current, " "))
		current = nil
	}
	for _, field := range fields {
		token := normalizeLayaCommandToken(field)
		if token == "или" || token == "or" {
			flush()
			continue
		}
		current = append(current, field)
	}
	flush()
	return cleanLayaOptions(raw)
}

func cleanLayaOptions(raw []string) []string {
	options := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, option := range raw {
		option = strings.TrimSpace(strings.Trim(option, " \t\r\n.,;:!?\"'`"))
		if option == "" {
			continue
		}
		key := strings.ToLower(option)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		options = append(options, option)
	}
	return options
}

func (a *App) randomLayaMember(c tele.Context, excludeID int64) (MemberRecord, bool) {
	if a == nil || a.store == nil || c == nil {
		return MemberRecord{}, false
	}
	chat := c.Chat()
	if chat == nil || chat.Type == tele.ChatPrivate {
		return MemberRecord{}, false
	}
	activeWindow := mustDuration(a.activeQuoteConfigForChat(chat).ActiveUserWindow, 7*24*time.Hour)
	return a.store.RandomActiveMember(chat.ID, activeWindow, time.Now().UTC(), excludeID)
}

func randomString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[randomIndex(len(values))]
}

func randomIndex(limit int) int {
	if limit <= 1 {
		return 0
	}
	n, err := crand.Int(crand.Reader, big.NewInt(int64(limit)))
	if err == nil {
		return int(n.Int64())
	}
	return int(time.Now().UnixNano() % int64(limit))
}

func firstNonNil(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode"

	tele "gopkg.in/telebot.v3"
)

func spamLinkSignatures(msg *tele.Message) []string {
	var out []string
	seen := make(map[string]struct{})
	add := func(raw string) {
		signature := normalizeSpamLink(raw)
		if signature == "" {
			return
		}
		if _, ok := seen[signature]; ok {
			return
		}
		seen[signature] = struct{}{}
		out = append(out, signature)
	}

	for _, entity := range append(msg.Entities, msg.CaptionEntities...) {
		if entity.Type == tele.EntityTextLink && entity.URL != "" {
			add(entity.URL)
		}
	}
	for _, raw := range spamURLPattern.FindAllString(msg.Text+" "+msg.Caption, -1) {
		add(raw)
	}
	return out
}

func normalizeSpamLink(raw string) string {
	raw = strings.TrimSpace(strings.TrimRight(raw, ".,!?;:)]}>'\""))
	if raw == "" {
		return ""
	}
	normalized := raw
	if !strings.Contains(normalized, "://") {
		normalized = "https://" + normalized
	}
	parsed, err := url.Parse(normalized)
	if err != nil {
		return strings.ToLower(raw)
	}
	host := strings.ToLower(parsed.Hostname())
	if strings.HasPrefix(host, "www.") {
		host = strings.TrimPrefix(host, "www.")
	}
	if host == "" {
		return ""
	}

	path := strings.Trim(parsed.EscapedPath(), "/")
	if (host == "t.me" || host == "telegram.me") && path != "" {
		first, _, _ := strings.Cut(path, "/")
		if first != "" {
			return host + "/" + strings.ToLower(first)
		}
	}
	return host
}

func spamMentionSignatures(msg *tele.Message) []string {
	var out []string
	seen := make(map[string]struct{})
	add := func(value string) {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}

	for _, entity := range append(msg.Entities, msg.CaptionEntities...) {
		if entity.Type == tele.EntityTMention && entity.User != nil {
			add(fmt.Sprintf("user:%d", entity.User.ID))
		}
	}
	for _, raw := range spamMentionPattern.FindAllString(msg.Text+" "+msg.Caption, -1) {
		add(raw)
	}
	return out
}

func isMediaMessage(msg *tele.Message) bool {
	return msg.Photo != nil || msg.Video != nil || msg.Animation != nil || msg.VideoNote != nil
}

func isOtherSpamCandidate(msg *tele.Message) bool {
	return msg.Audio != nil ||
		msg.Document != nil ||
		msg.Voice != nil ||
		msg.Contact != nil ||
		msg.Location != nil ||
		msg.Venue != nil ||
		msg.Poll != nil ||
		msg.Game != nil ||
		msg.Dice != nil
}

func normalizeSpamText(value string) string {
	value = strings.ToLower(strings.ReplaceAll(value, "ё", "е"))
	var b strings.Builder
	lastSpace := true
	for _, r := range value {
		switch {
		case isZeroWidthRune(r):
			continue
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastSpace = false
		default:
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func pruneSpamEvents(events []spamEvent, cutoff time.Time) []spamEvent {
	n := 0
	for _, event := range events {
		if event.At.Before(cutoff) {
			continue
		}
		events[n] = event
		n++
	}
	for i := n; i < len(events); i++ {
		events[i] = spamEvent{}
	}
	return events[:n]
}

func pruneSpamStrikes(strikes []spamStrike, cutoff time.Time) []spamStrike {
	n := 0
	for _, strike := range strikes {
		if strike.At.Before(cutoff) {
			continue
		}
		strikes[n] = strike
		n++
	}
	for i := n; i < len(strikes); i++ {
		strikes[i] = spamStrike{}
	}
	return strikes[:n]
}

func countSpamStrikes(strikes []spamStrike, confidence string) int {
	count := 0
	for _, strike := range strikes {
		if strike.Confidence == confidence {
			count++
		}
	}
	return count
}

func countSpamEvents(events []spamEvent, cutoff time.Time, kind string) int {
	count := 0
	for _, event := range events {
		if event.At.Before(cutoff) {
			continue
		}
		if event.Kind != kind {
			continue
		}
		if event.Count > 0 {
			count += event.Count
		} else {
			count++
		}
	}
	return count
}

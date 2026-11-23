package main

import (
	"strings"
	"unicode"
)

func defaultFAQRecords() map[string]FAQRecord {
	return map[string]FAQRecord{
		"season_release": {
			ID: "season_release",
			Patterns: []string{
				"когда выйдет сезон",
				"когда сезон",
				"дата выхода сезона",
				"season release",
			},
			Answer:  "Примерная дата выхода: 1 января 2027.",
			Enabled: true,
		},
	}
}

func normalizeFAQPatterns(patterns []string) []string {
	seen := make(map[string]struct{}, len(patterns))
	out := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		key := normalizeFAQText(pattern)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, pattern)
	}
	return out
}

func normalizeFAQText(value string) string {
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

func faqMatchScore(question string, patterns []string) int {
	best := 0
	for _, pattern := range patterns {
		pattern = normalizeFAQText(pattern)
		if pattern == "" {
			continue
		}
		if question == pattern {
			if score := 1000 + len(strings.Fields(pattern)); score > best {
				best = score
			}
			continue
		}
		if strings.Contains(question, pattern) {
			if score := 800 + len(strings.Fields(pattern)); score > best {
				best = score
			}
			continue
		}
		words := strings.Fields(pattern)
		if len(words) < 2 {
			continue
		}
		matched := 0
		for _, word := range words {
			if strings.Contains(" "+question+" ", " "+word+" ") {
				matched++
			}
		}
		if matched == len(words) {
			if score := 500 + matched; score > best {
				best = score
			}
		}
	}
	return best
}

func isZeroWidthRune(r rune) bool {
	switch r {
	case '\u200b', '\u200c', '\u200d', '\u2060', '\ufeff':
		return true
	default:
		return false
	}
}

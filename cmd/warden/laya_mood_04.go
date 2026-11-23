package main

import (
	"strings"
)

func isDumbLayaQuestion(text, botUsername string) bool {
	normalized := normalizeFAQText(text)
	if normalized == "" {
		return false
	}
	if botNormalized := normalizeFAQText(botUsername); botNormalized != "" {
		normalized = strings.TrimSpace(strings.ReplaceAll(" "+normalized+" ", " "+botNormalized+" ", " "))
	}
	for _, phrase := range dumbQuestionPhrases {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	if !strings.Contains(text, "?") {
		return false
	}
	words := stripLayaAddressWords(strings.Fields(normalized), botUsername)
	if len(words) == 0 || len(words) > 2 {
		return false
	}
	for _, word := range words {
		if !tinyQuestionWords[word] {
			return false
		}
	}
	return true
}

func stripLayaAddressWords(words []string, botUsername string) []string {
	botUsername = normalizeUsername(botUsername)
	out := make([]string, 0, len(words))
	for _, word := range words {
		switch {
		case word == "лайя" || word == "laya":
			continue
		case botUsername != "" && normalizeUsername(word) == botUsername:
			continue
		default:
			out = append(out, word)
		}
	}
	return out
}

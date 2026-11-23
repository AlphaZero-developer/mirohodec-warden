package main

import (
	"strings"
)

func layaDuplicateTextKey(text string, minLen int) string {
	text = strings.TrimSpace(text)
	text = strings.Trim(text, " \t\"'«»“”„`*_-.!?,;:…")
	text = strings.ToLower(strings.Join(strings.Fields(text), " "))
	if len([]rune(text)) < minLen {
		return ""
	}
	return text
}

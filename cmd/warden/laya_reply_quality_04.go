package main

import (
	"strings"
)

func stripLayaSpeakerPrefix(reply string) string {
	for i := 0; i < 4; i++ {
		trimmed := strings.TrimSpace(reply)
		next, ok := stripLayaLeadingLabel(trimmed)
		if !ok {
			return trimmed
		}
		reply = next
	}
	return strings.TrimSpace(reply)
}

func stripLayaLeadingLabel(reply string) (string, bool) {
	clean := strings.TrimLeft(reply, "*_` \t")
	if rest, ok := stripLayaLeadingLabelBySeparator(clean, ":"); ok {
		return rest, true
	}
	if rest, ok := stripLayaLeadingLabelBySeparator(clean, "："); ok {
		return rest, true
	}
	for _, sep := range []string{" — ", " – ", " - ", "— ", "– ", "- "} {
		if rest, ok := stripLayaLeadingLabelBySeparator(clean, sep); ok {
			return rest, true
		}
	}
	return reply, false
}

func stripLayaLeadingLabelBySeparator(reply, sep string) (string, bool) {
	idx := strings.Index(reply, sep)
	if idx < 0 || idx > 48 {
		return reply, false
	}
	label := strings.ToLower(cleanLayaInlineLabel(reply[:idx]))
	if !isLayaSpeakerLabel(label) && !isLayaReplyLabel(label) {
		return reply, false
	}
	rest := strings.TrimSpace(reply[idx+len(sep):])
	rest = strings.TrimLeft(rest, "*_` \t-—–")
	return strings.TrimSpace(rest), true
}

func cleanLayaInlineLabel(label string) string {
	label = strings.TrimSpace(label)
	label = strings.Trim(label, "*_`[]() \t")
	label = strings.TrimRight(label, ":：.-—– \t")
	return strings.TrimSpace(label)
}

func isLayaSpeakerLabel(label string) bool {
	if label == "лайя" || label == "лая" || label == "laya" || label == "layla" {
		return true
	}
	if strings.HasPrefix(label, "лайя ") || strings.HasPrefix(label, "лая ") || strings.HasPrefix(label, "laya ") {
		return true
	}
	return label == "assistant" || label == "ассистент" || label == "bot" || label == "бот"
}

func isLayaReplyLabel(label string) bool {
	switch label {
	case "ответ", "ответ лайи", "ответ лаи", "ответ laya", "финальный ответ", "итоговый ответ",
		"сообщение", "сообщение лайи", "ответ пользователю", "текст ответа",
		"вариант", "вариант ответа", "готовый ответ", "ответ в чат",
		"финал", "итог",
		"reply", "answer", "response", "final", "final answer":
		return true
	default:
		return false
	}
}

func isLayaAnalysisLabel(label string) bool {
	switch label {
	case "analysis", "reasoning", "thought", "thinking", "разбор", "анализ", "рассуждение", "мысли":
		return true
	default:
		return false
	}
}

func isLayaDialoguePromptLabel(label string) bool {
	switch label {
	case "user", "human", "client", "пользователь", "юзер", "человек", "собеседник",
		"system", "developer", "инструкция", "контекст", "запрос":
		return true
	default:
		return false
	}
}

func stripLayaMarkdownWrapper(reply string) string {
	reply = strings.TrimSpace(reply)
	for i := 0; i < 3; i++ {
		changed := false
		for _, pair := range [][2]string{
			{"**", "**"},
			{"__", "__"},
			{"~~", "~~"},
			{"*", "*"},
			{"_", "_"},
			{"`", "`"},
		} {
			if strings.HasPrefix(reply, pair[0]) && strings.HasSuffix(reply, pair[1]) {
				inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(reply, pair[0]), pair[1]))
				if inner != "" {
					reply = inner
					changed = true
					break
				}
			}
		}
		if !changed {
			return reply
		}
	}
	return reply
}

func stripLayaOuterQuotes(reply string) string {
	reply = strings.TrimSpace(reply)
	pairs := [][2]string{
		{`"`, `"`},
		{"'", "'"},
		{"«", "»"},
		{"“", "”"},
		{"„", "“"},
		{"`", "`"},
	}
	for _, pair := range pairs {
		if strings.HasPrefix(reply, pair[0]) && strings.HasSuffix(reply, pair[1]) {
			inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(reply, pair[0]), pair[1]))
			if inner != "" {
				return inner
			}
		}
	}
	return reply
}

func collapseLayaAdjacentDuplicateParagraphs(reply string) string {
	paragraphs := strings.Split(strings.TrimSpace(reply), "\n\n")
	if len(paragraphs) < 2 {
		return reply
	}
	out := make([]string, 0, len(paragraphs))
	prevKey := ""
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		key := layaDuplicateTextKey(paragraph, 16)
		if key != "" && key == prevKey {
			continue
		}
		out = append(out, paragraph)
		prevKey = key
	}
	return strings.TrimSpace(strings.Join(out, "\n\n"))
}

func collapseLayaAdjacentDuplicateLines(reply string) string {
	lines := strings.Split(strings.TrimSpace(reply), "\n")
	if len(lines) < 2 {
		return reply
	}
	out := make([]string, 0, len(lines))
	prevKey := ""
	for _, line := range lines {
		key := layaDuplicateTextKey(line, 8)
		if key != "" && key == prevKey {
			continue
		}
		out = append(out, line)
		if strings.TrimSpace(line) == "" {
			prevKey = ""
		} else {
			prevKey = key
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

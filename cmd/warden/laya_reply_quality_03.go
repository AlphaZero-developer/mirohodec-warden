package main

import (
	"strings"
)

func stripLayaFinalChannelSection(reply string) string {
	lines := strings.Split(strings.TrimSpace(reply), "\n")
	if len(lines) < 2 {
		return reply
	}
	seenAnalysis := false
	for i, line := range lines {
		label, rest, ok := layaLineLabelAndRest(line)
		if !ok {
			continue
		}
		if isLayaAnalysisLabel(label) {
			seenAnalysis = true
			continue
		}
		if seenAnalysis && (isLayaReplyLabel(label) || isLayaSpeakerLabel(label)) {
			candidate := append([]string{rest}, lines[i+1:]...)
			return strings.TrimSpace(strings.Join(candidate, "\n"))
		}
	}
	return reply
}

func stripLayaDialogueEcho(reply string) string {
	lines := strings.Split(strings.TrimSpace(reply), "\n")
	if len(lines) < 2 {
		return reply
	}
	seenPrompt := false
	replyLine := -1
	replyRest := ""
	for i, line := range lines {
		label, rest, ok := layaLineLabelAndRest(line)
		if !ok {
			continue
		}
		if isLayaDialoguePromptLabel(label) {
			seenPrompt = true
			continue
		}
		if seenPrompt && (isLayaSpeakerLabel(label) || isLayaReplyLabel(label)) && strings.TrimSpace(rest) != "" {
			replyLine = i
			replyRest = rest
		}
	}
	if replyLine < 0 {
		return reply
	}
	candidate := append([]string{replyRest}, lines[replyLine+1:]...)
	return strings.TrimSpace(strings.Join(candidate, "\n"))
}

func stripLayaDanglingRoleTail(reply string) string {
	lines := strings.Split(strings.TrimSpace(reply), "\n")
	for len(lines) > 1 {
		last := strings.TrimSpace(lines[len(lines)-1])
		label, rest, ok := layaLineLabelAndRest(last)
		if !ok || strings.TrimSpace(rest) != "" {
			break
		}
		if !isLayaDialoguePromptLabel(label) && !isLayaSpeakerLabel(label) && !isLayaReplyLabel(label) && !isLayaAnalysisLabel(label) {
			break
		}
		lines = lines[:len(lines)-1]
		for len(lines) > 1 && strings.TrimSpace(lines[len(lines)-1]) == "" {
			lines = lines[:len(lines)-1]
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func stripLayaTrailingServiceLines(reply string) string {
	lines := strings.Split(strings.TrimSpace(reply), "\n")
	for len(lines) > 1 {
		last := strings.TrimSpace(lines[len(lines)-1])
		if last == "" || isLayaServiceLeadLine(last) {
			lines = lines[:len(lines)-1]
			continue
		}
		break
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func isLayaServiceLeadLine(line string) bool {
	clean := cleanLayaLineLabel(line)
	if clean == "" || clean == "---" || clean == "___" || clean == "***" {
		return true
	}
	lower := strings.ToLower(clean)
	if isLayaSpeakerLabel(lower) || isLayaReplyLabel(lower) {
		return true
	}
	for _, prefix := range []string{
		"вот ответ",
		"держи ответ",
		"готовый ответ",
		"вариант ответа",
		"вариант",
		"я бы ответила так",
		"можно ответить так",
		"ответ можно сделать таким",
		"response",
		"here is the answer",
		"final answer",
	} {
		if lower == prefix {
			return true
		}
	}
	return false
}

func cleanLayaLineLabel(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimLeft(line, "#> \t")
	line = strings.TrimSpace(stripLayaMarkdownWrapper(line))
	line = strings.Trim(line, "*_` \t")
	line = strings.TrimRight(line, ":：.-—– \t")
	return strings.TrimSpace(line)
}

func layaLineLabelAndRest(line string) (string, string, bool) {
	clean := strings.TrimLeft(strings.TrimSpace(line), "*_`#> \t")
	for _, sep := range []string{":", "：", " — ", " – ", " - ", "— ", "– ", "- "} {
		idx := strings.Index(clean, sep)
		if idx < 0 || idx > 48 {
			continue
		}
		label := strings.ToLower(cleanLayaInlineLabel(clean[:idx]))
		rest := strings.TrimSpace(clean[idx+len(sep):])
		rest = strings.TrimLeft(rest, "*_` \t-—–")
		return label, strings.TrimSpace(rest), label != ""
	}
	return "", "", false
}

func stripLayaQuoteBlock(reply string) string {
	lines := strings.Split(strings.TrimSpace(reply), "\n")
	nonEmpty := 0
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			out = append(out, "")
			continue
		}
		if !strings.HasPrefix(trimmed, ">") {
			return reply
		}
		nonEmpty++
		out = append(out, strings.TrimSpace(strings.TrimLeft(trimmed, "> \t")))
	}
	if nonEmpty == 0 {
		return reply
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func stripLayaSingleListMarker(reply string) string {
	lines := strings.Split(strings.TrimSpace(reply), "\n")
	nonEmpty := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmpty++
		}
	}
	if nonEmpty != 1 {
		return reply
	}
	line := strings.TrimSpace(reply)
	for _, marker := range []string{"- ", "– ", "— ", "• ", "* "} {
		if strings.HasPrefix(line, marker) {
			return strings.TrimSpace(line[len(marker):])
		}
	}
	for i := 1; i <= 9; i++ {
		prefix := string(rune('0'+i)) + ". "
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(line[len(prefix):])
		}
	}
	return reply
}

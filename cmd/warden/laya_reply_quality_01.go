package main

import (
	"html"
	"strings"
)

func polishLayaReply(reply string) string {
	reply = normalizeLayaReplyText(reply)
	reply = stripLayaThinkingBlocks(reply)
	reply = stripLayaHTMLComments(reply)
	for i := 0; i < 4; i++ {
		before := reply
		reply = stripLayaCodeFence(reply)
		reply = stripLayaJSONEnvelope(reply)
		reply = stripLayaXMLReplyEnvelope(reply)
		reply = stripLayaHTMLReplyEnvelope(reply)
		reply = stripLayaHTMLComments(reply)
		reply = stripLayaInlineHTMLMarkup(reply)
		reply = stripLayaMarkdownImages(reply)
		reply = stripLayaFinalChannelSection(reply)
		reply = stripLayaDialogueEcho(reply)
		reply = stripLayaDanglingRoleTail(reply)
		reply = stripLayaTrailingServiceLines(reply)
		reply = stripLayaServiceLeadLines(reply)
		reply = stripLayaQuoteBlock(reply)
		reply = stripLayaSingleListMarker(reply)
		reply = stripLayaSpeakerPrefix(reply)
		reply = stripLayaMarkdownWrapper(reply)
		reply = stripLayaOuterQuotes(reply)
		reply = normalizeLayaReplyText(reply)
		reply = collapseLayaAdjacentDuplicateParagraphs(reply)
		reply = collapseLayaAdjacentDuplicateLines(reply)
		if reply == before {
			break
		}
	}
	return reply
}

func normalizeLayaReplyText(reply string) string {
	replacer := strings.NewReplacer(
		"\r\n", "\n",
		"\r", "\n",
		"\u00a0", " ",
		"\u200b", "",
		"\u200c", "",
		"\u200d", "",
		"\u2060", "",
		"\ufeff", "",
	)
	reply = replacer.Replace(reply)
	reply = html.UnescapeString(reply)
	reply = stripLayaChatTemplateTokens(reply)
	reply = normalizeLayaHTMLLineBreaks(reply)
	lines := strings.Split(reply, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return compactLayaBlankLines(strings.TrimSpace(strings.Join(lines, "\n")))
}

func stripLayaChatTemplateTokens(reply string) string {
	replacer := strings.NewReplacer(
		"<|begin_of_text|>", "",
		"<|end_of_text|>", "",
		"<|endoftext|>", "",
		"<|eot_id|>", "",
		"<|start_header_id|>", "",
		"<|end_header_id|>", "\n",
		"<|im_start|>", "",
		"<|im_end|>", "",
		"<s>", "",
		"</s>", "",
		"[INST]", "",
		"[/INST]", "",
	)
	return replacer.Replace(reply)
}

func normalizeLayaHTMLLineBreaks(reply string) string {
	replacer := strings.NewReplacer(
		"<br>", "\n",
		"<br/>", "\n",
		"<br />", "\n",
		"<BR>", "\n",
		"<BR/>", "\n",
		"<BR />", "\n",
	)
	return replacer.Replace(reply)
}

func compactLayaBlankLines(reply string) string {
	lines := strings.Split(reply, "\n")
	out := make([]string, 0, len(lines))
	blank := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blank++
			if blank <= 1 {
				out = append(out, "")
			}
			continue
		}
		blank = 0
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func stripLayaThinkingBlocks(reply string) string {
	for _, tag := range []string{"think", "thinking", "reasoning", "analysis"} {
		reply = stripLayaXMLBlock(reply, tag)
	}
	return strings.TrimSpace(reply)
}

func stripLayaHTMLComments(reply string) string {
	for {
		start := strings.Index(reply, "<!--")
		if start < 0 {
			return strings.TrimSpace(reply)
		}
		end := strings.Index(reply[start+4:], "-->")
		if end < 0 {
			return strings.TrimSpace(reply)
		}
		end += start + 4 + len("-->")
		reply = strings.TrimSpace(reply[:start] + "\n" + reply[end:])
	}
}

func stripLayaXMLBlock(reply, tag string) string {
	close := "</" + tag + ">"
	for {
		lower := strings.ToLower(reply)
		start := findLayaOpenXMLTag(lower, tag)
		if start < 0 {
			return strings.TrimSpace(reply)
		}
		openEnd := strings.IndexByte(lower[start:], '>')
		if openEnd < 0 {
			return strings.TrimSpace(reply)
		}
		contentStart := start + openEnd + 1
		closeStart := strings.Index(lower[contentStart:], close)
		if closeStart < 0 {
			return strings.TrimSpace(reply)
		}
		end := contentStart + closeStart + len(close)
		reply = strings.TrimSpace(reply[:start] + "\n" + reply[end:])
	}
}

func findLayaOpenXMLTag(lower, tag string) int {
	needle := "<" + tag
	offset := 0
	for {
		idx := strings.Index(lower[offset:], needle)
		if idx < 0 {
			return -1
		}
		idx += offset
		after := idx + len(needle)
		if after < len(lower) {
			switch lower[after] {
			case '>', ' ', '\t', '\n', '\r':
				return idx
			}
		}
		offset = after
	}
}

func stripLayaCodeFence(reply string) string {
	reply = strings.TrimSpace(reply)
	for _, fence := range []string{"```", "~~~"} {
		if !strings.HasPrefix(reply, fence) {
			continue
		}
		lines := strings.Split(reply, "\n")
		if len(lines) < 2 {
			return reply
		}
		lines = lines[1:]
		if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), fence) {
			lines = lines[:len(lines)-1]
		}
		return strings.TrimSpace(strings.Join(lines, "\n"))
	}
	return reply
}

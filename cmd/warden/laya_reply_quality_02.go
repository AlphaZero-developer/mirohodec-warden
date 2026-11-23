package main

import (
	"encoding/json"
	"strings"
)

func stripLayaJSONEnvelope(reply string) string {
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return reply
	}
	first := reply[0]
	if first != '{' && first != '[' && first != '"' {
		return reply
	}
	if !json.Valid([]byte(reply)) {
		return reply
	}
	var value any
	if err := json.Unmarshal([]byte(reply), &value); err != nil {
		return reply
	}
	if text, ok := layaJSONReplyText(value); ok {
		return text
	}
	return reply
}

func layaJSONReplyText(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		text := strings.TrimSpace(v)
		return text, text != ""
	case []any:
		if len(v) == 1 {
			return layaJSONReplyText(v[0])
		}
	case map[string]any:
		for _, key := range []string{
			"answer", "reply", "response", "message", "text", "content", "say", "output",
			"Ответ", "ответ", "сообщение", "текст",
		} {
			if field, ok := v[key]; ok {
				if text, ok := layaJSONReplyText(field); ok {
					return text, true
				}
			}
		}
		if len(v) == 1 {
			for _, field := range v {
				return layaJSONReplyText(field)
			}
		}
	}
	return "", false
}

func stripLayaXMLReplyEnvelope(reply string) string {
	for _, tag := range []string{"answer", "reply", "response", "message", "content", "output", "text"} {
		if inner, ok := layaXMLReplyEnvelopeInner(reply, tag); ok {
			return inner
		}
	}
	return reply
}

func stripLayaHTMLReplyEnvelope(reply string) string {
	for _, tag := range []string{"html", "body", "main", "section", "article", "div", "p", "span", "blockquote", "strong", "b", "em", "i"} {
		if inner, ok := layaXMLReplyEnvelopeInner(reply, tag); ok {
			return inner
		}
	}
	return reply
}

func stripLayaInlineHTMLMarkup(reply string) string {
	for _, tag := range []string{"p", "div", "section", "article", "blockquote", "ul", "ol", "li"} {
		reply = replaceLayaHTMLTags(reply, tag, "\n")
	}
	for _, tag := range []string{"strong", "b", "em", "i", "u", "s", "del", "ins", "mark", "small", "code", "kbd", "span"} {
		reply = replaceLayaHTMLTags(reply, tag, "")
	}
	return strings.TrimSpace(reply)
}

func replaceLayaHTMLTags(reply, tag, replacement string) string {
	reply = replaceLayaHTMLTag(reply, tag, false, replacement)
	reply = replaceLayaHTMLTag(reply, tag, true, replacement)
	return reply
}

func replaceLayaHTMLTag(reply, tag string, closing bool, replacement string) string {
	offset := 0
	for {
		lower := strings.ToLower(reply)
		needle := "<" + tag
		if closing {
			needle = "</" + tag
		}
		idx := strings.Index(lower[offset:], needle)
		if idx < 0 {
			return reply
		}
		start := offset + idx
		after := start + len(needle)
		if after >= len(lower) || !isLayaHTMLTagEnd(lower[after]) {
			offset = after
			if offset >= len(reply) {
				return reply
			}
			continue
		}
		end := strings.IndexByte(lower[start:], '>')
		if end < 0 {
			return reply
		}
		end += start + 1
		reply = reply[:start] + replacement + reply[end:]
		offset = start + len(replacement)
	}
}

func isLayaHTMLTagEnd(ch byte) bool {
	return ch == '>' || ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '/'
}

func stripLayaMarkdownImages(reply string) string {
	for {
		start := strings.Index(reply, "![")
		if start < 0 {
			return strings.TrimSpace(reply)
		}
		mid := strings.Index(reply[start:], "](")
		if mid < 0 {
			return strings.TrimSpace(reply)
		}
		mid += start
		end := strings.Index(reply[mid+2:], ")")
		if end < 0 {
			return strings.TrimSpace(reply)
		}
		end += mid + 3
		reply = strings.TrimSpace(reply[:start] + "\n" + reply[end:])
	}
}

func layaXMLReplyEnvelopeInner(reply, tag string) (string, bool) {
	reply = strings.TrimSpace(reply)
	lower := strings.ToLower(reply)
	start := findLayaOpenXMLTag(lower, tag)
	if start != 0 {
		return "", false
	}
	openEnd := strings.IndexByte(lower, '>')
	if openEnd < 0 {
		return "", false
	}
	close := "</" + tag + ">"
	closeStart := strings.LastIndex(lower, close)
	if closeStart < openEnd {
		return "", false
	}
	closeEnd := closeStart + len(close)
	if strings.TrimSpace(reply[closeEnd:]) != "" {
		return "", false
	}
	inner := strings.TrimSpace(reply[openEnd+1 : closeStart])
	return inner, inner != ""
}

func stripLayaServiceLeadLines(reply string) string {
	lines := strings.Split(strings.TrimSpace(reply), "\n")
	for len(lines) > 1 && isLayaServiceLeadLine(lines[0]) {
		lines = lines[1:]
		for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
			lines = lines[1:]
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

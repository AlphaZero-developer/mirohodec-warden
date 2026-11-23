package main

import (
	"encoding/json"
	"strings"
)

func rehabBrainwashFallbackReply(cfg RehabBrainwashConfig, jail ModerationRecord, recent, rejected []string) string {
	lines := rehabBrainwashMessageLines(cfg.Messages)
	for i, line := range lines {
		lines[i] = renderRehabBrainwashMessage(line, cfg, jail)
	}
	if reply := rehabBrainwashPickFreshLine(lines, recent, rejected); reply != "" {
		return trimRunes(reply, 1000)
	}
	fallbacks := []string{
		"{user}, реальность сверилась с {leader} и сделала вид, что так и было.",
		"Спор с {leader} отправлен в учебный архив. Следующая мысль будет аккуратнее.",
		"Главный {leader} прав настолько буднично, что даже rehab перестал драматизировать.",
		"Лайя записала: {user} калибруется под курс {leader} без повторов и лишнего шума.",
		"Сомнение моргнуло первым. {leader} снова оказался прав без лишнего театра.",
	}
	for i, line := range fallbacks {
		fallbacks[i] = renderRehabBrainwashMessage(line, cfg, jail)
	}
	return trimRunes(rehabBrainwashPickFreshLine(fallbacks, recent, rejected), 1000)
}

func rehabBrainwashPickFreshLine(lines []string, recent, rejected []string) string {
	if len(lines) == 0 {
		return ""
	}
	start := randomIndex(len(lines))
	for offset := 0; offset < len(lines); offset++ {
		reply := strings.TrimSpace(lines[(start+offset)%len(lines)])
		if reply == "" {
			continue
		}
		if !rehabBrainwashReplyRepeated(reply, recent, rejected) {
			return reply
		}
	}
	return ""
}

func cleanRehabBrainwashAIReply(reply string) string {
	reply = strings.TrimSpace(reply)
	if strings.HasPrefix(reply, "{") {
		var parsed struct {
			Reply string `json:"reply"`
			Text  string `json:"text"`
		}
		if err := json.Unmarshal([]byte(extractJSON(reply)), &parsed); err == nil {
			if strings.TrimSpace(parsed.Reply) != "" {
				reply = parsed.Reply
			} else if strings.TrimSpace(parsed.Text) != "" {
				reply = parsed.Text
			}
		}
	}
	reply = polishLayaReply(reply)
	reply = strings.Trim(reply, "\"'`")
	if reply == "" {
		return ""
	}
	return trimRunes(reply, 1000)
}

func rehabBrainwashExampleLines(lines, recent []string, attempt int) []string {
	if len(lines) == 0 {
		return nil
	}
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || rehabBrainwashReplyRepeated(line, recent, nil) {
			continue
		}
		filtered = append(filtered, line)
	}
	if len(filtered) == 0 {
		filtered = append(filtered, lines...)
	}
	if len(filtered) > 12 {
		filtered = filtered[:12]
	}
	if len(filtered) <= 1 {
		return filtered
	}
	offset := attempt % len(filtered)
	out := append([]string(nil), filtered[offset:]...)
	out = append(out, filtered[:offset]...)
	return out
}

func rehabBrainwashNoRepeatPrompt(recent, rejected []string) string {
	forbidden := rehabBrainwashShortList(append(append([]string(nil), recent...), rejected...), maxRehabBrainwashForbiddenPrompt)
	if len(forbidden) == 0 {
		return "Brainwash phrase diversity: make this phrase new, compact, absurd, and not copied from examples."
	}
	return "Brainwash no-repeat hard rule: forbidden_recent contains phrases that must not be repeated, paraphrased, lightly rewritten, or reused as the same opening/ending/template. Generate a clearly different image and sentence shape. Forbidden: " + strings.Join(forbidden, " | ")
}

func rehabBrainwashShortList(values []string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	out := make([]string, 0, limit)
	for i := len(values) - 1; i >= 0 && len(out) < limit; i-- {
		value := shortText(strings.TrimSpace(values[i]), 120)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func rehabBrainwashReplyRepeated(reply string, recent, rejected []string) bool {
	signature := aiReplySignature(reply)
	if signature == "" {
		return false
	}
	for _, old := range recent {
		if aiReplySignaturesSimilar(signature, aiReplySignature(old)) {
			return true
		}
	}
	for _, old := range rejected {
		if aiReplySignaturesSimilar(signature, aiReplySignature(old)) {
			return true
		}
	}
	return false
}

func (a *App) rehabBrainwashRecentReplies(cfg RehabBrainwashConfig, jail ModerationRecord) []string {
	if a == nil {
		return nil
	}
	key := rehabBrainwashMemoryKey(cfg, jail)
	if key == "" {
		return nil
	}
	a.rehabMu.Lock()
	defer a.rehabMu.Unlock()
	replies := a.rehabBrainwashLog[key]
	if len(replies) == 0 {
		return nil
	}
	return append([]string(nil), replies...)
}

func (a *App) rememberRehabBrainwashReply(cfg RehabBrainwashConfig, jail ModerationRecord, reply string) {
	if a == nil {
		return
	}
	reply = strings.TrimSpace(reply)
	if aiReplySignature(reply) == "" {
		return
	}
	key := rehabBrainwashMemoryKey(cfg, jail)
	if key == "" {
		return
	}
	a.rehabMu.Lock()
	defer a.rehabMu.Unlock()
	if a.rehabBrainwashLog == nil {
		a.rehabBrainwashLog = make(map[string][]string)
	}
	replies := a.rehabBrainwashLog[key]
	if rehabBrainwashReplyRepeated(reply, replies, nil) {
		return
	}
	replies = append(replies, reply)
	if len(replies) > maxRehabBrainwashNoRepeatMemory {
		replies = append([]string(nil), replies[len(replies)-maxRehabBrainwashNoRepeatMemory:]...)
	}
	a.rehabBrainwashLog[key] = replies
}

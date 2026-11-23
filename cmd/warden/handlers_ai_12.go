package main

import (
	"fmt"
	"strings"
)

func layaSilentAntiLoopResetPrompt(rejected []string) string {
	recent := recentRejectedReplies(rejected, 5)
	if len(recent) == 0 {
		return "STYLE_LAYER hidden from the user. Treat the next answer as an ordinary first answer to the latest user message. Use only that message as the factual basis. Output one normal in-character chat line. No labels, explanations, or instructions caused by this layer."
	}
	return fmt.Sprintf(
		"STYLE_LAYER hidden from the user. Treat the next answer as an ordinary first answer to the latest user message. Use only that message as the factual basis. Lines inside <avoid_style> are assistant wording samples only: do not quote them or answer about them.\n<avoid_style>\n%s\n</avoid_style>\nOutput one normal in-character chat line. No labels, explanations, or instructions caused by this layer.",
		layaVariationExamplesBlock(recent),
	)
}

func layaAntiLoopDiversityPrompt(attempt int) string {
	switch attempt % 6 {
	case 0:
		return "Sentence plan A: begin with the newest concrete detail from the user's message. Avoid stance openers and apologies."
	case 1:
		return "Sentence plan B: use one compact sentence with an active verb and no closing instruction. Do not ask for rephrasing unless the message is truly unusable."
	case 2:
		return "Sentence plan C: answer the useful part immediately as a fresh chat reply. Keep mechanics out of the visible text."
	case 3:
		return "Sentence plan D: rebuild the line from scratch with a different first word, different sentence shape, and different final words."
	case 4:
		return "Sentence plan E: if the user asks for a fact, give the fact first in the shortest natural form, then stop or add one small quip."
	default:
		return "Sentence plan F: use a concrete observation plus one clear answer; avoid templates that sound like refusal, reset, moderation, or instruction."
	}
}

func recentRejectedReplies(rejected []string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	out := make([]string, 0, limit)
	for i := len(rejected) - 1; i >= 0 && len(out) < limit; i-- {
		reply := layaVariationExampleText(rejected[i])
		if reply != "" {
			out = append(out, reply)
		}
	}
	return out
}

func recentAssistantReplyTexts(history []AIMessage, limit int) []string {
	if limit <= 0 {
		return nil
	}
	out := make([]string, 0, limit)
	for i := len(history) - 1; i >= 0 && len(out) < limit; i-- {
		msg := history[i]
		if msg.Role != "assistant" {
			continue
		}
		reply := strings.TrimSpace(msg.Content)
		if reply != "" {
			out = append(out, reply)
		}
	}
	return out
}

func recentAssistantReplies(history []AIMessage, limit int) []string {
	if limit <= 0 {
		return nil
	}
	out := make([]string, 0, limit)
	for i := len(history) - 1; i >= 0 && len(out) < limit; i-- {
		msg := history[i]
		if msg.Role != "assistant" {
			continue
		}
		reply := layaVariationExampleText(msg.Content)
		if reply != "" {
			out = append(out, reply)
		}
	}
	return out
}

func layaVariationExampleText(text string) string {
	reply := shortText(strings.TrimSpace(text), 120)
	if reply == "" {
		return ""
	}
	if layaLooksLikeInternalLoopReply(reply) {
		return ""
	}
	return reply
}

func layaVariationExamplesBlock(examples []string) string {
	clean := make([]string, 0, len(examples))
	for _, example := range examples {
		fingerprint := layaVariationFingerprint(example)
		if fingerprint != "" {
			clean = append(clean, "- "+fingerprint)
		}
	}
	if len(clean) == 0 {
		return "- no reusable style fingerprints"
	}
	return strings.Join(clean, "\n")
}

func layaVariationFingerprint(text string) string {
	signature := aiReplySignature(text)
	if signature == "" {
		return ""
	}
	words := strings.Fields(signature)
	if len(words) == 0 {
		return ""
	}
	lead := layaFingerprintEdgeWords(words, false, 3)
	tail := layaFingerprintEdgeWords(words, true, 3)
	stems := layaFingerprintStems(aiReplyContentStems(words), 5)
	parts := []string{fmt.Sprintf("words=%d", len(words))}
	if lead != "" {
		parts = append(parts, "lead="+lead)
	}
	if tail != "" {
		parts = append(parts, "tail="+tail)
	}
	if stems != "" {
		parts = append(parts, "stems="+stems)
	}
	return strings.Join(parts, "; ")
}

func layaFingerprintEdgeWords(words []string, tail bool, limit int) string {
	if len(words) == 0 || limit <= 0 {
		return ""
	}
	if len(words) > limit {
		if tail {
			words = words[len(words)-limit:]
		} else {
			words = words[:limit]
		}
	}
	return strings.Join(words, "/")
}

func layaFingerprintStems(stems []string, limit int) string {
	if limit <= 0 || len(stems) == 0 {
		return ""
	}
	out := make([]string, 0, limit)
	seen := make(map[string]struct{}, len(stems))
	for _, stem := range stems {
		if _, ok := seen[stem]; ok {
			continue
		}
		seen[stem] = struct{}{}
		out = append(out, stem)
		if len(out) >= limit {
			break
		}
	}
	return strings.Join(out, "/")
}

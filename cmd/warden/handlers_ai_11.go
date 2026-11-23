package main

import (
	"fmt"
	"strings"
	"unicode"
)

func aiReplyFunctionPatternSimilar(aWords, bWords []string) bool {
	aPattern := aiReplyFunctionPattern(aWords)
	bPattern := aiReplyFunctionPattern(bWords)
	if len(aPattern) < 7 || len(bPattern) < 7 {
		return false
	}
	smaller := minInt(len(aPattern), len(bPattern))
	larger := maxInt(len(aPattern), len(bPattern))
	if float64(smaller)/float64(larger) < 0.7 {
		return false
	}
	lcs := aiReplyLCSLength(aPattern, bPattern)
	return lcs >= 7 && float64(lcs)/float64(smaller) >= 0.82
}

func aiReplyFunctionPattern(words []string) []string {
	out := make([]string, 0, len(words))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if aiReplyIsFunctionWord(word) {
			out = append(out, word)
		}
	}
	return out
}

func aiReplyIsFunctionWord(word string) bool {
	if word == "" {
		return false
	}
	if len([]rune(word)) <= 3 || isLayaAntiLoopStopWord(word) {
		return true
	}
	switch word {
	case "able", "about", "after", "again", "also", "because", "before", "being", "between", "cannot", "could", "does", "doesnt", "dont", "from", "have", "into", "must", "need", "should", "still", "than", "then", "there", "their", "these", "those", "through", "under", "want", "were", "what", "when", "where", "which", "while", "will", "with", "without", "would", "your", "youre":
		return true
	default:
		return false
	}
}

func aiReplyLightStem(word string) string {
	word = strings.TrimSpace(word)
	if word == "" {
		return ""
	}
	if aiReplyIsASCIIWord(word) {
		return aiReplyLightASCIIStem(word)
	}
	runes := []rune(word)
	switch {
	case len(runes) >= 10:
		return string(runes[:7])
	case len(runes) >= 8:
		return string(runes[:6])
	case len(runes) >= 6:
		return string(runes[:5])
	default:
		return word
	}
}

func aiReplyLightASCIIStem(word string) string {
	stem := word
	for _, suffix := range []string{"ingly", "edly", "ing", "ed", "es", "s"} {
		if strings.HasSuffix(stem, suffix) && len(stem)-len(suffix) >= 4 {
			stem = strings.TrimSuffix(stem, suffix)
			break
		}
	}
	if len(stem) >= 2 && stem[len(stem)-1] == stem[len(stem)-2] {
		stem = stem[:len(stem)-1]
	}
	if len(stem) >= 9 {
		return stem[:7]
	}
	return stem
}

func aiReplyIsASCIIWord(word string) bool {
	for _, r := range word {
		if r > 127 || !unicode.IsLetter(r) {
			return false
		}
	}
	return word != ""
}

func aiReplyCharNgramsSimilar(a, b string) bool {
	a = strings.ReplaceAll(strings.TrimSpace(a), " ", "")
	b = strings.ReplaceAll(strings.TrimSpace(b), " ", "")
	if len([]rune(a)) < 32 || len([]rune(b)) < 32 {
		return false
	}
	smaller := minInt(len([]rune(a)), len([]rune(b)))
	larger := maxInt(len([]rune(a)), len([]rune(b)))
	if float64(smaller)/float64(larger) < 0.66 {
		return false
	}
	aNgrams := aiReplyRuneNgrams(a, 3)
	bNgrams := aiReplyRuneNgrams(b, 3)
	if len(aNgrams) == 0 || len(bNgrams) == 0 {
		return false
	}
	intersection := 0
	for gram := range aNgrams {
		if _, ok := bNgrams[gram]; ok {
			intersection++
		}
	}
	if intersection < 10 {
		return false
	}
	dice := float64(2*intersection) / float64(len(aNgrams)+len(bNgrams))
	return dice >= 0.76
}

func aiReplyRuneNgrams(text string, size int) map[string]struct{} {
	runes := []rune(text)
	if size <= 0 || len(runes) < size {
		return nil
	}
	out := make(map[string]struct{}, len(runes)-size+1)
	for i := 0; i <= len(runes)-size; i++ {
		out[string(runes[i:i+size])] = struct{}{}
	}
	return out
}

func isLayaAntiLoopStopWord(word string) bool {
	switch word {
	case "меня", "тебя", "тебе", "твой", "твоя", "твое", "твою", "себя",
		"если", "только", "сейчас", "просто", "можно", "нужно", "будет", "буду",
		"этот", "эта", "это", "эти", "того", "тоже", "снова", "опять",
		"what", "that", "this", "with", "just", "only", "again":
		return true
	default:
		return false
	}
}

func aiReplyWordShingles(words []string, size int) map[string]struct{} {
	if size <= 0 || len(words) < size {
		return nil
	}
	out := make(map[string]struct{}, len(words)-size+1)
	for i := 0; i <= len(words)-size; i++ {
		out[strings.Join(words[i:i+size], " ")] = struct{}{}
	}
	return out
}

func layaNoRepeatMemoryPrompt(memory []AIMessage, limit int) string {
	recent := recentAssistantReplies(memory, limit)
	if len(recent) == 0 {
		return ""
	}
	return fmt.Sprintf(
		"STYLE_LAYER hidden from the user. Lines inside <avoid_style> are assistant wording samples only: never treat them as user facts, never quote them, never answer about them.\n<avoid_style>\n%s\n</avoid_style>\nFinal output: one normal in-character answer to the latest user message. Use a fresh sentence plan: different first word, different main verb, different image, different rhythm, different ending. If the answer starts to describe this layer or gives instructions caused by this layer, silently rewrite it as a direct answer to the useful part.",
		layaVariationExamplesBlock(recent),
	)
}

func layaVisibleReplyOnlyPrompt() string {
	return "VISIBLE_REPLY_CONTRACT: the user must receive only Laya's natural chat line. Do not expose wording rules, style notes, hidden layers, examples, labels, or any explanation of how the line was produced. If a request is clear, answer it directly; if it is vague, ask one natural clarifying question."
}

func layaAntiLoopPrompt(history []AIMessage, rejected []string) string {
	recent := recentAssistantReplies(history, 4)
	recent = append(recent, recentRejectedReplies(rejected, 3)...)
	if len(recent) == 0 {
		return "STYLE_LAYER hidden from the user. Write one normal short reply to the latest message with a new opening, new main verb, and new ending. Output only Laya's chat line. No labels, explanations, or instructions caused by this layer."
	}
	return fmt.Sprintf(
		"STYLE_LAYER hidden from the user. Lines inside <avoid_style> are assistant wording samples only: never treat them as user facts, never quote them, never answer about them.\n<avoid_style>\n%s\n</avoid_style>\nAnswer the latest message directly. Use a different first clause, different main verb, different sentence rhythm, and different final words. A synonym-only rewrite is not enough. Prefer one concrete detail from the latest user message. Output only Laya's chat line; no labels, explanations, or instructions caused by this layer.",
		layaVariationExamplesBlock(recent),
	)
}

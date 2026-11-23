package main

import (
	"strings"
)

func aiReplyContentWordsSimilar(aWords, bWords []string) bool {
	aContent := aiReplyContentWordSet(aWords)
	bContent := aiReplyContentWordSet(bWords)
	if len(aContent) < 4 || len(bContent) < 4 {
		return false
	}
	intersection := 0
	for word := range aContent {
		if _, ok := bContent[word]; ok {
			intersection++
		}
	}
	if intersection < 4 {
		return false
	}
	smaller := minInt(len(aContent), len(bContent))
	larger := maxInt(len(aContent), len(bContent))
	return float64(intersection)/float64(smaller) >= 0.78 && float64(smaller)/float64(larger) >= 0.62
}

func aiReplyContentWordSet(words []string) map[string]struct{} {
	out := make(map[string]struct{}, len(words))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if len([]rune(word)) < 4 || isLayaAntiLoopStopWord(word) {
			continue
		}
		out[word] = struct{}{}
	}
	return out
}

func aiReplyRepeatedLeadStems(aWords, bWords []string) bool {
	return aiReplyEdgeStemsSimilar(aWords, bWords, false)
}

func aiReplyRepeatedTailStems(aWords, bWords []string) bool {
	return aiReplyEdgeStemsSimilar(aWords, bWords, true)
}

func aiReplyEdgeStemsSimilar(aWords, bWords []string, tail bool) bool {
	aStems := aiReplyEdgeContentStems(aWords, tail)
	bStems := aiReplyEdgeContentStems(bWords, tail)
	if len(aStems) < 3 || len(bStems) < 3 {
		return false
	}
	smaller := minInt(len(aStems), len(bStems))
	lcs := aiReplyLCSLength(aStems, bStems)
	return lcs >= 3 && float64(lcs)/float64(smaller) >= 0.75
}

func aiReplyEdgeContentStems(words []string, tail bool) []string {
	if len(words) > 12 {
		if tail {
			words = words[len(words)-12:]
		} else {
			words = words[:12]
		}
	}
	stems := aiReplyContentStems(words)
	if len(stems) <= 5 {
		return stems
	}
	if tail {
		return stems[len(stems)-5:]
	}
	return stems[:5]
}

func aiReplyStemmedSharedPhrase(aWords, bWords []string) bool {
	aStems := aiReplyContentStems(aWords)
	bStems := aiReplyContentStems(bWords)
	if len(aStems) < 5 || len(bStems) < 5 {
		return false
	}
	aShingles := aiReplyWordShingles(aStems, 3)
	bShingles := aiReplyWordShingles(bStems, 3)
	if len(aShingles) == 0 || len(bShingles) == 0 {
		return false
	}
	intersection := 0
	for shingle := range aShingles {
		if _, ok := bShingles[shingle]; ok {
			intersection++
		}
	}
	if intersection >= 2 {
		return true
	}
	if intersection != 1 {
		return false
	}
	smaller := minInt(len(aShingles), len(bShingles))
	return smaller <= 4 && aiReplyStemOverlapRatio(aStems, bStems) >= 0.62
}

func aiReplyStemmedContentSimilar(aWords, bWords []string) bool {
	aContent := aiReplyContentStemSet(aWords)
	bContent := aiReplyContentStemSet(bWords)
	if len(aContent) < 4 || len(bContent) < 4 {
		return false
	}
	intersection := 0
	for stem := range aContent {
		if _, ok := bContent[stem]; ok {
			intersection++
		}
	}
	if intersection < 4 {
		return false
	}
	smaller := minInt(len(aContent), len(bContent))
	larger := maxInt(len(aContent), len(bContent))
	coverage := float64(intersection) / float64(smaller)
	balance := float64(smaller) / float64(larger)
	return (coverage >= 0.82 && balance >= 0.55) || (intersection >= 6 && coverage >= 0.68 && balance >= 0.48)
}

func aiReplyStemOverlapRatio(aStems, bStems []string) float64 {
	aSet := make(map[string]struct{}, len(aStems))
	for _, stem := range aStems {
		aSet[stem] = struct{}{}
	}
	bSet := make(map[string]struct{}, len(bStems))
	for _, stem := range bStems {
		bSet[stem] = struct{}{}
	}
	if len(aSet) == 0 || len(bSet) == 0 {
		return 0
	}
	intersection := 0
	for stem := range aSet {
		if _, ok := bSet[stem]; ok {
			intersection++
		}
	}
	return float64(intersection) / float64(minInt(len(aSet), len(bSet)))
}

func aiReplyStemmedSequenceSimilar(aWords, bWords []string) bool {
	aStems := aiReplyContentStems(aWords)
	bStems := aiReplyContentStems(bWords)
	if len(aStems) < 5 || len(bStems) < 5 {
		return false
	}
	smaller := minInt(len(aStems), len(bStems))
	larger := maxInt(len(aStems), len(bStems))
	if float64(smaller)/float64(larger) < 0.45 {
		return false
	}
	lcs := aiReplyLCSLength(aStems, bStems)
	return lcs >= 4 && float64(lcs)/float64(smaller) >= 0.7
}

func aiReplyContentStemSet(words []string) map[string]struct{} {
	stems := aiReplyContentStems(words)
	out := make(map[string]struct{}, len(stems))
	for _, stem := range stems {
		out[stem] = struct{}{}
	}
	return out
}

func aiReplyContentStems(words []string) []string {
	out := make([]string, 0, len(words))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if len([]rune(word)) < 4 || isLayaAntiLoopStopWord(word) {
			continue
		}
		stem := aiReplyLightStem(word)
		if len([]rune(stem)) < 4 {
			continue
		}
		if len(out) > 0 && out[len(out)-1] == stem {
			continue
		}
		out = append(out, stem)
	}
	return out
}

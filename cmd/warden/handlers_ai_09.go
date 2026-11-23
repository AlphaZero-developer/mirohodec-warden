package main

import (
	"strings"
)

func aiReplySignature(value string) string {
	value = normalizeFAQText(value)
	if value == "" {
		return ""
	}
	return strings.Join(strings.Fields(value), " ")
}

func aiReplySignatureSeen(seen []string, signature string) bool {
	signature = strings.TrimSpace(signature)
	if signature == "" {
		return false
	}
	for _, old := range seen {
		if old == signature || aiReplySignaturesSimilar(old, signature) {
			return true
		}
	}
	return false
}

func aiReplySignaturesSimilar(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	if !aiReplySignatureHasLoopSignal(a) || !aiReplySignatureHasLoopSignal(b) {
		return false
	}
	aWords := strings.Fields(a)
	bWords := strings.Fields(b)
	if a == b {
		return true
	}
	if aiReplyRepeatedOpening(aWords, bWords) ||
		aiReplyRepeatedEnding(aWords, bWords) ||
		aiReplySharedPhrase(aWords, bWords) ||
		aiReplyOrderedOverlap(aWords, bWords) ||
		aiReplyContentWordsSimilar(aWords, bWords) ||
		aiReplyRepeatedLeadStems(aWords, bWords) ||
		aiReplyRepeatedTailStems(aWords, bWords) ||
		aiReplyStemmedSharedPhrase(aWords, bWords) ||
		aiReplyStemmedContentSimilar(aWords, bWords) ||
		aiReplyStemmedSequenceSimilar(aWords, bWords) ||
		aiReplyFunctionPatternSimilar(aWords, bWords) ||
		aiReplyCharNgramsSimilar(a, b) {
		return true
	}
	if len(aWords) < 5 || len(bWords) < 5 {
		return false
	}
	aSet := make(map[string]struct{}, len(aWords))
	for _, word := range aWords {
		aSet[word] = struct{}{}
	}
	bSet := make(map[string]struct{}, len(bWords))
	for _, word := range bWords {
		bSet[word] = struct{}{}
	}
	intersection := 0
	for word := range aSet {
		if _, ok := bSet[word]; ok {
			intersection++
		}
	}
	union := len(aSet) + len(bSet) - intersection
	if union == 0 {
		return false
	}
	jaccard := float64(intersection) / float64(union)
	coverageA := float64(intersection) / float64(len(aSet))
	coverageB := float64(intersection) / float64(len(bSet))
	return jaccard >= 0.82 || (coverageA >= 0.9 && coverageB >= 0.9)
}

func aiReplySignatureHasLoopSignal(signature string) bool {
	words := strings.Fields(strings.TrimSpace(signature))
	if len(words) < 5 {
		return false
	}
	compact := strings.Join(words, "")
	return len([]rune(compact)) >= 24
}

func aiReplyRepeatedOpening(aWords, bWords []string) bool {
	if len(aWords) < 7 || len(bWords) < 7 {
		return false
	}
	limit := minInt(minInt(len(aWords), len(bWords)), 7)
	matches := 0
	for i := 0; i < limit; i++ {
		if aWords[i] != bWords[i] {
			break
		}
		matches++
	}
	return matches >= 5
}

func aiReplyRepeatedEnding(aWords, bWords []string) bool {
	if len(aWords) < 7 || len(bWords) < 7 {
		return false
	}
	limit := minInt(minInt(len(aWords), len(bWords)), 7)
	matches := 0
	for i := 1; i <= limit; i++ {
		if aWords[len(aWords)-i] != bWords[len(bWords)-i] {
			break
		}
		matches++
	}
	return matches >= 5
}

func aiReplySharedPhrase(aWords, bWords []string) bool {
	if len(aWords) < 8 || len(bWords) < 8 {
		return false
	}
	aShingles := aiReplyWordShingles(aWords, 4)
	bShingles := aiReplyWordShingles(bWords, 4)
	if len(aShingles) == 0 || len(bShingles) == 0 {
		return false
	}
	intersection := 0
	for shingle := range aShingles {
		if _, ok := bShingles[shingle]; ok {
			intersection++
		}
	}
	if intersection < 2 {
		return false
	}
	smaller := minInt(len(aShingles), len(bShingles))
	return float64(intersection)/float64(smaller) >= 0.45
}

func aiReplyOrderedOverlap(aWords, bWords []string) bool {
	if len(aWords) < 7 || len(bWords) < 7 {
		return false
	}
	smaller := minInt(len(aWords), len(bWords))
	larger := maxInt(len(aWords), len(bWords))
	if float64(smaller)/float64(larger) < 0.58 {
		return false
	}
	lcs := aiReplyLCSLength(aWords, bWords)
	return lcs >= 6 && float64(lcs)/float64(smaller) >= 0.72
}

func aiReplyLCSLength(aWords, bWords []string) int {
	if len(aWords) == 0 || len(bWords) == 0 {
		return 0
	}
	prev := make([]int, len(bWords)+1)
	for i := 1; i <= len(aWords); i++ {
		curr := make([]int, len(bWords)+1)
		for j := 1; j <= len(bWords); j++ {
			if aWords[i-1] == bWords[j-1] {
				curr[j] = prev[j-1] + 1
				continue
			}
			if prev[j] >= curr[j-1] {
				curr[j] = prev[j]
			} else {
				curr[j] = curr[j-1]
			}
		}
		prev = curr
	}
	return prev[len(bWords)]
}

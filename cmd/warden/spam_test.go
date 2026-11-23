package main

import (
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func TestSpamLinkSignaturesDeduplicateEntitiesAndText(t *testing.T) {
	msg := &tele.Message{
		Text: "https://example.com/path https://example.com/other",
		Entities: tele.Entities{
			{Type: tele.EntityTextLink, URL: "https://example.com/path"},
		},
	}
	links := spamLinkSignatures(msg)
	if len(links) != 1 || links[0] != "example.com" {
		t.Fatalf("unexpected links: %#v", links)
	}
}

func TestSpamMentionSignaturesDeduplicate(t *testing.T) {
	msg := &tele.Message{
		Text: "@SomeUser @someuser",
		Entities: tele.Entities{
			{Type: tele.EntityTMention, User: &tele.User{ID: 42}},
			{Type: tele.EntityTMention, User: &tele.User{ID: 42}},
		},
	}
	mentions := spamMentionSignatures(msg)
	if len(mentions) != 2 {
		t.Fatalf("expected user mention and username mention, got %#v", mentions)
	}
}

func TestNormalizeSpamTextCollapsesNoise(t *testing.T) {
	a := normalizeSpamText("ПРИВЕТ!!!\u200b  сезон?")
	b := normalizeSpamText("привет сезон")
	if a != b {
		t.Fatalf("expected equal normalized text, got %q vs %q", a, b)
	}
}

func TestActiveUserSpamMultiplierOnlyChangesRateLimits(t *testing.T) {
	if got := applySpamRateMultiplier(12, 2); got != 24 {
		t.Fatalf("unexpected multiplied limit: %d", got)
	}
	if got := applySpamRateMultiplier(0, 2); got != 0 {
		t.Fatalf("zero limit should stay zero, got %d", got)
	}
}

func TestSpamStrikeConfidenceCounting(t *testing.T) {
	strikes := []spamStrike{
		{At: time.Now(), Confidence: spamConfidenceLow},
		{At: time.Now(), Confidence: spamConfidenceHigh},
		{At: time.Now(), Confidence: spamConfidenceHigh},
	}
	if got := countSpamStrikes(strikes, spamConfidenceHigh); got != 2 {
		t.Fatalf("expected 2 high strikes, got %d", got)
	}
}

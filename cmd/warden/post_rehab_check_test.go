package main

import (
	"strings"
	"testing"
	"time"
)

func TestRandomPostRehabCheckDelayBounds(t *testing.T) {
	cfg := RehabConfig{
		PostReleaseCheckMinDelay: "3m",
		PostReleaseCheckMaxDelay: "5h",
	}
	for i := 0; i < 100; i++ {
		got := randomPostRehabCheckDelay(cfg)
		if got < 3*time.Minute || got > 5*time.Hour {
			t.Fatalf("delay %s outside configured bounds", got)
		}
	}
}

func TestObfuscatePostRehabQuestionHidesOriginalText(t *testing.T) {
	question := "Как самочувствие после лечения?"
	got := obfuscatePostRehabQuestion(question)
	if strings.TrimSpace(got) == "" {
		t.Fatal("obfuscated text is empty")
	}
	if strings.Contains(got, "самочувствие") || strings.Contains(got, "лечение") {
		t.Fatalf("obfuscated text leaked source words: %q", got)
	}
	if len([]rune(got)) <= len([]rune(question)) {
		t.Fatalf("obfuscated text should include noise, got %q", got)
	}
}

func TestFallbackPostRehabAnswerLiked(t *testing.T) {
	tests := []struct {
		name   string
		answer string
		want   bool
	}{
		{name: "positive", answer: "нормально, спасибо", want: true},
		{name: "bad treatment", answer: "не понравилось, отстань", want: false},
		{name: "empty", answer: "", want: false},
		{name: "insult", answer: "шлюхабот", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fallbackPostRehabAnswerLiked(tt.answer); got != tt.want {
				t.Fatalf("fallbackPostRehabAnswerLiked() = %t, want %t", got, tt.want)
			}
		})
	}
}

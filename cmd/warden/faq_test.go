package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFAQStoreCreatesDefaultAndMatchesSeason(t *testing.T) {
	path := filepath.Join(t.TempDir(), "faq.json")
	store, err := NewFAQStore(path, nil)
	if err != nil {
		t.Fatalf("NewFAQStore: %v", err)
	}

	rec, ok := store.Match("Лайя, когда выйдет сезон?")
	if !ok {
		t.Fatal("expected FAQ match")
	}
	if rec.ID != "season_release" {
		t.Fatalf("unexpected record id: %s", rec.ID)
	}
	if rec.Answer != "Примерная дата выхода: 1 января 2027." {
		t.Fatalf("unexpected answer: %q", rec.Answer)
	}
}

func TestFAQStoreLoadsLegacyStringMap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "faq.json")
	raw := []byte(`{"season":"Примерная дата выхода: 1 января 2027."}`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	store, err := NewFAQStore(path, nil)
	if err != nil {
		t.Fatalf("NewFAQStore: %v", err)
	}
	rec, ok := store.Match("season")
	if !ok {
		t.Fatal("expected legacy FAQ match")
	}
	if rec.Answer != "Примерная дата выхода: 1 января 2027." {
		t.Fatalf("unexpected answer: %q", rec.Answer)
	}
}

func TestFAQInputFromArgs(t *testing.T) {
	patterns, answer := faqInputFromArgs([]string{"когда", "сезон;", "дата", "сезона", "|", "1", "января"})
	if len(patterns) != 2 {
		t.Fatalf("expected 2 patterns, got %d: %#v", len(patterns), patterns)
	}
	if answer != "1 января" {
		t.Fatalf("unexpected answer: %q", answer)
	}
}

func TestMentionsLaya(t *testing.T) {
	for _, text := range []string{"Лайя когда сезон?", "laya when season?", "Лайетана Каталонес, ты тут?", "эй @layetana_bot"} {
		if !mentionsLaya(text, "layetana_bot") {
			t.Fatalf("expected mention in %q", text)
		}
	}
	if mentionsLaya("когда сезон?", "layetana_bot") {
		t.Fatal("did not expect mention")
	}
}

func TestFAQMatchBaldherKeywords(t *testing.T) {
	data, err := parseFAQData([]byte(`{
		"faqs": {
			"baldher": {
				"id": "baldher",
				"patterns": [
					"Почему ты убила Вальдхара?",
					"Зачем ты убила Балдхэра?",
					"убила Балдхэра",
					"Балдхэр"
				],
				"answer": "Тебя это волнует? Убила и убила",
				"enabled": true
			}
		}
	}`))
	if err != nil {
		t.Fatalf("parseFAQData: %v", err)
	}
	store := &FAQStore{data: data}
	rec, ok := store.Match("Зачем ты убила Балдхэра?")
	if !ok {
		t.Fatal("expected Baldher FAQ match")
	}
	if rec.ID != "baldher" {
		t.Fatalf("unexpected FAQ id: %s", rec.ID)
	}
}

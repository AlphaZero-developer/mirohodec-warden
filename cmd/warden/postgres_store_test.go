package main

import (
	"os"
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func TestPostgresStoreCRUDWhenDSNProvided(t *testing.T) {
	dsn := os.Getenv("WARDEN_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("WARDEN_TEST_POSTGRES_DSN is not set")
	}
	cfg := Config{
		Storage: StorageConfig{
			Driver:      "postgres",
			AutoMigrate: true,
			Postgres:    PostgresConfig{DSN: dsn},
		},
	}
	store, err := NewPostgresStore(cfg, nil)
	if err != nil {
		t.Fatalf("NewPostgresStore: %v", err)
	}
	chatID := time.Now().UnixNano()
	user := &tele.User{ID: chatID + 1, FirstName: "PG"}
	store.TrackMember(chatID, user, 7, 11, true)
	if rec, ok := store.Member(chatID, user.ID); !ok || rec.Messages != 1 {
		t.Fatalf("unexpected member: %#v ok=%v", rec, ok)
	}
	if _, err := store.AddWarning(chatID, WarningRecord{UserID: user.ID, DisplayName: "PG", ExpiresUnix: time.Now().Add(time.Hour).Unix()}); err != nil {
		t.Fatalf("AddWarning: %v", err)
	}
	if count := store.ActiveWarningCount(chatID, user.ID, time.Now().UTC()); count != 1 {
		t.Fatalf("unexpected warnings: %d", count)
	}
	if err := store.UpsertChatSettings(ChatSettings{ChatID: chatID, Title: "pg chat", Enabled: true, RehabTopicID: 9, SpamNoRehabAction: spamNoRehabMuteOnDelete}); err != nil {
		t.Fatalf("UpsertChatSettings: %v", err)
	}
	if settings, ok := store.ChatSettings(chatID); !ok || settings.RehabTopicID != 9 || !settings.Enabled {
		t.Fatalf("unexpected chat settings: %#v ok=%v", settings, ok)
	}
	score, err := store.AddKarma(chatID, user.ID, 2, "manual", user, 11, time.Now().UTC())
	if err != nil || score != 2 {
		t.Fatalf("AddKarma score=%d err=%v", score, err)
	}

	quotes := &PostgresQuoteStore{base: store}
	q, err := quotes.Add("quote", "author", []string{"test"}, user)
	if err != nil {
		t.Fatalf("quote add: %v", err)
	}
	if _, ok := quotes.Random("test"); !ok {
		t.Fatal("expected random quote")
	}
	if _, ok, err := quotes.Delete(q.ID); err != nil || !ok {
		t.Fatalf("quote delete ok=%v err=%v", ok, err)
	}

	faq := &PostgresFAQStore{base: store}
	f, err := faq.Add([]string{"question"}, "answer")
	if err != nil {
		t.Fatalf("faq add: %v", err)
	}
	if _, ok := faq.Match("question"); !ok {
		t.Fatal("expected faq match")
	}
	if _, ok, err := faq.Delete(f.ID); err != nil || !ok {
		t.Fatalf("faq delete ok=%v err=%v", ok, err)
	}
}

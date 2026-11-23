package main

import (
	"path/filepath"
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func TestActivityScoreUsesTodayKarmaAndWarnings(t *testing.T) {
	now := time.Now().UTC()
	rec := MemberRecord{
		ID:            42,
		Messages:      100,
		MessagesToday: 5,
		ActivityDay:   now.Format("2006-01-02"),
	}

	got := activityScore(rec, 3, 1, now)
	want := int64(116) // 100 total + 5*4 today + 3*7 positive karma - 25 warning.
	if got != want {
		t.Fatalf("activityScore = %d, want %d", got, want)
	}

	if got := activityScore(rec, -50, 10, now); got != 0 {
		t.Fatalf("activityScore should not go negative, got %d", got)
	}
}

func TestTopActivityRowsSortBySocialScore(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	app := &App{store: store}
	chatID := int64(-1001)
	now := time.Now().UTC()

	store.TrackMember(chatID, &tele.User{ID: 1, FirstName: "Total"}, 0, 1, true)
	store.TrackMember(chatID, &tele.User{ID: 2, FirstName: "Karma"}, 0, 2, true)
	if _, err := store.AddKarma(chatID, 2, 50, "manual", &tele.User{ID: 9, FirstName: "giver"}, 3, now); err != nil {
		t.Fatalf("AddKarma: %v", err)
	}

	rows := app.topActivityRows(chatID, 2, now)
	if len(rows) != 2 {
		t.Fatalf("rows len = %d, want 2", len(rows))
	}
	if rows[0].Member.ID != 2 || rows[0].Rank != 1 {
		t.Fatalf("karma-rich user should lead activity rows: %#v", rows)
	}
}

func TestKarmaAutoProgressLine(t *testing.T) {
	cfg := KarmaConfig{Enabled: true, ActivityMessagesPerPoint: 10, ActivityDailyLimit: 5}
	if got := karmaAutoProgressLine(24, cfg); got != "🤖 Авто-карма сегодня: 2/5, до следующего +1: 6 сообщ." {
		t.Fatalf("unexpected progress line: %q", got)
	}
	if got := karmaAutoProgressLine(80, cfg); got != "🤖 Авто-карма сегодня: 5/5, дневной лимит взят." {
		t.Fatalf("unexpected capped progress line: %q", got)
	}
}

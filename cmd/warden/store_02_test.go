package main

import (
	"path/filepath"
	"testing"

	tele "gopkg.in/telebot.v3"
)

func TestRehabEscapeChancePersistsOnMember(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	chatID := int64(-1001)
	userID := int64(42)
	store.TrackMember(chatID, &tele.User{ID: userID, FirstName: "user"}, 0, 0, false)

	if got := store.RehabEscapeChance(chatID, userID); got != 5 {
		t.Fatalf("initial chance = %d, want 5", got)
	}
	store.RecordRehabEscapeSuccess(chatID, userID)
	if got := store.RehabEscapeChance(chatID, userID); got != 4 {
		t.Fatalf("chance after success = %d, want 4", got)
	}
	member, ok := store.Member(chatID, userID)
	if !ok || member.RehabEscapePenalty != 1 {
		t.Fatalf("escape penalty not saved: %#v ok=%v", member, ok)
	}
}

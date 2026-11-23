package main

import (
	"path/filepath"
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func TestParseLayaForwardRestrictionIntent(t *testing.T) {
	tests := []struct {
		name string
		text string
		want layaForwardRestrictionIntent
		ok   bool
	}{
		{
			name: "block explicit target",
			text: "Лайя, этому запрещено пересылать сообщения",
			want: layaForwardRestrictionBlock,
			ok:   true,
		},
		{
			name: "block imperative",
			text: "Лайя, запрети пересылку",
			want: layaForwardRestrictionBlock,
			ok:   true,
		},
		{
			name: "block typo",
			text: "Лайя, этому запрешено пересылать сообщения",
			want: layaForwardRestrictionBlock,
			ok:   true,
		},
		{
			name: "allow explicit target",
			text: "Лайя, этому можно пересылать сообщения",
			want: layaForwardRestrictionAllow,
			ok:   true,
		},
		{
			name: "allow remove ban",
			text: "Лайя, сними запрет на пересылку",
			want: layaForwardRestrictionAllow,
			ok:   true,
		},
		{
			name: "ordinary question",
			text: "Лайя, можно ли пересылать сообщения?",
			want: layaForwardRestrictionNone,
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseLayaForwardRestrictionIntent(tt.text, "")
			if got != tt.want || ok != tt.ok {
				t.Fatalf("parseLayaForwardRestrictionIntent() = %v, %t; want %v, %t", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestForwardRestrictionForMessageMatchesSenderAndOriginalUser(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	app := &App{store: store}
	chatID := int64(-1001)
	targetID := int64(42)
	now := time.Now().UTC()
	store.SetForwardRestriction(chatID, ModerationRecord{
		UserID:      targetID,
		DisplayName: "Target",
		SinceUnix:   now.Unix(),
	})

	if _, _, ok := app.forwardRestrictionForMessage(chatID, &tele.User{ID: targetID}, &tele.Message{Via: &tele.User{ID: 777}}, now); !ok {
		t.Fatal("expected via-bot message from restricted sender to match")
	}
	if _, matched, ok := app.forwardRestrictionForMessage(chatID, &tele.User{ID: 99}, &tele.Message{Origin: &tele.MessageOrigin{Sender: &tele.User{ID: targetID}}}, now); !ok || matched != targetID {
		t.Fatalf("expected forward from restricted original user to match, got ok=%t matched=%d", ok, matched)
	}
	if isForwardRestrictionPayload(&tele.Message{Text: "regular"}) {
		t.Fatal("regular message should not be a forward restriction payload")
	}
}

func TestStoreForwardRestrictionLifecycle(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	chatID := int64(-1001)
	userID := int64(42)
	now := time.Now().UTC()

	if _, ok := store.ActiveForwardRestriction(chatID, userID, now); ok {
		t.Fatal("did not expect active restriction initially")
	}
	store.SetForwardRestriction(chatID, ModerationRecord{UserID: userID, DisplayName: "Target", SinceUnix: now.Unix()})
	if _, ok := store.ActiveForwardRestriction(chatID, userID, now); !ok {
		t.Fatal("expected active restriction")
	}
	if _, ok := store.RemoveForwardRestriction(chatID, userID); !ok {
		t.Fatal("expected restriction removal")
	}
	if _, ok := store.ActiveForwardRestriction(chatID, userID, now); ok {
		t.Fatal("restriction should be removed")
	}
}

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func TestRandomActiveMemberExcludesBotsAndInactiveUsers(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	chatID := int64(1)
	store.TrackMember(chatID, &tele.User{ID: 1, FirstName: "human"}, 10, 100, true)
	store.TrackMember(chatID, &tele.User{ID: 2, FirstName: "bot", IsBot: true}, 10, 101, true)

	rec, ok := store.RandomActiveMember(chatID, 24*time.Hour, time.Now().UTC(), 0)
	if !ok {
		t.Fatal("expected active member")
	}
	if rec.ID != 1 || rec.LastMessageID != 100 || rec.LastThreadID != 10 {
		t.Fatalf("unexpected member: %#v", rec)
	}
}

func TestUserHasActiveRestriction(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	app := &App{store: store}
	chatID := int64(1)
	userID := int64(2)
	now := time.Now().UTC()

	if app.userHasActiveRestriction(chatID, userID, now) {
		t.Fatal("did not expect restriction initially")
	}
	store.SetJail(chatID, ModerationRecord{UserID: userID, DisplayName: "user", UntilUnix: now.Add(time.Minute).Unix()})
	if !app.userHasActiveRestriction(chatID, userID, now) {
		t.Fatal("expected active jail restriction")
	}
	store.RemoveJail(chatID, userID)
	store.SetMute(chatID, ModerationRecord{UserID: userID, DisplayName: "user", UntilUnix: now.Add(time.Minute).Unix()})
	if !app.userHasActiveRestriction(chatID, userID, now) {
		t.Fatal("expected active mute restriction")
	}
}

func TestChatSettingsKeepPersistedRehabTopic(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	chatID := int64(-1001)
	if err := store.UpsertChatSettings(ChatSettings{ChatID: chatID, Enabled: true, RehabTopicID: 7229}); err != nil {
		t.Fatalf("set topic: %v", err)
	}
	if err := store.UpsertChatSettings(ChatSettings{ChatID: chatID, Enabled: true, RehabTopicID: 0}); err != nil {
		t.Fatalf("zero overwrite: %v", err)
	}
	settings, ok := store.ChatSettings(chatID)
	if !ok {
		t.Fatal("settings missing")
	}
	if settings.RehabTopicID != 7229 || settings.PersistedRehabTopicID != 7229 {
		t.Fatalf("topic was not kept: %#v", settings)
	}
}

func TestStoreMigratesRehabStartedAtAndPersistedTopic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	raw := `{
  "chats": {
    "-1001": {
      "members": {},
      "jailed": {
        "42": {
          "user_id": 42,
          "display_name": "user",
          "since_unix": 1700000000,
          "until_unix": 4700000000
        }
      }
    }
  },
  "managed_chats": {
    "-1001": {
      "chat_id": -1001,
      "enabled": true,
      "rehab_topic_id": 0,
      "persisted_rehab_topic_id": 7229
    }
  }
}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("write state: %v", err)
	}
	store, err := NewStore(path, nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	rec, ok := store.ActiveJail(-1001, 42, time.Now().UTC())
	if !ok {
		t.Fatal("expected active jail")
	}
	if rec.StartedAtUnix != rec.SinceUnix {
		t.Fatalf("started_at not migrated: %#v", rec)
	}
	settings, ok := store.ChatSettings(-1001)
	if !ok || settings.RehabTopicID != 7229 {
		t.Fatalf("rehab topic not restored: %#v ok=%v", settings, ok)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if !strings.Contains(string(saved), `"started_at": 1700000000`) {
		t.Fatalf("started_at was not persisted: %s", saved)
	}
}

func TestDiscussionReplyMarkAndMigration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	raw := `{
  "chats": {},
  "discussion_replies": [
    {"chat_id": -1001, "topic_id": 7, "posted_unix": 1700000000},
    {"chat_id": -1001, "topic_id": 7, "posted_unix": 1700000001},
    {"chat_id": 0, "topic_id": 8}
  ]
}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("write state: %v", err)
	}
	store, err := NewStore(path, nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if !store.DiscussionReplyPosted(-1001, 7) {
		t.Fatal("expected migrated discussion reply")
	}
	if created, err := store.MarkDiscussionReply(-1001, 7, time.Now().UTC()); err != nil || created {
		t.Fatalf("duplicate mark created=%v err=%v", created, err)
	}
	if created, err := store.MarkDiscussionReply(-1001, 9, time.Now().UTC()); err != nil || !created {
		t.Fatalf("new mark created=%v err=%v", created, err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	if strings.Count(string(saved), `"topic_id": 7`) != 1 || !strings.Contains(string(saved), `"topic_id": 9`) {
		t.Fatalf("discussion replies were not normalized/persisted: %s", saved)
	}
}

func TestRemoveJailStoresLastRehabDuration(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	chatID := int64(-1001)
	userID := int64(42)
	store.TrackMember(chatID, &tele.User{ID: userID, FirstName: "user"}, 0, 0, false)
	started := time.Now().UTC().Add(-30 * time.Minute)
	store.SetJail(chatID, ModerationRecord{
		UserID:        userID,
		DisplayName:   "user",
		SinceUnix:     started.Unix(),
		StartedAtUnix: started.Unix(),
		UntilUnix:     time.Now().UTC().Add(time.Hour).Unix(),
	})
	store.RemoveJail(chatID, userID)
	member, ok := store.Member(chatID, userID)
	if !ok {
		t.Fatal("member missing")
	}
	if member.LastRehabDuration < int64((29*time.Minute)/time.Second) {
		t.Fatalf("last rehab duration too small: %d", member.LastRehabDuration)
	}
}

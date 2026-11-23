package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func TestReleaseLineShowsRelativeAndExactTime(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	line := releaseLine(now.Add(72*time.Hour).Unix(), now)
	if !strings.Contains(line, "через 3 дня") {
		t.Fatalf("expected relative release time, got %q", line)
	}
	if !strings.Contains(line, "2026-04-29T") {
		t.Fatalf("expected exact release time, got %q", line)
	}
	if got := releaseLine(0, now); !strings.Contains(got, "вручную") {
		t.Fatalf("expected manual release line, got %q", got)
	}
}

func TestManagedChatLegacyFallbackAndDisable(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	app := &App{store: store}
	app.applyConfig(Config{
		Chat:     ChatConfig{ID: -1001, RehabTopicID: 77},
		Telegram: TelegramConfig{ShowDebugCommands: true},
	})

	legacy := &tele.Chat{ID: -1001, Type: tele.ChatSuperGroup, Title: "legacy"}
	if got := app.rehabTopicID(legacy); got != 77 {
		t.Fatalf("legacy chat should use configured rehab topic, got %d", got)
	}

	other := &tele.Chat{ID: -1002, Type: tele.ChatSuperGroup, Title: "second"}
	if got := app.rehabTopicID(other); got != 0 {
		t.Fatalf("other chat should not inherit legacy rehab topic, got %d", got)
	}
	app.rememberManagedChat(other)
	settings, ok := store.ChatSettings(other.ID)
	if !ok || !settings.Enabled || settings.RehabTopicID != 0 {
		t.Fatalf("unexpected discovered settings: %#v ok=%v", settings, ok)
	}

	settings.Enabled = false
	if err := store.UpsertChatSettings(settings); err != nil {
		t.Fatalf("UpsertChatSettings: %v", err)
	}
	if app.isManagedChat(other) {
		t.Fatal("disabled managed chat should not be active")
	}
}

func TestManagedChatStoredTopicWinsOverConfig(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	app := &App{store: store}
	app.applyConfig(Config{
		Chat: ChatConfig{ID: -1001, RehabTopicID: 77},
	})

	chat := &tele.Chat{ID: -1001, Type: tele.ChatSuperGroup, Title: "legacy"}
	if err := store.UpsertChatSettings(ChatSettings{ChatID: chat.ID, Enabled: true, RehabTopicID: 12}); err != nil {
		t.Fatalf("UpsertChatSettings: %v", err)
	}
	if got := app.rehabTopicID(chat); got != 12 {
		t.Fatalf("stored rehab topic should win over config, got %d", got)
	}

	app.rememberManagedChat(chat)
	settings, ok := store.ChatSettings(chat.ID)
	if !ok || settings.RehabTopicID != 12 || settings.PersistedRehabTopicID != 12 {
		t.Fatalf("rememberManagedChat overwrote stored topic: %#v ok=%v", settings, ok)
	}
}

func TestUnscopedConfigTopicDoesNotApplyToDiscoveredGroups(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	app := &App{store: store}
	app.applyConfig(Config{
		Chat: ChatConfig{ID: 0, RehabTopicID: 77},
	})

	chat := &tele.Chat{ID: -1002, Type: tele.ChatSuperGroup, Title: "second"}
	app.rememberManagedChat(chat)
	settings, ok := store.ChatSettings(chat.ID)
	if !ok || !settings.Enabled || settings.RehabTopicID != 0 || settings.PersistedRehabTopicID != 0 {
		t.Fatalf("unscoped config topic should not be copied to discovered chat: %#v ok=%v", settings, ok)
	}
	if got := app.rehabTopicID(chat); got != 0 {
		t.Fatalf("unscoped config topic should not be effective for discovered chat, got %d", got)
	}
}

func TestAIRequiresExplicitWhitelistedChat(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	app := &App{store: store}
	app.applyConfig(Config{AI: AIConfig{Enabled: true}})
	chat := &tele.Chat{ID: -1002, Type: tele.ChatSuperGroup, Title: "second"}

	app.rememberManagedChat(chat)
	if app.aiEnabledForChat(chat) {
		t.Fatal("auto-discovered chat should not be AI-whitelisted")
	}
	if err := store.UpsertChatSettings(ChatSettings{ChatID: chat.ID, Enabled: true, AIEnabledMode: "on"}); err != nil {
		t.Fatalf("UpsertChatSettings: %v", err)
	}
	if !app.aiEnabledForChat(chat) {
		t.Fatal("explicit AI-enabled chat should be whitelisted")
	}
}

func TestLegacyConfigChatIsAIWhitelisted(t *testing.T) {
	app := &App{}
	app.applyConfig(Config{Chat: ChatConfig{ID: -1001}, AI: AIConfig{Enabled: true}})

	if !app.aiEnabledForChat(&tele.Chat{ID: -1001, Type: tele.ChatSuperGroup}) {
		t.Fatal("configured legacy chat should be AI-whitelisted")
	}
	if app.aiEnabledForChat(&tele.Chat{ID: -1002, Type: tele.ChatSuperGroup}) {
		t.Fatal("non-configured chat should not be AI-whitelisted")
	}
}

func TestShouldMuteNoRehabSpamPolicy(t *testing.T) {
	cfg := SpamConfig{AutoMute: true}
	settings := ChatSettings{SpamNoRehabAction: spamNoRehabMuteOnDelete}
	if !shouldMuteNoRehabSpam(cfg, settings, 0, true, false, false) {
		t.Fatal("expected immediate mute for delete-worthy spam without rehab")
	}
	if shouldMuteNoRehabSpam(cfg, settings, 10, true, false, false) {
		t.Fatal("rehab-enabled chat should use normal thresholds")
	}
	settings.SpamNoRehabAction = "normal"
	if shouldMuteNoRehabSpam(cfg, settings, 0, true, false, false) {
		t.Fatal("normal spam policy should not force immediate mute")
	}
}

func TestWebSupergroupAddAndUseKnownTopicAsRehab(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	app := &App{store: store}
	app.applyConfig(Config{})

	req := httptest.NewRequest(http.MethodPost, "/supergroups/add", strings.NewReader("chat_id=-1003&title=Main&enabled=on&spam_no_rehab_action=mute_on_delete"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	app.webSupergroupAdd(httptest.NewRecorder(), req)
	settings, ok := store.ChatSettings(-1003)
	if !ok || settings.Title != "Main" || !settings.Enabled {
		t.Fatalf("unexpected settings after add: %#v ok=%v", settings, ok)
	}

	store.TrackMember(-1003, &tele.User{ID: 50, FirstName: "User"}, 42, 100, true)
	page := httptest.NewRecorder()
	app.webSupergroups(page, httptest.NewRequest(http.MethodGet, "/supergroups", nil))
	if !strings.Contains(page.Body.String(), `topic_id" value="42"`) {
		t.Fatalf("expected use-as-rehab topic button, body=%s", page.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/supergroups/rehab-topic", strings.NewReader("chat_id=-1003&topic_id=42"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	app.webSupergroupRehabTopic(httptest.NewRecorder(), req)
	settings, ok = store.ChatSettings(-1003)
	if !ok || settings.RehabTopicID != 42 {
		t.Fatalf("expected rehab topic 42, got %#v ok=%v", settings, ok)
	}
}

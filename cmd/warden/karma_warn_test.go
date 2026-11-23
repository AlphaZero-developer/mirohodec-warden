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

func TestFormatQuoteOmitsHeader(t *testing.T) {
	app := &App{}
	app.applyConfig(Config{I18n: I18nConfig{Locale: "en"}})
	got := app.formatQuote(QuoteRecord{Text: "Soon.", Author: "Laya"})
	if got == "Laya quote:\nSoon.\nAuthor: Laya" {
		t.Fatal("quote still has legacy header")
	}
	if got != "Soon.\nAuthor: Laya" {
		t.Fatalf("unexpected quote format: %q", got)
	}
}

func TestKnownCommandAndManualKarmaDelta(t *testing.T) {
	if !knownCommand("warn") || !knownCommand("/topkarma") {
		t.Fatal("new commands should be known")
	}
	if knownCommand("definitely_unknown") {
		t.Fatal("unknown command detected as known")
	}
	if delta, ok := manualKarmaDelta("+карма"); !ok || delta != 1 {
		t.Fatalf("unexpected +karma parse: %d %v", delta, ok)
	}
	if delta, ok := manualKarmaDelta("-karma"); !ok || delta != -1 {
		t.Fatalf("unexpected -karma parse: %d %v", delta, ok)
	}
}

func TestJSONStoreWarningsAndActivityKarma(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	chatID := int64(100)
	user := &tele.User{ID: 200, FirstName: "User"}
	for i := 0; i < 10; i++ {
		store.TrackMember(chatID, user, 1, i+1, true)
	}
	if score, ok := store.MaybeAwardActivityKarma(chatID, user.ID, 10, 5, time.Now().UTC()); !ok || score != 1 {
		t.Fatalf("expected first activity karma award, score=%d ok=%v", score, ok)
	}
	if _, ok := store.MaybeAwardActivityKarma(chatID, user.ID, 10, 5, time.Now().UTC()); ok {
		t.Fatal("did not expect duplicate activity karma award")
	}
	warn, err := store.AddWarning(chatID, WarningRecord{UserID: user.ID, DisplayName: "User", ExpiresUnix: time.Now().Add(time.Hour).Unix()})
	if err != nil {
		t.Fatalf("AddWarning: %v", err)
	}
	if count := store.ActiveWarningCount(chatID, user.ID, time.Now().UTC()); count != 1 {
		t.Fatalf("unexpected warning count: %d", count)
	}
	if _, ok, err := store.DeactivateWarning(chatID, warn.ID); err != nil || !ok {
		t.Fatalf("DeactivateWarning ok=%v err=%v", ok, err)
	}
	if count := store.ActiveWarningCount(chatID, user.ID, time.Now().UTC()); count != 0 {
		t.Fatalf("unexpected warning count after deactivate: %d", count)
	}
}

func TestWebLoginRejectsBadToken(t *testing.T) {
	app := &App{}
	handler := app.webLogin(WebConfig{AdminToken: "secret", SessionTTL: time.Hour})
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("token=bad"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if rec.Result().Cookies() != nil && len(rec.Result().Cookies()) > 0 {
		t.Fatal("bad token should not set cookies")
	}
}

func TestWebLoginPermanentSession(t *testing.T) {
	app := &App{}
	handler := app.webLogin(WebConfig{AdminToken: "secret", SessionTTL: -1})
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("token=secret"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %d", len(cookies))
	}
	if cookies[0].MaxAge < int((9 * 365 * 24 * time.Hour).Seconds()) {
		t.Fatalf("permanent cookie MaxAge = %d, want about 10 years", cookies[0].MaxAge)
	}
}

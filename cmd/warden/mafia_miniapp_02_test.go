package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	wardenweb "mirohodec-warden/internal/web"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"
)

func miniAppPlayerHasRole(players []wardenweb.MafiaMiniAppPlayer, id int64, role string) bool {
	for _, player := range players {
		if player.ID == id {
			return player.Role == role
		}
	}
	return false
}

func TestValidateTelegramMiniAppInitData(t *testing.T) {
	token := "123456:secret"
	values := url.Values{}
	values.Set("auth_date", "1893456000")
	values.Set("query_id", "q1")
	values.Set("user", `{"id":777,"first_name":"Mira","username":"mira"}`)
	values.Set("hash", telegramMiniAppHash(values, token))
	user, err := validateTelegramMiniAppInitData(values.Encode(), token, time.Hour)
	if err != nil {
		t.Fatalf("validateTelegramMiniAppInitData: %v", err)
	}
	if user.ID != 777 || user.Username != "mira" {
		t.Fatalf("unexpected user: %+v", user)
	}
	values.Set("user", `{"id":778,"first_name":"Mallory"}`)
	if _, err := validateTelegramMiniAppInitData(values.Encode(), token, time.Hour); err == nil {
		t.Fatal("tampered initData accepted")
	}
}

func TestMafiaMiniAppURLAddsRoomID(t *testing.T) {
	t.Setenv(mafiaMiniAppEnvURL, "")
	var app App
	app.cfg.Store(Config{Web: WebConfig{MafiaMiniAppURL: "https://example.com/mafia-mini-app/"}})
	got := app.mafiaMiniAppURL("mf-test")
	want := "https://example.com/mafia-mini-app/?room_id=MF-TEST"
	if got != want {
		t.Fatalf("mafiaMiniAppURL = %q, want %q", got, want)
	}
}

func TestMafiaMiniAppURLFallsBackToPublicURL(t *testing.T) {
	t.Setenv(mafiaMiniAppEnvURL, "")
	var app App
	app.cfg.Store(Config{Web: WebConfig{PublicURL: "https://example.com/panel/"}})
	got := app.mafiaMiniAppURL("")
	want := "https://example.com/panel/mafia-mini-app/"
	if got != want {
		t.Fatalf("mafiaMiniAppURL fallback = %q, want %q", got, want)
	}
}

func TestMafiaMiniAppEntryMarkupUsesWebAppButton(t *testing.T) {
	t.Setenv(mafiaMiniAppEnvURL, "https://example.com/app")
	var app App
	markup := app.mafiaMiniAppEntryMarkup("MF-ROOM")
	if markup == nil || len(markup.InlineKeyboard) != 1 || len(markup.InlineKeyboard[0]) != 1 {
		t.Fatalf("unexpected markup: %#v", markup)
	}
	button := markup.InlineKeyboard[0][0]
	if button.WebApp == nil || button.WebApp.URL != "https://example.com/app?room_id=MF-ROOM" {
		t.Fatalf("unexpected web app button: %#v", button)
	}
}

func TestMafiaMiniAppEntryMarkupRequiresHTTPS(t *testing.T) {
	t.Setenv(mafiaMiniAppEnvURL, "")
	var app App
	app.cfg.Store(Config{Web: WebConfig{PublicURL: "http://5.34.208.171:8090"}})
	if markup := app.mafiaMiniAppEntryMarkup(""); markup != nil {
		t.Fatalf("http mini app url should not become Telegram WebApp button: %#v", markup)
	}
	warning := app.mafiaMiniAppHTTPSWarning("")
	if !strings.Contains(warning, "Telegram") || !strings.Contains(warning, "HTTPS") {
		t.Fatalf("expected HTTPS warning, got %q", warning)
	}
	if !strings.Contains(warning, "https://5-34-208-171.sslip.io/mafia-mini-app/") {
		t.Fatalf("expected free sslip suggestion, got %q", warning)
	}
	if strings.Contains(warning, "-HttpsDomain <домен>") {
		t.Fatalf("warning should not suggest paid/custom domain placeholder: %q", warning)
	}
}

func telegramMiniAppHash(values url.Values, token string) string {
	copied := url.Values{}
	for key, raw := range values {
		if key == "hash" {
			continue
		}
		copied[key] = append([]string(nil), raw...)
	}
	keys := make([]string, 0, len(copied))
	for key := range copied {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+"="+copied.Get(key))
	}
	secretHMAC := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = secretHMAC.Write([]byte(token))
	checkHMAC := hmac.New(sha256.New, secretHMAC.Sum(nil))
	_, _ = checkHMAC.Write([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(checkHMAC.Sum(nil))
}

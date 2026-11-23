package main

import (
	"path/filepath"
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func TestParseLayaScriptwriterRequestAdd(t *testing.T) {
	action, profession, ok := parseLayaScriptwriterRequest("Лайя, это сценарист, его не муть", "")
	if !ok || action != layaScriptwriterAdd || profession != "сценарист" {
		t.Fatalf("parseLayaScriptwriterRequest add = %v, %q, %t", action, profession, ok)
	}
}

func TestParseLayaScriptwriterRequestRemove(t *testing.T) {
	action, profession, ok := parseLayaScriptwriterRequest("Лайя, он больше не сценарист, можно мутить", "")
	if !ok || action != layaScriptwriterRemove || profession != "сценарист" {
		t.Fatalf("parseLayaScriptwriterRequest remove = %v, %q, %t", action, profession, ok)
	}
}

func TestParseLayaScriptwriterRequestProfessions(t *testing.T) {
	tests := []struct {
		text       string
		profession string
	}{
		{"Лайя, это художник, его не муть", "художник"},
		{"Лайя, это строитель, его не трогай", "строитель"},
		{"Лайя, это кодер, не бань за флуд", "кодер"},
		{"Лайя, это программист, его не муть", "кодер"},
	}

	for _, tt := range tests {
		action, profession, ok := parseLayaScriptwriterRequest(tt.text, "")
		if !ok || action != layaScriptwriterAdd || profession != tt.profession {
			t.Fatalf("parseLayaScriptwriterRequest(%q) = %v, %q, %t", tt.text, action, profession, ok)
		}
	}
}

func TestScriptwriterExemptionStoredAndSpamExempt(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	app := NewApp("", Config{}, nil, store, nil, nil, nil, nil, nil, nil)
	rec := ScriptwriterExemption{
		ChatID:      -1001,
		UserID:      42,
		Username:    "Writer",
		DisplayName: "Writer",
		Profession:  "кодер",
		AddedUnix:   time.Now().UTC().Unix(),
		AddedByID:   117,
		AddedByName: "Admin",
	}
	if err := store.SetScriptwriterExemption(rec); err != nil {
		t.Fatalf("SetScriptwriterExemption: %v", err)
	}

	if !app.isSpamExempt(-1001, 42) {
		t.Fatalf("scriptwriter was not treated as spam-exempt")
	}
	if app.isSpamExempt(-1002, 42) {
		t.Fatalf("scriptwriter exemption leaked to another chat")
	}
	stored, ok := store.ScriptwriterExemption(-1001, 42)
	if !ok || stored.Profession != "кодер" {
		t.Fatalf("stored exemption profession = %#v, %t", stored, ok)
	}
}

func TestLayaScriptwriterTargetUsesReply(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	app := NewApp("", Config{}, nil, store, nil, nil, nil, nil, nil, nil)
	msg := &tele.Message{
		Text:   "Лайя, это сценарист, его не муть",
		Sender: &tele.User{ID: 117, FirstName: "Admin"},
		ReplyTo: &tele.Message{
			Sender: &tele.User{ID: 42, Username: "writer", FirstName: "Writer"},
		},
	}

	target, ok := app.layaScriptwriterTarget(-1001, msg, 117)
	if !ok || target.ID != 42 || target.Username != "writer" {
		t.Fatalf("layaScriptwriterTarget() = %#v, %t", target, ok)
	}
}

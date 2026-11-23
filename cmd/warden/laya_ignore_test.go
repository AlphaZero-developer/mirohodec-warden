package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestParseLayaIgnoreRequestAdd(t *testing.T) {
	action, args, ok := parseLayaIgnoreRequest("Лайя, игнорируй этого человека", "")
	if !ok || action != layaIgnoreAdd {
		t.Fatalf("parseLayaIgnoreRequest add = %v, %v, %t", action, args, ok)
	}
	if len(args) != 0 {
		t.Fatalf("reply-target command should not invent args: %#v", args)
	}
}

func TestParseLayaIgnoreRequestTargetArg(t *testing.T) {
	action, args, ok := parseLayaIgnoreRequest("@LayaBot ignore @TargetUser", "LayaBot")
	if !ok || action != layaIgnoreAdd {
		t.Fatalf("parseLayaIgnoreRequest target = %v, %v, %t", action, args, ok)
	}
	if len(args) != 1 || args[0] != "@TargetUser" {
		t.Fatalf("target args = %#v", args)
	}
}

func TestParseLayaIgnoreRequestRemove(t *testing.T) {
	tests := []string{
		"Лайя, не игнорируй этого человека",
		"Лайя, перестань игнорировать @TargetUser",
		"Лайя, отвечай 42",
	}
	for _, text := range tests {
		action, _, ok := parseLayaIgnoreRequest(text, "")
		if !ok || action != layaIgnoreRemove {
			t.Fatalf("parseLayaIgnoreRequest(%q) = %v, %t", text, action, ok)
		}
	}
}

func TestLayaIgnoredUserStoredPerChat(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	app := NewApp("", Config{}, nil, store, nil, nil, nil, nil, nil, nil)
	rec := LayaIgnoredUser{
		ChatID:      -1001,
		UserID:      42,
		Username:    "Target",
		DisplayName: "Target",
		AddedUnix:   time.Now().UTC().Unix(),
		AddedByID:   117,
		AddedByName: "Admin",
	}
	if err := store.SetLayaIgnoredUser(rec); err != nil {
		t.Fatalf("SetLayaIgnoredUser: %v", err)
	}
	if !app.layaIsIgnoring(-1001, 42) {
		t.Fatalf("laya ignore flag was not stored")
	}
	if app.layaIsIgnoring(-1002, 42) {
		t.Fatalf("laya ignore leaked to another chat")
	}
	stored, ok := store.LayaIgnoredUser(-1001, 42)
	if !ok || stored.Username != "target" {
		t.Fatalf("stored ignored user = %#v, %t", stored, ok)
	}
	if _, ok, err := store.RemoveLayaIgnoredUser(-1001, 42); err != nil || !ok {
		t.Fatalf("RemoveLayaIgnoredUser ok=%t err=%v", ok, err)
	}
	if app.layaIsIgnoring(-1001, 42) {
		t.Fatalf("laya ignore flag survived removal")
	}
}

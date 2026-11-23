package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func TestIdeaTextFromArgsOrReply(t *testing.T) {
	if got := ideaTextFrom([]string{"новая", "идея"}, nil); got != "новая идея" {
		t.Fatalf("unexpected args idea: %q", got)
	}
	msg := &tele.Message{ReplyTo: &tele.Message{Text: "идея из reply"}}
	if got := ideaTextFrom(nil, msg); got != "идея из reply" {
		t.Fatalf("unexpected reply idea: %q", got)
	}
	if got := ideaTextFrom(nil, &tele.Message{}); got != "" {
		t.Fatalf("expected empty idea, got %q", got)
	}
}

func TestFormatIdeaReportIncludesContext(t *testing.T) {
	report := formatIdeaReport(
		&tele.Chat{ID: -1001, Title: "chat"},
		&tele.Message{ID: 42, ThreadID: 7},
		&tele.User{ID: 10, Username: "user", FirstName: "Name"},
		"идея",
	)
	for _, want := range []string{"Новая идея", "Автор: Name", "User ID: 10", "Username: @user", "Чат: chat (-1001)", "Тема: 7", "Сообщение: 42", "Текст:", "идея"} {
		if !strings.Contains(report, want) {
			t.Fatalf("report does not contain %q:\n%s", want, report)
		}
	}
}

func TestFormatOwnerReportUsesKindTitle(t *testing.T) {
	report := formatOwnerReport(
		ownerReportTitle("appeal"),
		&tele.Chat{ID: -1001, Title: "chat"},
		&tele.Message{ID: 42},
		&tele.User{ID: 10, FirstName: "Name"},
		"прошу пересмотреть",
	)
	if !strings.Contains(report, "Обжалование решения") {
		t.Fatalf("missing appeal title:\n%s", report)
	}
	if !strings.Contains(report, "прошу пересмотреть") {
		t.Fatalf("missing appeal body:\n%s", report)
	}
}

func TestOwnerReportTitlesForJobAndDonate(t *testing.T) {
	if got := ownerReportTitle("job"); !strings.Contains(got, "Заявка на работу") {
		t.Fatalf("job title = %q", got)
	}
	if got := ownerReportTitle("donate"); !strings.Contains(got, "Донат") {
		t.Fatalf("donate title = %q", got)
	}
}

func TestIdeaCooldownRemaining(t *testing.T) {
	app := &App{
		lastIdeaUser: make(map[string]time.Time),
		lastIdeaChat: make(map[string]time.Time),
	}
	now := time.Unix(1000, 0)
	cfg := IdeaConfig{UserCooldown: "15m", ChatCooldown: "30s", MaxLength: 1500}
	if wait := app.ideaCooldownRemaining(1, 2, now, cfg); wait != 0 {
		t.Fatalf("unexpected initial wait: %v", wait)
	}
	app.rememberIdeaCooldown(1, 2, now)
	if wait := app.ideaCooldownRemaining(1, 2, now.Add(10*time.Second), cfg); wait <= 14*time.Minute {
		t.Fatalf("expected user cooldown to dominate, got %v", wait)
	}
	if wait := app.ideaCooldownRemaining(1, 3, now.Add(10*time.Second), cfg); wait != 20*time.Second {
		t.Fatalf("expected chat cooldown, got %v", wait)
	}
}

func TestAppealCooldownRemaining(t *testing.T) {
	app := &App{lastAppeal: make(map[string]time.Time)}
	now := time.Unix(1000, 0)
	if wait := app.appealCooldownRemaining(1, 2, now); wait != 0 {
		t.Fatalf("unexpected initial appeal wait: %v", wait)
	}

	app.rememberAppealCooldown(1, 2, now)
	if wait := app.appealCooldownRemaining(1, 2, now.Add(time.Hour)); wait != 23*time.Hour {
		t.Fatalf("unexpected appeal wait: %v", wait)
	}
	if wait := app.appealCooldownRemaining(1, 2, now.Add(25*time.Hour)); wait != 0 {
		t.Fatalf("expired appeal cooldown should be clear, got %v", wait)
	}
}

func TestAppealCooldownUsesStore(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	now := time.Unix(1000, 0)
	if err := store.SetOwnerReportLastSent("appeal", 1, 2, now); err != nil {
		t.Fatalf("SetOwnerReportLastSent: %v", err)
	}
	app := &App{store: store}
	if wait := app.appealCooldownRemaining(1, 2, now.Add(time.Hour)); wait != 23*time.Hour {
		t.Fatalf("unexpected stored appeal wait: %v", wait)
	}
}

func TestJailedUserCannotSendBugOrIdea(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	app := &App{store: store}
	chatID := int64(1)
	userID := int64(2)
	now := time.Unix(1000, 0)

	if app.jailedUserCannotSendReport(chatID, userID, "bug", now) {
		t.Fatal("user without jail should be allowed")
	}
	store.SetJail(chatID, ModerationRecord{UserID: userID, DisplayName: "user", UntilUnix: now.Add(time.Hour).Unix()})
	for _, kind := range []string{"idea", "bug", "reportbug"} {
		if !app.jailedUserCannotSendReport(chatID, userID, kind, now) {
			t.Fatalf("jailed user should not send %s", kind)
		}
	}
	if !app.jailedUserCannotSendReport(999, userID, "bug", now) {
		t.Fatal("jailed user should not send bug from another chat context")
	}
	if app.jailedUserCannotSendReport(chatID, userID, "appeal", now) {
		t.Fatal("appeal should stay available for jailed users")
	}
}

func TestOwnerReportCommandMessage(t *testing.T) {
	for _, text := range []string{"/idea текст", "/bug текст", "/reportbug текст", "/appeal текст"} {
		if !isOwnerReportCommandMessage(&tele.Message{Text: text}) {
			t.Fatalf("%q should be an owner report command", text)
		}
	}
	if isOwnerReportCommandMessage(&tele.Message{Text: "/ward"}) {
		t.Fatal("/ward should not be treated as an owner report command")
	}
}

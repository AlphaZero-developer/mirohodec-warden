package main

import (
	"context"
	wardenweb "mirohodec-warden/internal/web"
	"path/filepath"
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func newWebPanelTestService(t *testing.T) (webPanelService, *App) {
	t.Helper()
	dir := t.TempDir()
	store, err := NewStore(filepath.Join(dir, "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	photos, err := NewPhotoStore(filepath.Join(dir, "photos.json"), nil)
	if err != nil {
		t.Fatalf("NewPhotoStore: %v", err)
	}
	quotes, err := NewQuoteStore(filepath.Join(dir, "quotes.json"), nil)
	if err != nil {
		t.Fatalf("NewQuoteStore: %v", err)
	}
	faq, err := NewFAQStore(filepath.Join(dir, "faq.json"), nil)
	if err != nil {
		t.Fatalf("NewFAQStore: %v", err)
	}
	app := NewApp(filepath.Join(dir, "config.yaml"), Config{
		Chat:     ChatConfig{ID: -1001, RehabTopicID: 77},
		Telegram: TelegramConfig{Mode: "polling", ShowDebugCommands: true},
		Web:      WebConfig{PublicURL: "http://127.0.0.1:8090"},
	}, nil, store, photos, quotes, faq, nil, nil, nil)
	return webPanelService{app: app}, app
}

func TestWebPanelServiceSupergroupCRUD(t *testing.T) {
	service, app := newWebPanelTestService(t)
	ctx := context.Background()

	if err := service.AddGroup(ctx, wardenGroupInput(-1002, "Second", 12)); err != nil {
		t.Fatalf("AddGroup: %v", err)
	}
	settings, ok := app.store.ChatSettings(-1002)
	if !ok || settings.Title != "Second" || settings.RehabTopicID != 12 || !settings.Enabled {
		t.Fatalf("unexpected settings after add: %#v ok=%v", settings, ok)
	}
	groups, err := service.Groups(ctx)
	if err != nil {
		t.Fatalf("Groups: %v", err)
	}
	var found *wardenweb.Group
	for i := range groups {
		if groups[i].ID == -1002 {
			found = &groups[i]
			break
		}
	}
	if found == nil ||
		!webGroupHasSetting(*found, "Discussion", "auto link reply") ||
		!webGroupHasSetting(*found, "Психцентр", "release exam") ||
		!webGroupHasSetting(*found, "Spam", "without center topic") {
		t.Fatalf("expected detailed group settings, group=%#v", found)
	}

	input := wardenGroupInput(-1002, "Renamed", 34)
	input.Enabled = false
	input.AIEnabledMode = "off"
	input.SpamEnabledMode = "off"
	input.WarnEnabledMode = "on"
	input.WarnExpireAfter = "14d"
	input.WarnMuteThreshold = 2
	input.WarnMuteDuration = "30m"
	input.WarnRehabThreshold = 4
	input.WarnRehabDuration = "6h"
	input.KarmaEnabledMode = "on"
	input.KarmaMessagesPerPoint = 7
	input.KarmaDailyLimit = 3
	input.KarmaManualCooldown = "2h"
	input.FAQEnabledMode = "off"
	input.QuotesEnabledMode = "off"
	input.PhotosEnabledMode = "off"
	if err := service.UpdateGroup(ctx, input); err != nil {
		t.Fatalf("UpdateGroup: %v", err)
	}
	settings, ok = app.store.ChatSettings(-1002)
	if !ok || settings.Title != "Renamed" || settings.RehabTopicID != 34 || settings.Enabled {
		t.Fatalf("unexpected settings after update: %#v ok=%v", settings, ok)
	}
	if settings.AIEnabledMode != "off" || settings.SpamEnabledMode != "off" || settings.WarnEnabledMode != "on" ||
		settings.WarnExpireAfter != "14d" || settings.WarnMuteThreshold != 2 || settings.WarnMuteDuration != "30m" ||
		settings.WarnRehabThreshold != 4 || settings.WarnRehabDuration != "6h" ||
		settings.KarmaEnabledMode != "on" || settings.KarmaMessagesPerPoint != 7 || settings.KarmaDailyLimit != 3 ||
		settings.KarmaManualCooldown != "2h" || settings.FAQEnabledMode != "off" ||
		settings.QuotesEnabledMode != "off" || settings.PhotosEnabledMode != "off" {
		t.Fatalf("group runtime overrides were not stored: %#v", settings)
	}
	warnCfg := app.activeWarnConfigForChat(-1002)
	karmaCfg := app.activeKarmaConfigForChat(-1002)
	if !warnCfg.Enabled || warnCfg.MuteThreshold != 2 || warnCfg.RehabDuration != "6h" ||
		!karmaCfg.Enabled || karmaCfg.ActivityMessagesPerPoint != 7 || karmaCfg.ManualCooldown != "2h" ||
		app.aiEnabledForChat(&tele.Chat{ID: -1002}) || app.faqEnabledForChat(&tele.Chat{ID: -1002}) {
		t.Fatalf("group runtime overrides are not active: warn=%#v karma=%#v settings=%#v", warnCfg, karmaCfg, settings)
	}

	if err := service.SetGroupDebug(ctx, -1002, true); err != nil {
		t.Fatalf("SetGroupDebug: %v", err)
	}
	settings, _ = app.store.ChatSettings(-1002)
	if !settings.DebugCommands {
		t.Fatal("debug commands should be enabled")
	}

	if err := service.DeleteGroup(ctx, -1002); err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}
	if _, ok := app.store.ChatSettings(-1002); ok {
		t.Fatal("group should be deleted")
	}
}

func TestWebPanelServiceRehabAddRelease(t *testing.T) {
	service, app := newWebPanelTestService(t)
	ctx := context.Background()
	app.store.TrackMember(-1001, &tele.User{ID: 42, FirstName: "User"}, 9, 1, true)

	if err := service.AddRehab(ctx, wardenRehabInput(-1001, 42)); err != nil {
		t.Fatalf("AddRehab: %v", err)
	}
	rec, ok := app.store.ActiveJail(-1001, 42, time.Now().UTC())
	if !ok || rec.UserID != 42 || rec.Reason != "test" {
		t.Fatalf("expected active jail, rec=%#v ok=%v", rec, ok)
	}

	if err := service.ReleaseRehab(ctx, -1001, 42); err != nil {
		t.Fatalf("ReleaseRehab: %v", err)
	}
	if _, ok := app.store.ActiveJail(-1001, 42, time.Now().UTC()); ok {
		t.Fatal("jail should be released")
	}

	if err := service.UpdateRehabBrainwash(ctx, wardenweb.RehabBrainwashInput{
		Enabled:      true,
		AIEnabled:    true,
		LeaderUserID: 117,
		LeaderName:   "Ruslan",
		AIPrompt:     "AI {leader}",
		Messages:     "Главный {leader}",
	}); err != nil {
		t.Fatalf("UpdateRehabBrainwash: %v", err)
	}
	brainwash := app.activeRehabBrainwashConfig()
	if !brainwash.Enabled || !brainwash.AIEnabled || brainwash.LeaderUserID != 117 || brainwash.LeaderName != "Ruslan" || brainwash.AIPrompt != "AI {leader}" || brainwash.Messages != "Главный {leader}" {
		t.Fatalf("unexpected brainwash config: %#v", brainwash)
	}
	page, err := service.Rehab(ctx, -1001)
	if err != nil {
		t.Fatalf("Rehab: %v", err)
	}
	if !page.Brainwash.Enabled || !page.Brainwash.AIEnabled || page.Brainwash.LeaderUserID != 117 || page.Brainwash.LeaderName != "Ruslan" || page.Brainwash.AIPrompt != "AI {leader}" {
		t.Fatalf("unexpected rehab brainwash page: %#v", page.Brainwash)
	}
}

func TestWebPanelServiceUsersAndForeverBan(t *testing.T) {
	service, app := newWebPanelTestService(t)
	ctx := context.Background()
	app.store.TrackMember(-1001, &tele.User{ID: 42, Username: "mira", FirstName: "Mira"}, 9, 1, true)
	app.store.TrackMember(-1001, &tele.User{ID: 43, Username: "kai", FirstName: "Kai"}, 0, 2, true)

	rows, err := service.Users(ctx, -1001)
	if err != nil {
		t.Fatalf("Users: %v", err)
	}
	if len(rows) != 2 || rows[0].ID != 43 || !rows[0].CanBan {
		t.Fatalf("expected all known users sorted by activity with ban action, rows=%#v", rows)
	}

	if err := service.AddForeverBan(ctx, wardenweb.ForeverBanInput{ChatID: -1001, UserID: 43, Reason: "panel"}); err != nil {
		t.Fatalf("AddForeverBan: %v", err)
	}
	rec, ok := app.store.Ban(-1001, 43)
	if !ok || rec.Reason != "panel" || rec.ByAdminName != "web panel" {
		t.Fatalf("expected forever ban record, rec=%#v ok=%v", rec, ok)
	}
}

package main

import (
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func TestLayaInlineMarkupIsDisabled(t *testing.T) {
	app := &App{}
	markup := app.layaInlineMarkup(&tele.Chat{ID: -1001, Type: tele.ChatSuperGroup}, &tele.User{ID: 42})
	if markup != nil && len(markup.InlineKeyboard) != 0 {
		t.Fatalf("regular laya replies should not expose inline menu buttons: %#v", markup.InlineKeyboard)
	}
}

func TestLayaMenuMarkupHasWorkingCallbacks(t *testing.T) {
	app := &App{}
	markup := app.layaMenuMarkup(&tele.Chat{ID: -1001, Type: tele.ChatSuperGroup}, &tele.User{ID: 42})
	for _, want := range []string{"laya:selfmenu", "laya:selfmute:5m", "laya:selfjail:15m", "laya:profile", "laya:shopmenu", "laya:statsmenu"} {
		if !markupHasCallbackData(markup, want) {
			t.Fatalf("laya markup missing callback %q", want)
		}
	}
	for _, hidden := range []string{"laya:openmenu", "laya:hidemenu"} {
		if markupHasCallbackData(markup, hidden) {
			t.Fatalf("command menu should not expose callback %q", hidden)
		}
	}
}

func TestLayaSubmenusExposeConcreteActions(t *testing.T) {
	app := &App{}
	for _, tt := range []struct {
		name   string
		markup *tele.ReplyMarkup
		want   []string
	}{
		{name: "self", markup: app.layaSelfMarkup(nil, nil), want: []string{"laya:selfmute:15m", "laya:selfjail:6h", "laya:selfban:1h"}},
		{name: "shop", markup: app.layaShopMarkup(), want: []string{"laya:buy:charm", "laya:buy:quote", "laya:buy:title:legend"}},
		{name: "stats", markup: app.layaStatsMarkup(), want: []string{"laya:karma", "laya:topactive", "laya:topkarma"}},
	} {
		for _, want := range tt.want {
			if !markupHasCallbackData(tt.markup, want) {
				t.Fatalf("%s markup missing callback %q", tt.name, want)
			}
		}
	}
}

func TestLayaRestrictionMarkupHidesBugInRehab(t *testing.T) {
	app := &App{}
	rehabMarkup := app.layaRestrictionMarkup(true)
	if !markupHasCallbackData(rehabMarkup, "laya:appeal_quick") {
		t.Fatal("rehab restriction markup should keep appeal")
	}
	if markupHasCallbackData(rehabMarkup, "laya:bug_quick") {
		t.Fatal("rehab restriction markup should hide bug report")
	}

	muteMarkup := app.layaRestrictionMarkup(false)
	if !markupHasCallbackData(muteMarkup, "laya:bug_quick") {
		t.Fatal("mute restriction markup should keep bug report")
	}
}

func TestParseLayaSelfCallback(t *testing.T) {
	action, d, ok := parseLayaSelfCallback("laya:confirm:selfjail:15m", "laya:confirm:")
	if !ok || action != "selfjail" || d != 15*time.Minute {
		t.Fatalf("parse confirm callback = (%q, %s, %t)", action, d, ok)
	}
	if _, _, ok := parseLayaSelfCallback("laya:confirm:selfban:forever", "laya:confirm:"); ok {
		t.Fatal("forever self callback should be rejected")
	}
}

func TestParseSelfModerationDuration(t *testing.T) {
	got, err := parseSelfModerationDuration(nil, 5*time.Minute, time.Hour)
	if err != nil || got != 5*time.Minute {
		t.Fatalf("default duration = %s, %v", got, err)
	}
	if _, err := parseSelfModerationDuration([]string{"forever"}, 5*time.Minute, time.Hour); err == nil {
		t.Fatal("forever self duration should be rejected")
	}
	if _, err := parseSelfModerationDuration([]string{"2h"}, 5*time.Minute, time.Hour); err == nil {
		t.Fatal("duration over max should be rejected")
	}
}

func markupHasCallbackData(markup *tele.ReplyMarkup, data string) bool {
	if markup == nil {
		return false
	}
	for _, row := range markup.InlineKeyboard {
		for _, btn := range row {
			if btn.Data == data {
				return true
			}
		}
	}
	return false
}

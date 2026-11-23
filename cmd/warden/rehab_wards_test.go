package main

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeRehabWard(t *testing.T) {
	tests := map[string]string{
		"quiet":      rehabWardQuiet,
		"тихая":      rehabWardQuiet,
		"буйная":     rehabWardRowdy,
		"палата=vip": rehabWardVIP,
		"вип":        rehabWardVIP,
	}

	for value, want := range tests {
		if got := normalizeRehabWard(value); got != want {
			t.Fatalf("normalizeRehabWard(%q) = %q, want %q", value, got, want)
		}
	}
}

func TestParseJailOptionsWithWard(t *testing.T) {
	before := time.Now().UTC()
	until, ward, reason, err := parseJailOptions([]string{"vip", "30m", "за", "дело"}, true)
	if err != nil {
		t.Fatalf("parseJailOptions returned error: %v", err)
	}
	if ward != rehabWardVIP {
		t.Fatalf("ward = %q, want %q", ward, rehabWardVIP)
	}
	if reason != "за дело" {
		t.Fatalf("reason = %q, want %q", reason, "за дело")
	}
	remaining := time.Unix(until, 0).UTC().Sub(before)
	if remaining < 29*time.Minute || remaining > 31*time.Minute {
		t.Fatalf("remaining = %s, want about 30m", remaining)
	}
}

func TestParseJailOptionsWithDurationThenWard(t *testing.T) {
	_, ward, reason, err := parseJailOptions([]string{"10", "минут", "буйная", "флуд"}, true)
	if err != nil {
		t.Fatalf("parseJailOptions returned error: %v", err)
	}
	if ward != rehabWardRowdy {
		t.Fatalf("ward = %q, want %q", ward, rehabWardRowdy)
	}
	if reason != "флуд" {
		t.Fatalf("reason = %q, want %q", reason, "флуд")
	}
}

func TestRehabWardLine(t *testing.T) {
	line := rehabWardLine(ModerationRecord{RehabWard: rehabWardVIP})
	if !strings.Contains(line, "VIP") {
		t.Fatalf("ward line = %q, want VIP marker", line)
	}
}

func TestAdjustRehabEscapeChanceForWard(t *testing.T) {
	if got := adjustRehabEscapeChanceForWard(5, ModerationRecord{RehabWard: rehabWardRowdy}); got != 4 {
		t.Fatalf("rowdy escape chance = %d, want 4", got)
	}
	if got := adjustRehabEscapeChanceForWard(5, ModerationRecord{RehabWard: rehabWardVIP}); got != 7 {
		t.Fatalf("vip escape chance = %d, want 7", got)
	}
	if got := adjustRehabEscapeChanceForWard(0, ModerationRecord{RehabWard: rehabWardRowdy}); got != 0 {
		t.Fatalf("escape chance floor = %d, want 0", got)
	}
}

func TestRehabBribeCostAndChance(t *testing.T) {
	member := MemberRecord{Messages: 600, MessagesToday: 50, ActivityDay: time.Now().UTC().Format("2006-01-02")}
	if got := rehabBribeCost(member, ModerationRecord{RehabWard: rehabWardVIP}); got != 1 {
		t.Fatalf("vip active bribe cost = %d, want 1", got)
	}
	rowdyCost := rehabBribeCost(MemberRecord{}, ModerationRecord{RehabWard: rehabWardRowdy})
	if rowdyCost != rehabBribeBaseCost+1 {
		t.Fatalf("rowdy bribe cost = %d, want %d", rowdyCost, rehabBribeBaseCost+1)
	}
	if got := rehabBribeChance(member, 20, ModerationRecord{RehabWard: rehabWardVIP}); got <= rehabBribeBaseChance {
		t.Fatalf("active vip bribe chance = %d, want above base %d", got, rehabBribeBaseChance)
	}
}

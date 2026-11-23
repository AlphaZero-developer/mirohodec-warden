package main

import (
	"reflect"
	"testing"
	"time"
)

func TestParseDurationTokenLocalizedRussian(t *testing.T) {
	tests := []struct {
		value string
		want  time.Duration
	}{
		{value: "10м", want: 10 * time.Minute},
		{value: "2ч", want: 2 * time.Hour},
		{value: "3д", want: 72 * time.Hour},
		{value: "1н", want: 7 * 24 * time.Hour},
		{value: "45сек", want: 45 * time.Second},
	}

	for _, tt := range tests {
		got, permanent, ok, err := parseDurationToken(tt.value)
		if err != nil || !ok || permanent || got != tt.want {
			t.Fatalf("parseDurationToken(%q) = (%s, permanent=%t, ok=%t, err=%v), want %s", tt.value, got, permanent, ok, err, tt.want)
		}
	}
}

func TestParseDurationTokenLocalizedPermanent(t *testing.T) {
	got, permanent, ok, err := parseDurationToken("навсегда")
	if err != nil || !ok || !permanent || got != 0 {
		t.Fatalf("parseDurationToken(навсегда) = (%s, permanent=%t, ok=%t, err=%v), want permanent", got, permanent, ok, err)
	}
}

func TestParseOptionalUntilLocalizedTwoTokenDuration(t *testing.T) {
	before := time.Now().UTC()
	until, reason, err := parseOptionalUntil([]string{"10", "минут", "шум"}, true)
	if err != nil {
		t.Fatalf("parseOptionalUntil returned error: %v", err)
	}
	if reason != "шум" {
		t.Fatalf("reason = %q, want шум", reason)
	}
	remaining := time.Unix(until, 0).UTC().Sub(before)
	if remaining < 9*time.Minute || remaining > 11*time.Minute {
		t.Fatalf("remaining = %s, want about 10m", remaining)
	}
}

func TestMaybeDurationFirstJailArgs(t *testing.T) {
	tests := []struct {
		args []string
		want []string
	}{
		{args: []string{"30m", "@user", "reason"}, want: []string{"@user", "30m", "reason"}},
		{args: []string{"10", "минут", "@user", "reason"}, want: []string{"@user", "10", "минут", "reason"}},
	}

	for _, tt := range tests {
		got, ok := maybeDurationFirstJailArgs(tt.args)
		if !ok || !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("maybeDurationFirstJailArgs(%v) = (%v, %t), want (%v, true)", tt.args, got, ok, tt.want)
		}
	}
}

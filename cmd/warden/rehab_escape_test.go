package main

import (
	"testing"
	"time"
)

func TestIsRehabEscapeCommand(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{text: "побег из дурки", want: true},
		{text: "Побег из дурки!", want: true},
		{text: "сбежать с дурки", want: true},
		{text: "/escape", want: true},
		{text: "побег из психушки", want: true},
		{text: "побег из комнаты", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			if got := isRehabEscapeCommand(tt.text); got != tt.want {
				t.Fatalf("isRehabEscapeCommand(%q) = %t, want %t", tt.text, got, tt.want)
			}
		})
	}
}

func TestRehabEscapeCommandScope(t *testing.T) {
	scope := rehabEscapeCommandScope(-1001, 42)
	if scope.Type != "chat_member" || scope.ChatID != -1001 || scope.UserID != 42 {
		t.Fatalf("unexpected scope: %#v", scope)
	}
	commands := rehabUserCommands()
	if len(commands) != 2 || commands[0].Text != "escape" || commands[1].Text != "bribe" {
		t.Fatalf("unexpected rehab commands: %#v", commands)
	}
}

func TestIsRehabBribeCommand(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{text: "/bribe", want: true},
		{text: "подкуп санитара", want: true},
		{text: "подкупить санитара!", want: true},
		{text: "просто сообщение", want: false},
	}

	for _, tt := range tests {
		if got := isRehabBribeCommand(tt.text); got != tt.want {
			t.Fatalf("isRehabBribeCommand(%q) = %t, want %t", tt.text, got, tt.want)
		}
	}
}

func TestRehabEscapeChanceFromPenalty(t *testing.T) {
	if got := rehabEscapeChanceFromPenalty(0); got != 5 {
		t.Fatalf("chance with no penalty = %d, want 5", got)
	}
	if got := rehabEscapeChanceFromPenalty(1); got != 4 {
		t.Fatalf("chance after one success = %d, want 4", got)
	}
	if got := rehabEscapeChanceFromPenalty(99); got != 0 {
		t.Fatalf("chance floor = %d, want 0", got)
	}
}

func TestRehabEscapeCooldownDuration(t *testing.T) {
	tests := []struct {
		level int
		want  time.Duration
	}{
		{level: 0, want: time.Minute},
		{level: 1, want: time.Minute},
		{level: 3, want: 3 * time.Minute},
		{level: 99, want: 5 * time.Minute},
	}

	for _, tt := range tests {
		if got := rehabEscapeCooldownDuration(tt.level); got != tt.want {
			t.Fatalf("cooldown duration for level %d = %s, want %s", tt.level, got, tt.want)
		}
	}
}

func TestRehabEscapeCooldownDurationForRowdyWard(t *testing.T) {
	jail := ModerationRecord{RehabWard: rehabWardRowdy}
	tests := []struct {
		level int
		want  time.Duration
	}{
		{level: 1, want: 2 * time.Minute},
		{level: 2, want: 4 * time.Minute},
		{level: 3, want: 5 * time.Minute},
		{level: 99, want: 5 * time.Minute},
	}

	for _, tt := range tests {
		if got := rehabEscapeCooldownDurationForWard(tt.level, jail); got != tt.want {
			t.Fatalf("rowdy cooldown duration for level %d = %s, want %s", tt.level, got, tt.want)
		}
	}
}

func TestRehabEscapeCooldownEscalatesAndResetsPerJail(t *testing.T) {
	app := &App{}
	jail := ModerationRecord{SinceUnix: 100}
	now := time.Unix(1000, 0).UTC()

	for i, want := range []time.Duration{
		time.Minute,
		2 * time.Minute,
		3 * time.Minute,
		4 * time.Minute,
		5 * time.Minute,
		5 * time.Minute,
	} {
		got := app.recordRehabEscapeCooldown(-1001, 42, jail, now)
		if got != want {
			t.Fatalf("failed attempt %d cooldown = %s, want %s", i+1, got, want)
		}
		remaining := app.rehabEscapeCooldownRemaining(-1001, 42, jail, now.Add(10*time.Second))
		if remaining <= 0 || remaining > want {
			t.Fatalf("failed attempt %d remaining = %s, want within %s", i+1, remaining, want)
		}
		now = now.Add(want + time.Second)
	}

	newJail := ModerationRecord{SinceUnix: 200}
	if got := app.recordRehabEscapeCooldown(-1001, 42, newJail, now); got != time.Minute {
		t.Fatalf("new jail cooldown = %s, want %s", got, time.Minute)
	}
}

func TestRehabEscapeCooldownCheckDoesNotEscalate(t *testing.T) {
	app := &App{}
	jail := ModerationRecord{StartedAtUnix: 100}
	now := time.Unix(1000, 0).UTC()

	if got := app.recordRehabEscapeCooldown(-1001, 42, jail, now); got != time.Minute {
		t.Fatalf("first cooldown = %s, want %s", got, time.Minute)
	}
	for i := 0; i < 3; i++ {
		remaining := app.rehabEscapeCooldownRemaining(-1001, 42, jail, now.Add(10*time.Second))
		if remaining != 50*time.Second {
			t.Fatalf("remaining after check %d = %s, want 50s", i+1, remaining)
		}
	}
	if got := app.recordRehabEscapeCooldown(-1001, 42, jail, now.Add(time.Minute+time.Second)); got != 2*time.Minute {
		t.Fatalf("second failed cooldown = %s, want %s", got, 2*time.Minute)
	}
}

func TestRehabEscapeCooldownClear(t *testing.T) {
	app := &App{}
	jail := ModerationRecord{SinceUnix: 100}
	now := time.Unix(1000, 0).UTC()

	app.recordRehabEscapeCooldown(-1001, 42, jail, now)
	app.clearRehabEscapeCooldown(-1001, 42)
	if remaining := app.rehabEscapeCooldownRemaining(-1001, 42, jail, now.Add(10*time.Second)); remaining != 0 {
		t.Fatalf("remaining after clear = %s, want 0", remaining)
	}
}

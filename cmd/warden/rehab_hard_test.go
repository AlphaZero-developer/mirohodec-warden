package main

import (
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func TestRehabReminderSlot(t *testing.T) {
	cfg := RehabConfig{
		Timezone:       "Europe/Moscow",
		ReminderTimes:  []string{"09:00", "14:00", "20:00"},
		ReminderWindow: "15m",
	}

	tests := []struct {
		name     string
		nowUTC   time.Time
		wantSlot string
		wantOK   bool
	}{
		{
			name:     "morning exact",
			nowUTC:   time.Date(2026, 4, 27, 6, 0, 0, 0, time.UTC),
			wantSlot: "2026-04-27 09:00",
			wantOK:   true,
		},
		{
			name:     "inside window",
			nowUTC:   time.Date(2026, 4, 27, 11, 14, 59, 0, time.UTC),
			wantSlot: "2026-04-27 14:00",
			wantOK:   true,
		},
		{
			name:   "outside window",
			nowUTC: time.Date(2026, 4, 27, 11, 15, 0, 0, time.UTC),
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSlot, gotOK := rehabReminderSlot(cfg, tt.nowUTC)
			if gotOK != tt.wantOK || gotSlot != tt.wantSlot {
				t.Fatalf("slot = %q, %t; want %q, %t", gotSlot, gotOK, tt.wantSlot, tt.wantOK)
			}
		})
	}
}

func TestChatMemberBecamePresent(t *testing.T) {
	user := &tele.User{ID: 42}

	tests := []struct {
		name string
		old  *tele.ChatMember
		next *tele.ChatMember
		want bool
	}{
		{
			name: "left to member",
			old:  &tele.ChatMember{User: user, Role: tele.Left},
			next: &tele.ChatMember{User: user, Role: tele.Member},
			want: true,
		},
		{
			name: "member to admin is not join",
			old:  &tele.ChatMember{User: user, Role: tele.Member},
			next: &tele.ChatMember{User: user, Role: tele.Administrator},
			want: false,
		},
		{
			name: "restricted non-member to restricted member",
			old:  &tele.ChatMember{User: user, Role: tele.Restricted, Member: false},
			next: &tele.ChatMember{User: user, Role: tele.Restricted, Member: true},
			want: true,
		},
		{
			name: "restricted non-member is not present",
			old:  &tele.ChatMember{User: user, Role: tele.Left},
			next: &tele.ChatMember{User: user, Role: tele.Restricted, Member: false},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chatMemberBecamePresent(tt.old, tt.next); got != tt.want {
				t.Fatalf("chatMemberBecamePresent() = %t, want %t", got, tt.want)
			}
		})
	}
}

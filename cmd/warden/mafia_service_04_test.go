package main

import (
	"strings"
	"testing"
	"time"
)

func TestMafiaPostGameBlocksAreSeparatedAndFinal(t *testing.T) {
	room := MafiaRoom{
		ID:       "MF-FINAL",
		Status:   MafiaStatusFinished,
		Winner:   "mafia",
		Settings: defaultMafiaRoomSettings(),
		Players: []MafiaPlayer{
			{TelegramID: 9101, DisplayName: "Джон", Role: MafiaRoleDetective, Alive: false},
			{TelegramID: 9102, DisplayName: "Арчибальд", Role: MafiaRoleMafia, Alive: true},
		},
	}
	achievements := mafiaPostGameAchievementsText(room)
	if !strings.Contains(achievements, "--- ДОСТИЖЕНИЯ ---") || strings.Contains(achievements, "--- ФИНАЛ ---") {
		t.Fatalf("achievements must be separated from final:\n%s", achievements)
	}
	fallback := "Победитель: мафия.\n\nПартия завершилась."
	badReport := "Мафия пока ведёт, но игра ещё не закончена. Следующий день наступил. Кто будет следующей жертвой?"
	if got := sanitizeMafiaPostGameReport(badReport, fallback); got != fallback {
		t.Fatalf("bad post-game report was not replaced:\n%s", got)
	}
}

func TestMafiaVotingMarkupCarriesPhaseKey(t *testing.T) {
	room := MafiaRoom{
		ID:             "MF-VOTE",
		Status:         MafiaStatusVoting,
		DayNumber:      2,
		PhaseStartedAt: 1715000000,
		Players: []MafiaPlayer{
			{TelegramID: 9201, DisplayName: "Джон", Alive: true},
			{TelegramID: 9202, DisplayName: "София", Alive: true},
		},
	}
	markup := mafiaVotingMarkup(room, false)
	if markup == nil || len(markup.InlineKeyboard) < 3 {
		t.Fatalf("voting markup is empty")
	}
	wantVote := "mafia:vote:MF-VOTE:1715000000:9201"
	if got := markup.InlineKeyboard[0][0].Data; got != wantVote {
		t.Fatalf("vote button data = %q, want %q", got, wantVote)
	}
	wantSkip := "mafia:voteskip:MF-VOTE:1715000000"
	if got := markup.InlineKeyboard[2][0].Data; got != wantSkip {
		t.Fatalf("skip vote button data = %q, want %q", got, wantSkip)
	}
}

func TestMafiaPhaseTimerText(t *testing.T) {
	now := time.Unix(1715000030, 0).UTC()
	room := MafiaRoom{
		ID:             "MF-TIMER",
		Status:         MafiaStatusDayDiscussion,
		DayNumber:      1,
		PhaseStartedAt: 1715000000,
		Settings:       defaultMafiaRoomSettings(),
	}
	text := mafiaPhaseTimerText(room, now)
	if !strings.Contains(text, "До начала голосования: 2:30") {
		t.Fatalf("day timer text = %q", text)
	}

	room.Status = MafiaStatusNight
	text = mafiaPhaseTimerText(room, now)
	if !strings.Contains(text, "До конца ночи: 1:00") {
		t.Fatalf("night timer text = %q", text)
	}

	room.Status = MafiaStatusVoting
	text = mafiaPhaseTimerText(room, now)
	if !strings.Contains(text, "До конца голосования: 0:30") {
		t.Fatalf("voting timer text = %q", text)
	}
}

func TestMafiaEliminatedDMTextIsLoudAndClear(t *testing.T) {
	room := MafiaRoom{ID: "MF-DEAD"}
	text := mafiaEliminatedDMText(room, "Джон Кеннеди", "Этой ночью мафия выбрала вас жертвой.")
	for _, want := range []string{
		"ВЫ ВЫБЫЛИ ИЗ ПАРТИИ",
		"вы мертвы",
		"мафия выбрала вас жертвой",
		"писать в общий чат",
		"голосовать",
		"делать ночные действия",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("eliminated DM missing %q:\n%s", want, text)
		}
	}
}

func TestMafiaPublicTalkBlockedAtNight(t *testing.T) {
	tests := []struct {
		status MafiaRoomStatus
		want   bool
	}{
		{status: MafiaStatusNight, want: true},
		{status: MafiaStatusRoleDistribution, want: true},
		{status: MafiaStatusDayDiscussion, want: false},
		{status: MafiaStatusVoting, want: false},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := mafiaRoomStatusBlocksPublicTalk(tt.status); got != tt.want {
				t.Fatalf("mafiaRoomStatusBlocksPublicTalk(%s) = %t, want %t", tt.status, got, tt.want)
			}
		})
	}
}

func TestMafiaGroupSkipPhaseText(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{text: "Скип фазы", want: true},
		{text: "⏭ Скип фазы", want: true},
		{text: "  скипнуть   фазу  ", want: true},
		{text: "пропустить фазу", want: true},
		{text: "скип", want: false},
		{text: "я хочу скип фазы", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			if got := mafiaIsGroupSkipPhaseText(tt.text); got != tt.want {
				t.Fatalf("mafiaIsGroupSkipPhaseText(%q) = %t, want %t", tt.text, got, tt.want)
			}
		})
	}
}

func TestMafiaPublicPhaseSkipAllowedOnlyForPublicDayPhases(t *testing.T) {
	tests := []struct {
		status MafiaRoomStatus
		want   bool
	}{
		{status: MafiaStatusDayDiscussion, want: true},
		{status: MafiaStatusVoting, want: true},
		{status: MafiaStatusNight, want: false},
		{status: MafiaStatusRoleDistribution, want: false},
		{status: MafiaStatusLobby, want: false},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := mafiaRoomAllowsPublicPhaseSkip(tt.status); got != tt.want {
				t.Fatalf("mafiaRoomAllowsPublicPhaseSkip(%s) = %t, want %t", tt.status, got, tt.want)
			}
		})
	}
}

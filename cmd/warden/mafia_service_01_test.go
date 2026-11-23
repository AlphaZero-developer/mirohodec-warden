package main

import (
	"strings"
	"testing"
	"time"
)

func TestRunMafiaFullTestPasses(t *testing.T) {
	report := RunMafiaFullTest()
	if !strings.Contains(report.Text, "Результат: механика работает.") {
		t.Fatalf("full mafia test did not pass:\n%s", report.Text)
	}
}

func TestMafiaRoleRevealDetection(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
		role MafiaRole
	}{
		{name: "direct mafia", text: "я мафия", want: true, role: MafiaRoleMafia},
		{name: "night heal", text: "я лечил его ночью", want: true, role: MafiaRoleDoctor},
		{name: "suspicion", text: "я думаю, он мафия", want: false},
		{name: "denial", text: "я не мафия", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectMafiaRoleReveal(tt.text)
			if got.IsRoleReveal != tt.want {
				t.Fatalf("DetectMafiaRoleReveal(%q).IsRoleReveal = %t, want %t", tt.text, got.IsRoleReveal, tt.want)
			}
			if tt.want && got.DetectedRole != tt.role {
				t.Fatalf("DetectMafiaRoleReveal(%q).DetectedRole = %q, want %q", tt.text, got.DetectedRole, tt.role)
			}
		})
	}
}

func TestMafiaCriticalLeakDetectionOnlyProofLikeLeaks(t *testing.T) {
	tests := []struct {
		name string
		text string
		want MafiaRuleViolationType
	}{
		{name: "ally with me", text: "Mira со мной мафия", want: MafiaRuleViolationAllyLeak},
		{name: "we are mafia", text: "мы с Shadow мафы", want: MafiaRuleViolationAllyLeak},
		{name: "shadow channel decision", text: "в теневом канале решили убить Alex", want: MafiaRuleViolationGameBreakingLeak},
		{name: "ordinary accusation", text: "Mira тоже мафия", want: MafiaRuleViolationNone},
		{name: "ordinary suspicion", text: "я думаю, Mira мафия", want: MafiaRuleViolationNone},
		{name: "ordinary group suspicion", text: "мы с Иваном думаем что Mira мафия", want: MafiaRuleViolationNone},
		{name: "ambiguous second mafia accusation", text: "второй маф София", want: MafiaRuleViolationNone},
		{name: "self claim with second mafia", text: "я мафия, и второй маф София", want: MafiaRuleViolationAllyLeak},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectMafiaCriticalLeak(tt.text)
			if got.ViolationType != tt.want {
				t.Fatalf("DetectMafiaCriticalLeak(%q).ViolationType = %q, want %q", tt.text, got.ViolationType, tt.want)
			}
		})
	}
}

func TestMafiaShadowChatRecipientsOnlyMafia(t *testing.T) {
	store := newMafiaTestStore(t)
	service, room := startMafiaTestRoom(t, store, 8)
	mafia := mafiaPlayers(room)
	if len(mafia) != 2 {
		t.Fatalf("expected 2 mafia players, got %d", len(mafia))
	}

	room, _, recipients, err := service.SendShadowMessage(room.ID, mafia[0].TelegramID, "Убираем Shadow.", time.Now().UTC())
	if err != nil {
		t.Fatalf("SendShadowMessage: %v", err)
	}
	if len(recipients) != len(mafia) {
		t.Fatalf("recipients = %v, want %d mafia recipients", recipients, len(mafia))
	}
	for _, recipientID := range recipients {
		idx := mafiaRoomPlayerIndex(room, recipientID)
		if idx < 0 || room.Players[idx].Role != MafiaRoleMafia {
			t.Fatalf("non-mafia recipient leaked into shadow channel: %d", recipientID)
		}
	}
}

func TestMafiaFinishUpdatesStats(t *testing.T) {
	store := newMafiaTestStore(t)
	service, room := startMafiaTestRoom(t, store, 5)
	mafia := mafiaPlayers(room)[0]

	room, err := service.PunishRoleReveal(room.ID, mafia.TelegramID, time.Now().UTC())
	if err != nil {
		t.Fatalf("PunishRoleReveal: %v", err)
	}
	if room.Status != MafiaStatusFinished || room.Winner != "civilian" {
		t.Fatalf("room status=%q winner=%q, want finished/civilian", room.Status, room.Winner)
	}

	mafiaUser, ok := store.MafiaUser(mafia.TelegramID)
	if !ok {
		t.Fatalf("mafia user not found")
	}
	if mafiaUser.Stats.Games != 1 || mafiaUser.Stats.Deaths != 1 || mafiaUser.Stats.RoleReveals != 1 {
		t.Fatalf("unexpected mafia stats: %#v", mafiaUser.Stats)
	}
	for _, player := range room.Players {
		if player.Role == MafiaRoleMafia {
			continue
		}
		user, ok := store.MafiaUser(player.TelegramID)
		if !ok {
			t.Fatalf("civilian user %d not found", player.TelegramID)
		}
		if user.Stats.Games != 1 || user.Stats.Wins != 1 || user.Stats.CivilianWins != 1 {
			t.Fatalf("unexpected civilian stats for %d: %#v", player.TelegramID, user.Stats)
		}
	}
}

func TestNormalizeMafiaCallbackData(t *testing.T) {
	tests := map[string]string{
		"create_mafia_room":                   "mafia:create",
		"mafia_full_test":                     "mafia:test",
		"mafia_night_kill_MF-4821_123":        "mafia:kill:MF-4821:123",
		"mafia_doctor_heal_MF-4821_123":       "mafia:heal:MF-4821:123",
		"mafia_detective_check_MF-4821_123":   "mafia:check:MF-4821:123",
		"mafia_vote_MF-4821_123":              "mafia:vote:MF-4821:123",
		"mafia_vote_skip_MF-4821":             "mafia:voteskip:MF-4821",
		"open_mafia_shadow_chat_MF-4821":      "mafia:shadow:MF-4821",
		"open_mafia_players_MF-4821":          "mafia:players:MF-4821",
		"mafia_invite_menu_MF-4821":           "mafia:invite_menu:MF-4821",
		"mafia_recruitment_open_MF-4821":      "mafia:recruit_open:MF-4821",
		"mafia_join_request_MF-4821":          "mafia:join_request:MF-4821",
		"mafia_request_accept_MF-4821_123":    "mafia:request_accept:MF-4821:123",
		"mafia_ready_queue_auto_fill_MF-4821": "mafia:queue_autofill:MF-4821",
		"mafia:vote:MF-4821:123":              "mafia:vote:MF-4821:123",
	}
	for input, want := range tests {
		if got := normalizeMafiaCallbackData(input); got != want {
			t.Fatalf("normalizeMafiaCallbackData(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseMafiaVoteCallbackData(t *testing.T) {
	tests := []struct {
		name       string
		data       string
		wantRoom   string
		wantPhase  string
		wantTarget string
		wantOK     bool
	}{
		{name: "old vote", data: "mafia:vote:MF-1:42", wantRoom: "MF-1", wantTarget: "42", wantOK: true},
		{name: "new vote", data: "mafia:vote:MF-1:1715000000:42", wantRoom: "MF-1", wantPhase: "1715000000", wantTarget: "42", wantOK: true},
		{name: "old skip", data: "mafia:voteskip:MF-1", wantRoom: "MF-1", wantTarget: "skip", wantOK: true},
		{name: "new skip", data: "mafia:voteskip:MF-1:1715000000", wantRoom: "MF-1", wantPhase: "1715000000", wantTarget: "skip", wantOK: true},
		{name: "bad", data: "mafia:vote:MF-1:phase:42:extra", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseMafiaVoteCallbackData(tt.data)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got.roomID != tt.wantRoom || got.phaseKey != tt.wantPhase || got.target != tt.wantTarget {
				t.Fatalf("payload = %#v, want room=%q phase=%q target=%q", got, tt.wantRoom, tt.wantPhase, tt.wantTarget)
			}
		})
	}
}

func TestMafiaVoteCallbacksNeedManualResponse(t *testing.T) {
	for _, data := range []string{
		"mafia:vote:MF-1:1715000000:42",
		"mafia:voteskip:MF-1:1715000000",
	} {
		if !mafiaCallbackNeedsManualResponse(data) {
			t.Fatalf("%s should need manual response", data)
		}
	}
	if mafiaCallbackNeedsManualResponse("mafia:status:MF-1") {
		t.Fatalf("status callback should use default response")
	}
}

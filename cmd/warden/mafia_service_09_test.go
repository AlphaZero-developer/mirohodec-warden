package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func TestMafiaPhaseSkipVoteAdvancesByMajority(t *testing.T) {
	store := newMafiaTestStore(t)
	service := NewMafiaService(store)
	now := time.Now().UTC()
	room := MafiaRoom{
		ID:          "SKIP-DAY",
		OwnerID:     9650,
		GroupChatID: -1009650,
		Status:      MafiaStatusDayDiscussion,
		Settings:    defaultMafiaRoomSettings(),
		DayNumber:   1,
		Players: []MafiaPlayer{
			{TelegramID: 9650, DisplayName: "Owner", Role: MafiaRoleCivilian, Alive: true},
			{TelegramID: 9651, DisplayName: "Mafia", Role: MafiaRoleMafia, Alive: true},
			{TelegramID: 9652, DisplayName: "Civilian", Role: MafiaRoleCivilian, Alive: true},
			{TelegramID: mafiaTestBotBaseID - 25, DisplayName: "Bot", Role: MafiaRoleCivilian, Alive: true, IsTestBot: true},
		},
		CreatedAt: now.Unix(),
		UpdatedAt: now.Unix(),
	}
	if err := store.SaveMafiaRoom(room); err != nil {
		t.Fatalf("SaveMafiaRoom: %v", err)
	}
	updated, votes, needed, reached, err := service.VotePhaseSkip(room.ID, 9650)
	if err != nil {
		t.Fatalf("VotePhaseSkip first: %v", err)
	}
	if votes != 1 || needed != 2 || reached {
		t.Fatalf("first skip vote = votes %d needed %d reached %v, want 1/2 false", votes, needed, reached)
	}
	if updated.UpdatedAt != room.UpdatedAt {
		t.Fatalf("skip vote must not reset phase timer")
	}
	updated, votes, needed, reached, err = service.VotePhaseSkip(room.ID, 9651)
	if err != nil {
		t.Fatalf("VotePhaseSkip second: %v", err)
	}
	if votes != 2 || needed != 2 || !reached {
		t.Fatalf("second skip vote = votes %d needed %d reached %v, want 2/2 true", votes, needed, reached)
	}
	app := &App{store: store}
	if err := app.advanceMafiaRoomAsLaya(room.ID, now); err != nil {
		t.Fatalf("advanceMafiaRoomAsLaya: %v", err)
	}
	advanced, ok := store.MafiaRoom(room.ID)
	if !ok {
		t.Fatalf("room not found")
	}
	if advanced.Status != MafiaStatusVoting {
		t.Fatalf("status=%s, want voting", advanced.Status)
	}
	if len(advanced.PhaseSkipVotes) != 0 {
		t.Fatalf("phase skip votes must be cleared on phase change: %#v", advanced.PhaseSkipVotes)
	}
}

func TestMafiaActionDoesNotExtendPhaseTimer(t *testing.T) {
	store := newMafiaTestStore(t)
	service := NewMafiaService(store)
	now := time.Now().UTC()
	phaseStart := now.Add(-2 * time.Minute)
	room := MafiaRoom{
		ID:             "TIMER-NIGHT",
		OwnerID:        9660,
		GroupChatID:    -1009660,
		Status:         MafiaStatusNight,
		Settings:       defaultMafiaRoomSettings(),
		DayNumber:      1,
		PhaseStartedAt: phaseStart.Unix(),
		Players: []MafiaPlayer{
			{TelegramID: 9660, DisplayName: "Owner", Role: MafiaRoleCivilian, Alive: true},
			{TelegramID: 9661, DisplayName: "Mafia", Role: MafiaRoleMafia, Alive: true},
			{TelegramID: 9662, DisplayName: "Civilian", Role: MafiaRoleCivilian, Alive: true},
			{TelegramID: 9663, DisplayName: "Doctor", Role: MafiaRoleDoctor, Alive: true},
		},
		NightActions: MafiaNightActions{MafiaVotes: map[string]int64{}, DetectiveChecks: map[string]int64{}, DetectiveResults: map[string]bool{}},
		CreatedAt:    phaseStart.Unix(),
		UpdatedAt:    phaseStart.Unix(),
	}
	if err := store.SaveMafiaRoom(room); err != nil {
		t.Fatalf("SaveMafiaRoom: %v", err)
	}
	updated, err := service.MafiaKill(room.ID, 9661, 9662, now)
	if err != nil {
		t.Fatalf("MafiaKill: %v", err)
	}
	if updated.UpdatedAt == updated.PhaseStartedAt {
		t.Fatalf("test setup expected action to update UpdatedAt separately")
	}
	if !mafiaPhaseDue(updated, now, time.Duration(normalizeMafiaSettings(updated.Settings).NightActionSeconds)*time.Second) {
		t.Fatalf("phase timer must use PhaseStartedAt, not UpdatedAt")
	}
}

func TestMafiaAutoHostResolvesNightWithTestBots(t *testing.T) {
	store := newMafiaTestStore(t)
	now := time.Now().UTC()
	room := MafiaRoom{
		ID:          "AUTO-NIGHT",
		OwnerID:     9700,
		GroupChatID: -1009700,
		Status:      MafiaStatusNight,
		Settings:    defaultMafiaRoomSettings(),
		DayNumber:   1,
		Players: []MafiaPlayer{
			{TelegramID: 9700, DisplayName: "Owner", Role: MafiaRoleCivilian, Alive: false},
			{TelegramID: mafiaTestBotBaseID - 30, DisplayName: "Bot Mafia", Role: MafiaRoleMafia, Alive: true, IsTestBot: true},
			{TelegramID: mafiaTestBotBaseID - 31, DisplayName: "Bot One", Role: MafiaRoleCivilian, Alive: true, IsTestBot: true},
			{TelegramID: mafiaTestBotBaseID - 32, DisplayName: "Bot Two", Role: MafiaRoleCivilian, Alive: true, IsTestBot: true},
			{TelegramID: mafiaTestBotBaseID - 33, DisplayName: "Bot Three", Role: MafiaRoleCivilian, Alive: true, IsTestBot: true},
		},
		NightActions: MafiaNightActions{MafiaVotes: map[string]int64{}, DetectiveChecks: map[string]int64{}, DetectiveResults: map[string]bool{}},
		CreatedAt:    now.Add(-10 * time.Minute).Unix(),
		UpdatedAt:    now.Add(-10 * time.Minute).Unix(),
	}
	if err := store.SaveMafiaRoom(room); err != nil {
		t.Fatalf("SaveMafiaRoom: %v", err)
	}
	app := &App{store: store}
	if err := app.advanceMafiaRoomAsLaya(room.ID, now); err != nil {
		t.Fatalf("advanceMafiaRoomAsLaya: %v", err)
	}
	updated, ok := store.MafiaRoom(room.ID)
	if !ok {
		t.Fatalf("room not found")
	}
	if updated.Status == MafiaStatusNight {
		t.Fatalf("auto host should move the game out of night")
	}
	killedCivilian := false
	for _, player := range updated.Players {
		if player.IsTestBot && !isMafiaTeam(player.Role) && !player.Alive {
			killedCivilian = true
			break
		}
	}
	if !killedCivilian {
		t.Fatalf("test bot mafia should kill a non-mafia player: %#v", updated.Players)
	}
}

func mafiaReplyMarkupContains(markup *tele.ReplyMarkup, text string) bool {
	if markup == nil {
		return false
	}
	for _, row := range markup.ReplyKeyboard {
		for _, button := range row {
			if strings.Contains(button.Text, text) {
				return true
			}
		}
	}
	return false
}

func newMafiaTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return store
}

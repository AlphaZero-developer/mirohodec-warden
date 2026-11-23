package main

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func TestMafiaResolveVotingResultKeepsVoteProtocol(t *testing.T) {
	store := newMafiaTestStore(t)
	service := NewMafiaService(store)
	now := time.Now().UTC()
	room := MafiaRoom{
		ID:          "VOTE-RESULT",
		OwnerID:     9031,
		GroupChatID: -1009031,
		Status:      MafiaStatusVoting,
		Settings:    defaultMafiaRoomSettings(),
		DayNumber:   1,
		Players: []MafiaPlayer{
			{TelegramID: 9031, DisplayName: "Арчибальд", Role: MafiaRoleCivilian, Alive: true},
			{TelegramID: 9032, DisplayName: "София", Role: MafiaRoleMafia, Alive: true},
			{TelegramID: 9033, DisplayName: "Марта", Role: MafiaRoleCivilian, Alive: true},
			{TelegramID: 9034, DisplayName: "Григорий", Role: MafiaRoleDoctor, Alive: true},
		},
		Votes: map[string]string{
			idKey(9031): idKey(9032),
			idKey(9032): idKey(9031),
			idKey(9033): idKey(9032),
			idKey(9034): "skip",
		},
		CreatedAt: now.Unix(),
		UpdatedAt: now.Unix(),
	}
	if err := store.SaveMafiaRoom(room); err != nil {
		t.Fatalf("SaveMafiaRoom: %v", err)
	}
	updated, result, err := service.ResolveVoting(room.ID, now)
	if err != nil {
		t.Fatalf("ResolveVoting: %v", err)
	}
	if result.ExecutedID != 9032 {
		t.Fatalf("executed=%d, want 9032", result.ExecutedID)
	}
	if len(result.Votes) != len(room.Votes) || result.Votes[idKey(9032)] != idKey(9031) {
		t.Fatalf("result votes were not preserved: %#v", result.Votes)
	}
	text := mafiaVotingResultText(updated, result)
	for _, want := range []string{
		"Прогресс голосования: 4/4",
		"София → Арчибальд",
		"Арчибальд → София",
		"Марта → София",
		"Григорий → Пропуск",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("voting result text missing %q:\n%s", want, text)
		}
	}
}

func TestMafiaCanStartWithFourPlayers(t *testing.T) {
	store := newMafiaTestStore(t)
	_, room := startMafiaTestRoom(t, store, 4)
	if len(room.Players) != 4 {
		t.Fatalf("players=%d, want 4", len(room.Players))
	}
	roles := map[MafiaRole]int{}
	for _, player := range room.Players {
		roles[player.Role]++
	}
	if roles[MafiaRoleMafia] != 1 || roles[MafiaRoleDoctor] != 1 || roles[MafiaRoleDetective] != 1 || roles[MafiaRoleCivilian] != 1 {
		t.Fatalf("unexpected roles for 4-player room: %#v", roles)
	}
}

func TestMafiaTestBotsFillRoomToMinimum(t *testing.T) {
	store := newMafiaTestStore(t)
	service := NewMafiaService(store)
	service.rng = rand.New(rand.NewSource(11))
	now := time.Now().UTC()
	owner := &tele.User{ID: 9100, FirstName: "Owner"}
	if _, err := service.RegisterUser(owner, now); err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}
	room, err := service.CreateRoom(owner, now)
	if err != nil {
		t.Fatalf("CreateRoom: %v", err)
	}
	room, added, err := service.FillTestBotsToMinimum(room.ID, owner.ID, now)
	if err != nil {
		t.Fatalf("FillTestBotsToMinimum: %v", err)
	}
	if len(added) != 3 || len(room.Players) != 4 {
		t.Fatalf("added=%d players=%d, want 3/4", len(added), len(room.Players))
	}
	for _, player := range added {
		if !player.IsTestBot || player.TelegramID >= 0 {
			t.Fatalf("invalid test bot: %#v", player)
		}
		if _, ok := store.MafiaUser(player.TelegramID); ok {
			t.Fatalf("test bot %d must not be registered as a real user", player.TelegramID)
		}
	}
	if _, err := service.BindGroup(room.ID, owner.ID, -1009100, now); err != nil {
		t.Fatalf("BindGroup: %v", err)
	}
	room, err = service.StartGame(room.ID, owner.ID, now)
	if err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	if room.Status != MafiaStatusNight {
		t.Fatalf("status=%s, want night", room.Status)
	}
}

func TestMafiaTestBotsResolveNightAutomatically(t *testing.T) {
	store := newMafiaTestStore(t)
	service := NewMafiaService(store)
	service.rng = rand.New(rand.NewSource(21))
	now := time.Now().UTC()
	room := MafiaRoom{
		ID:          "BOT-NIGHT",
		OwnerID:     9200,
		GroupChatID: -1009200,
		Status:      MafiaStatusNight,
		Settings:    defaultMafiaRoomSettings(),
		DayNumber:   1,
		Players: []MafiaPlayer{
			{TelegramID: 9200, DisplayName: "Owner", Role: MafiaRoleCivilian, Alive: false},
			{TelegramID: mafiaTestBotBaseID - 1, DisplayName: "Bot Mafia", Role: MafiaRoleMafia, Alive: true, IsTestBot: true},
			{TelegramID: mafiaTestBotBaseID - 2, DisplayName: "Bot One", Role: MafiaRoleCivilian, Alive: true, IsTestBot: true},
			{TelegramID: mafiaTestBotBaseID - 3, DisplayName: "Bot Two", Role: MafiaRoleCivilian, Alive: true, IsTestBot: true},
		},
		NightActions: MafiaNightActions{MafiaVotes: map[string]int64{}, DetectiveChecks: map[string]int64{}, DetectiveResults: map[string]bool{}},
		CreatedAt:    now.Unix(),
		UpdatedAt:    now.Unix(),
	}
	if err := store.SaveMafiaRoom(room); err != nil {
		t.Fatalf("SaveMafiaRoom: %v", err)
	}
	updated, result, err := service.ResolveNight(room.ID, now)
	if err != nil {
		t.Fatalf("ResolveNight: %v", err)
	}
	if result.KilledID == 0 || result.Saved || result.Quiet {
		t.Fatalf("test bot mafia should kill automatically, result=%#v", result)
	}
	if updated.Status != MafiaStatusFinished && updated.Status != MafiaStatusDayDiscussion {
		t.Fatalf("status=%s, want day or finished", updated.Status)
	}
}

func TestMafiaTestBotsVoteAutomatically(t *testing.T) {
	store := newMafiaTestStore(t)
	service := NewMafiaService(store)
	service.rng = rand.New(rand.NewSource(31))
	now := time.Now().UTC()
	room := MafiaRoom{
		ID:          "BOT-VOTE",
		OwnerID:     9300,
		GroupChatID: -1009300,
		Status:      MafiaStatusDayDiscussion,
		Settings:    defaultMafiaRoomSettings(),
		DayNumber:   1,
		Players: []MafiaPlayer{
			{TelegramID: 9300, DisplayName: "Owner", Role: MafiaRoleCivilian, Alive: true},
			{TelegramID: mafiaTestBotBaseID - 10, DisplayName: "Bot Mafia", Role: MafiaRoleMafia, Alive: true, IsTestBot: true},
			{TelegramID: mafiaTestBotBaseID - 11, DisplayName: "Bot One", Role: MafiaRoleCivilian, Alive: true, IsTestBot: true},
			{TelegramID: mafiaTestBotBaseID - 12, DisplayName: "Bot Two", Role: MafiaRoleCivilian, Alive: true, IsTestBot: true},
		},
		CreatedAt: now.Unix(),
		UpdatedAt: now.Unix(),
	}
	if err := store.SaveMafiaRoom(room); err != nil {
		t.Fatalf("SaveMafiaRoom: %v", err)
	}
	updated, err := service.StartVoting(room.ID, now)
	if err != nil {
		t.Fatalf("StartVoting: %v", err)
	}
	for _, player := range updated.Players {
		if !player.IsTestBot || !player.Alive {
			continue
		}
		if got := updated.Votes[idKey(player.TelegramID)]; got == "" {
			t.Fatalf("test bot %d did not vote: %#v", player.TelegramID, updated.Votes)
		}
	}
}

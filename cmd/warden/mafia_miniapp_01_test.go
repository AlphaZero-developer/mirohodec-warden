package main

import (
	"context"
	"errors"
	wardenweb "mirohodec-warden/internal/web"
	"path/filepath"
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func TestMafiaMiniAppStateDoesNotLeakRoles(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	owner := &tele.User{ID: 100, FirstName: "Host"}
	civilian := &tele.User{ID: 101, FirstName: "Civil"}
	mafia := &tele.User{ID: 102, FirstName: "Shade"}
	for _, user := range []*tele.User{owner, civilian, mafia} {
		if _, err := store.RegisterMafiaUser(user, time.Now().UTC()); err != nil {
			t.Fatalf("RegisterMafiaUser: %v", err)
		}
	}
	room := MafiaRoom{
		ID:           "MF-LEAK",
		OwnerID:      owner.ID,
		Status:       MafiaStatusDayDiscussion,
		DayNumber:    1,
		Settings:     defaultMafiaRoomSettings(),
		CreatedAt:    time.Now().UTC().Unix(),
		UpdatedAt:    time.Now().UTC().Unix(),
		Players:      []MafiaPlayer{{TelegramID: owner.ID, DisplayName: "Host", Role: MafiaRoleDetective, Alive: true}, {TelegramID: civilian.ID, DisplayName: "Civil", Role: MafiaRoleCivilian, Alive: true}, {TelegramID: mafia.ID, DisplayName: "Shade", Role: MafiaRoleMafia, Alive: true}},
		Votes:        map[string]string{},
		NightActions: MafiaNightActions{MafiaVotes: map[string]int64{}, DetectiveChecks: map[string]int64{}, DetectiveResults: map[string]bool{}},
	}
	if err := store.SaveMafiaRoom(room); err != nil {
		t.Fatalf("SaveMafiaRoom: %v", err)
	}
	service := webPanelService{app: &App{store: store}}
	state, err := service.mafiaMiniStateForPlayer(room.ID, civilian.ID)
	if err != nil {
		t.Fatalf("mafiaMiniStateForPlayer: %v", err)
	}
	for _, player := range state.Players {
		if player.ID == mafia.ID && player.Role != "" {
			t.Fatalf("civilian sees mafia role: %+v", player)
		}
	}
	for _, event := range state.Log {
		if event.Kind == "host" {
			t.Fatalf("player should not receive host log events: %+v", state.Log)
		}
	}
	ownerState, err := service.mafiaMiniStateForPlayer(room.ID, owner.ID)
	if err != nil {
		t.Fatalf("owner state: %v", err)
	}
	if ownerState.Me.IsHost {
		t.Fatalf("room owner should not get host powers unless lobby setting enables them: %+v", ownerState.Me)
	}
	if ownerState.Me.Role != string(MafiaRoleDetective) {
		t.Fatalf("owner player should still see own role: %+v", ownerState.Me)
	}
	if ownerState.Action.Targets == nil {
		t.Fatalf("empty action must encode targets as [] instead of null: %+v", ownerState.Action)
	}
	seenHostChannel := false
	for _, channel := range ownerState.Channels {
		if channel.ID == mafiaMiniChannelHost {
			seenHostChannel = true
		}
	}
	if seenHostChannel {
		t.Fatalf("owner should not receive host channel while host powers are disabled: %+v", ownerState.Channels)
	}
	seenMafia := false
	for _, player := range ownerState.Players {
		if player.ID == mafia.ID && player.Role == string(MafiaRoleMafia) {
			seenMafia = true
		}
	}
	if seenMafia {
		t.Fatalf("owner should not see all roles while host powers are disabled: %+v", ownerState.Players)
	}

	room.Settings.EnableHostPowers = true
	if err := store.SaveMafiaRoom(room); err != nil {
		t.Fatalf("SaveMafiaRoom with host powers: %v", err)
	}
	hostState, err := service.mafiaMiniStateForPlayer(room.ID, owner.ID)
	if err != nil {
		t.Fatalf("owner state with host powers: %v", err)
	}
	if !hostState.Me.IsHost {
		t.Fatalf("room owner should become host after enabling host powers: %+v", hostState.Me)
	}
	seenHostChannel = false
	for _, channel := range hostState.Channels {
		if channel.ID == mafiaMiniChannelHost {
			seenHostChannel = true
		}
	}
	if !seenHostChannel {
		t.Fatalf("host should receive host channel after enabling host powers: %+v", hostState.Channels)
	}
	seenMafia = false
	for _, player := range hostState.Players {
		if player.ID == mafia.ID && player.Role == string(MafiaRoleMafia) {
			seenMafia = true
		}
	}
	if !seenMafia {
		t.Fatalf("host should see all roles after enabling host powers: %+v", hostState.Players)
	}
}

func TestMafiaMiniAppDetectiveResultStaysPrivateAfterNight(t *testing.T) {
	t.Setenv("DEV_AUTH", "true")
	store, err := NewStore(filepath.Join(t.TempDir(), "state.json"), nil)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	now := time.Now().UTC()
	room := MafiaRoom{
		ID:        "MF-CHECK",
		OwnerID:   200,
		Status:    MafiaStatusNight,
		DayNumber: 1,
		Settings:  defaultMafiaRoomSettings(),
		CreatedAt: now.Unix(),
		UpdatedAt: now.Unix(),
		Players: []MafiaPlayer{
			{TelegramID: 200, DisplayName: "Host", Role: MafiaRoleCivilian, Alive: true},
			{TelegramID: 201, DisplayName: "Detective", Role: MafiaRoleDetective, Alive: true},
			{TelegramID: 202, DisplayName: "Shade", Role: MafiaRoleMafia, Alive: true},
			{TelegramID: 203, DisplayName: "Civil", Role: MafiaRoleCivilian, Alive: true},
		},
		Votes:        map[string]string{},
		NightActions: MafiaNightActions{MafiaVotes: map[string]int64{}, DetectiveChecks: map[string]int64{}, DetectiveResults: map[string]bool{}},
	}
	if err := store.SaveMafiaRoom(room); err != nil {
		t.Fatalf("SaveMafiaRoom: %v", err)
	}
	service := webPanelService{app: &App{store: store}}
	auth := wardenweb.MafiaMiniAppAuthInput{DevUserID: 201, DevName: "Detective"}
	state, err := service.MafiaMiniAppSubmitNightAction(context.Background(), auth, wardenweb.MafiaMiniAppNightActionInput{RoomID: room.ID, Action: "check", TargetID: 202})
	if err != nil {
		t.Fatalf("MafiaMiniAppSubmitNightAction: %v", err)
	}
	if !miniAppMessagesContain(state.Messages, "Проверка Лайи: Shade — мафия.") {
		t.Fatalf("detective did not receive private check result: %+v", state.Messages)
	}
	if !miniAppPlayerHasRole(state.Players, 202, string(MafiaRoleMafia)) {
		t.Fatalf("detective should see checked target role in players tab: %+v", state.Players)
	}
	if miniAppPlayerHasRole(state.Players, 203, string(MafiaRoleCivilian)) {
		t.Fatalf("detective should not see unchecked target role: %+v", state.Players)
	}
	if _, err := service.MafiaMiniAppSubmitNightAction(context.Background(), auth, wardenweb.MafiaMiniAppNightActionInput{RoomID: room.ID, Action: "check", TargetID: 203}); !errors.Is(err, errMafiaNightActionExists) {
		t.Fatalf("second detective check err = %v, want errMafiaNightActionExists", err)
	}
	civilState, err := service.mafiaMiniStateForPlayer(room.ID, 203)
	if err != nil {
		t.Fatalf("civilian state: %v", err)
	}
	if miniAppMessagesContain(civilState.Messages, "Проверка Лайи: Shade — мафия.") {
		t.Fatalf("civilian received detective result: %+v", civilState.Messages)
	}
	if _, _, err := service.app.mafiaService().ResolveNight(room.ID, now.Add(time.Minute)); err != nil {
		t.Fatalf("ResolveNight: %v", err)
	}
	dayState, err := service.mafiaMiniStateForPlayer(room.ID, 201)
	if err != nil {
		t.Fatalf("day detective state: %v", err)
	}
	if !miniAppMessagesContain(dayState.Messages, "Проверка Лайи: Shade — мафия.") {
		t.Fatalf("detective result did not survive night resolve: %+v", dayState.Messages)
	}
	if !miniAppPlayerHasRole(dayState.Players, 202, string(MafiaRoleMafia)) {
		t.Fatalf("checked target role did not survive night resolve: %+v", dayState.Players)
	}
}

func miniAppMessagesContain(messages []wardenweb.MafiaMiniAppMessage, text string) bool {
	for _, msg := range messages {
		if msg.Text == text {
			return true
		}
	}
	return false
}

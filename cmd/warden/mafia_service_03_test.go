package main

import (
	"errors"
	"strings"
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func TestMafiaReadyQueueAutoFill(t *testing.T) {
	store := newMafiaTestStore(t)
	service := NewMafiaService(store)
	now := time.Now().UTC()
	owner := &tele.User{ID: 7101, FirstName: "Owner"}
	queued := []*tele.User{
		{ID: 7102, FirstName: "Ready1"},
		{ID: 7103, FirstName: "Ready2"},
	}
	if _, err := service.RegisterUser(owner, now); err != nil {
		t.Fatalf("Register owner: %v", err)
	}
	for _, user := range queued {
		if _, err := service.RegisterUser(user, now); err != nil {
			t.Fatalf("Register queued: %v", err)
		}
		if _, err := service.JoinReadyQueue(user, now); err != nil {
			t.Fatalf("JoinReadyQueue: %v", err)
		}
	}
	room, err := service.CreateRoom(owner, now)
	if err != nil {
		t.Fatalf("CreateRoom: %v", err)
	}
	room, added, err := service.AutoFillFromQueue(room.ID, owner.ID, now)
	if err != nil {
		t.Fatalf("AutoFillFromQueue: %v", err)
	}
	if len(added) != 2 || len(room.Players) != 3 {
		t.Fatalf("added=%d players=%d, want 2/3", len(added), len(room.Players))
	}
	if len(store.MafiaReadyQueue()) != 0 {
		t.Fatalf("ready queue should be empty after auto fill")
	}
}

func TestMafiaInviteBroadcastRespectsSettingsAndLimits(t *testing.T) {
	store := newMafiaTestStore(t)
	service := NewMafiaService(store)
	now := time.Now().UTC()
	owner := &tele.User{ID: 7201, FirstName: "Owner"}
	allowed := &tele.User{ID: 7202, FirstName: "Allowed"}
	disabled := &tele.User{ID: 7203, FirstName: "Disabled"}
	for _, user := range []*tele.User{owner, allowed, disabled} {
		if _, err := service.RegisterUser(user, now); err != nil {
			t.Fatalf("RegisterUser: %v", err)
		}
	}
	if err := service.SetInvitePermission(disabled.ID, false); err != nil {
		t.Fatalf("SetInvitePermission: %v", err)
	}
	room, err := service.CreateRoom(owner, now)
	if err != nil {
		t.Fatalf("CreateRoom: %v", err)
	}
	room, recipients, err := service.PrepareInviteBroadcast(room.ID, owner.ID, false, now)
	if err != nil {
		t.Fatalf("PrepareInviteBroadcast: %v", err)
	}
	if len(recipients) != 1 || recipients[0].TelegramID != allowed.ID {
		t.Fatalf("recipients=%v, want only allowed user", recipients)
	}
	if _, _, err := service.PrepareInviteBroadcast(room.ID, owner.ID, false, now.Add(time.Minute)); !errors.Is(err, errMafiaInviteLimited) {
		t.Fatalf("second broadcast err=%v, want invite limit", err)
	}
}

func TestMafiaInviteMenuDoesNotListAllUsers(t *testing.T) {
	room := MafiaRoom{ID: "MF-4821", Status: MafiaStatusLobby, Settings: defaultMafiaRoomSettings(), RecruitmentMode: MafiaRecruitmentRequests}
	markup := mafiaInviteMenuMarkup(room)
	if markup == nil || len(markup.InlineKeyboard) > 8 {
		t.Fatalf("invite menu should stay compact, rows=%d", len(markup.InlineKeyboard))
	}
	for _, row := range markup.InlineKeyboard {
		for _, button := range row {
			if strings.Contains(button.Text, "User_799") {
				t.Fatalf("invite menu leaked a registered user list")
			}
		}
	}
}

func TestMafiaStartButtonVisibleWithMissingRequirements(t *testing.T) {
	room := MafiaRoom{
		ID:       "MF-START",
		OwnerID:  9001,
		Status:   MafiaStatusLobby,
		Settings: defaultMafiaRoomSettings(),
		Players: []MafiaPlayer{
			{TelegramID: 9001, DisplayName: "Owner", Alive: true},
		},
	}
	if !mafiaMarkupContains(mafiaRoomMarkup(room, room.OwnerID), "Начать игру") {
		t.Fatalf("owner inline lobby menu must show start button even before all requirements are met")
	}
	if !mafiaReplyMarkupContains(mafiaRoomReplyKeyboard(room, room.OwnerID), "Начать игру") {
		t.Fatalf("owner reply lobby menu must show start button even before all requirements are met")
	}
}

func TestMafiaRosterTextShowsAliveAndDeadPlayers(t *testing.T) {
	room := MafiaRoom{
		ID:       "ROSTER",
		OwnerID:  9011,
		Status:   MafiaStatusDayDiscussion,
		Settings: defaultMafiaRoomSettings(),
		Players: []MafiaPlayer{
			{TelegramID: 9011, DisplayName: "Арчибальд", Alive: true},
			{TelegramID: 9012, DisplayName: "София", Alive: false},
			{TelegramID: 9013, DisplayName: "Марта", Alive: true},
		},
	}
	text := mafiaRosterText(room)
	for _, want := range []string{"Живые игроки (2)", "1. Арчибальд", "2. Марта", "Выбыли (1)", "1. София"} {
		if !strings.Contains(text, want) {
			t.Fatalf("roster text missing %q:\n%s", want, text)
		}
	}
}

func TestMafiaRosterAndDeathTextShowUsernames(t *testing.T) {
	room := MafiaRoom{
		ID:       "ROSTER-USERNAMES",
		OwnerID:  9016,
		Status:   MafiaStatusDayDiscussion,
		Settings: defaultMafiaRoomSettings(),
		Players: []MafiaPlayer{
			{TelegramID: 9016, Username: "john_kennedy", DisplayName: "Джон Кеннеди", Alive: false},
			{TelegramID: 9017, Username: "@sofia", DisplayName: "София", Alive: true},
			{TelegramID: 9018, DisplayName: "Без Юза", Alive: true},
			{TelegramID: mafiaTestBotBaseID - 90, Username: "test_bot", DisplayName: "Тестовый бот", Alive: true, IsTestBot: true},
		},
	}
	roster := mafiaRosterText(room)
	for _, want := range []string{
		"София(@sofia)",
		"Джон Кеннеди(@john_kennedy)",
		"Без Юза",
		"Тестовый бот",
	} {
		if !strings.Contains(roster, want) {
			t.Fatalf("roster missing %q:\n%s", want, roster)
		}
	}
	if strings.Contains(roster, "Тестовый бот(@test_bot)") {
		t.Fatalf("test bot username should not be shown:\n%s", roster)
	}
	death := mafiaNightResultText(room, MafiaNightResult{KilledID: 9016})
	if !strings.Contains(death, "Джон Кеннеди(@john_kennedy)") {
		t.Fatalf("death text should include username:\n%s", death)
	}
	room.Settings.NightKillMessage = "Наступает утро.\nПогиб - {player}.\nКомната: {room}, день {day}."
	customDeath := mafiaNightResultText(room, MafiaNightResult{KilledID: 9016})
	for _, want := range []string{"Погиб - Джон Кеннеди(@john_kennedy).", "Комната: ROSTER-USERNAMES"} {
		if !strings.Contains(customDeath, want) {
			t.Fatalf("custom death text missing %q:\n%s", want, customDeath)
		}
	}
}

func TestMafiaPhaseMessageSeparatesHeaderAndBody(t *testing.T) {
	got := mafiaPhaseMessage("ДЕНЬ", "Город просыпается.")
	for _, want := range []string{
		"━━━━━━━━━━━━━━━━━━━━",
		"--- ДЕНЬ ---",
		"\n\n\nГород просыпается.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("phase message missing %q:\n%s", want, got)
		}
	}
}

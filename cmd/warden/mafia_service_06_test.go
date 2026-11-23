package main

import (
	"strings"
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func TestMafiaGroupMessageRoomFallsBackToNewestGroupRoom(t *testing.T) {
	store := newMafiaTestStore(t)
	now := time.Now().UTC()
	oldOwner := &tele.User{ID: 9051, FirstName: "Old"}
	newOwner := &tele.User{ID: 9052, FirstName: "New"}
	if _, err := store.RegisterMafiaUser(oldOwner, now); err != nil {
		t.Fatalf("RegisterMafiaUser old: %v", err)
	}
	if _, err := store.RegisterMafiaUser(newOwner, now); err != nil {
		t.Fatalf("RegisterMafiaUser new: %v", err)
	}
	oldRoom := MafiaRoom{
		ID:          "OLD-GROUP-ROOM",
		OwnerID:     oldOwner.ID,
		GroupChatID: -1009050,
		Status:      MafiaStatusDayDiscussion,
		Settings:    defaultMafiaRoomSettings(),
		Players: []MafiaPlayer{
			{TelegramID: oldOwner.ID, DisplayName: "Old", Alive: true},
		},
		CreatedAt:      now.Unix(),
		UpdatedAt:      now.Unix(),
		PhaseStartedAt: now.Unix(),
	}
	nightRoom := MafiaRoom{
		ID:          "NIGHT-GROUP-ROOM",
		OwnerID:     newOwner.ID,
		GroupChatID: -1009050,
		Status:      MafiaStatusNight,
		Settings:    defaultMafiaRoomSettings(),
		Players: []MafiaPlayer{
			{TelegramID: newOwner.ID, DisplayName: "New", Alive: true},
		},
		CreatedAt:      now.Unix(),
		UpdatedAt:      now.Add(time.Second).Unix(),
		PhaseStartedAt: now.Add(time.Second).Unix(),
	}
	if err := store.SaveMafiaRoom(oldRoom); err != nil {
		t.Fatalf("SaveMafiaRoom old: %v", err)
	}
	if err := store.SaveMafiaRoom(nightRoom); err != nil {
		t.Fatalf("SaveMafiaRoom night: %v", err)
	}
	app := &App{store: store}
	got, ok := app.mafiaRoomForGroupMessage(-1009050, 999999)
	if !ok || got.ID != nightRoom.ID {
		t.Fatalf("mafiaRoomForGroupMessage picked %#v, ok=%t, want %s", got, ok, nightRoom.ID)
	}
}

func TestMafiaDiscussionChatSilencesOrdinaryLayaReplies(t *testing.T) {
	store := newMafiaTestStore(t)
	now := time.Now().UTC()
	room := MafiaRoom{
		ID:          "SILENCE-GROUP",
		OwnerID:     9061,
		GroupChatID: -1009060,
		Status:      MafiaStatusDayDiscussion,
		Settings:    defaultMafiaRoomSettings(),
		Players: []MafiaPlayer{
			{TelegramID: 9061, DisplayName: "Owner", Alive: true},
		},
		CreatedAt:      now.Unix(),
		UpdatedAt:      now.Unix(),
		PhaseStartedAt: now.Unix(),
	}
	if err := store.SaveMafiaRoom(room); err != nil {
		t.Fatalf("SaveMafiaRoom: %v", err)
	}
	app := &App{store: store}
	chat := &tele.Chat{ID: -1009060, Type: tele.ChatSuperGroup}
	if !app.shouldSilenceMafiaDiscussionMessage(chat, &tele.Message{Text: "Лайя привет"}) {
		t.Fatalf("ordinary mafia discussion message should silence normal Laya replies")
	}
	if app.shouldSilenceMafiaDiscussionMessage(chat, &tele.Message{Text: "/mafia_status"}) {
		t.Fatalf("commands in mafia discussion chat must stay available")
	}
	if app.shouldSilenceMafiaDiscussionMessage(&tele.Chat{ID: -1009061, Type: tele.ChatSuperGroup}, &tele.Message{Text: "Лайя привет"}) {
		t.Fatalf("non-mafia chat should not be silenced")
	}
}

func TestMafiaRememberedDiscussionChatSilencesAfterFinal(t *testing.T) {
	store := newMafiaTestStore(t)
	now := time.Now().UTC()
	user := &tele.User{ID: 9065, FirstName: "Owner"}
	if _, err := store.RegisterMafiaUser(user, now); err != nil {
		t.Fatalf("RegisterMafiaUser: %v", err)
	}
	settings := normalizeMafiaUserSettings(MafiaUserSettings{})
	settings.LastDiscussionGroupChatID = -1009065
	if err := store.SetMafiaUserSettings(user.ID, settings); err != nil {
		t.Fatalf("SetMafiaUserSettings: %v", err)
	}
	app := &App{store: store}
	chat := &tele.Chat{ID: -1009065, Type: tele.ChatSuperGroup}
	if !app.shouldSilenceMafiaDiscussionMessage(chat, &tele.Message{Text: "Лайя ты тут?"}) {
		t.Fatalf("remembered mafia discussion chat should silence ordinary Laya replies after final")
	}
	if app.shouldSilenceMafiaDiscussionMessage(chat, &tele.Message{Text: "/mafia"}) {
		t.Fatalf("commands must remain available in remembered mafia discussion chat")
	}
}

func TestMafiaVotingProgressShowsVotesAndTally(t *testing.T) {
	room := MafiaRoom{
		ID:       "VOTE-PROGRESS",
		OwnerID:  9021,
		Status:   MafiaStatusVoting,
		Settings: defaultMafiaRoomSettings(),
		Players: []MafiaPlayer{
			{TelegramID: 9021, DisplayName: "Арчибальд", Alive: true},
			{TelegramID: 9022, DisplayName: "София", Alive: true},
			{TelegramID: 9023, DisplayName: "Марта", Alive: true},
		},
	}
	text := mafiaVotingProgressText(room, map[string]string{
		idKey(9021): idKey(9022),
		idKey(9022): "skip",
	})
	for _, want := range []string{
		"Прогресс голосования: 2/3",
		"Счёт:",
		"София — 1",
		"Пропуск — 1",
		"Арчибальд → София",
		"София → Пропуск",
		"Марта → ещё не голосовал",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("voting progress missing %q:\n%s", want, text)
		}
	}
}

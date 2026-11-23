package main

import (
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func TestMafiaActivePartyBlocksForwardedAndMediaMessages(t *testing.T) {
	activeStatuses := []MafiaRoomStatus{
		MafiaStatusRoleDistribution,
		MafiaStatusNight,
		MafiaStatusDayDiscussion,
		MafiaStatusVoting,
	}
	for _, status := range activeStatuses {
		if !mafiaRoomStatusHasActiveParty(status) {
			t.Fatalf("status %s must be treated as active party", status)
		}
	}
	inactiveStatuses := []MafiaRoomStatus{MafiaStatusLobby, MafiaStatusFinished, MafiaStatusCancelled}
	for _, status := range inactiveStatuses {
		if mafiaRoomStatusHasActiveParty(status) {
			t.Fatalf("status %s must not be treated as active party", status)
		}
	}
	if !mafiaIsForwardedMessage(&tele.Message{AutomaticForward: true}) {
		t.Fatalf("automatic forward must be blocked during active mafia party")
	}
	if !mafiaIsForwardedMessage(&tele.Message{OriginalSenderName: "Лайя"}) {
		t.Fatalf("forwarded message must be blocked during active mafia party")
	}
	if !mafiaIsMediaMessage(&tele.Message{Photo: &tele.Photo{}}) {
		t.Fatalf("photo must be blocked during active mafia party")
	}
	if !mafiaIsMediaMessage(&tele.Message{Document: &tele.Document{}}) {
		t.Fatalf("document must be blocked during active mafia party")
	}
	if !mafiaIsMediaMessage(&tele.Message{Sticker: &tele.Sticker{}}) {
		t.Fatalf("sticker must be blocked during active mafia party")
	}
	if mafiaIsMediaMessage(&tele.Message{Text: "обычное сообщение"}) {
		t.Fatalf("plain text must not be treated as media")
	}
}

func TestMafiaGroupNoticeOnlyOncePerPhase(t *testing.T) {
	room := MafiaRoom{
		ID:             "NOTICE",
		Status:         MafiaStatusDayDiscussion,
		DayNumber:      2,
		PhaseStartedAt: 12345,
	}
	app := &App{}
	now := time.Now().UTC()
	if !app.shouldSendMafiaGroupNotice(room, 9059, "dead_talking", now) {
		t.Fatalf("first notice should be sent")
	}
	if app.shouldSendMafiaGroupNotice(room, 9059, "dead_talking", now.Add(time.Second)) {
		t.Fatalf("second notice in the same phase should be suppressed")
	}
	room.PhaseStartedAt++
	if !app.shouldSendMafiaGroupNotice(room, 9059, "dead_talking", now.Add(2*time.Second)) {
		t.Fatalf("notice should reset on a new phase")
	}
}

func TestMafiaPrepareRematchLobbyKeepsRoomAndPlayers(t *testing.T) {
	store := newMafiaTestStore(t)
	service := NewMafiaService(store)
	now := time.Now().UTC()
	room := MafiaRoom{
		ID:                "MF-KEEP",
		OwnerID:           9400,
		GroupChatID:       -1009400,
		Status:            MafiaStatusFinished,
		Settings:          defaultMafiaRoomSettings(),
		DayNumber:         3,
		Winner:            "mafia",
		Votes:             map[string]string{"9400": "9401"},
		PhaseSkipVotes:    map[string]bool{"9400": true},
		NightActions:      MafiaNightActions{MafiaVotes: map[string]int64{"9401": 9400}, DoctorTargetID: 9400, DetectiveChecks: map[string]int64{"9402": 9401}, DetectiveResults: map[string]bool{"9401": true}},
		MafiaPrivateChat:  []MafiaPrivateMessage{{ID: "shadow", RoomID: "MF-KEEP", SenderID: 9401, SenderAlias: "Shade", Text: "тишина", CreatedAt: now.Unix()}},
		CreatedAt:         now.Add(-time.Hour).Unix(),
		UpdatedAt:         now.Add(-time.Minute).Unix(),
		PhaseStartedAt:    now.Add(-time.Minute).Unix(),
		RecruitmentStatus: MafiaRecruitmentOpen,
		JoinRequests:      []MafiaJoinRequest{{TelegramID: 9499, CharacterName: "Гость", RequestedAt: now.Unix(), Status: MafiaJoinRequestPending}},
		Players: []MafiaPlayer{
			{TelegramID: 9400, DisplayName: "Джон", Role: MafiaRoleDetective, MafiaAlias: "Don", Alive: false, WarnedForReveal: true, NightActionDone: true, Will: "завещание"},
			{TelegramID: 9401, DisplayName: "София", Role: MafiaRoleMafia, MafiaAlias: "Shade", Alive: true, NightActionDone: true, Will: "план"},
		},
	}
	if err := store.SaveMafiaRoom(room); err != nil {
		t.Fatalf("SaveMafiaRoom: %v", err)
	}
	lobby, err := service.PrepareRematchLobby(room.ID, now)
	if err != nil {
		t.Fatalf("PrepareRematchLobby: %v", err)
	}
	if lobby.ID != room.ID || lobby.GroupChatID != room.GroupChatID || lobby.OwnerID != room.OwnerID {
		t.Fatalf("room identity was not preserved: %#v", lobby)
	}
	if lobby.Status != MafiaStatusLobby {
		t.Fatalf("status=%s, want lobby", lobby.Status)
	}
	if lobby.DayNumber != 0 || lobby.Winner != "" || lobby.PhaseStartedAt != 0 {
		t.Fatalf("round state was not cleared: %#v", lobby)
	}
	if len(lobby.Votes) != 0 || len(lobby.PhaseSkipVotes) != 0 || len(lobby.MafiaPrivateChat) != 0 || len(lobby.JoinRequests) != 0 {
		t.Fatalf("transient room data was not cleared: %#v", lobby)
	}
	if lobby.RecruitmentStatus != MafiaRecruitmentClosed {
		t.Fatalf("recruitment status=%s, want closed", lobby.RecruitmentStatus)
	}
	for _, player := range lobby.Players {
		if !player.Alive || player.Role != "" || player.MafiaAlias != "" || player.WarnedForReveal || player.NightActionDone || player.Will != "" {
			t.Fatalf("player was not reset for rematch: %#v", player)
		}
	}
}

func TestMafiaGroupMessageRoomPrefersSenderActiveRoom(t *testing.T) {
	store := newMafiaTestStore(t)
	now := time.Now().UTC()
	wrongRoom := MafiaRoom{
		ID:          "WRONG-GROUP-ROOM",
		OwnerID:     9042,
		GroupChatID: -1009040,
		Status:      MafiaStatusDayDiscussion,
		Settings:    defaultMafiaRoomSettings(),
		Players: []MafiaPlayer{
			{TelegramID: 9041, DisplayName: "Другой игрок", Alive: true},
		},
		CreatedAt: now.Unix(),
		UpdatedAt: now.Add(2 * time.Second).Unix(),
	}
	currentRoom := MafiaRoom{
		ID:          "CURRENT-GROUP-ROOM",
		OwnerID:     9042,
		GroupChatID: -1009040,
		Status:      MafiaStatusNight,
		Settings:    defaultMafiaRoomSettings(),
		Players: []MafiaPlayer{
			{TelegramID: 9042, DisplayName: "Джон", Alive: false},
			{TelegramID: 9043, DisplayName: "София", Alive: true},
		},
		CreatedAt: now.Unix(),
		UpdatedAt: now.Add(time.Second).Unix(),
	}
	if err := store.SaveMafiaRoom(wrongRoom); err != nil {
		t.Fatalf("SaveMafiaRoom wrong: %v", err)
	}
	if err := store.SaveMafiaRoom(currentRoom); err != nil {
		t.Fatalf("SaveMafiaRoom current: %v", err)
	}
	app := &App{store: store}
	got, ok := app.mafiaRoomForGroupMessage(-1009040, 9042)
	if !ok || got.ID != currentRoom.ID {
		t.Fatalf("mafiaRoomForGroupMessage picked %#v, ok=%t, want %s", got, ok, currentRoom.ID)
	}
}

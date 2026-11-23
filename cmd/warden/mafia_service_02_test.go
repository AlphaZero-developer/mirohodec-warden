package main

import (
	"strings"
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func TestMafiaInlineButtonsUseRawCallbackData(t *testing.T) {
	markup := mafiaProfileMarkup(true)
	if markup == nil || len(markup.InlineKeyboard) == 0 || len(markup.InlineKeyboard[0]) == 0 {
		t.Fatalf("profile markup is empty")
	}
	button := markup.InlineKeyboard[0][0]
	if button.Unique != "" {
		t.Fatalf("profile inline button uses telebot unique %q, want raw callback data", button.Unique)
	}
	if button.Data != "mafia:profile_edit_name" {
		t.Fatalf("profile inline button data = %q, want mafia:profile_edit_name", button.Data)
	}
	if !mafiaMarkupContains(markup, "Достижения") {
		t.Fatalf("profile markup must expose achievements")
	}
}

func TestMafiaRegistrationIntroHasAcknowledgementButton(t *testing.T) {
	text := mafiaRegistrationIntroText()
	if !strings.Contains(text, "Как работает MafiaGame") || !strings.Contains(text, "Ночью общий чат молчит") {
		t.Fatalf("registration intro is missing game principles:\n%s", text)
	}
	markup := mafiaRegistrationIntroMarkup()
	if markup == nil || !mafiaMarkupContains(markup, "Ознакомился") {
		t.Fatalf("registration intro must include acknowledgement button")
	}
}

func TestMafiaRevealSettingsExplainAllowedAndForbidden(t *testing.T) {
	room := MafiaRoom{
		ID:       "MF-RULES",
		Settings: defaultMafiaRoomSettings(),
	}
	room.Settings.RoleRevealPolicy = MafiaRoleRevealBanSystemProof
	text := mafiaRevealSettingsText(room)
	for _, want := range []string{"Текущий режим: 🟡 Обычный", "Сейчас разрешено", "Сейчас запрещено", "у меня кнопка лечения"} {
		if !strings.Contains(text, want) {
			t.Fatalf("reveal settings text must contain %q:\n%s", want, text)
		}
	}
	markup := mafiaRevealSettingsMarkup(room)
	if !mafiaMarkupContains(markup, "🟡 Обычный режим ✅") {
		t.Fatalf("current reveal policy button must be marked")
	}
}

func TestMafiaProfileTextIncludesPlayerInfoAndAchievements(t *testing.T) {
	user := MafiaRegisteredUser{
		TelegramID:   7010,
		Username:     "kennedy",
		DisplayName:  "Telegram Name",
		RegisteredAt: time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC).Unix(),
		Profile: MafiaProfile{
			CharacterName: "Джон Кеннеди",
			Backstory:     "Слишком много знал.",
		},
		Stats: MafiaStats{
			Games:        4,
			Wins:         2,
			Deaths:       1,
			MafiaWins:    1,
			CivilianWins: 1,
			Achievements: []string{"first_game", "last_thought"},
		},
		Settings: MafiaUserSettings{Initialized: true, AllowMafiaInvites: true, LastDiscussionGroupChatID: -1007010},
	}
	text := mafiaProfileText(user)
	for _, want := range []string{
		"Джон Кеннеди",
		"Telegram Name (@kennedy)",
		"Игр: 4",
		"Винрейт: 50%",
		"Первая партия",
		"Последняя мысль",
		"Чат обсуждений: запомнен",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("profile text missing %q:\n%s", want, text)
		}
	}
}

func TestMafiaPlayerAdminTitleUsesCharacterName(t *testing.T) {
	tests := []struct {
		name string
		in   MafiaPlayer
		want string
	}{
		{name: "plain", in: MafiaPlayer{DisplayName: "Джон Кеннеди"}, want: "Джон Кеннеди"},
		{name: "emoji removed", in: MafiaPlayer{DisplayName: "Масьнька❤️‍🔥"}, want: "Масьнька"},
		{name: "long trimmed", in: MafiaPlayer{DisplayName: "Никто и не Зовут на районе"}, want: "Никто и не Зовут"},
		{name: "fallback", in: MafiaPlayer{DisplayName: "❤️‍🔥"}, want: "Игрок"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mafiaPlayerAdminTitle(tt.in); got != tt.want {
				t.Fatalf("mafiaPlayerAdminTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMafiaHomeMarkupExposesAchievements(t *testing.T) {
	if !mafiaMarkupContains(mafiaHomeMarkup(false, false), "Достижения") {
		t.Fatalf("home menu must expose achievements")
	}
}

func TestMafiaRecruitmentRequestsAndAutoAccept(t *testing.T) {
	store := newMafiaTestStore(t)
	service := NewMafiaService(store)
	now := time.Now().UTC()
	owner := &tele.User{ID: 7001, FirstName: "Owner"}
	guest := &tele.User{ID: 7002, FirstName: "Guest"}
	autoGuest := &tele.User{ID: 7003, FirstName: "Auto"}
	for _, user := range []*tele.User{owner, guest, autoGuest} {
		if _, err := service.RegisterUser(user, now); err != nil {
			t.Fatalf("RegisterUser: %v", err)
		}
	}
	room, err := service.CreateRoom(owner, now)
	if err != nil {
		t.Fatalf("CreateRoom: %v", err)
	}
	room, err = service.OpenRecruitment(room.ID, owner.ID, MafiaRecruitmentRequests, now)
	if err != nil {
		t.Fatalf("OpenRecruitment requests: %v", err)
	}
	room, added, err := service.SubmitJoinRequest(room.ID, guest, now)
	if err != nil {
		t.Fatalf("SubmitJoinRequest: %v", err)
	}
	if added || len(mafiaPendingJoinRequests(room)) != 1 || len(room.Players) != 1 {
		t.Fatalf("request mode should create pending request without adding player: added=%v players=%d requests=%d", added, len(room.Players), len(mafiaPendingJoinRequests(room)))
	}
	room, err = service.AcceptJoinRequest(room.ID, owner.ID, guest.ID, now)
	if err != nil {
		t.Fatalf("AcceptJoinRequest: %v", err)
	}
	if mafiaRoomPlayerIndex(room, guest.ID) < 0 {
		t.Fatalf("guest was not added after request accept")
	}

	room, err = service.SetRecruitmentMode(room.ID, owner.ID, MafiaRecruitmentAutoAccept, now)
	if err != nil {
		t.Fatalf("SetRecruitmentMode: %v", err)
	}
	room, added, err = service.SubmitJoinRequest(room.ID, autoGuest, now)
	if err != nil {
		t.Fatalf("SubmitJoinRequest auto: %v", err)
	}
	if !added || mafiaRoomPlayerIndex(room, autoGuest.ID) < 0 {
		t.Fatalf("auto accept should add guest immediately")
	}
}

func TestMafiaCreateRoomRemembersOwnerDiscussionGroup(t *testing.T) {
	store := newMafiaTestStore(t)
	service := NewMafiaService(store)
	now := time.Now().UTC()
	owner := &tele.User{ID: 7051, FirstName: "Owner"}
	if _, err := service.RegisterUser(owner, now); err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}
	first, err := service.CreateRoom(owner, now)
	if err != nil {
		t.Fatalf("CreateRoom first: %v", err)
	}
	if _, err := service.BindGroup(first.ID, owner.ID, -1007051, now); err != nil {
		t.Fatalf("BindGroup: %v", err)
	}
	if err := store.DeleteMafiaRoom(first.ID); err != nil {
		t.Fatalf("DeleteMafiaRoom: %v", err)
	}
	second, err := service.CreateRoom(owner, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("CreateRoom second: %v", err)
	}
	if second.GroupChatID != -1007051 {
		t.Fatalf("second room group=%d, want remembered group", second.GroupChatID)
	}
}

package web

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestStartServesHTML(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	errs := make(chan error, 1)
	go func() {
		errs <- Start(Config{
			Web:       WebConfig{Enabled: true, Listen: addr, AdminToken: "secret", SessionTTL: time.Hour},
			Context:   ctx,
			Service:   fakeService{},
			OnStarted: func() { close(started) },
		})
	}()

	select {
	case <-started:
	case err := <-errs:
		t.Fatalf("Start returned early: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("server did not start")
	}

	resp, err := http.Get("http://" + addr + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}

type fakeService struct{}

func (fakeService) MafiaMiniAppAuth(context.Context, MafiaMiniAppAuthInput) (MafiaMiniAppAuthPayload, error) {
	return MafiaMiniAppAuthPayload{User: MafiaMiniAppUser{ID: 1, DisplayName: "Dev"}, DevAuth: true}, nil
}

func (fakeService) MafiaMiniAppUpdateProfile(context.Context, MafiaMiniAppAuthInput, MafiaMiniAppProfileInput) (MafiaMiniAppAuthPayload, error) {
	return MafiaMiniAppAuthPayload{User: MafiaMiniAppUser{ID: 1, DisplayName: "Dev"}, DevAuth: true}, nil
}

func (fakeService) MafiaMiniAppProfile(context.Context, MafiaMiniAppAuthInput, int64) (MafiaMiniAppProfilePayload, error) {
	return MafiaMiniAppProfilePayload{Profile: MafiaMiniAppProfile{UserID: 1, DisplayName: "Dev"}}, nil
}

func (fakeService) MafiaMiniAppLobby(context.Context, MafiaMiniAppAuthInput) (MafiaMiniAppLobbyPayload, error) {
	return MafiaMiniAppLobbyPayload{User: MafiaMiniAppUser{ID: 1, DisplayName: "Dev"}}, nil
}

func (fakeService) MafiaMiniAppCreateRoom(context.Context, MafiaMiniAppAuthInput) (MafiaMiniAppState, error) {
	return MafiaMiniAppState{}, nil
}

func (fakeService) MafiaMiniAppJoinRoom(context.Context, MafiaMiniAppAuthInput, string) (MafiaMiniAppState, error) {
	return MafiaMiniAppState{}, nil
}

func (fakeService) MafiaMiniAppRoomState(context.Context, MafiaMiniAppAuthInput, string) (MafiaMiniAppState, error) {
	return MafiaMiniAppState{}, nil
}

func (fakeService) MafiaMiniAppSendMessage(context.Context, MafiaMiniAppAuthInput, MafiaMiniAppMessageInput) (MafiaMiniAppState, error) {
	return MafiaMiniAppState{}, nil
}

func (fakeService) MafiaMiniAppSaveWill(context.Context, MafiaMiniAppAuthInput, MafiaMiniAppWillInput) (MafiaMiniAppState, error) {
	return MafiaMiniAppState{}, nil
}

func (fakeService) MafiaMiniAppSkipPhase(context.Context, MafiaMiniAppAuthInput, string) (MafiaMiniAppState, error) {
	return MafiaMiniAppState{}, nil
}

func (fakeService) MafiaMiniAppSubmitVote(context.Context, MafiaMiniAppAuthInput, MafiaMiniAppVoteInput) (MafiaMiniAppState, error) {
	return MafiaMiniAppState{}, nil
}

func (fakeService) MafiaMiniAppSubmitNightAction(context.Context, MafiaMiniAppAuthInput, MafiaMiniAppNightActionInput) (MafiaMiniAppState, error) {
	return MafiaMiniAppState{}, nil
}

func (fakeService) MafiaMiniAppStartGame(context.Context, MafiaMiniAppAuthInput, string) (MafiaMiniAppState, error) {
	return MafiaMiniAppState{}, nil
}

func (fakeService) MafiaMiniAppNextPhase(context.Context, MafiaMiniAppAuthInput, string) (MafiaMiniAppState, error) {
	return MafiaMiniAppState{}, nil
}

func (fakeService) MafiaMiniAppUpdateSettings(context.Context, MafiaMiniAppAuthInput, string, MafiaMiniAppSettingsInput) (MafiaMiniAppState, error) {
	return MafiaMiniAppState{}, nil
}

func (fakeService) MafiaMiniAppKickPlayer(context.Context, MafiaMiniAppAuthInput, MafiaMiniAppKickInput) (MafiaMiniAppState, error) {
	return MafiaMiniAppState{}, nil
}

func (fakeService) MafiaMiniAppUpdateTestBots(context.Context, MafiaMiniAppAuthInput, MafiaMiniAppTestBotsInput) (MafiaMiniAppState, error) {
	return MafiaMiniAppState{}, nil
}

func (fakeService) MafiaMiniAppLeaveRoom(context.Context, MafiaMiniAppAuthInput, string) (MafiaMiniAppLobbyPayload, error) {
	return MafiaMiniAppLobbyPayload{}, nil
}

func (fakeService) MafiaMiniAppCloseRoom(context.Context, MafiaMiniAppAuthInput, string) (MafiaMiniAppLobbyPayload, error) {
	return MafiaMiniAppLobbyPayload{}, nil
}

func (fakeService) Dashboard(context.Context) (Dashboard, error) {
	return Dashboard{BotName: "@laya", Mode: "polling", State: "running", KnownUsers: 3}, nil
}

func (fakeService) ChatSelection(context.Context, int64) (ChatSelection, error) {
	return ChatSelection{CurrentID: -1001, Options: []ChatOption{{ID: -1001, Label: "Main", Selected: true}}}, nil
}

func (fakeService) Groups(context.Context) ([]Group, error) {
	return []Group{{ID: -1001, Title: "Main", Enabled: true, RehabTopicID: 7}}, nil
}

func (fakeService) AddGroup(context.Context, GroupInput) error { return nil }

func (fakeService) UpdateGroup(context.Context, GroupInput) error { return nil }

func (fakeService) DeleteGroup(context.Context, int64) error { return nil }

func (fakeService) SetGroupDebug(context.Context, int64, bool) error { return nil }

func (fakeService) SetGroupRehabTopic(context.Context, int64, int) error { return nil }

func (fakeService) Rehab(context.Context, int64) (RehabPage, error) { return RehabPage{}, nil }

func (fakeService) AddRehab(_ context.Context, input RehabInput) error {
	if input.UserID == 0 {
		return errors.New("user_id is required")
	}
	return nil
}

func (fakeService) ReleaseRehab(context.Context, int64, int64) error { return nil }

func (fakeService) AddForeverBan(context.Context, ForeverBanInput) error {
	return nil
}

func (fakeService) UpdateRehabBrainwash(context.Context, RehabBrainwashInput) error {
	return nil
}

func (fakeService) UpdateRehabReleaseExam(context.Context, RehabReleaseExamInput) error {
	return nil
}

func (fakeService) Users(context.Context, int64) ([]UserRow, error) { return nil, nil }

func (fakeService) Moderation(context.Context, int64) (ModerationPage, error) {
	return ModerationPage{}, nil
}

func (fakeService) Warnings(context.Context, int64) ([]WarningRow, error) { return nil, nil }

func (fakeService) Karma(context.Context, int64) ([]KarmaRow, error) { return nil, nil }

func (fakeService) OwnerSleep(context.Context) (OwnerSleepPage, error) { return OwnerSleepPage{}, nil }

func (fakeService) ClearOwnerSleep(context.Context, int64) error { return nil }

func (fakeService) LayaControl(context.Context) (LayaControlPage, error) {
	return LayaControlPage{ConfigSystemPrompt: "config", ActiveSystemPrompt: "active"}, nil
}

func (fakeService) UpdateLayaControl(context.Context, LayaControlInput) error { return nil }

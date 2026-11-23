package main

import (
	"context"
	"errors"
	"fmt"
	wardenweb "mirohodec-warden/internal/web"
	"time"
)

const (
	mafiaMiniChannelPublic = "PUBLIC"
	mafiaMiniChannelMafia  = "MAFIA"
	mafiaMiniChannelDead   = "DEAD"
	mafiaMiniChannelHost   = "HOST"
	mafiaMiniChannelSystem = "SYSTEM"
)

type mafiaMiniAuthUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	IsBot     bool   `json:"is_bot"`
}

func (s webPanelService) MafiaMiniAppAuth(ctx context.Context, input wardenweb.MafiaMiniAppAuthInput) (wardenweb.MafiaMiniAppAuthPayload, error) {
	user, dev, err := s.mafiaMiniAppUser(ctx, input)
	if err != nil {
		return wardenweb.MafiaMiniAppAuthPayload{}, err
	}
	if err := s.mafiaMiniEnsureRegistered(user); err != nil {
		return wardenweb.MafiaMiniAppAuthPayload{}, err
	}
	return wardenweb.MafiaMiniAppAuthPayload{
		User:        mafiaMiniAppWebUser(user),
		Rooms:       s.mafiaMiniRoomSummaries(user.ID),
		PublicRooms: s.mafiaMiniPublicRoomSummaries(user.ID),
		Leaderboard: s.mafiaMiniLeaderboard(10),
		Profile:     s.mafiaMiniProfile(user.ID),
		History:     s.mafiaMiniHistory(user.ID),
		DevAuth:     dev,
	}, nil
}

func (s webPanelService) MafiaMiniAppUpdateProfile(ctx context.Context, auth wardenweb.MafiaMiniAppAuthInput, input wardenweb.MafiaMiniAppProfileInput) (wardenweb.MafiaMiniAppAuthPayload, error) {
	user, dev, err := s.mafiaMiniAppUser(ctx, auth)
	if err != nil {
		return wardenweb.MafiaMiniAppAuthPayload{}, err
	}
	if err := s.mafiaMiniEnsureRegistered(user); err != nil {
		return wardenweb.MafiaMiniAppAuthPayload{}, err
	}
	rec, ok := s.app.store.MafiaUser(user.ID)
	if !ok {
		return wardenweb.MafiaMiniAppAuthPayload{}, errMafiaUserNotRegistered
	}
	profile := rec.Profile
	profile.AvatarURL = input.AvatarURL
	profile.AvatarText = input.AvatarText
	profile.AvatarTone = input.AvatarTone
	if err := s.app.store.SetMafiaProfile(user.ID, profile); err != nil {
		return wardenweb.MafiaMiniAppAuthPayload{}, err
	}
	return wardenweb.MafiaMiniAppAuthPayload{
		User:        mafiaMiniAppWebUser(user),
		Rooms:       s.mafiaMiniRoomSummaries(user.ID),
		PublicRooms: s.mafiaMiniPublicRoomSummaries(user.ID),
		Leaderboard: s.mafiaMiniLeaderboard(10),
		Profile:     s.mafiaMiniProfile(user.ID),
		History:     s.mafiaMiniHistory(user.ID),
		DevAuth:     dev,
	}, nil
}

func (s webPanelService) MafiaMiniAppProfile(ctx context.Context, auth wardenweb.MafiaMiniAppAuthInput, userID int64) (wardenweb.MafiaMiniAppProfilePayload, error) {
	user, _, err := s.mafiaMiniAppUser(ctx, auth)
	if err != nil {
		return wardenweb.MafiaMiniAppProfilePayload{}, err
	}
	if err := s.mafiaMiniEnsureRegistered(user); err != nil {
		return wardenweb.MafiaMiniAppProfilePayload{}, err
	}
	if userID == 0 {
		return wardenweb.MafiaMiniAppProfilePayload{}, errors.New("user id is required")
	}
	return wardenweb.MafiaMiniAppProfilePayload{Profile: s.mafiaMiniProfile(userID)}, nil
}

func (s webPanelService) MafiaMiniAppLobby(ctx context.Context, auth wardenweb.MafiaMiniAppAuthInput) (wardenweb.MafiaMiniAppLobbyPayload, error) {
	user, _, err := s.mafiaMiniAppUser(ctx, auth)
	if err != nil {
		return wardenweb.MafiaMiniAppLobbyPayload{}, err
	}
	if err := s.mafiaMiniEnsureRegistered(user); err != nil {
		return wardenweb.MafiaMiniAppLobbyPayload{}, err
	}
	return wardenweb.MafiaMiniAppLobbyPayload{
		User:        mafiaMiniAppWebUser(user),
		Rooms:       s.mafiaMiniRoomSummaries(user.ID),
		PublicRooms: s.mafiaMiniPublicRoomSummaries(user.ID),
		Leaderboard: s.mafiaMiniLeaderboard(10),
		Profile:     s.mafiaMiniProfile(user.ID),
		History:     s.mafiaMiniHistory(user.ID),
	}, nil
}

func (s webPanelService) MafiaMiniAppCreateRoom(ctx context.Context, auth wardenweb.MafiaMiniAppAuthInput) (wardenweb.MafiaMiniAppState, error) {
	user, _, err := s.mafiaMiniAppUser(ctx, auth)
	if err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	if err := s.mafiaMiniEnsureRegistered(user); err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	room, err := s.app.mafiaService().CreateRoom(mafiaMiniTeleUser(user), time.Now().UTC())
	if err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	_ = s.app.mafiaService().AddMiniAppSystemMessage(room.ID, "Лайя открыла комнату. Игроки могут входить по коду.", true, time.Now().UTC())
	return s.mafiaMiniStateForPlayer(room.ID, user.ID)
}

func (s webPanelService) MafiaMiniAppJoinRoom(ctx context.Context, auth wardenweb.MafiaMiniAppAuthInput, roomID string) (wardenweb.MafiaMiniAppState, error) {
	user, _, err := s.mafiaMiniAppUser(ctx, auth)
	if err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	if err := s.mafiaMiniEnsureRegistered(user); err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	room, err := s.app.mafiaService().JoinRoom(roomID, mafiaMiniTeleUser(user), time.Now().UTC())
	if err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	_ = s.app.mafiaService().AddMiniAppSystemMessage(room.ID, fmt.Sprintf("%s вошёл в лобби.", mafiaMiniDisplayName(user)), false, time.Now().UTC())
	return s.mafiaMiniStateForPlayer(room.ID, user.ID)
}

func (s webPanelService) MafiaMiniAppRoomState(ctx context.Context, auth wardenweb.MafiaMiniAppAuthInput, roomID string) (wardenweb.MafiaMiniAppState, error) {
	user, _, err := s.mafiaMiniAppUser(ctx, auth)
	if err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	return s.mafiaMiniStateForPlayer(roomID, user.ID)
}

func (s webPanelService) MafiaMiniAppSendMessage(ctx context.Context, auth wardenweb.MafiaMiniAppAuthInput, input wardenweb.MafiaMiniAppMessageInput) (wardenweb.MafiaMiniAppState, error) {
	user, _, err := s.mafiaMiniAppUser(ctx, auth)
	if err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	if _, err := s.app.mafiaService().AddMiniAppMessage(input.RoomID, user.ID, input.Channel, input.Text, input.ReplyToID, time.Now().UTC()); err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	return s.mafiaMiniStateForPlayer(input.RoomID, user.ID)
}

func (s webPanelService) MafiaMiniAppSaveWill(ctx context.Context, auth wardenweb.MafiaMiniAppAuthInput, input wardenweb.MafiaMiniAppWillInput) (wardenweb.MafiaMiniAppState, error) {
	user, _, err := s.mafiaMiniAppUser(ctx, auth)
	if err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	if _, err := s.app.mafiaService().SaveWill(input.RoomID, user.ID, input.Text, time.Now().UTC()); err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	return s.mafiaMiniStateForPlayer(input.RoomID, user.ID)
}

func (s webPanelService) MafiaMiniAppSkipPhase(ctx context.Context, auth wardenweb.MafiaMiniAppAuthInput, roomID string) (wardenweb.MafiaMiniAppState, error) {
	user, _, err := s.mafiaMiniAppUser(ctx, auth)
	if err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	now := time.Now().UTC()
	updated, votes, needed, reached, err := s.app.mafiaService().VotePhaseSkip(roomID, user.ID)
	if err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	text := fmt.Sprintf("Голос за пропуск фазы принят: %d/%d.", votes, needed)
	if reached {
		text = fmt.Sprintf("Лайя пропускает фазу: %d/%d игроков готовы.", votes, needed)
	}
	_ = s.app.mafiaService().AddMiniAppSystemMessage(updated.ID, text, reached, now)
	if reached {
		if err := s.app.advanceMafiaRoomAsLaya(updated.ID, now); err != nil {
			return wardenweb.MafiaMiniAppState{}, err
		}
	}
	return s.mafiaMiniStateForPlayer(updated.ID, user.ID)
}

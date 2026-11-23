package main

import (
	"context"
	"errors"
	"fmt"
	wardenweb "mirohodec-warden/internal/web"
	"strings"
	"time"
)

func (s webPanelService) MafiaMiniAppLeaveRoom(ctx context.Context, auth wardenweb.MafiaMiniAppAuthInput, roomID string) (wardenweb.MafiaMiniAppLobbyPayload, error) {
	user, _, err := s.mafiaMiniAppUser(ctx, auth)
	if err != nil {
		return wardenweb.MafiaMiniAppLobbyPayload{}, err
	}
	room, ok := s.app.store.MafiaRoom(roomID)
	if !ok {
		return wardenweb.MafiaMiniAppLobbyPayload{}, errMafiaRoomNotFound
	}
	if room.Status != MafiaStatusLobby {
		return wardenweb.MafiaMiniAppLobbyPayload{}, errMafiaSettingsLocked
	}
	if room.OwnerID == user.ID {
		return wardenweb.MafiaMiniAppLobbyPayload{}, errors.New("создатель может закрыть лобби, но не выйти из него")
	}
	idx := mafiaRoomPlayerIndex(room, user.ID)
	if idx < 0 {
		return wardenweb.MafiaMiniAppLobbyPayload{}, errMafiaPlayerNotFound
	}
	leftName := mafiaRoomPlayerLabel(room, room.Players[idx])
	room.Players = append(room.Players[:idx], room.Players[idx+1:]...)
	room.UpdatedAt = time.Now().UTC().Unix()
	if err := s.app.store.SaveMafiaRoom(room); err != nil {
		return wardenweb.MafiaMiniAppLobbyPayload{}, err
	}
	_ = s.app.mafiaService().AddMiniAppSystemMessage(room.ID, fmt.Sprintf("%s покинул лобби.", leftName), false, time.Now().UTC())
	return s.mafiaMiniLobbyPayload(user), nil
}

func (s webPanelService) MafiaMiniAppCloseRoom(ctx context.Context, auth wardenweb.MafiaMiniAppAuthInput, roomID string) (wardenweb.MafiaMiniAppLobbyPayload, error) {
	user, _, err := s.mafiaMiniAppUser(ctx, auth)
	if err != nil {
		return wardenweb.MafiaMiniAppLobbyPayload{}, err
	}
	room, ok := s.app.store.MafiaRoom(roomID)
	if !ok {
		return wardenweb.MafiaMiniAppLobbyPayload{}, errMafiaRoomNotFound
	}
	if room.OwnerID != user.ID {
		return wardenweb.MafiaMiniAppLobbyPayload{}, errMafiaNotRoomOwner
	}
	if room.Status == MafiaStatusLobby || room.Status == MafiaStatusCancelled || room.Status == MafiaStatusFinished {
		if room.Status == MafiaStatusFinished && room.Winner != "" && !room.EloProcessed {
			finishedAt := time.Now().UTC()
			if room.FinishedAt > 0 {
				finishedAt = time.Unix(room.FinishedAt, 0).UTC()
			} else if room.UpdatedAt > 0 {
				finishedAt = time.Unix(room.UpdatedAt, 0).UTC()
			}
			if _, _, _, _, err := s.app.store.ApplyMafiaGameResult(room.ID, room.Winner, finishedAt); err != nil {
				return wardenweb.MafiaMiniAppLobbyPayload{}, err
			}
		}
		if err := s.app.store.DeleteMafiaRoom(room.ID); err != nil {
			return wardenweb.MafiaMiniAppLobbyPayload{}, err
		}
		return s.mafiaMiniLobbyPayload(user), nil
	}
	if _, err := s.app.mafiaService().EndGameByOwner(room.ID, user.ID, time.Now().UTC()); err != nil {
		return wardenweb.MafiaMiniAppLobbyPayload{}, err
	}
	return s.mafiaMiniLobbyPayload(user), nil
}

func (s webPanelService) mafiaMiniLobbyPayload(user mafiaMiniAuthUser) wardenweb.MafiaMiniAppLobbyPayload {
	return wardenweb.MafiaMiniAppLobbyPayload{
		User:        mafiaMiniAppWebUser(user),
		Rooms:       s.mafiaMiniRoomSummaries(user.ID),
		PublicRooms: s.mafiaMiniPublicRoomSummaries(user.ID),
		Leaderboard: s.mafiaMiniLeaderboard(10),
		Profile:     s.mafiaMiniProfile(user.ID),
		History:     s.mafiaMiniHistory(user.ID),
	}
}

func (s webPanelService) mafiaMiniAppUser(ctx context.Context, input wardenweb.MafiaMiniAppAuthInput) (mafiaMiniAuthUser, bool, error) {
	if s.app == nil || s.app.store == nil {
		return mafiaMiniAuthUser{}, false, errors.New("app is not ready")
	}
	if mafiaMiniDevAuthEnabled() {
		userID := input.DevUserID
		if userID == 0 {
			userID = 1000001
		}
		name := strings.TrimSpace(input.DevName)
		if name == "" {
			name = "Dev Player"
		}
		username := strings.TrimPrefix(strings.TrimSpace(input.DevUsername), "@")
		return mafiaMiniAuthUser{ID: userID, Username: username, FirstName: name}, true, nil
	}
	token := strings.TrimSpace(s.app.config().Telegram.Token)
	if token == "" {
		return mafiaMiniAuthUser{}, false, errors.New("telegram token is not configured")
	}
	user, err := validateTelegramMiniAppInitData(input.InitData, token, 24*time.Hour)
	if err != nil {
		return mafiaMiniAuthUser{}, false, err
	}
	if user.ID == 0 || user.IsBot {
		return mafiaMiniAuthUser{}, false, errMafiaUserNotRegistered
	}
	return user, false, nil
}

func (s webPanelService) mafiaMiniEnsureRegistered(user mafiaMiniAuthUser) error {
	if s.app == nil || s.app.store == nil {
		return errors.New("app is not ready")
	}
	if user.ID == 0 || user.IsBot {
		return errMafiaUserNotRegistered
	}
	if _, ok := s.app.store.MafiaUser(user.ID); ok {
		return nil
	}
	_, err := s.app.store.RegisterMafiaUser(mafiaMiniTeleUser(user), time.Now().UTC())
	return err
}

func (s webPanelService) mafiaMiniStateForPlayer(roomID string, viewerID int64) (wardenweb.MafiaMiniAppState, error) {
	room, ok := s.app.store.MafiaRoom(roomID)
	if !ok {
		return wardenweb.MafiaMiniAppState{}, errMafiaRoomNotFound
	}
	viewerIdx := mafiaRoomPlayerIndex(room, viewerID)
	hasOwnerAccess := room.OwnerID == viewerID
	if !hasOwnerAccess && viewerIdx < 0 {
		return wardenweb.MafiaMiniAppState{}, errMafiaPlayerNotFound
	}
	now := time.Now().UTC()
	settings := normalizeMafiaSettings(room.Settings)
	isHost := hasOwnerAccess && settings.EnableHostPowers
	me := s.mafiaMiniMe(room, viewerID, viewerIdx, isHost)
	channels := s.mafiaMiniChannels(room, viewerIdx, isHost)
	action := s.mafiaMiniAction(room, viewerID, viewerIdx, isHost)
	return wardenweb.MafiaMiniAppState{
		Room:              mafiaMiniRoom(room, settings, now, viewerID, viewerIdx, hasOwnerAccess, isHost),
		Me:                me,
		Channels:          channels,
		AvailableChannels: mafiaMiniAvailableChannelIDs(channels),
		Messages:          s.mafiaMiniMessages(room, viewerID, viewerIdx, isHost),
		Players:           s.mafiaMiniPlayers(room, viewerID, viewerIdx, isHost),
		Log:               s.mafiaMiniLog(room, viewerID, viewerIdx, isHost),
		Action:            action,
		AvailableActions:  mafiaMiniAvailableActions(action),
		ServerTime:        now.Unix(),
		FinalResult:       s.mafiaMiniFinalResult(room, viewerID, viewerIdx),
		Warnings:          mafiaMiniWarnings(room),
	}, nil
}

func (s webPanelService) mafiaMiniRoomSummaries(viewerID int64) []wardenweb.MafiaMiniAppRoomSummary {
	rooms := s.app.store.MafiaRoomsForUser(viewerID)
	out := make([]wardenweb.MafiaMiniAppRoomSummary, 0, len(rooms))
	for _, room := range rooms {
		out = append(out, mafiaMiniRoomSummary(room, viewerID))
	}
	return out
}

func (s webPanelService) mafiaMiniPublicRoomSummaries(viewerID int64) []wardenweb.MafiaMiniAppRoomSummary {
	rooms := s.app.store.MafiaRooms()
	out := make([]wardenweb.MafiaMiniAppRoomSummary, 0, len(rooms))
	for _, room := range rooms {
		settings := normalizeMafiaSettings(room.Settings)
		if room.Status != MafiaStatusLobby || !settings.PublicLobby || len(room.Players) >= settings.MaxPlayers {
			continue
		}
		out = append(out, mafiaMiniRoomSummary(room, viewerID))
	}
	if len(out) > 30 {
		return out[:30]
	}
	return out
}

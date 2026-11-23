package main

import (
	"context"
	"errors"
	"fmt"
	wardenweb "mirohodec-warden/internal/web"
	"strings"
	"time"
)

func (s webPanelService) MafiaMiniAppSubmitVote(ctx context.Context, auth wardenweb.MafiaMiniAppAuthInput, input wardenweb.MafiaMiniAppVoteInput) (wardenweb.MafiaMiniAppState, error) {
	user, _, err := s.mafiaMiniAppUser(ctx, auth)
	if err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	target := strings.TrimSpace(input.Target)
	if target == "" {
		return wardenweb.MafiaMiniAppState{}, errors.New("target is required")
	}
	if _, err := s.app.mafiaService().Vote(input.RoomID, user.ID, target, time.Now().UTC()); err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	return s.mafiaMiniStateForPlayer(input.RoomID, user.ID)
}

func (s webPanelService) MafiaMiniAppSubmitNightAction(ctx context.Context, auth wardenweb.MafiaMiniAppAuthInput, input wardenweb.MafiaMiniAppNightActionInput) (wardenweb.MafiaMiniAppState, error) {
	user, _, err := s.mafiaMiniAppUser(ctx, auth)
	if err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	switch strings.TrimSpace(input.Action) {
	case "kill":
		_, err = s.app.mafiaService().MafiaKill(input.RoomID, user.ID, input.TargetID, time.Now().UTC())
	case "heal":
		_, err = s.app.mafiaService().DoctorHeal(input.RoomID, user.ID, input.TargetID, time.Now().UTC())
	case "check":
		var updated MafiaRoom
		var checked bool
		updated, checked, err = s.app.mafiaService().DetectiveCheck(input.RoomID, user.ID, input.TargetID, time.Now().UTC())
		if err == nil {
			_ = s.app.mafiaService().AddMiniAppPrivateSystemMessage(updated.ID, user.ID, mafiaMiniDetectiveResultText(updated, input.TargetID, checked), true, time.Now().UTC())
		}
	default:
		err = errors.New("unknown night action")
	}
	if err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	return s.mafiaMiniStateForPlayer(input.RoomID, user.ID)
}

func (s webPanelService) MafiaMiniAppStartGame(ctx context.Context, auth wardenweb.MafiaMiniAppAuthInput, roomID string) (wardenweb.MafiaMiniAppState, error) {
	user, _, err := s.mafiaMiniAppUser(ctx, auth)
	if err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	room, err := s.app.mafiaService().StartGame(roomID, user.ID, time.Now().UTC())
	if err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	_ = s.app.mafiaService().AddMiniAppSystemMessage(room.ID, "Игра началась. Наступила Ночь 1.", true, time.Now().UTC())
	s.app.announceMafiaNight(room)
	return s.mafiaMiniStateForPlayer(room.ID, user.ID)
}

func (s webPanelService) MafiaMiniAppNextPhase(ctx context.Context, auth wardenweb.MafiaMiniAppAuthInput, roomID string) (wardenweb.MafiaMiniAppState, error) {
	user, _, err := s.mafiaMiniAppUser(ctx, auth)
	if err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	room, ok := s.app.store.MafiaRoom(roomID)
	if !ok {
		return wardenweb.MafiaMiniAppState{}, errMafiaRoomNotFound
	}
	if room.OwnerID != user.ID {
		return wardenweb.MafiaMiniAppState{}, errMafiaNotRoomOwner
	}
	if !normalizeMafiaSettings(room.Settings).EnableHostPowers {
		return wardenweb.MafiaMiniAppState{}, errors.New("host powers are disabled for this room")
	}
	now := time.Now().UTC()
	switch room.Status {
	case MafiaStatusNight:
		updated, result, err := s.app.mafiaService().ResolveNight(room.ID, now)
		if err != nil {
			return wardenweb.MafiaMiniAppState{}, err
		}
		s.app.announceMafiaDayByLaya(updated, result)
	case MafiaStatusDayDiscussion:
		updated, err := s.app.mafiaService().StartVoting(room.ID, now)
		if err != nil {
			return wardenweb.MafiaMiniAppState{}, err
		}
		s.app.announceMafiaVotingByLaya(updated)
	case MafiaStatusVoting:
		updated, result, err := s.app.mafiaService().ResolveVoting(room.ID, now)
		if err != nil {
			return wardenweb.MafiaMiniAppState{}, err
		}
		s.app.announceMafiaVotingResolvedByLaya(updated, result)
	default:
		return wardenweb.MafiaMiniAppState{}, errMafiaWrongPhase
	}
	return s.mafiaMiniStateForPlayer(roomID, user.ID)
}

func (s webPanelService) MafiaMiniAppUpdateSettings(ctx context.Context, auth wardenweb.MafiaMiniAppAuthInput, roomID string, input wardenweb.MafiaMiniAppSettingsInput) (wardenweb.MafiaMiniAppState, error) {
	user, _, err := s.mafiaMiniAppUser(ctx, auth)
	if err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	room, err := s.app.mafiaService().UpdateLobbySettings(roomID, user.ID, time.Now().UTC(), func(settings MafiaRoomSettings) MafiaRoomSettings {
		return mafiaMiniMergeSettingsInput(settings, input)
	})
	if err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	_ = s.app.mafiaService().AddMiniAppSystemMessage(room.ID, "Лайя обновила настройки лобби.", false, time.Now().UTC())
	return s.mafiaMiniStateForPlayer(room.ID, user.ID)
}

func (s webPanelService) MafiaMiniAppKickPlayer(ctx context.Context, auth wardenweb.MafiaMiniAppAuthInput, input wardenweb.MafiaMiniAppKickInput) (wardenweb.MafiaMiniAppState, error) {
	user, _, err := s.mafiaMiniAppUser(ctx, auth)
	if err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	room, ok := s.app.store.MafiaRoom(input.RoomID)
	if !ok {
		return wardenweb.MafiaMiniAppState{}, errMafiaRoomNotFound
	}
	if room.OwnerID != user.ID {
		return wardenweb.MafiaMiniAppState{}, errMafiaNotRoomOwner
	}
	if room.Status != MafiaStatusLobby {
		return wardenweb.MafiaMiniAppState{}, errMafiaSettingsLocked
	}
	if input.TargetID == 0 {
		return wardenweb.MafiaMiniAppState{}, errors.New("targetId is required")
	}
	if input.TargetID == user.ID {
		return wardenweb.MafiaMiniAppState{}, errors.New("нельзя исключить себя")
	}
	idx := mafiaRoomPlayerIndex(room, input.TargetID)
	if idx < 0 {
		return wardenweb.MafiaMiniAppState{}, errMafiaPlayerNotFound
	}
	removedName := mafiaRoomPlayerLabel(room, room.Players[idx])
	room.Players = append(room.Players[:idx], room.Players[idx+1:]...)
	room.UpdatedAt = time.Now().UTC().Unix()
	if err := s.app.store.SaveMafiaRoom(room); err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	_ = s.app.mafiaService().AddMiniAppSystemMessage(room.ID, fmt.Sprintf("%s исключён из лобби.", removedName), false, time.Now().UTC())
	return s.mafiaMiniStateForPlayer(room.ID, user.ID)
}

func (s webPanelService) MafiaMiniAppUpdateTestBots(ctx context.Context, auth wardenweb.MafiaMiniAppAuthInput, input wardenweb.MafiaMiniAppTestBotsInput) (wardenweb.MafiaMiniAppState, error) {
	user, _, err := s.mafiaMiniAppUser(ctx, auth)
	if err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	action := strings.ToLower(strings.TrimSpace(input.Action))
	now := time.Now().UTC()
	var room MafiaRoom
	var count int
	switch action {
	case "add":
		var added []MafiaPlayer
		room, added, err = s.app.mafiaService().AddTestBots(input.RoomID, user.ID, input.Count, now)
		count = len(added)
	case "fill":
		var added []MafiaPlayer
		room, added, err = s.app.mafiaService().FillTestBotsToMinimum(input.RoomID, user.ID, now)
		count = len(added)
	case "clear":
		room, count, err = s.app.mafiaService().RemoveTestBots(input.RoomID, user.ID, now)
	default:
		return wardenweb.MafiaMiniAppState{}, errors.New("unknown test bot action")
	}
	if err != nil {
		return wardenweb.MafiaMiniAppState{}, err
	}
	switch action {
	case "clear":
		_ = s.app.mafiaService().AddMiniAppSystemMessage(room.ID, fmt.Sprintf("Лайя убрала тест-ботов: %d.", count), false, now)
	default:
		_ = s.app.mafiaService().AddMiniAppSystemMessage(room.ID, fmt.Sprintf("Лайя добавила тест-ботов: %d.", count), false, now)
	}
	return s.mafiaMiniStateForPlayer(room.ID, user.ID)
}

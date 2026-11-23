package main

import (
	"context"
	"time"

	"go.uber.org/zap"
)

const mafiaAutoHostTick = 5 * time.Second

func (a *App) StartMafiaAutoHost(ctx context.Context) {
	ticker := time.NewTicker(mafiaAutoHostTick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			a.runMafiaAutoHost(now.UTC())
		}
	}
}

func (a *App) runMafiaAutoHost(now time.Time) {
	if a == nil || a.store == nil {
		return
	}
	for _, room := range a.mafiaAutoHostRooms() {
		if err := a.advanceMafiaRoomAsLaya(room.ID, now); err != nil && a.log != nil {
			a.log.Warn("mafia auto host failed", zap.String("room", room.ID), zap.Error(err))
		}
	}
}

func (a *App) mafiaAutoHostRooms() []MafiaRoom {
	seen := map[string]struct{}{}
	var rooms []MafiaRoom
	for _, user := range a.store.MafiaUsers() {
		for _, room := range a.store.MafiaRoomsForUser(user.TelegramID) {
			if !mafiaAutoHostStatus(room.Status) {
				continue
			}
			if _, ok := seen[room.ID]; ok {
				continue
			}
			seen[room.ID] = struct{}{}
			rooms = append(rooms, room)
		}
	}
	return rooms
}

func mafiaAutoHostStatus(status MafiaRoomStatus) bool {
	switch status {
	case MafiaStatusNight, MafiaStatusDayDiscussion, MafiaStatusVoting:
		return true
	default:
		return false
	}
}

func (a *App) advanceMafiaRoomAsLaya(roomID string, now time.Time) error {
	room, ok := a.store.MafiaRoom(roomID)
	if !ok || !mafiaAutoHostStatus(room.Status) || room.PendingLeakOffenderID != 0 {
		return nil
	}
	settings := normalizeMafiaSettings(room.Settings)
	switch room.Status {
	case MafiaStatusNight:
		if !mafiaPhaseDue(room, now, time.Duration(settings.NightActionSeconds)*time.Second) && !mafiaNightReadyForLaya(room) && !mafiaPhaseSkipReached(room) {
			return nil
		}
		updated, result, err := a.mafiaService().ResolveNight(room.ID, now)
		if err != nil {
			return err
		}
		a.announceMafiaDayByLaya(updated, result)
	case MafiaStatusDayDiscussion:
		if !mafiaPhaseDue(room, now, time.Duration(settings.DayDiscussionSeconds)*time.Second) && !mafiaPhaseSkipReached(room) {
			return nil
		}
		updated, err := a.mafiaService().StartVoting(room.ID, now)
		if err != nil {
			return err
		}
		a.announceMafiaVotingByLaya(updated)
	case MafiaStatusVoting:
		if !mafiaPhaseDue(room, now, time.Duration(settings.VotingSeconds)*time.Second) && !mafiaVotingReadyForLaya(room) && !mafiaPhaseSkipReached(room) {
			return nil
		}
		updated, result, err := a.mafiaService().ResolveVoting(room.ID, now)
		if err != nil {
			return err
		}
		a.announceMafiaVotingResolvedByLaya(updated, result)
	}
	return nil
}

func mafiaPhaseSkipReached(room MafiaRoom) bool {
	needed := mafiaPhaseSkipNeeded(room)
	return needed > 0 && mafiaPhaseSkipVotesCount(room) >= needed
}

func mafiaPhaseSkipVotesCount(room MafiaRoom) int {
	count := 0
	for _, player := range room.Players {
		if !player.Alive || player.IsTestBot {
			continue
		}
		if room.PhaseSkipVotes[idKey(player.TelegramID)] {
			count++
		}
	}
	return count
}

func mafiaPhaseSkipNeeded(room MafiaRoom) int {
	aliveReal := 0
	for _, player := range room.Players {
		if player.Alive && !player.IsTestBot {
			aliveReal++
		}
	}
	if aliveReal <= 0 {
		return 0
	}
	return aliveReal/2 + 1
}

func mafiaPhaseDue(room MafiaRoom, now time.Time, duration time.Duration) bool {
	if duration <= 0 {
		return true
	}
	startedAt := room.PhaseStartedAt
	if startedAt <= 0 {
		startedAt = room.UpdatedAt
	}
	if startedAt <= 0 {
		return true
	}
	return !now.Before(time.Unix(startedAt, 0).Add(duration))
}

func mafiaNightReadyForLaya(room MafiaRoom) bool {
	if room.Status != MafiaStatusNight {
		return false
	}
	hasAction := false
	for _, player := range room.Players {
		if !player.Alive {
			continue
		}
		switch player.Role {
		case MafiaRoleMafia:
			hasAction = true
			if player.IsTestBot {
				continue
			}
			if _, ok := room.NightActions.MafiaVotes[idKey(player.TelegramID)]; !ok {
				return false
			}
		case MafiaRoleDoctor:
			hasAction = true
			if player.IsTestBot {
				continue
			}
			if room.NightActions.DoctorTargetID == 0 {
				return false
			}
		case MafiaRoleDetective:
			hasAction = true
			if player.IsTestBot {
				continue
			}
			if _, ok := room.NightActions.DetectiveChecks[idKey(player.TelegramID)]; !ok {
				return false
			}
		}
	}
	return hasAction
}

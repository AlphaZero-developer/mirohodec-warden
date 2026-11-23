package main

import (
	"time"
)

func (s *MafiaService) AutoFillFromQueue(roomID string, ownerID int64, now time.Time) (MafiaRoom, []int64, error) {
	room, err := s.ownerLobbyRoom(roomID, ownerID)
	if err != nil {
		return MafiaRoom{}, nil, err
	}
	settings := normalizeMafiaSettings(room.Settings)
	if len(room.Players) >= settings.MaxPlayers {
		return MafiaRoom{}, nil, errMafiaRoomFull
	}
	queue := s.store.MafiaReadyQueue()
	remaining := make([]MafiaReadyQueueEntry, 0, len(queue))
	var added []int64
	for _, entry := range queue {
		if len(room.Players) >= settings.MaxPlayers {
			remaining = append(remaining, entry)
			continue
		}
		if mafiaRoomPlayerIndex(room, entry.TelegramID) >= 0 {
			continue
		}
		if s.store.MafiaHostBlocked(room.OwnerID, entry.TelegramID) {
			continue
		}
		rec, ok := s.store.MafiaUser(entry.TelegramID)
		if !ok {
			continue
		}
		room.Players = append(room.Players, mafiaPlayerFromUser(rec, now))
		added = append(added, rec.TelegramID)
	}
	room.UpdatedAt = now.UTC().Unix()
	if err := s.store.SaveMafiaReadyQueue(remaining); err != nil {
		return MafiaRoom{}, nil, err
	}
	return room, added, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) PrepareInviteBroadcast(roomID string, ownerID int64, favoritesOnly bool, now time.Time) (MafiaRoom, []MafiaRegisteredUser, error) {
	room, err := s.ownerLobbyRoom(roomID, ownerID)
	if err != nil {
		return MafiaRoom{}, nil, err
	}
	if len(room.Players) >= normalizeMafiaSettings(room.Settings).MaxPlayers {
		return MafiaRoom{}, nil, errMafiaRoomFull
	}
	if !favoritesOnly {
		if room.InviteBroadcastCount >= mafiaInviteLimits.MaxInviteBroadcastPerRoom ||
			(room.LastInviteBroadcastAt != 0 && now.UTC().Unix()-room.LastInviteBroadcastAt < int64(mafiaInviteLimits.MinSecondsBetweenInviteBroadcasts)) {
			return MafiaRoom{}, nil, errMafiaInviteLimited
		}
	}
	owner, _ := s.store.MafiaUser(ownerID)
	favorites := int64Set(owner.FavoriteMafiaUserIDs)
	today := now.UTC().Format("2006-01-02")
	recipients := make([]MafiaRegisteredUser, 0, 16)
	for _, user := range s.store.MafiaUsers() {
		if user.TelegramID == ownerID || mafiaRoomPlayerIndex(room, user.TelegramID) >= 0 {
			continue
		}
		if s.store.MafiaHostBlocked(ownerID, user.TelegramID) {
			continue
		}
		if favoritesOnly {
			if _, ok := favorites[user.TelegramID]; !ok {
				continue
			}
		}
		settings := normalizeMafiaUserSettings(user.Settings)
		if !settings.AllowMafiaInvites {
			continue
		}
		if settings.InviteOnlyFromFavorites {
			if _, ok := favorites[user.TelegramID]; !ok {
				continue
			}
		}
		if settings.InviteDay != today {
			settings.InviteDay = today
			settings.InvitesToday = 0
		}
		if settings.InvitesToday >= mafiaInviteLimits.MaxInvitesPerUserPerDay {
			continue
		}
		settings.InvitesToday++
		settings.LastInviteAt = now.UTC().Unix()
		user.Settings = settings
		if err := s.store.SetMafiaUserSettings(user.TelegramID, settings); err != nil {
			continue
		}
		recipients = append(recipients, user)
	}
	if !favoritesOnly {
		room.InviteBroadcastCount++
		room.LastInviteBroadcastAt = now.UTC().Unix()
	}
	room.UpdatedAt = now.UTC().Unix()
	return room, recipients, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) SetInvitePermission(userID int64, allow bool) error {
	rec, ok := s.store.MafiaUser(userID)
	if !ok {
		return errMafiaUserNotRegistered
	}
	settings := normalizeMafiaUserSettings(rec.Settings)
	settings.AllowMafiaInvites = allow
	return s.store.SetMafiaUserSettings(userID, settings)
}

func (s *MafiaService) ToggleInvitePermission(userID int64) (MafiaUserSettings, error) {
	rec, ok := s.store.MafiaUser(userID)
	if !ok {
		return MafiaUserSettings{}, errMafiaUserNotRegistered
	}
	settings := normalizeMafiaUserSettings(rec.Settings)
	settings.AllowMafiaInvites = !settings.AllowMafiaInvites
	if err := s.store.SetMafiaUserSettings(userID, settings); err != nil {
		return MafiaUserSettings{}, err
	}
	return settings, nil
}

func (s *MafiaService) AddFavorite(ownerID, userID int64) error {
	owner, ok := s.store.MafiaUser(ownerID)
	if !ok {
		return errMafiaUserNotRegistered
	}
	if _, ok := s.store.MafiaUser(userID); !ok {
		return errMafiaUserNotRegistered
	}
	favorites := append(append([]int64(nil), owner.FavoriteMafiaUserIDs...), userID)
	return s.store.SetMafiaUserFavorites(ownerID, favorites)
}

func (s *MafiaService) recordMafiaRoleReveal(userID int64) {
	if s == nil || s.store == nil {
		return
	}
	rec, ok := s.store.MafiaUser(userID)
	if !ok {
		return
	}
	rec.Stats.RoleReveals++
	_ = s.store.SetMafiaUserStats(userID, rec.Stats)
}

func (s *MafiaService) recordMafiaCancelledGameCaused(userID int64) {
	if s == nil || s.store == nil {
		return
	}
	rec, ok := s.store.MafiaUser(userID)
	if !ok {
		return
	}
	rec.Stats.CancelledGamesCaused++
	_ = s.store.SetMafiaUserStats(userID, rec.Stats)
}

func (s *MafiaService) finishMafiaRoom(room *MafiaRoom, winner string) {
	if s == nil || s.store == nil || room == nil || winner == "" || room.Status == MafiaStatusFinished {
		return
	}
	finishedAt := time.Now().UTC()
	if room.UpdatedAt > 0 {
		finishedAt = time.Unix(room.UpdatedAt, 0).UTC()
	}
	updated, _, _, _, err := s.store.ApplyMafiaGameResult(room.ID, winner, finishedAt)
	if err != nil {
		room.Status = MafiaStatusFinished
		room.Winner = winner
		room.FinishedAt = finishedAt.Unix()
		room.EloProcessed = true
		return
	}
	*room = updated
}

func mafiaNormalizedElo(value int) int {
	if value <= 0 {
		return 1000
	}
	return value
}

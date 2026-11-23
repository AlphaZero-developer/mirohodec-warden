package main

import (
	"sort"
	"strings"
	"time"
)

func (a *App) searchMafiaUsersForInvite(room MafiaRoom, query string, limit int) []MafiaRegisteredUser {
	if a == nil || a.store == nil || limit <= 0 {
		return nil
	}
	query = normalizeFAQText(strings.TrimSpace(strings.TrimPrefix(query, "@")))
	if query == "" {
		return nil
	}
	var out []MafiaRegisteredUser
	for _, user := range a.store.MafiaUsers() {
		if user.TelegramID == room.OwnerID || mafiaRoomPlayerIndex(room, user.TelegramID) >= 0 {
			continue
		}
		if a.store.MafiaHostBlocked(room.OwnerID, user.TelegramID) {
			continue
		}
		name := normalizeFAQText(mafiaUserCharacterName(user))
		displayName := normalizeFAQText(user.DisplayName)
		username := normalizeFAQText(user.Username)
		if strings.Contains(name, query) || strings.Contains(displayName, query) || strings.Contains(username, query) {
			out = append(out, user)
			if len(out) >= limit {
				break
			}
		}
	}
	return out
}

func (a *App) sendDirectMafiaInvite(room MafiaRoom, ownerID, targetID int64) error {
	if a == nil || a.store == nil {
		return errMafiaUserNotRegistered
	}
	if room.OwnerID != ownerID {
		return errMafiaNotRoomOwner
	}
	if room.Status != MafiaStatusLobby {
		return errMafiaRoomNotLobby
	}
	if len(room.Players) >= normalizeMafiaSettings(room.Settings).MaxPlayers {
		return errMafiaRoomFull
	}
	if mafiaRoomPlayerIndex(room, targetID) >= 0 {
		return nil
	}
	if a.store.MafiaHostBlocked(ownerID, targetID) {
		return errMafiaHostBlocked
	}
	user, ok := a.store.MafiaUser(targetID)
	if !ok {
		return errMafiaUserNotRegistered
	}
	owner, _ := a.store.MafiaUser(ownerID)
	favorites := int64Set(owner.FavoriteMafiaUserIDs)
	settings := normalizeMafiaUserSettings(user.Settings)
	if !settings.AllowMafiaInvites {
		return errMafiaInvitesDisabled
	}
	if settings.InviteOnlyFromFavorites {
		if _, ok := favorites[targetID]; !ok {
			return errMafiaInvitesDisabled
		}
	}
	now := time.Now().UTC()
	today := now.Format("2006-01-02")
	if settings.InviteDay != today {
		settings.InviteDay = today
		settings.InvitesToday = 0
	}
	if settings.InvitesToday >= mafiaInviteLimits.MaxInvitesPerUserPerDay {
		return errMafiaInviteLimited
	}
	settings.InvitesToday++
	settings.LastInviteAt = now.Unix()
	if err := a.store.SetMafiaUserSettings(targetID, settings); err != nil {
		return err
	}
	a.sendMafiaDM(targetID, mafiaInviteText(room), mafiaInviteDMMarkup(room))
	return nil
}

func (a *App) activeMafiaRoomForUser(userID int64) (MafiaRoom, bool) {
	for _, room := range a.store.MafiaRoomsForUser(userID) {
		if room.Status != MafiaStatusFinished && room.Status != MafiaStatusCancelled && mafiaRoomPlayerIndex(room, userID) >= 0 {
			return room, true
		}
	}
	return MafiaRoom{}, false
}

func (a *App) mafiaRoomForGroupMessage(chatID, userID int64) (MafiaRoom, bool) {
	if a == nil || a.store == nil {
		return MafiaRoom{}, false
	}
	rooms := a.mafiaActiveRoomsForGroup(chatID)
	seen := make(map[string]bool, len(rooms))
	for _, room := range rooms {
		seen[normalizeMafiaRoomID(room.ID)] = true
	}
	for _, room := range a.store.MafiaRoomsForUser(userID) {
		if !mafiaGroupRoomAcceptsMessage(room, chatID) {
			continue
		}
		id := normalizeMafiaRoomID(room.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		rooms = append(rooms, room)
	}
	sort.Slice(rooms, func(i, j int) bool {
		return mafiaRoomActivityScore(rooms[i]) > mafiaRoomActivityScore(rooms[j])
	})
	var best MafiaRoom
	bestScore := int64(-1)
	for _, room := range rooms {
		if mafiaRoomPlayerIndex(room, userID) < 0 {
			continue
		}
		score := mafiaRoomActivityScore(room)
		if score > bestScore {
			best = room
			bestScore = score
		}
	}
	if bestScore >= 0 {
		return best, true
	}
	for _, room := range rooms {
		score := mafiaRoomActivityScore(room)
		if score > bestScore {
			best = room
			bestScore = score
		}
	}
	if bestScore >= 0 {
		return best, true
	}
	return MafiaRoom{}, false
}

func (a *App) mafiaActiveRoomsForGroup(chatID int64) []MafiaRoom {
	if a == nil || a.store == nil {
		return nil
	}
	seen := map[string]bool{}
	var rooms []MafiaRoom
	add := func(room MafiaRoom) {
		if !mafiaGroupRoomAcceptsMessage(room, chatID) {
			return
		}
		id := normalizeMafiaRoomID(room.ID)
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		rooms = append(rooms, room)
	}
	if room, ok := a.store.MafiaRoomByGroup(chatID); ok {
		add(room)
	}
	for _, user := range a.store.MafiaUsers() {
		for _, room := range a.store.MafiaRoomsForUser(user.TelegramID) {
			add(room)
		}
	}
	sort.Slice(rooms, func(i, j int) bool {
		return mafiaRoomActivityScore(rooms[i]) > mafiaRoomActivityScore(rooms[j])
	})
	return rooms
}

func mafiaGroupRoomAcceptsMessage(room MafiaRoom, chatID int64) bool {
	return room.GroupChatID == chatID && room.Status != MafiaStatusFinished && room.Status != MafiaStatusCancelled && mafiaRoomStatusAcceptsPublicDiscussion(room.Status)
}

func mafiaRoomActivityScore(room MafiaRoom) int64 {
	score := room.UpdatedAt
	if room.PhaseStartedAt > score {
		score = room.PhaseStartedAt
	}
	if room.CreatedAt > score {
		score = room.CreatedAt
	}
	return score
}

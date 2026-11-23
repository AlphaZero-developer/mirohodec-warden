package main

import (
	"fmt"
	"time"
)

func mafiaNewTestBotPlayer(room MafiaRoom, now time.Time) MafiaPlayer {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	index := mafiaTestBotCount(room)
	profile := mafiaTestBotProfiles[index%len(mafiaTestBotProfiles)]
	round := index/len(mafiaTestBotProfiles) + 1
	name := profile.Name
	if round > 1 {
		name = fmt.Sprintf("%s %d", profile.Name, round)
	}
	return MafiaPlayer{
		TelegramID:  mafiaNextTestBotID(room),
		Username:    fmt.Sprintf("test_bot_%d", index+1),
		DisplayName: name,
		Backstory:   profile.Backstory,
		IsTestBot:   true,
		Alive:       true,
		JoinedAt:    now.UTC().Unix(),
	}
}

func mafiaTestBotCount(room MafiaRoom) int {
	count := 0
	for _, player := range room.Players {
		if player.IsTestBot {
			count++
		}
	}
	return count
}

func mafiaNextTestBotID(room MafiaRoom) int64 {
	used := make(map[int64]struct{}, len(room.Players))
	for _, player := range room.Players {
		used[player.TelegramID] = struct{}{}
	}
	for i := int64(1); i < 10000; i++ {
		id := mafiaTestBotBaseID - i
		if _, ok := used[id]; !ok {
			return id
		}
	}
	return mafiaTestBotBaseID - time.Now().UnixNano()
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func (s *MafiaService) applyTestBotNightActions(room *MafiaRoom) {
	if s == nil || room == nil || room.Status != MafiaStatusNight {
		return
	}
	if room.NightActions.MafiaVotes == nil {
		room.NightActions.MafiaVotes = map[string]int64{}
	}
	if room.NightActions.DetectiveChecks == nil {
		room.NightActions.DetectiveChecks = map[string]int64{}
	}
	if room.NightActions.DetectiveResults == nil {
		room.NightActions.DetectiveResults = map[string]bool{}
	}
	for i := range room.Players {
		player := room.Players[i]
		if !player.IsTestBot || !player.Alive {
			continue
		}
		switch player.Role {
		case MafiaRoleMafia:
			if _, ok := room.NightActions.MafiaVotes[idKey(player.TelegramID)]; ok {
				continue
			}
			candidates := mafiaAliveCandidatePlayers(room, player.TelegramID, false)
			targetID, ok := s.mafiaAIBotTarget(room, player, "night_kill", candidates)
			if !ok {
				if normalizeMafiaSettings(room.Settings).EnableAIBots {
					targetID, _ = mafiaAIBotPreferredKillTarget(*room, candidates, player)
				}
				if targetID == 0 {
					targetID = s.randomAlivePlayer(room, player.TelegramID, false)
				}
			}
			if targetID != 0 {
				room.NightActions.MafiaVotes[idKey(player.TelegramID)] = targetID
				room.Players[i].NightActionDone = true
			}
		case MafiaRoleDoctor:
			if room.NightActions.DoctorTargetID != 0 {
				continue
			}
			candidates := mafiaAliveCandidatePlayers(room, 0, true)
			targetID, ok := s.mafiaAIBotTarget(room, player, "doctor_heal", candidates)
			if !ok {
				targetID = s.randomAlivePlayer(room, 0, true)
			}
			if targetID != 0 {
				room.NightActions.DoctorTargetID = targetID
				room.Players[i].NightActionDone = true
			}
		case MafiaRoleDetective:
			if _, ok := room.NightActions.DetectiveChecks[idKey(player.TelegramID)]; ok {
				continue
			}
			candidates := mafiaAliveCandidatePlayers(room, player.TelegramID, true)
			targetID, ok := s.mafiaAIBotTarget(room, player, "detective_check", candidates)
			if !ok {
				targetID = s.randomAlivePlayer(room, player.TelegramID, true)
			}
			if targetID != 0 {
				idx := mafiaRoomPlayerIndex(*room, targetID)
				room.NightActions.DetectiveChecks[idKey(player.TelegramID)] = targetID
				room.NightActions.DetectiveResults[idKey(player.TelegramID)] = idx >= 0 && room.Players[idx].Role == MafiaRoleMafia
				room.Players[i].NightActionDone = true
			}
		}
	}
}

func (s *MafiaService) applyTestBotVotes(room *MafiaRoom) {
	if s == nil || room == nil || room.Status != MafiaStatusVoting {
		return
	}
	if room.Votes == nil {
		room.Votes = map[string]string{}
	}
	for _, player := range room.Players {
		if !player.IsTestBot || !player.Alive {
			continue
		}
		key := idKey(player.TelegramID)
		if _, exists := room.Votes[key]; exists {
			continue
		}
		candidates := mafiaAliveCandidatePlayers(room, player.TelegramID, true)
		targetID, skip, say, ok := s.mafiaAIBotVote(room, player, candidates)
		if ok {
			if skip {
				room.Votes[key] = "skip"
			} else {
				room.Votes[key] = idKey(targetID)
			}
			appendMafiaAIBotMessage(room, player, say)
			continue
		}
		targetID = s.randomAlivePlayer(room, player.TelegramID, true)
		if targetID == 0 {
			room.Votes[key] = "skip"
			continue
		}
		room.Votes[key] = idKey(targetID)
	}
}

func (s *MafiaService) randomAlivePlayer(room *MafiaRoom, excludeID int64, allowMafia bool) int64 {
	if s == nil || room == nil {
		return 0
	}
	candidates := make([]int64, 0, len(room.Players))
	for _, player := range room.Players {
		if !player.Alive || player.TelegramID == excludeID {
			continue
		}
		if !allowMafia && isMafiaTeam(player.Role) {
			continue
		}
		candidates = append(candidates, player.TelegramID)
	}
	if len(candidates) == 0 {
		return 0
	}
	return candidates[s.rng.Intn(len(candidates))]
}

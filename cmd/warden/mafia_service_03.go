package main

import (
	"strings"
	"time"
)

func (s *MafiaService) DetectiveCheck(roomID string, detectiveID, targetID int64, now time.Time) (MafiaRoom, bool, error) {
	room, _, err := s.playerActionRoom(roomID, detectiveID, MafiaRoleDetective, MafiaStatusNight)
	if err != nil {
		return MafiaRoom{}, false, err
	}
	idx := mafiaRoomPlayerIndex(room, targetID)
	if idx < 0 {
		return MafiaRoom{}, false, errMafiaPlayerNotFound
	}
	if !room.Players[idx].Alive {
		return MafiaRoom{}, false, errMafiaPlayerDead
	}
	result := room.Players[idx].Role == MafiaRoleMafia
	if room.NightActions.DetectiveChecks == nil {
		room.NightActions.DetectiveChecks = map[string]int64{}
	}
	if room.NightActions.DetectiveResults == nil {
		room.NightActions.DetectiveResults = map[string]bool{}
	}
	if _, exists := room.NightActions.DetectiveChecks[idKey(detectiveID)]; exists {
		return MafiaRoom{}, false, errMafiaNightActionExists
	}
	room.NightActions.DetectiveChecks[idKey(detectiveID)] = targetID
	room.NightActions.DetectiveResults[idKey(detectiveID)] = result
	mafiaRememberDetectiveKnownRole(&room, detectiveID, targetID, room.Players[idx].Role)
	room.Players[mafiaRoomPlayerIndex(room, detectiveID)].NightActionDone = true
	room.UpdatedAt = now.UTC().Unix()
	return room, result, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) ResolveNight(roomID string, now time.Time) (MafiaRoom, MafiaNightResult, error) {
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, MafiaNightResult{}, errMafiaRoomNotFound
	}
	if room.Status != MafiaStatusNight {
		return MafiaRoom{}, MafiaNightResult{}, errMafiaWrongPhase
	}
	s.applyTestBotNightActions(&room)
	appendMiniAppDetectiveResultMessages(&room, now)
	targetID := mafiaMajorityTarget(room.NightActions.MafiaVotes, s.rng)
	result := MafiaNightResult{Quiet: targetID == 0}
	if targetID != 0 {
		if targetID == room.NightActions.DoctorTargetID {
			result.Saved = true
		} else if idx := mafiaRoomPlayerIndex(room, targetID); idx >= 0 {
			room.Players[idx].Alive = false
			result.KilledID = targetID
		}
	}
	room.Status = MafiaStatusDayDiscussion
	room.PhaseSkipVotes = map[string]bool{}
	room.PhaseStartedAt = now.UTC().Unix()
	for i := range room.Players {
		room.Players[i].NightActionDone = false
	}
	appendMiniAppSystemMessageToRoom(&room, mafiaNightResultText(room, result), true, now)
	room.UpdatedAt = now.UTC().Unix()
	if winner := mafiaWinner(room); winner != "" {
		s.finishMafiaRoom(&room, winner)
	}
	return room, result, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) StartVoting(roomID string, now time.Time) (MafiaRoom, error) {
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, errMafiaRoomNotFound
	}
	if room.Status != MafiaStatusDayDiscussion {
		return MafiaRoom{}, errMafiaWrongPhase
	}
	room.Status = MafiaStatusVoting
	room.Votes = map[string]string{}
	room.PhaseSkipVotes = map[string]bool{}
	s.applyTestBotVotes(&room)
	room.PhaseStartedAt = now.UTC().Unix()
	room.UpdatedAt = now.UTC().Unix()
	return room, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) Vote(roomID string, voterID int64, target string, now time.Time) (MafiaRoom, error) {
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, errMafiaRoomNotFound
	}
	if room.Status != MafiaStatusVoting {
		return MafiaRoom{}, errMafiaWrongPhase
	}
	idx := mafiaRoomPlayerIndex(room, voterID)
	if idx < 0 {
		return MafiaRoom{}, errMafiaPlayerNotFound
	}
	if !room.Players[idx].Alive {
		return MafiaRoom{}, errMafiaPlayerDead
	}
	if room.Votes == nil {
		room.Votes = map[string]string{}
	}
	key := idKey(voterID)
	if _, exists := room.Votes[key]; exists && !normalizeMafiaSettings(room.Settings).AllowVoteChange {
		return MafiaRoom{}, errMafiaVoteExists
	}
	target = strings.TrimSpace(target)
	if target != "skip" {
		targetID, err := strconvParseInt64(target)
		targetIdx := mafiaRoomPlayerIndex(room, targetID)
		if err != nil || targetIdx < 0 {
			return MafiaRoom{}, errMafiaPlayerNotFound
		}
		if !room.Players[targetIdx].Alive {
			return MafiaRoom{}, errMafiaPlayerDead
		}
	}
	room.Votes[key] = target
	room.UpdatedAt = now.UTC().Unix()
	return room, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) ResolveVoting(roomID string, now time.Time) (MafiaRoom, MafiaVoteResult, error) {
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, MafiaVoteResult{}, errMafiaRoomNotFound
	}
	if room.Status != MafiaStatusVoting {
		return MafiaRoom{}, MafiaVoteResult{}, errMafiaWrongPhase
	}
	s.applyTestBotVotes(&room)
	result := MafiaVoteResult{Votes: cloneStringMap(room.Votes)}
	leaders := mafiaVoteLeaders(room.Votes)
	if len(leaders) == 1 && leaders[0] != "skip" {
		result.ExecutedID = mustParseInt64(leaders[0])
	} else if len(leaders) > 1 {
		result.Tie = true
		if normalizeMafiaSettings(room.Settings).TieMode == MafiaTieRandomBetweenLeaders {
			result.ExecutedID = mustParseInt64(leaders[s.rng.Intn(len(leaders))])
		}
	}
	if result.ExecutedID != 0 {
		if idx := mafiaRoomPlayerIndex(room, result.ExecutedID); idx >= 0 {
			room.Players[idx].Alive = false
		}
	}
	if winner := mafiaWinner(room); winner != "" {
		s.finishMafiaRoom(&room, winner)
	} else {
		room.Status = MafiaStatusNight
		room.DayNumber++
		room.NightActions = MafiaNightActions{MafiaVotes: map[string]int64{}, DetectiveChecks: map[string]int64{}, DetectiveResults: map[string]bool{}}
		room.PhaseStartedAt = now.UTC().Unix()
	}
	room.Votes = map[string]string{}
	room.PhaseSkipVotes = map[string]bool{}
	room.UpdatedAt = now.UTC().Unix()
	return room, result, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) VotePhaseSkip(roomID string, voterID int64) (MafiaRoom, int, int, bool, error) {
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, 0, 0, false, errMafiaRoomNotFound
	}
	if !mafiaAutoHostStatus(room.Status) {
		return MafiaRoom{}, 0, 0, false, errMafiaWrongPhase
	}
	idx := mafiaRoomPlayerIndex(room, voterID)
	if idx < 0 {
		return MafiaRoom{}, 0, 0, false, errMafiaPlayerNotFound
	}
	if !room.Players[idx].Alive {
		return MafiaRoom{}, 0, 0, false, errMafiaPlayerDead
	}
	if room.Players[idx].IsTestBot {
		return MafiaRoom{}, 0, 0, false, errMafiaWrongRole
	}
	if room.PhaseSkipVotes == nil {
		room.PhaseSkipVotes = map[string]bool{}
	}
	room.PhaseSkipVotes[idKey(voterID)] = true
	votes := mafiaPhaseSkipVotesCount(room)
	needed := mafiaPhaseSkipNeeded(room)
	reached := votes >= needed && needed > 0
	return room, votes, needed, reached, s.store.SaveMafiaRoom(room)
}

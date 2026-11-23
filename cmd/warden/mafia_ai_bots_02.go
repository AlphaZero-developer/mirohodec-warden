package main

import (
	"context"
	"strings"
	"time"
)

func (s *MafiaService) MaybeAddAIBotKillRecommendations(roomID string, now time.Time) (MafiaRoom, []MafiaPrivateMessage, []int64, error) {
	if s == nil || s.store == nil {
		return MafiaRoom{}, nil, nil, errMafiaRoomNotFound
	}
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, nil, nil, errMafiaRoomNotFound
	}
	if !s.mafiaAIBotsReady(&room) || room.Status != MafiaStatusNight {
		return room, nil, nil, nil
	}
	settings := normalizeMafiaSettings(room.Settings)
	if settings.MafiaChatMode == MafiaChatDisabled {
		return room, nil, nil, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	hintKey := mafiaAIBotKillHintKey(room)
	if hintKey != "" && room.AIBotLastKillHintKey == hintKey {
		return room, nil, nil, nil
	}
	if room.NightActions.MafiaVotes == nil {
		room.NightActions.MafiaVotes = map[string]int64{}
	}
	recipients := mafiaShadowRecipients(room, settings)
	if len(recipients) == 0 {
		return room, nil, nil, nil
	}
	var messages []MafiaPrivateMessage
	for i := range room.Players {
		player := room.Players[i]
		if !player.IsTestBot || !player.Alive || player.Role != MafiaRoleMafia {
			continue
		}
		candidates := mafiaAliveCandidatePlayers(&room, player.TelegramID, false)
		if len(candidates) == 0 {
			continue
		}
		decision, ok := s.mafiaAIBotDecision(&room, player, "night_kill", candidates, false)
		targetID := int64(0)
		reason := ""
		if ok {
			targetID = mafiaAIBotValidTargetID(decision.TargetID, candidates)
			if targetID == 0 {
				targetID = mafiaAIBotVoteTargetID(decision.Vote, candidates)
			}
			reason = decision.Reason
		}
		if targetID == 0 {
			targetID, reason = mafiaAIBotPreferredKillTarget(room, candidates, player)
		}
		if targetID == 0 {
			continue
		}
		room.NightActions.MafiaVotes[idKey(player.TelegramID)] = targetID
		room.Players[i].NightActionDone = true
		text := mafiaAIBotKillRecommendationText(room, targetID, reason)
		msg := appendMafiaShadowMessage(&room, player.TelegramID, player.MafiaAlias, text, now)
		messages = append(messages, msg)
	}
	if len(messages) == 0 {
		return room, nil, nil, nil
	}
	room.AIBotLastKillHintKey = hintKey
	room.UpdatedAt = now.UTC().Unix()
	if err := s.store.SaveMafiaRoom(room); err != nil {
		return room, nil, nil, err
	}
	return room, messages, recipients, nil
}

func (s *MafiaService) RecordAIBotSuspicion(roomID string, speakerID int64, messageID int, text string, now time.Time) (MafiaRoom, bool, error) {
	if s == nil || s.store == nil {
		return MafiaRoom{}, false, errMafiaRoomNotFound
	}
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, false, errMafiaRoomNotFound
	}
	settings := normalizeMafiaSettings(room.Settings)
	if !settings.EnableAIBots || (room.Status != MafiaStatusDayDiscussion && room.Status != MafiaStatusVoting) {
		return room, false, nil
	}
	speakerIdx := mafiaRoomPlayerIndex(room, speakerID)
	if speakerIdx < 0 || !room.Players[speakerIdx].Alive || room.Players[speakerIdx].IsTestBot {
		return room, false, nil
	}
	text = trimRunes(strings.TrimSpace(text), 420)
	if !mafiaLooksLikeSuspicionText(text) {
		return room, false, nil
	}
	targetIDs, targetNames := mafiaSuspicionTargets(room, text, speakerID)
	if len(targetIDs) == 0 && !mafiaHasStrongSuspicionWord(text) {
		return room, false, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	memory := MafiaAIBotMemory{
		DayNumber:   room.DayNumber,
		Phase:       room.Status,
		MessageID:   messageID,
		SpeakerID:   speakerID,
		SpeakerName: room.Players[speakerIdx].DisplayName,
		Text:        text,
		TargetIDs:   targetIDs,
		TargetNames: targetNames,
		CreatedAt:   now.UTC().Unix(),
	}
	if mafiaAIBotMemoryExists(room.AIBotMemory, memory) {
		return room, false, nil
	}
	room.AIBotMemory = append(room.AIBotMemory, memory)
	room.AIBotMemory = pruneMafiaAIBotMemory(room.AIBotMemory, room.DayNumber)
	return room, true, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) MaybeAddAIBotDiscussionMessage(roomID string, now time.Time) (MafiaRoom, *MafiaAIBotMessage, error) {
	if s == nil || s.store == nil {
		return MafiaRoom{}, nil, errMafiaRoomNotFound
	}
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, nil, errMafiaRoomNotFound
	}
	if !s.mafiaAIBotsReady(&room) || (room.Status != MafiaStatusDayDiscussion && room.Status != MafiaStatusVoting) {
		return room, nil, nil
	}
	if len(mafiaAIBotMemoryPayload(room, 1)) == 0 {
		return room, nil, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if room.AIBotLastSpokeUnix != 0 && now.UTC().Sub(time.Unix(room.AIBotLastSpokeUnix, 0)) < mafiaAIBotTalkCooldown {
		return room, nil, nil
	}
	bots := mafiaAliveAIBotPlayers(room)
	if len(bots) == 0 {
		return room, nil, nil
	}
	player := bots[s.rng.Intn(len(bots))]
	ctx, cancel := context.WithTimeout(context.Background(), mafiaAIBotDecisionTimeout)
	defer cancel()
	text, err := NewMafiaAIBotService(s.ai).Discuss(ctx, s.aiCfg, room, player)
	if err != nil || strings.TrimSpace(text) == "" {
		return room, nil, nil
	}
	room.AIBotLastSpokeUnix = now.UTC().Unix()
	appendMafiaAIBotMessage(&room, player, text)
	if len(room.AIBotMessages) == 0 {
		return room, nil, nil
	}
	message := room.AIBotMessages[len(room.AIBotMessages)-1]
	if err := s.store.SaveMafiaRoom(room); err != nil {
		return room, nil, err
	}
	return room, &message, nil
}

func mafiaAliveCandidatePlayers(room *MafiaRoom, excludeID int64, allowMafia bool) []MafiaPlayer {
	if room == nil {
		return nil
	}
	candidates := make([]MafiaPlayer, 0, len(room.Players))
	for _, player := range room.Players {
		if !player.Alive || player.TelegramID == excludeID {
			continue
		}
		if !allowMafia && isMafiaTeam(player.Role) {
			continue
		}
		candidates = append(candidates, player)
	}
	return candidates
}

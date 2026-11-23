package main

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

const (
	mafiaAIBotDecisionTimeout = 3 * time.Second
	mafiaAIBotTalkCooldown    = 45 * time.Second
	mafiaAIBotMemoryLimit     = 80
)

type MafiaAIBotService struct {
	ai AIService
}

type mafiaAIBotDecision struct {
	TargetID int64  `json:"target_id"`
	Vote     string `json:"vote"`
	Say      string `json:"say"`
	Reason   string `json:"reason"`
}

func NewMafiaAIBotService(ai AIService) MafiaAIBotService {
	return MafiaAIBotService{ai: ai}
}

func (s MafiaAIBotService) Decide(ctx context.Context, cfg AIConfig, room MafiaRoom, player MafiaPlayer, action string, candidates []MafiaPlayer, allowSkip bool) (mafiaAIBotDecision, error) {
	if s.ai == nil || !cfg.Enabled {
		return mafiaAIBotDecision{}, errAIUnavailable
	}
	cfg.SystemPrompt = "Ты AI-агент тест-бота в MafiaGame. Играй как живой игрок: выбирай цель по текущей фазе, не раскрывай скрытые роли, не меняй правила и не придумывай состояние игры. Используй только valid_targets. Для голосования можно вернуть короткую публичную реплику say, без оскорблений и без системной информации. Верни только JSON."
	if action == "night_kill" {
		cfg.SystemPrompt += " Сейчас ты выбираешь ночную жертву за мафию. Предпочитай kill_recommendations: тех, кто сильно подозревал мафию, давил на твоих союзников или может повести город против вас. В reason коротко объясни, почему цель опасна."
	}
	cfg.MaxReplyChars = 700
	if cfg.MaxTokens <= 0 || cfg.MaxTokens > 220 {
		cfg.MaxTokens = 220
	}
	payload := map[string]interface{}{
		"room_id":              room.ID,
		"phase":                room.Status,
		"day":                  room.DayNumber,
		"action":               action,
		"allow_skip":           allowSkip,
		"you":                  mafiaAIBotPlayerPayload(player, true),
		"alive_players":        mafiaAIBotPlayersPayload(mafiaAliveCandidatePlayers(&room, 0, true), player.TelegramID),
		"valid_targets":        mafiaAIBotPlayersPayload(candidates, player.TelegramID),
		"mafia_allies":         mafiaAIBotMafiaAlliesPayload(room, player),
		"suspicion_memory":     mafiaAIBotMemoryPayload(room, 16),
		"kill_recommendations": mafiaAIBotKillRecommendationsPayload(room, candidates, player),
		"response_schema": map[string]interface{}{
			"target_id": 123,
			"vote":      "target|skip",
			"say":       "short public phrase for voting only",
			"reason":    "private short reason",
		},
	}
	raw, _ := json.Marshal(payload)
	reply, err := s.ai.Reply(ctx, cfg, string(raw))
	if err != nil {
		return mafiaAIBotDecision{}, err
	}
	var decision mafiaAIBotDecision
	if err := json.Unmarshal([]byte(extractJSON(reply)), &decision); err != nil {
		return mafiaAIBotDecision{}, err
	}
	decision.Vote = strings.TrimSpace(decision.Vote)
	decision.Say = trimRunes(strings.TrimSpace(decision.Say), 260)
	decision.Say = formatAIReplyTextForEncoding(cfg, decision.Say)
	decision.Reason = trimRunes(strings.TrimSpace(decision.Reason), 260)
	return decision, nil
}

func (s MafiaAIBotService) Discuss(ctx context.Context, cfg AIConfig, room MafiaRoom, player MafiaPlayer) (string, error) {
	if s.ai == nil || !cfg.Enabled {
		return "", errAIUnavailable
	}
	cfg.SystemPrompt = "Ты AI-агент тест-бота в MafiaGame. Напиши короткую живую реплику в дневное обсуждение: предположение, вопрос или сомнение о том, кто может быть мафией. Опирайся на suspicion_memory, не раскрывай свою скрытую роль, не утверждай проверку/лечения без фактов, не оскорбляй и не пиши системные объяснения. Верни только JSON."
	cfg.MaxReplyChars = 700
	if cfg.MaxTokens <= 0 || cfg.MaxTokens > 180 {
		cfg.MaxTokens = 180
	}
	payload := map[string]interface{}{
		"room_id":          room.ID,
		"phase":            room.Status,
		"day":              room.DayNumber,
		"you":              mafiaAIBotPlayerPayload(player, true),
		"alive_players":    mafiaAIBotPlayersPayload(mafiaAliveCandidatePlayers(&room, 0, true), player.TelegramID),
		"mafia_allies":     mafiaAIBotMafiaAlliesPayload(room, player),
		"suspicion_memory": mafiaAIBotMemoryPayload(room, 18),
		"response_schema": map[string]interface{}{
			"say": "short public phrase",
		},
	}
	raw, _ := json.Marshal(payload)
	reply, err := s.ai.Reply(ctx, cfg, string(raw))
	if err != nil {
		return "", err
	}
	var parsed struct {
		Say  string `json:"say"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(extractJSON(reply)), &parsed); err == nil {
		if strings.TrimSpace(parsed.Say) != "" {
			reply = parsed.Say
		} else if strings.TrimSpace(parsed.Text) != "" {
			reply = parsed.Text
		}
	}
	reply = strings.Trim(strings.TrimSpace(reply), "\"'`")
	if reply == "" {
		return "", errAIEmptyReply
	}
	return formatAIReplyTextForEncoding(cfg, trimRunes(reply, 260)), nil
}

func (s *MafiaService) mafiaAIBotsReady(room *MafiaRoom) bool {
	return s != nil && room != nil && normalizeMafiaSettings(room.Settings).EnableAIBots && s.ai != nil && s.aiCfg.Enabled
}

func (s *MafiaService) mafiaAIBotTarget(room *MafiaRoom, player MafiaPlayer, action string, candidates []MafiaPlayer) (int64, bool) {
	decision, ok := s.mafiaAIBotDecision(room, player, action, candidates, false)
	if !ok {
		return 0, false
	}
	targetID := mafiaAIBotValidTargetID(decision.TargetID, candidates)
	if targetID == 0 {
		targetID = mafiaAIBotVoteTargetID(decision.Vote, candidates)
	}
	return targetID, targetID != 0
}

func (s *MafiaService) mafiaAIBotVote(room *MafiaRoom, player MafiaPlayer, candidates []MafiaPlayer) (int64, bool, string, bool) {
	decision, ok := s.mafiaAIBotDecision(room, player, "vote", candidates, true)
	if !ok {
		return 0, false, "", false
	}
	if strings.EqualFold(decision.Vote, "skip") || strings.EqualFold(decision.Vote, "abstain") {
		return 0, true, decision.Say, true
	}
	targetID := mafiaAIBotValidTargetID(decision.TargetID, candidates)
	if targetID == 0 {
		targetID = mafiaAIBotVoteTargetID(decision.Vote, candidates)
	}
	if targetID == 0 {
		return 0, false, "", false
	}
	return targetID, false, decision.Say, true
}

func (s *MafiaService) mafiaAIBotDecision(room *MafiaRoom, player MafiaPlayer, action string, candidates []MafiaPlayer, allowSkip bool) (mafiaAIBotDecision, bool) {
	if !s.mafiaAIBotsReady(room) || len(candidates) == 0 {
		if allowSkip && s.mafiaAIBotsReady(room) && len(candidates) == 0 {
			return mafiaAIBotDecision{Vote: "skip"}, true
		}
		return mafiaAIBotDecision{}, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), mafiaAIBotDecisionTimeout)
	defer cancel()
	decision, err := NewMafiaAIBotService(s.ai).Decide(ctx, s.aiCfg, *room, player, action, candidates, allowSkip)
	if err != nil {
		return mafiaAIBotDecision{}, false
	}
	return decision, true
}

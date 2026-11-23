package main

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

type MafiaModerationAIService struct {
	ai AIService
}

type MafiaNarratorService struct {
	ai AIService
}

type mafiaModerationAIResponse struct {
	IsRoleReveal bool    `json:"is_role_reveal"`
	Confidence   float64 `json:"confidence"`
	DetectedRole string  `json:"detected_role"`
	Reason       string  `json:"reason"`
}

func NewMafiaModerationAIService(ai AIService) MafiaModerationAIService {
	return MafiaModerationAIService{ai: ai}
}

func NewMafiaNarratorService(ai AIService) MafiaNarratorService {
	return MafiaNarratorService{ai: ai}
}

func (s MafiaNarratorService) Narrate(ctx context.Context, cfg AIConfig, event string, facts map[string]interface{}, fallback string) string {
	if s.ai == nil || !cfg.Enabled {
		return fallback
	}
	cfg.SystemPrompt = "Ты Лайя, AI-ведущая MafiaGame. Пиши короткое атмосферное сообщение на русском так, будто именно Лайя ведёт партию сама: объявляет фазы, держит темп и следит за правилами. Создатель комнаты не ведущий, он только владелец комнаты. Используй только переданные факты. Не раскрывай скрытые роли, не меняй состояние игры, не выбирай жертву и не объявляй победителя сам."
	if event == "post_game_report" {
		cfg.SystemPrompt += " Сейчас нужен финальный отчёт ПОСЛЕ завершения партии. Обязательно пиши в прошедшем времени. Запрещено говорить, что игра продолжается, следующий день наступил, будет следующая ночь или будет следующая жертва."
	}
	cfg.MaxReplyChars = 700
	if cfg.MaxTokens <= 0 || cfg.MaxTokens > 220 {
		cfg.MaxTokens = 220
	}
	payload := map[string]interface{}{
		"event": event,
		"facts": facts,
	}
	raw, _ := json.Marshal(payload)
	reply, err := s.ai.Reply(ctx, cfg, string(raw))
	if err != nil {
		return fallback
	}
	reply = strings.TrimSpace(reply)
	if reply == "" || len([]rune(reply)) > 900 {
		return fallback
	}
	return formatAIReplyTextForEncoding(cfg, reply)
}

func (a *App) mafiaNarration(event string, facts map[string]interface{}, fallback string) string {
	if a == nil {
		return fallback
	}
	ai := a.currentAI()
	if ai == nil {
		return fallback
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return NewMafiaNarratorService(ai).Narrate(ctx, a.config().AI, event, facts, fallback)
}

func (s MafiaModerationAIService) Detect(ctx context.Context, cfg AIConfig, text string, status MafiaRoomStatus) (MafiaRoleRevealResult, error) {
	if s.ai == nil || !cfg.Enabled || strings.TrimSpace(text) == "" {
		return MafiaRoleRevealResult{}, errAIUnavailable
	}
	cfg.SystemPrompt = "Ты вспомогательный AI-модератор игры MafiaGame. Определи, раскрывает ли игрок свою роль или личное ночное действие. Не решай исход игры. Верни только JSON."
	cfg.MaxReplyChars = 700
	if cfg.MaxTokens <= 0 || cfg.MaxTokens > 300 {
		cfg.MaxTokens = 300
	}
	payload := map[string]interface{}{
		"text":  text,
		"phase": status,
		"forbidden_examples": []string{
			"я мафия", "я доктор", "моя роль детектив", "я лечил его ночью", "я проверил его ночью",
		},
		"allowed_examples": []string{
			"я не мафия", "я думаю он мафия", "мне кажется она доктор", "проверьте меня", "он ведёт себя как мафия",
		},
		"response_schema": `{"is_role_reveal":true,"confidence":0.92,"detected_role":"doctor","reason":"..."}`,
	}
	raw, _ := json.Marshal(payload)
	reply, err := s.ai.Reply(ctx, cfg, string(raw))
	if err != nil {
		return MafiaRoleRevealResult{}, err
	}
	var parsed mafiaModerationAIResponse
	if err := json.Unmarshal([]byte(extractJSON(reply)), &parsed); err != nil {
		return MafiaRoleRevealResult{}, err
	}
	role := MafiaRole(strings.TrimSpace(parsed.DetectedRole))
	if role != MafiaRoleMafia && role != MafiaRoleDoctor && role != MafiaRoleDetective && role != MafiaRoleCivilian {
		role = ""
	}
	return MafiaRoleRevealResult{
		IsRoleReveal: parsed.IsRoleReveal,
		Confidence:   parsed.Confidence,
		DetectedRole: role,
		Reason:       strings.TrimSpace(parsed.Reason),
		Source:       MafiaRevealSourceAI,
	}, nil
}

func (a *App) detectMafiaRoleRevealAI(text string, status MafiaRoomStatus) MafiaRoleRevealResult {
	if a == nil || a.ai == nil || !looksLikeMafiaRevealCandidate(text) {
		return MafiaRoleRevealResult{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := NewMafiaModerationAIService(a.ai).Detect(ctx, a.config().AI, text, status)
	if err != nil {
		return MafiaRoleRevealResult{}
	}
	return result
}

func looksLikeMafiaRevealCandidate(text string) bool {
	normalized := normalizeFAQText(text)
	if normalized == "" {
		return false
	}
	if strings.Contains(" "+normalized+" ", " я не мафия ") ||
		strings.Contains(" "+normalized+" ", " я думаю ") ||
		strings.Contains(" "+normalized+" ", " мне кажется ") {
		return false
	}
	if !strings.Contains(" "+normalized+" ", " я ") &&
		!strings.Contains(normalized, "моя роль") &&
		!strings.Contains(normalized, "у меня роль") &&
		!strings.Contains(normalized, "мне выпал") {
		return false
	}
	for _, needle := range []string{"мафи", "доктор", "комиссар", "детектив", "мирн", "лечил", "проверил", "жертву", "ночью"} {
		if strings.Contains(normalized, needle) {
			return true
		}
	}
	return false
}

func extractJSON(text string) string {
	text = strings.TrimSpace(text)
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end >= start {
		return text[start : end+1]
	}
	return text
}

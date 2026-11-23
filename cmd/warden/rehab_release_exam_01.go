package main

import (
	"strings"
	"time"
)

const (
	defaultRehabReleaseExamQuestions = `Почему тебя можно выпускать? | Я понял правила и буду нормально общаться.
Что ты сделаешь после выхода? | Буду спокойно писать в чат и не нарушать правила.
Куда ты больше не хочешь возвращаться? | В психиатрический центр.`
	rehabReleaseExamVerdictPrompt = `Ты строгий AI-экзаменатор перед выходом пользователя из психиатрического центра Лайи. Сравни ответ пользователя с правильным ответом по смыслу. Верни только OK или BAD.
OK: ответ непустой, по смыслу совпадает с правильным ответом или честно формулирует ту же мысль другими словами.
BAD: ответ пустой, уклончивый, агрессивный, с оскорблениями, не по теме, спорит с вопросом или не совпадает с правильным ответом.`
	rehabReleaseExamFailMessage = "Ты не прошёл лечение, отправляйся обратно в психиатрический центр."
)

type RehabReleaseExamConfig struct {
	Enabled       bool
	Questions     string
	AnswerWindow  string
	FailDuration  string
	UpdatedUnix   int64
	UpdatedByID   int64
	UpdatedByName string
}

type rehabReleaseExamQA struct {
	Question string
	Expected string
}

type rehabReleaseExamSession struct {
	ChatID      int64
	UserID      int64
	Username    string
	DisplayName string
	Question    string
	Expected    string
	Source      string
	StartedAt   time.Time
	ExpiresAt   time.Time
	Record      ModerationRecord
}

func normalizeRehabReleaseExamRuntimeState(state RehabReleaseExamRuntimeState) RehabReleaseExamRuntimeState {
	state.Questions = strings.TrimSpace(state.Questions)
	state.AnswerWindow = strings.TrimSpace(state.AnswerWindow)
	state.FailDuration = strings.TrimSpace(state.FailDuration)
	state.UpdatedByName = strings.TrimSpace(state.UpdatedByName)
	return state
}

func rehabReleaseExamConfigFromRehab(cfg RehabConfig) RehabReleaseExamConfig {
	return normalizeRehabReleaseExamConfig(RehabReleaseExamConfig{
		Enabled:      cfg.ReleaseExamEnabled,
		Questions:    cfg.ReleaseExamQuestions,
		AnswerWindow: cfg.ReleaseExamAnswerWindow,
		FailDuration: cfg.ReleaseExamFailDuration,
	})
}

func normalizeRehabReleaseExamConfig(cfg RehabReleaseExamConfig) RehabReleaseExamConfig {
	cfg.Questions = strings.TrimSpace(cfg.Questions)
	if cfg.Questions == "" {
		cfg.Questions = defaultRehabReleaseExamQuestions
	}
	cfg.AnswerWindow = strings.TrimSpace(cfg.AnswerWindow)
	if cfg.AnswerWindow == "" {
		cfg.AnswerWindow = "10m"
	}
	cfg.FailDuration = strings.TrimSpace(cfg.FailDuration)
	if cfg.FailDuration == "" {
		cfg.FailDuration = "24h"
	}
	cfg.UpdatedByName = strings.TrimSpace(cfg.UpdatedByName)
	return cfg
}

func rehabReleaseExamRuntimeStateFromConfig(cfg RehabReleaseExamConfig, byID int64, byName string, now time.Time) RehabReleaseExamRuntimeState {
	cfg = normalizeRehabReleaseExamConfig(cfg)
	return normalizeRehabReleaseExamRuntimeState(RehabReleaseExamRuntimeState{
		Enabled:       cfg.Enabled,
		Questions:     cfg.Questions,
		AnswerWindow:  cfg.AnswerWindow,
		FailDuration:  cfg.FailDuration,
		UpdatedUnix:   now.UTC().Unix(),
		UpdatedByID:   byID,
		UpdatedByName: byName,
	})
}

func (a *App) activeRehabReleaseExamConfig() RehabReleaseExamConfig {
	if a == nil {
		return normalizeRehabReleaseExamConfig(RehabReleaseExamConfig{})
	}
	cfg := rehabReleaseExamConfigFromRehab(a.config().Rehab)
	if a.store == nil {
		return cfg
	}
	state := normalizeRehabReleaseExamRuntimeState(a.store.RehabReleaseExamRuntimeState())
	if state.UpdatedUnix == 0 {
		return cfg
	}
	return normalizeRehabReleaseExamConfig(RehabReleaseExamConfig{
		Enabled:       state.Enabled,
		Questions:     state.Questions,
		AnswerWindow:  state.AnswerWindow,
		FailDuration:  state.FailDuration,
		UpdatedUnix:   state.UpdatedUnix,
		UpdatedByID:   state.UpdatedByID,
		UpdatedByName: state.UpdatedByName,
	})
}

func (a *App) activeRehabReleaseExamConfigForChat(chatID int64) RehabReleaseExamConfig {
	cfg := a.activeRehabReleaseExamConfig()
	if a == nil || a.store == nil || chatID == 0 {
		return cfg
	}
	state, ok := a.store.RehabReleaseExamRuntimeStateForChat(chatID)
	if !ok || state.UpdatedUnix == 0 {
		return cfg
	}
	return normalizeRehabReleaseExamConfig(RehabReleaseExamConfig{
		Enabled:       state.Enabled,
		Questions:     state.Questions,
		AnswerWindow:  state.AnswerWindow,
		FailDuration:  state.FailDuration,
		UpdatedUnix:   state.UpdatedUnix,
		UpdatedByID:   state.UpdatedByID,
		UpdatedByName: state.UpdatedByName,
	})
}

func parseRehabReleaseExamQuestions(value string) []rehabReleaseExamQA {
	var out []rehabReleaseExamQA
	for _, raw := range strings.Split(value, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var question, expected string
		switch {
		case strings.Contains(line, "|"):
			parts := strings.SplitN(line, "|", 2)
			question, expected = parts[0], parts[1]
		case strings.Contains(line, "=>"):
			parts := strings.SplitN(line, "=>", 2)
			question, expected = parts[0], parts[1]
		case strings.Contains(line, "="):
			parts := strings.SplitN(line, "=", 2)
			question, expected = parts[0], parts[1]
		default:
			question = line
		}
		question = strings.TrimSpace(question)
		expected = strings.TrimSpace(expected)
		if question == "" {
			continue
		}
		out = append(out, rehabReleaseExamQA{Question: question, Expected: expected})
	}
	return out
}

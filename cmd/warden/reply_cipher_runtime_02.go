package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) addReplyCipherTarget(c tele.Context, state ReplyCipherRuntimeState, args []string, forcedCipher string) error {
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	target, rest, err := a.targetFromArgs(chat.ID, c, args)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	cipher := normalizeAIReplyCipher(forcedCipher)
	if cipher == "" && len(rest) > 0 {
		if replyCipherRemoveToken(rest[0]) {
			return a.removeReplyCipherTarget(c, state, args)
		}
		cipher = normalizeAIReplyCipher(rest[0])
		if cipher == "" {
			return c.Send("Не знаю такой шифр. Доступно: binary, hex, base64.")
		}
	}

	state = normalizeReplyCipherRuntimeState(state)
	if state.Targets == nil {
		state.Targets = make(map[string]ReplyCipherTarget)
	}
	now := time.Now().UTC()
	admin := c.Sender()
	rec := ReplyCipherTarget{
		UserID:      target.ID,
		Username:    target.Username,
		DisplayName: target.Name,
		Cipher:      cipher,
		AddedUnix:   now.Unix(),
	}
	if admin != nil {
		rec.AddedByID = admin.ID
		rec.AddedByName = displayNameFromUser(admin)
	}
	state.Targets[idKey(target.ID)] = normalizeReplyCipherTarget(rec)
	state = replyCipherRuntimeUpdated(state, admin, now)
	if err := a.store.SetReplyCipherRuntimeState(state); err != nil {
		return err
	}

	effective := cipher
	if effective == "" {
		effective = a.effectiveReplyCipherDefault(a.config().AI)
	}
	return c.Send(fmt.Sprintf("Готово. %s теперь получает ответы Лайи через %s.", target.Name, replyCipherDisplayName(effective)))
}

func (a *App) removeReplyCipherTarget(c tele.Context, state ReplyCipherRuntimeState, args []string) error {
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	target, _, err := a.targetFromArgs(chat.ID, c, args)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	state = normalizeReplyCipherRuntimeState(state)
	if state.Targets != nil {
		delete(state.Targets, idKey(target.ID))
		if len(state.Targets) == 0 {
			state.Targets = nil
		}
	}
	state = replyCipherRuntimeUpdated(state, c.Sender(), time.Now().UTC())
	if err := a.store.SetReplyCipherRuntimeState(state); err != nil {
		return err
	}
	return c.Send(fmt.Sprintf("Готово. %s снова получает обычный текст.", target.Name))
}

func (a *App) replyCipherStatusText() string {
	cfg := a.config().AI
	state := ReplyCipherRuntimeState{}
	if a != nil && a.store != nil {
		state = a.store.ReplyCipherRuntimeState()
	}
	defaultCipher := a.effectiveReplyCipherDefault(cfg)
	lines := []string{
		fmt.Sprintf("Шифр по умолчанию: %s", replyCipherDisplayName(defaultCipher)),
		"Боты: шифруются по умолчанию.",
	}
	targets := replyCipherTargetsSorted(state.Targets)
	if len(targets) == 0 && len(cfg.ReplyCipherUserIDs) == 0 && len(cfg.ReplyCipherUsernames) == 0 {
		lines = append(lines, "Отдельных пользователей пока нет.")
		return strings.Join(lines, "\n")
	}
	if len(targets) > 0 {
		lines = append(lines, "Пользователи:")
		for _, target := range targets {
			cipher := target.Cipher
			if cipher == "" {
				cipher = defaultCipher
			}
			name := target.DisplayName
			if name == "" {
				name = fmt.Sprintf("user %d", target.UserID)
			}
			if target.Username != "" {
				name = fmt.Sprintf("%s (@%s)", name, target.Username)
			}
			lines = append(lines, fmt.Sprintf("- %s: %s", name, replyCipherDisplayName(cipher)))
		}
	}
	if len(cfg.ReplyCipherUserIDs) > 0 || len(cfg.ReplyCipherUsernames) > 0 {
		lines = append(lines, "Из config.yaml тоже включены цели.")
	}
	return strings.Join(lines, "\n")
}

func replyCipherTargetsSorted(targets map[string]ReplyCipherTarget) []ReplyCipherTarget {
	out := make([]ReplyCipherTarget, 0, len(targets))
	for _, target := range targets {
		target = normalizeReplyCipherTarget(target)
		if target.UserID != 0 {
			out = append(out, target)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		left := strings.ToLower(out[i].DisplayName)
		right := strings.ToLower(out[j].DisplayName)
		if left == right {
			return out[i].UserID < out[j].UserID
		}
		return left < right
	})
	return out
}

func (a *App) effectiveReplyCipherDefault(cfg AIConfig) string {
	cipher := normalizeAIReplyCipher(cfg.ReplyCipher)
	if a != nil && a.store != nil {
		state := a.store.ReplyCipherRuntimeState()
		if state.DefaultCipher != "" {
			cipher = state.DefaultCipher
		}
	}
	if cipher == "" {
		cipher = defaultAIReplyCipher
	}
	return cipher
}

func (a *App) replyCipherForTelegramTarget(cfg AIConfig, chat *tele.Chat, sender *tele.User, msg *tele.Message) (string, bool) {
	defaultCipher := a.effectiveReplyCipherDefault(cfg)
	if cipher, ok := a.replyCipherForUser(cfg, sender, defaultCipher); ok {
		return cipher, true
	}
	if msg != nil && msg.ReplyTo != nil {
		if cipher, ok := a.replyCipherForUser(cfg, msg.ReplyTo.Sender, defaultCipher); ok {
			return cipher, true
		}
	}
	for _, username := range layaMentionUsernames(messageText(msg)) {
		if cipher, ok := replyCipherForConfigUsername(cfg, username, defaultCipher); ok {
			return cipher, true
		}
		if a != nil && a.store != nil && chat != nil {
			if target, ok := a.store.UserByUsername(chat.ID, username); ok {
				if cipher, ok := a.replyCipherForTargetUser(cfg, target, defaultCipher); ok {
					return cipher, true
				}
			}
		}
	}
	return "", false
}

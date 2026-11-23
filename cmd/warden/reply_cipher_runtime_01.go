package main

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

const defaultAIReplyCipher = "binary"

func normalizeAIReplyCipher(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "binary", "bin", "bits", "bit", "бинарный", "бинарка", "двоичный", "двоичный код":
		return "binary"
	case "hex", "hexadecimal", "base16", "16", "16x", "хекс", "шестнадцатеричный":
		return "hex"
	case "base64", "b64", "64":
		return "base64"
	default:
		return ""
	}
}

func normalizeAIReplyCipherOrDefault(value string) string {
	if cipher := normalizeAIReplyCipher(value); cipher != "" {
		return cipher
	}
	return defaultAIReplyCipher
}

func normalizeReplyCipherUsernames(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		username := normalizeUsername(value)
		if username == "" {
			continue
		}
		if _, ok := seen[username]; ok {
			continue
		}
		seen[username] = struct{}{}
		out = append(out, username)
	}
	return out
}

func normalizeReplyCipherRuntimeState(state ReplyCipherRuntimeState) ReplyCipherRuntimeState {
	state.DefaultCipher = normalizeAIReplyCipher(state.DefaultCipher)
	state.UpdatedByName = strings.TrimSpace(state.UpdatedByName)
	if len(state.Targets) == 0 {
		state.Targets = nil
		return state
	}
	next := make(map[string]ReplyCipherTarget, len(state.Targets))
	for _, target := range state.Targets {
		target = normalizeReplyCipherTarget(target)
		if target.UserID == 0 {
			continue
		}
		next[idKey(target.UserID)] = target
	}
	if len(next) == 0 {
		state.Targets = nil
	} else {
		state.Targets = next
	}
	return state
}

func normalizeReplyCipherTarget(target ReplyCipherTarget) ReplyCipherTarget {
	target.Username = normalizeUsername(target.Username)
	target.DisplayName = strings.TrimSpace(target.DisplayName)
	target.Cipher = normalizeAIReplyCipher(target.Cipher)
	target.AddedByName = strings.TrimSpace(target.AddedByName)
	return target
}

func emptyReplyCipherRuntimeState(state ReplyCipherRuntimeState) bool {
	state = normalizeReplyCipherRuntimeState(state)
	return state.DefaultCipher == "" && len(state.Targets) == 0 && state.UpdatedUnix == 0
}

func replyCipherRuntimeUpdated(state ReplyCipherRuntimeState, by *tele.User, now time.Time) ReplyCipherRuntimeState {
	state = normalizeReplyCipherRuntimeState(state)
	state.UpdatedUnix = now.UTC().Unix()
	if by != nil {
		state.UpdatedByID = by.ID
		state.UpdatedByName = displayNameFromUser(by)
	}
	return state
}

func (a *App) handleReplyCipher(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	if a.store == nil {
		return c.Send("Runtime store is not configured.")
	}

	args := c.Args()
	if len(args) == 0 || replyCipherListToken(args[0]) {
		return c.Send(a.replyCipherStatusText())
	}

	state := a.store.ReplyCipherRuntimeState()
	token := strings.ToLower(strings.TrimSpace(args[0]))
	if token == "help" || token == "usage" {
		return c.Send(replyCipherUsageText())
	}
	if replyCipherDefaultToken(token) {
		return a.setReplyCipherDefault(c, state, args[1:])
	}
	if replyCipherRemoveToken(token) {
		return a.removeReplyCipherTarget(c, state, args[1:])
	}
	if token == "add" || token == "on" {
		return a.addReplyCipherTarget(c, state, args[1:], "")
	}
	if cipher := normalizeAIReplyCipher(token); cipher != "" {
		if msg := c.Message(); msg != nil && msg.ReplyTo != nil && msg.ReplyTo.Sender != nil {
			return a.addReplyCipherTarget(c, state, args[1:], cipher)
		}
		if len(args) == 1 {
			return a.setReplyCipherDefault(c, state, args[:1])
		}
		return a.addReplyCipherTarget(c, state, args[1:], cipher)
	}

	return a.addReplyCipherTarget(c, state, args, "")
}

func replyCipherUsageText() string {
	return strings.Join([]string{
		"Шифр Лайи:",
		"/cipher list",
		"/cipher default binary|hex|base64",
		"/cipher @user [binary|hex|base64]",
		"/cipher off @user",
		"Можно ответить командой на сообщение: /cipher binary или /cipher off.",
	}, "\n")
}

func replyCipherListToken(token string) bool {
	switch strings.ToLower(strings.TrimSpace(token)) {
	case "list", "status", "show", "список", "статус":
		return true
	default:
		return false
	}
}

func replyCipherDefaultToken(token string) bool {
	switch strings.ToLower(strings.TrimSpace(token)) {
	case "default", "set", "mode", "cipher", "шифр":
		return true
	default:
		return false
	}
}

func replyCipherRemoveToken(token string) bool {
	switch strings.ToLower(strings.TrimSpace(token)) {
	case "off", "remove", "delete", "del", "clear", "выкл", "убрать":
		return true
	default:
		return false
	}
}

func (a *App) setReplyCipherDefault(c tele.Context, state ReplyCipherRuntimeState, args []string) error {
	if len(args) == 0 {
		return c.Send(replyCipherUsageText())
	}
	cipher := normalizeAIReplyCipher(args[0])
	if cipher == "" {
		return c.Send("Не знаю такой шифр. Доступно: binary, hex, base64.")
	}
	state.DefaultCipher = cipher
	state = replyCipherRuntimeUpdated(state, c.Sender(), time.Now().UTC())
	if err := a.store.SetReplyCipherRuntimeState(state); err != nil {
		return err
	}
	return c.Send(fmt.Sprintf("Готово. Шифр по умолчанию: %s.", replyCipherDisplayName(cipher)))
}

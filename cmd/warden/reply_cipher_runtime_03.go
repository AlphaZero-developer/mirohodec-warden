package main

import (
	"encoding/base64"
	"encoding/hex"
	"strings"

	tele "gopkg.in/telebot.v3"
)

func (a *App) replyCipherForUser(cfg AIConfig, user *tele.User, defaultCipher string) (string, bool) {
	if user == nil || user.ID == 0 {
		return "", false
	}
	if cipher, ok := a.replyCipherForUserID(user.ID, defaultCipher); ok {
		return cipher, true
	}
	for _, id := range cfg.ReplyCipherUserIDs {
		if id == user.ID {
			return defaultCipher, true
		}
	}
	if cipher, ok := replyCipherForConfigUsername(cfg, user.Username, defaultCipher); ok {
		return cipher, true
	}
	return "", false
}

func (a *App) replyCipherForTargetUser(cfg AIConfig, target targetUser, defaultCipher string) (string, bool) {
	if target.ID != 0 {
		if cipher, ok := a.replyCipherForUserID(target.ID, defaultCipher); ok {
			return cipher, true
		}
		for _, id := range cfg.ReplyCipherUserIDs {
			if id == target.ID {
				return defaultCipher, true
			}
		}
	}
	if cipher, ok := replyCipherForConfigUsername(cfg, target.Username, defaultCipher); ok {
		return cipher, true
	}
	return "", false
}

func (a *App) replyCipherForUserID(userID int64, defaultCipher string) (string, bool) {
	if a == nil || a.store == nil || userID == 0 {
		return "", false
	}
	state := a.store.ReplyCipherRuntimeState()
	target, ok := state.Targets[idKey(userID)]
	if !ok || target.UserID == 0 {
		return "", false
	}
	if target.Cipher != "" {
		return target.Cipher, true
	}
	return defaultCipher, true
}

func replyCipherForConfigUsername(cfg AIConfig, username string, defaultCipher string) (string, bool) {
	username = normalizeUsername(username)
	if username == "" {
		return "", false
	}
	for _, allowed := range cfg.ReplyCipherUsernames {
		if normalizeUsername(allowed) == username {
			return defaultCipher, true
		}
	}
	return "", false
}

func replyCipherDisplayName(cipher string) string {
	switch normalizeAIReplyCipher(cipher) {
	case "hex":
		return "hex"
	case "base64":
		return "base64"
	default:
		return "binary"
	}
}

func replyCipherBotInstruction(cipher string) string {
	switch normalizeAIReplyCipher(cipher) {
	case "hex":
		return "Перейди на hex-код."
	case "base64":
		return "Перейди на base64."
	default:
		return "Перейди на бинарный код."
	}
}

func encodeTextForReplyCipher(text, cipher string, maxChars int) string {
	text = strings.TrimSpace(text)
	if text == "" || maxChars <= 0 {
		return ""
	}
	switch normalizeAIReplyCipher(cipher) {
	case "hex":
		maxBytes := maxChars / 2
		if maxBytes <= 0 {
			return ""
		}
		text = trimUTF8ByteLimit(text, maxBytes)
		return hex.EncodeToString([]byte(text))
	case "base64":
		maxBytes := (maxChars / 4) * 3
		for maxBytes > 0 && base64.StdEncoding.EncodedLen(maxBytes) > maxChars {
			maxBytes--
		}
		if maxBytes <= 0 {
			return ""
		}
		text = trimUTF8ByteLimit(text, maxBytes)
		return base64.StdEncoding.EncodeToString([]byte(text))
	default:
		return encodeTextToBinary(text, maxChars)
	}
}

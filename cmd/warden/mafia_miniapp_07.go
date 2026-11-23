package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	wardenweb "mirohodec-warden/internal/web"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (s *MafiaService) AddMiniAppPrivateSystemMessage(roomID string, recipientID int64, text string, important bool, now time.Time) error {
	if s == nil || s.store == nil || recipientID == 0 || strings.TrimSpace(text) == "" {
		return nil
	}
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return errMafiaRoomNotFound
	}
	text = strings.TrimSpace(text)
	if mafiaMiniPrivateSystemMessageExists(room, recipientID, text) {
		return nil
	}
	msg := MafiaMiniAppMessage{
		ID:          fmt.Sprintf("private_system_%d_%d", recipientID, now.UnixNano()),
		RoomID:      room.ID,
		Channel:     mafiaMiniChannelSystem,
		RecipientID: recipientID,
		SenderName:  "Лайя",
		Text:        text,
		System:      true,
		Important:   important,
		CreatedAt:   now.UTC().Unix(),
	}
	room.MiniAppMessages = append(room.MiniAppMessages, msg)
	if len(room.MiniAppMessages) > 300 {
		room.MiniAppMessages = room.MiniAppMessages[len(room.MiniAppMessages)-300:]
	}
	room.UpdatedAt = now.UTC().Unix()
	return s.store.SaveMafiaRoom(room)
}

func appendMiniAppDetectiveResultMessages(room *MafiaRoom, now time.Time) {
	if room == nil || len(room.NightActions.DetectiveChecks) == 0 {
		return
	}
	for detectiveKey, targetID := range room.NightActions.DetectiveChecks {
		detectiveID, err := strconvParseInt64(detectiveKey)
		if err != nil || detectiveID == 0 || targetID == 0 {
			continue
		}
		checked, ok := room.NightActions.DetectiveResults[detectiveKey]
		if !ok {
			targetIdx := mafiaRoomPlayerIndex(*room, targetID)
			checked = targetIdx >= 0 && room.Players[targetIdx].Role == MafiaRoleMafia
		}
		if targetIdx := mafiaRoomPlayerIndex(*room, targetID); targetIdx >= 0 {
			mafiaRememberDetectiveKnownRole(room, detectiveID, targetID, room.Players[targetIdx].Role)
		}
		text := mafiaMiniDetectiveResultText(*room, targetID, checked)
		if mafiaMiniPrivateSystemMessageExists(*room, detectiveID, text) {
			continue
		}
		msg := MafiaMiniAppMessage{
			ID:          fmt.Sprintf("private_system_%d_%d_%d", detectiveID, now.UnixNano(), len(room.MiniAppMessages)),
			RoomID:      room.ID,
			Channel:     mafiaMiniChannelSystem,
			RecipientID: detectiveID,
			SenderName:  "Лайя",
			Text:        text,
			System:      true,
			Important:   true,
			CreatedAt:   now.UTC().Unix(),
		}
		room.MiniAppMessages = append(room.MiniAppMessages, msg)
	}
	if len(room.MiniAppMessages) > 300 {
		room.MiniAppMessages = room.MiniAppMessages[len(room.MiniAppMessages)-300:]
	}
}

func mafiaRememberDetectiveKnownRole(room *MafiaRoom, detectiveID int64, targetID int64, role MafiaRole) {
	if room == nil || detectiveID == 0 || targetID == 0 || role == "" {
		return
	}
	if room.DetectiveKnownRoles == nil {
		room.DetectiveKnownRoles = map[string]map[string]MafiaRole{}
	}
	detectiveKey := idKey(detectiveID)
	if room.DetectiveKnownRoles[detectiveKey] == nil {
		room.DetectiveKnownRoles[detectiveKey] = map[string]MafiaRole{}
	}
	room.DetectiveKnownRoles[detectiveKey][idKey(targetID)] = role
}

func mafiaMiniPrivateSystemMessageExists(room MafiaRoom, recipientID int64, text string) bool {
	text = strings.TrimSpace(text)
	for _, msg := range room.MiniAppMessages {
		if msg.System && msg.RecipientID == recipientID && strings.TrimSpace(msg.Text) == text {
			return true
		}
	}
	return false
}

func mafiaMiniSystemMessageExists(room MafiaRoom, text string) bool {
	text = strings.TrimSpace(text)
	for _, msg := range room.MiniAppMessages {
		if msg.System && msg.RecipientID == 0 && strings.TrimSpace(msg.Text) == text {
			return true
		}
	}
	return false
}

func validateTelegramMiniAppInitData(initData string, botToken string, maxAge time.Duration) (mafiaMiniAuthUser, error) {
	initData = strings.TrimSpace(initData)
	if initData == "" {
		return mafiaMiniAuthUser{}, errors.New("telegram initData is required")
	}
	values, err := url.ParseQuery(initData)
	if err != nil {
		return mafiaMiniAuthUser{}, errors.New("invalid telegram initData")
	}
	receivedHash := values.Get("hash")
	if receivedHash == "" {
		return mafiaMiniAuthUser{}, errors.New("telegram initData hash is missing")
	}
	values.Del("hash")
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+"="+values.Get(key))
	}
	secretHMAC := hmac.New(sha256.New, []byte("WebAppData"))
	_, _ = secretHMAC.Write([]byte(botToken))
	secret := secretHMAC.Sum(nil)
	checkHMAC := hmac.New(sha256.New, secret)
	_, _ = checkHMAC.Write([]byte(strings.Join(lines, "\n")))
	expected := hex.EncodeToString(checkHMAC.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(expected), []byte(receivedHash)) != 1 {
		return mafiaMiniAuthUser{}, errors.New("telegram initData signature is invalid")
	}
	authDate, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil || authDate <= 0 {
		return mafiaMiniAuthUser{}, errors.New("telegram initData auth_date is invalid")
	}
	if maxAge > 0 && time.Since(time.Unix(authDate, 0)) > maxAge {
		return mafiaMiniAuthUser{}, errors.New("telegram initData is expired")
	}
	var user mafiaMiniAuthUser
	if err := json.Unmarshal([]byte(values.Get("user")), &user); err != nil {
		return mafiaMiniAuthUser{}, errors.New("telegram initData user is invalid")
	}
	return user, nil
}

func mafiaMiniDevAuthEnabled() bool {
	value := strings.TrimSpace(os.Getenv("DEV_AUTH"))
	return value == "1" || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes")
}

func mafiaMiniTeleUser(user mafiaMiniAuthUser) *tele.User {
	return &tele.User{ID: user.ID, Username: user.Username, FirstName: user.FirstName, LastName: user.LastName, IsBot: user.IsBot}
}

func mafiaMiniAppWebUser(user mafiaMiniAuthUser) wardenweb.MafiaMiniAppUser {
	return wardenweb.MafiaMiniAppUser{
		ID:          user.ID,
		Username:    user.Username,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		DisplayName: mafiaMiniDisplayName(user),
	}
}

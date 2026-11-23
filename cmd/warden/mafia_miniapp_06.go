package main

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

func mafiaMiniReplySnapshot(room MafiaRoom, channel string, replyToID string) (string, string, string) {
	replyToID = strings.TrimSpace(replyToID)
	if replyToID == "" {
		return "", "", ""
	}
	channel = strings.ToUpper(strings.TrimSpace(channel))
	for _, msg := range room.MiniAppMessages {
		if msg.ID != replyToID || strings.ToUpper(msg.Channel) != channel {
			continue
		}
		return msg.ID, msg.SenderName, mafiaMiniReplyText(msg.Text)
	}
	return "", "", ""
}

func mafiaMiniReplyText(text string) string {
	const maxRunes = 120
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "..."
}

func mafiaMiniCleanAvatarURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) > 220000 {
		return ""
	}
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "data:image/") {
		if strings.Contains(lower[:minInt(len(lower), 80)], ";base64,") {
			return value
		}
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return ""
	}
	if parsed.Host == "" {
		return ""
	}
	if len(value) > 700 {
		return value[:700]
	}
	return value
}

func mafiaMiniCleanAvatarText(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	runes := []rune(value)
	if len(runes) > 3 {
		runes = runes[:3]
	}
	return string(runes)
}

func mafiaMiniCleanAvatarTone(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "teal", "red", "violet", "blue", "gold":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "teal"
	}
}

func (s *MafiaService) AddMiniAppMessage(roomID string, senderID int64, channel string, text string, replyToID string, now time.Time) (MafiaRoom, error) {
	if s == nil || s.store == nil {
		return MafiaRoom{}, errMafiaRoomNotFound
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return MafiaRoom{}, errors.New("message text is required")
	}
	if len([]rune(text)) > 1200 {
		return MafiaRoom{}, errors.New("message is too long")
	}
	channel = strings.ToUpper(strings.TrimSpace(channel))
	if channel == "" {
		channel = mafiaMiniChannelPublic
	}
	if channel == mafiaMiniChannelMafia {
		room, _, _, err := s.SendShadowMessage(roomID, senderID, text, now)
		return room, err
	}
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, errMafiaRoomNotFound
	}
	idx := mafiaRoomPlayerIndex(room, senderID)
	isHost := room.OwnerID == senderID && normalizeMafiaSettings(room.Settings).EnableHostPowers
	if !mafiaMiniCanWriteChannel(room, idx, isHost, channel) {
		return MafiaRoom{}, errors.New("you cannot write to this channel now")
	}
	if channel == mafiaMiniChannelPublic && idx >= 0 {
		settings := normalizeMafiaSettings(room.Settings)
		if DetectMafiaSystemProof(text).IsRoleReveal || DetectMafiaCriticalLeak(text).ShouldPunish || (settings.RoleRevealPolicy == MafiaRoleRevealBanAllClaims && DetectMafiaRoleReveal(text).IsRoleReveal) {
			return MafiaRoom{}, errors.New("Лайя заблокировала сообщение: похоже на слив роли или приватной информации")
		}
	}
	name := "Host"
	if idx >= 0 {
		name = mafiaRoomPlayerLabel(room, room.Players[idx])
	} else if user, ok := s.store.MafiaUser(senderID); ok {
		name = mafiaUserCharacterName(user)
	}
	replyToID, replyToSenderName, replyToText := mafiaMiniReplySnapshot(room, channel, replyToID)
	msg := MafiaMiniAppMessage{
		ID:                fmt.Sprintf("mini_%d_%d", now.UnixNano(), senderID),
		RoomID:            room.ID,
		Channel:           channel,
		SenderID:          senderID,
		SenderName:        name,
		Text:              text,
		ReplyToID:         replyToID,
		ReplyToSenderName: replyToSenderName,
		ReplyToText:       replyToText,
		CreatedAt:         now.UTC().Unix(),
	}
	room.MiniAppMessages = append(room.MiniAppMessages, msg)
	if len(room.MiniAppMessages) > 300 {
		room.MiniAppMessages = room.MiniAppMessages[len(room.MiniAppMessages)-300:]
	}
	room.UpdatedAt = now.UTC().Unix()
	return room, s.store.SaveMafiaRoom(room)
}

func (s *MafiaService) AddMiniAppSystemMessage(roomID string, text string, important bool, now time.Time) error {
	if s == nil || s.store == nil || strings.TrimSpace(text) == "" {
		return nil
	}
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return errMafiaRoomNotFound
	}
	appendMiniAppSystemMessageToRoom(&room, text, important, now)
	room.UpdatedAt = now.UTC().Unix()
	return s.store.SaveMafiaRoom(room)
}

func appendMiniAppSystemMessageToRoom(room *MafiaRoom, text string, important bool, now time.Time) {
	if room == nil || strings.TrimSpace(text) == "" {
		return
	}
	text = strings.TrimSpace(text)
	if mafiaMiniSystemMessageExists(*room, text) {
		return
	}
	msg := MafiaMiniAppMessage{
		ID:         fmt.Sprintf("system_%d_%d", now.UnixNano(), len(room.MiniAppMessages)),
		RoomID:     room.ID,
		Channel:    mafiaMiniChannelSystem,
		SenderName: "Лайя",
		Text:       text,
		System:     true,
		Important:  important,
		CreatedAt:  now.UTC().Unix(),
	}
	room.MiniAppMessages = append(room.MiniAppMessages, msg)
	if len(room.MiniAppMessages) > 300 {
		room.MiniAppMessages = room.MiniAppMessages[len(room.MiniAppMessages)-300:]
	}
}

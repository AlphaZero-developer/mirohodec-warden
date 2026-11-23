package main

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) shouldSilenceMafiaDiscussionMessage(chat *tele.Chat, msg *tele.Message) bool {
	if a == nil || chat == nil || msg == nil || chat.Type == tele.ChatPrivate {
		return false
	}
	if commandName(msg.Text) != "" {
		return false
	}
	if len(a.mafiaActiveRoomsForGroup(chat.ID)) > 0 {
		return true
	}
	return a.mafiaDiscussionChatRemembered(chat.ID)
}

func (a *App) mafiaDiscussionChatRemembered(chatID int64) bool {
	if a == nil || a.store == nil || chatID == 0 {
		return false
	}
	for _, user := range a.store.MafiaUsers() {
		settings := normalizeMafiaUserSettings(user.Settings)
		if settings.LastDiscussionGroupChatID == chatID {
			return true
		}
	}
	return false
}

func mafiaIsGroupSkipPhaseText(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	normalized = strings.TrimSpace(strings.TrimPrefix(normalized, "⏭"))
	normalized = strings.Join(strings.Fields(normalized), " ")
	switch normalized {
	case "скип фазы", "скипнуть фазу", "пропустить фазу":
		return true
	default:
		return false
	}
}

func mafiaRoomAllowsPublicPhaseSkip(status MafiaRoomStatus) bool {
	switch status {
	case MafiaStatusDayDiscussion, MafiaStatusVoting:
		return true
	default:
		return false
	}
}

func (a *App) handleMafiaGroupPhaseSkip(c tele.Context, room MafiaRoom, voter MafiaPlayer) {
	if !mafiaRoomAllowsPublicPhaseSkip(room.Status) {
		_ = c.Send("⛔ Сейчас эту фазу нельзя скипнуть из общего чата.")
		return
	}
	alreadyVoted := room.PhaseSkipVotes != nil && room.PhaseSkipVotes[idKey(voter.TelegramID)]
	now := time.Now().UTC()
	updated, votes, needed, reached, err := a.mafiaService().VotePhaseSkip(room.ID, voter.TelegramID)
	if err != nil {
		_ = c.Send("⛔ " + a.localizeMafiaErr(err))
		return
	}
	voterName := mafiaRoomPlayerLabelWithUsername(updated, voter)
	if alreadyVoted {
		_ = c.Send(fmt.Sprintf("⏭ %s уже голосовал за скип фазы.\n\nГолосов: %d/%d.", voterName, votes, needed))
		return
	}
	if reached {
		_ = c.Send(fmt.Sprintf("⏭ %s голосует за скип фазы.\n\nГолосов: %d/%d.\nЛайя переводит игру дальше.", voterName, votes, needed))
		if err := a.advanceMafiaRoomAsLaya(updated.ID, now); err != nil {
			_ = c.Send("⛔ " + a.localizeMafiaErr(err))
		}
		return
	}
	_ = c.Send(fmt.Sprintf("⏭ %s голосует за скип фазы.\n\nГолосов: %d/%d.", voterName, votes, needed))
}

func (a *App) shouldSendMafiaGroupNotice(room MafiaRoom, userID int64, kind string, now time.Time) bool {
	if a == nil {
		return true
	}
	a.mafiaNoticeMu.Lock()
	defer a.mafiaNoticeMu.Unlock()
	if a.lastMafiaNotice == nil {
		a.lastMafiaNotice = make(map[string]time.Time)
	}
	key := mafiaGroupNoticeKey(room, userID, kind)
	if key == "" {
		return true
	}
	if _, ok := a.lastMafiaNotice[key]; ok {
		return false
	}
	a.lastMafiaNotice[key] = now
	for noticeKey, ts := range a.lastMafiaNotice {
		if now.Sub(ts) > 6*time.Hour {
			delete(a.lastMafiaNotice, noticeKey)
		}
	}
	return true
}

func mafiaGroupNoticeKey(room MafiaRoom, userID int64, kind string) string {
	kind = strings.TrimSpace(kind)
	roomID := normalizeMafiaRoomID(room.ID)
	if roomID == "" || kind == "" {
		return ""
	}
	phaseKey := room.PhaseStartedAt
	if phaseKey <= 0 {
		phaseKey = int64(room.DayNumber)
	}
	return fmt.Sprintf("%s:%s:%d:%d:%s", roomID, room.Status, phaseKey, userID, kind)
}

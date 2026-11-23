package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func rehabReminderSlot(cfg RehabConfig, now time.Time) (string, bool) {
	loc := moscowLocation(cfg.Timezone)
	local := now.In(loc)
	window := mustDuration(cfg.ReminderWindow, 15*time.Minute)
	if window <= 0 {
		window = 15 * time.Minute
	}

	for _, raw := range cfg.ReminderTimes {
		hour, minute, ok := parseClock(raw)
		if !ok {
			continue
		}
		scheduled := time.Date(local.Year(), local.Month(), local.Day(), hour, minute, 0, 0, loc)
		if !local.Before(scheduled) && local.Sub(scheduled) < window {
			return fmt.Sprintf("%s %02d:%02d", local.Format("2006-01-02"), hour, minute), true
		}
	}
	return "", false
}

func moscowLocation(name string) *time.Location {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Europe/Moscow"
	}
	loc, err := time.LoadLocation(name)
	if err == nil {
		return loc
	}
	return time.FixedZone("MSK", 3*60*60)
}

func parseClock(value string) (int, int, bool) {
	value = strings.TrimSpace(value)
	var hour, minute int
	if _, err := fmt.Sscanf(value, "%d:%d", &hour, &minute); err != nil {
		return 0, 0, false
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, false
	}
	return hour, minute, true
}

func (a *App) claimRehabNudge(chatID, userID int64, slot string) bool {
	key := rehabStateKey(chatID, userID)
	a.rehabMu.Lock()
	defer a.rehabMu.Unlock()
	if a.rehabNudgeSlots[key] == slot {
		return false
	}
	a.rehabNudgeSlots[key] = slot
	return true
}

func (a *App) sendRehabPrivateReminder(chatID int64, rec ModerationRecord) {
	cfg := a.config().Rehab
	text := strings.TrimSpace(cfg.PrivateMessage)
	if text == "" {
		text = "Привет, когда ты уже выздоровеешь?"
	}
	text = strings.ReplaceAll(text, "{name}", rec.DisplayName)

	_, err := a.bot.Send(&tele.User{ID: rec.UserID}, text)
	if err == nil {
		a.setRehabDMBlocked(chatID, rec.UserID, false)
		if a.metrics != nil {
			a.metrics.IncModeration("hard_rehab_private_reminder")
		}
		return
	}

	if telegramPrivateUnavailable(err) {
		a.setRehabDMBlocked(chatID, rec.UserID, true)
	} else if a.log != nil {
		a.log.Warn("send rehab private reminder failed",
			zap.Int64("chat", chatID),
			zap.Int64("user", rec.UserID),
			zap.Error(err),
		)
	}
}

func telegramPrivateUnavailable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, tele.ErrBlockedByUser) || errors.Is(err, tele.ErrNotStartedByUser) || errors.Is(err, tele.ErrUserIsDeactivated) {
		return true
	}
	var teleErr *tele.Error
	if errors.As(err, &teleErr) && teleErr.Code == 403 {
		return true
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "bot was blocked") ||
		strings.Contains(text, "can't initiate conversation") ||
		strings.Contains(text, "user is deactivated")
}

func (a *App) setRehabDMBlocked(chatID, userID int64, blocked bool) {
	key := rehabStateKey(chatID, userID)
	a.rehabMu.Lock()
	defer a.rehabMu.Unlock()
	if blocked {
		a.rehabDMBlocked[key] = true
		return
	}
	delete(a.rehabDMBlocked, key)
}

func (a *App) shouldUseHardRehabReply(chatID, userID int64) bool {
	if !a.config().Rehab.HardMode {
		return false
	}
	key := rehabStateKey(chatID, userID)
	a.rehabMu.Lock()
	defer a.rehabMu.Unlock()
	return a.rehabDMBlocked[key]
}

func rehabStateKey(chatID, userID int64) string {
	return fmt.Sprintf("%d:%d", chatID, userID)
}

func joinedUsersFromMessage(msg *tele.Message) []*tele.User {
	if msg == nil {
		return nil
	}
	if msg.UserJoined != nil {
		return []*tele.User{msg.UserJoined}
	}
	users := make([]*tele.User, 0, len(msg.UsersJoined))
	for i := range msg.UsersJoined {
		users = append(users, &msg.UsersJoined[i])
	}
	return users
}

func chatMemberBecamePresent(oldMember, newMember *tele.ChatMember) bool {
	if newMember == nil || newMember.User == nil {
		return false
	}
	return !chatMemberPresent(oldMember) && chatMemberPresent(newMember)
}

func chatMemberPresent(member *tele.ChatMember) bool {
	if member == nil {
		return false
	}
	switch member.Role {
	case tele.Member, tele.Administrator, tele.Creator:
		return true
	case tele.Restricted:
		return member.Member
	default:
		return false
	}
}

func (a *App) handleJoinedUserEvent(chat *tele.Chat, user, actor *tele.User, inviteLink *tele.ChatInviteLink, viaFolder bool, threadID int, source string) {
	if a == nil || chat == nil || user == nil || user.IsBot {
		return
	}
	a.processJoinedUser(chat, user)
	if !a.claimJoinEvent(chat.ID, user.ID, time.Now().UTC()) {
		return
	}
	a.sendJoinLog(chat, user, actor, inviteLink, viaFolder, source)
	a.sendWelcomeReaction(chat, user, actor, threadID)
}

func (a *App) claimJoinEvent(chatID, userID int64, now time.Time) bool {
	key := rehabStateKey(chatID, userID)
	a.joinMu.Lock()
	defer a.joinMu.Unlock()
	if last := a.lastJoinEvents[key]; !last.IsZero() && now.Sub(last) < 30*time.Second {
		return false
	}
	a.lastJoinEvents[key] = now
	return true
}

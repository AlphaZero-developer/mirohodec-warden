package main

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) announceMafiaLobbyPlayersAdded(room MafiaRoom, userIDs []int64) {
	if room.Status != MafiaStatusLobby || room.GroupChatID == 0 || len(userIDs) == 0 {
		return
	}
	a.syncMafiaGroupPlayerTitles(room)
	var names []string
	for _, userID := range userIDs {
		if userID <= 0 {
			continue
		}
		names = append(names, mafiaPlayerPublicName(room, userID))
	}
	if len(names) == 0 {
		return
	}
	a.sendMafiaGroup(room, fmt.Sprintf("🎭 В лобби добавлены игроки:\n%s\n\nКомната: %s\nИгроков: %d/%d", strings.Join(names, "\n"), room.ID, len(room.Players), normalizeMafiaSettings(room.Settings).MaxPlayers), nil)
}

func (a *App) syncMafiaGroupPlayerTitles(room MafiaRoom) {
	if a == nil || a.bot == nil || room.GroupChatID == 0 {
		return
	}
	chat := &tele.Chat{ID: room.GroupChatID}
	for _, player := range room.Players {
		if player.IsTestBot || player.TelegramID <= 0 {
			continue
		}
		title := mafiaPlayerAdminTitle(player)
		if title == "" {
			continue
		}
		member, err := a.bot.ChatMemberOf(chat, &tele.User{ID: player.TelegramID})
		if err != nil {
			if a.log != nil {
				a.log.Debug("check mafia player title eligibility failed", zap.String("room", room.ID), zap.Int64("chat", room.GroupChatID), zap.Int64("user", player.TelegramID), zap.Error(err))
			}
			continue
		}
		if member == nil || member.Role != tele.Administrator {
			continue
		}
		if strings.TrimSpace(member.Title) == title {
			continue
		}
		if err := a.bot.SetAdminTitle(chat, &tele.User{ID: player.TelegramID}, title); err != nil {
			if a.log != nil {
				a.log.Debug("set mafia player admin title failed", zap.String("room", room.ID), zap.Int64("chat", room.GroupChatID), zap.Int64("user", player.TelegramID), zap.String("title", title), zap.Error(err))
			}
			continue
		}
		if a.metrics != nil {
			a.metrics.IncModeration("mafia_player_title")
		}
	}
}

func mafiaPlayerAdminTitle(player MafiaPlayer) string {
	raw := mafiaPlayerLabel(player)
	var b strings.Builder
	space := false
	for _, r := range raw {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(r)
			space = false
		case unicode.IsSpace(r) || r == '-' || r == '_' || r == '\'':
			if !space && b.Len() > 0 {
				b.WriteRune(' ')
				space = true
			}
		}
	}
	title := strings.Join(strings.Fields(b.String()), " ")
	runes := []rune(title)
	if len(runes) > 16 {
		title = strings.TrimSpace(string(runes[:16]))
	}
	if title == "" {
		return "Игрок"
	}
	return title
}

func (a *App) syncMafiaGroupRestrictions(room MafiaRoom) {
	if a == nil || a.bot == nil || room.GroupChatID == 0 {
		return
	}
	now := time.Now().UTC()
	for _, player := range room.Players {
		if player.IsTestBot || player.TelegramID <= 0 {
			continue
		}
		switch {
		case room.Status == MafiaStatusNight || room.Status == MafiaStatusRoleDistribution:
			until := mafiaTemporaryMuteUntil(room, player, now)
			a.restrictMafiaGroupPlayer(room, player, until, "mafia_phase_mute")
		case room.Status == MafiaStatusDayDiscussion || room.Status == MafiaStatusVoting:
			if player.Alive {
				a.unrestrictMafiaGroupPlayer(room, player, now, "mafia_phase_unmute")
			} else {
				a.restrictMafiaGroupPlayer(room, player, now.Add(24*time.Hour), "mafia_dead_mute")
			}
		case room.Status == MafiaStatusLobby || room.Status == MafiaStatusFinished || room.Status == MafiaStatusCancelled:
			a.unrestrictMafiaGroupPlayer(room, player, now, "mafia_game_unmute")
		}
	}
}

func mafiaTemporaryMuteUntil(room MafiaRoom, player MafiaPlayer, now time.Time) time.Time {
	if !player.Alive {
		return now.Add(24 * time.Hour)
	}
	settings := normalizeMafiaSettings(room.Settings)
	seconds := settings.NightActionSeconds
	if room.Status == MafiaStatusRoleDistribution || seconds <= 0 {
		seconds = 300
	}
	if seconds < 60 {
		seconds = 60
	}
	return now.Add(time.Duration(seconds+180) * time.Second)
}

func (a *App) restrictMafiaGroupPlayer(room MafiaRoom, player MafiaPlayer, until time.Time, action string) {
	rights := tele.NoRights()
	rights.Independent = true
	member := &tele.ChatMember{
		User:            &tele.User{ID: player.TelegramID},
		Rights:          rights,
		RestrictedUntil: until.UTC().Unix(),
	}
	if err := a.bot.Restrict(&tele.Chat{ID: room.GroupChatID}, member); err != nil {
		if a.log != nil {
			a.log.Warn("restrict mafia group player failed", zap.String("room", room.ID), zap.Int64("chat", room.GroupChatID), zap.Int64("user", player.TelegramID), zap.String("action", action), zap.Error(err))
		}
		return
	}
	if a.metrics != nil {
		a.metrics.IncModeration(action)
	}
}

func (a *App) unrestrictMafiaGroupPlayer(room MafiaRoom, player MafiaPlayer, now time.Time, action string) {
	if a.store != nil {
		if _, ok := a.store.ActiveMute(room.GroupChatID, player.TelegramID, now); ok {
			return
		}
		if _, ok := a.store.ActiveJail(room.GroupChatID, player.TelegramID, now); ok {
			return
		}
		if _, ok := a.store.Ban(room.GroupChatID, player.TelegramID); ok {
			return
		}
	}
	member := &tele.ChatMember{
		User:   &tele.User{ID: player.TelegramID},
		Rights: tele.NoRestrictions(),
	}
	if err := a.bot.Restrict(&tele.Chat{ID: room.GroupChatID}, member); err != nil {
		if a.log != nil {
			a.log.Warn("unrestrict mafia group player failed", zap.String("room", room.ID), zap.Int64("chat", room.GroupChatID), zap.Int64("user", player.TelegramID), zap.String("action", action), zap.Error(err))
		}
		return
	}
	if a.metrics != nil {
		a.metrics.IncModeration(action)
	}
}

package main

import (
	"context"
	"errors"
	"fmt"
	wardenweb "mirohodec-warden/internal/web"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (s webPanelService) Users(ctx context.Context, chatID int64) ([]wardenweb.UserRow, error) {
	chatID = s.resolveChatID(chatID)
	adminRoles := s.webTelegramAdminRoles(chatID)
	members := s.webKnownMembers(chatID)
	out := make([]wardenweb.UserRow, 0, len(members))
	for _, rec := range members {
		role := s.webUserRoleLabel(chatID, rec.ID, adminRoles)
		_, banned := s.app.store.Ban(chatID, rec.ID)
		out = append(out, wardenweb.UserRow{
			ID:            rec.ID,
			DisplayName:   rec.DisplayName,
			Username:      rec.Username,
			Role:          role,
			CanBan:        s.webCanForeverBanUser(rec.ID, role, banned),
			Banned:        banned,
			Messages:      rec.Messages,
			MessagesToday: messagesToday(rec),
			Karma:         s.app.store.Karma(chatID, rec.ID),
			LastMessage:   webFormatUnix(rec.LastMessage),
			LastThreadID:  rec.LastThreadID,
		})
	}
	return out, nil
}

func (s webPanelService) webKnownMembers(chatID int64) []MemberRecord {
	if s.app == nil || s.app.store == nil {
		return nil
	}
	members := s.app.store.Members(chatID)
	sort.SliceStable(members, func(i, j int) bool {
		if members[i].Messages == members[j].Messages {
			return strings.ToLower(members[i].DisplayName) < strings.ToLower(members[j].DisplayName)
		}
		return members[i].Messages > members[j].Messages
	})
	return members
}

func (s webPanelService) webTelegramAdminRoles(chatID int64) map[int64]string {
	if s.app == nil || s.app.bot == nil || chatID == 0 {
		return nil
	}
	admins, err := s.app.bot.AdminsOf(&tele.Chat{ID: chatID, Type: tele.ChatSuperGroup})
	if err != nil {
		if s.app.log != nil {
			s.app.log.Warn("web admin role lookup failed", zap.Int64("chat", chatID), zap.Error(err))
		}
		return nil
	}
	roles := make(map[int64]string, len(admins))
	for _, admin := range admins {
		if admin.User == nil || admin.User.ID == 0 {
			continue
		}
		switch admin.Role {
		case tele.Creator:
			roles[admin.User.ID] = "создатель группы"
		case tele.Administrator:
			roles[admin.User.ID] = "админ"
		}
	}
	return roles
}

func (s webPanelService) webUserRoleLabel(chatID, userID int64, adminRoles map[int64]string) string {
	if s.app == nil || userID == 0 {
		return ""
	}
	var labels []string
	add := func(label string) {
		label = strings.TrimSpace(label)
		if label == "" {
			return
		}
		for _, existing := range labels {
			if existing == label {
				return
			}
		}
		labels = append(labels, label)
	}
	add(adminRoles[userID])
	if s.app.bot != nil && s.app.bot.Me != nil && userID == s.app.bot.Me.ID {
		add("бот")
	}
	if s.app.isConfiguredAdmin(userID) {
		add("админ бота")
	}
	if s.app.isProtectedUser(userID) {
		add("защищён")
	}
	if s.app.store != nil {
		if _, banned := s.app.store.Ban(chatID, userID); banned {
			add("вечный бан")
		}
	}
	return strings.Join(labels, ", ")
}

func (s webPanelService) webCanForeverBanUser(userID int64, role string, banned bool) bool {
	if s.app == nil || userID == 0 || banned {
		return false
	}
	if s.app.bot != nil && s.app.bot.Me != nil && userID == s.app.bot.Me.ID {
		return false
	}
	if s.app.isConfiguredAdmin(userID) || s.app.isProtectedUser(userID) {
		return false
	}
	role = strings.ToLower(role)
	return !strings.Contains(role, "создатель группы") && !strings.Contains(role, "админ")
}

func (s webPanelService) Moderation(ctx context.Context, chatID int64) (wardenweb.ModerationPage, error) {
	chatID = s.resolveChatID(chatID)
	now := time.Now().UTC()
	page := wardenweb.ModerationPage{ChatID: chatID}
	for _, member := range s.webKnownMembers(chatID) {
		page.Users = append(page.Users, wardenweb.UserChoice{ID: member.ID, Label: member.DisplayName})
	}
	for _, rec := range s.app.store.ModerationRecords(chatID, "jail", true, now) {
		page.Jails = append(page.Jails, moderationRow("rehab", rec))
	}
	for _, rec := range s.app.store.ModerationRecords(chatID, "mute", true, now) {
		page.Mutes = append(page.Mutes, moderationRow("mute", rec))
	}
	for _, rec := range s.app.store.ModerationRecords(chatID, "ban", true, now) {
		page.Bans = append(page.Bans, moderationRow("ban", rec))
	}
	return page, nil
}

func (s webPanelService) AddForeverBan(ctx context.Context, input wardenweb.ForeverBanInput) error {
	if s.app == nil || s.app.store == nil {
		return errors.New("app store is not ready")
	}
	if input.ChatID == 0 || input.UserID == 0 {
		return errors.New("chat_id and user_id are required")
	}
	member, _ := s.app.store.Member(input.ChatID, input.UserID)
	name := strings.TrimSpace(member.DisplayName)
	if name == "" {
		name = fmt.Sprintf("user %d", input.UserID)
	}
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		reason = "web panel"
	}
	target := targetUser{ID: input.UserID, Username: member.Username, Name: name, IsBot: member.IsBot}
	chat := &tele.Chat{ID: input.ChatID, Type: tele.ChatSuperGroup}
	return s.app.banForeverTarget(chat, target, reason, 0, "web panel", true)
}

func (s webPanelService) Warnings(ctx context.Context, chatID int64) ([]wardenweb.WarningRow, error) {
	chatID = s.resolveChatID(chatID)
	rows := s.app.store.Warnings(chatID, 0, false, 100, time.Now().UTC())
	out := make([]wardenweb.WarningRow, 0, len(rows))
	for _, rec := range rows {
		out = append(out, wardenweb.WarningRow{
			ID:          rec.ID,
			UserID:      rec.UserID,
			DisplayName: rec.DisplayName,
			Reason:      rec.Reason,
			Created:     webFormatUnix(rec.CreatedUnix),
			Expires:     webFormatUnix(rec.ExpiresUnix),
			By:          rec.ByAdminName,
			Active:      rec.Active,
		})
	}
	return out, nil
}

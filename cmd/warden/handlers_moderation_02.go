package main

import (
	"fmt"
	"html"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) handleBanForever(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	target, rest, err := a.targetFromArgs(chat.ID, c, c.Args())
	if err != nil {
		return c.Send(a.t("usage_banforever"))
	}
	if err := a.ensureModeratable(chat, target.ID); err != nil {
		return c.Send(a.localizeErr(err))
	}

	reason := strings.Join(rest, " ")
	admin := c.Sender()
	if err := a.banForeverTarget(chat, target, reason, admin.ID, displayNameFromUser(admin), true); err != nil {
		return c.Send(a.t("telegram_ban_failed", err))
	}
	return nil
}

func (a *App) banForeverTarget(chat *tele.Chat, target targetUser, reason string, byAdminID int64, byAdminName string, sendTaunt bool) error {
	if a == nil || a.store == nil {
		return fmt.Errorf("store is not ready")
	}
	if chat == nil || chat.ID == 0 || target.ID == 0 {
		return fmt.Errorf("chat_id and user_id are required")
	}
	if strings.TrimSpace(target.Name) == "" {
		target.Name = fmt.Sprintf("user %d", target.ID)
	}
	if strings.TrimSpace(byAdminName) == "" {
		byAdminName = "web panel"
	}
	if a.bot != nil {
		if err := a.ensureModeratable(chat, target.ID); err != nil {
			return err
		}
		member := &tele.ChatMember{
			User:            &tele.User{ID: target.ID},
			RestrictedUntil: tele.Forever(),
		}
		if err := a.bot.Ban(chat, member, true); err != nil {
			return err
		}
	}
	a.store.SetBan(chat.ID, ModerationRecord{
		UserID:      target.ID,
		Username:    target.Username,
		DisplayName: target.Name,
		Reason:      reason,
		ByAdminID:   byAdminID,
		ByAdminName: byAdminName,
		SinceUnix:   time.Now().UTC().Unix(),
		UntilUnix:   0,
	})
	if _, released := a.store.RemoveJail(chat.ID, target.ID); released {
		a.hideRehabEscapeCommand(chat.ID, target.ID)
		a.clearRehabEscapeCooldown(chat.ID, target.ID)
	}
	a.store.RemoveMute(chat.ID, target.ID)
	if a.metrics != nil {
		a.metrics.IncModeration("banforever")
	}
	a.sendOwnerLog("Forever ban", logField{"User", fmt.Sprintf("%s (%d)", target.Name, target.ID)}, logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)}, logField{"Reason", textOrDash(reason)}, logField{"By", byAdminName})
	if sendTaunt {
		a.sendForeverBanTaunt(chat)
	}
	return nil
}

func (a *App) sendForeverBanTaunt(chat *tele.Chat) {
	if a == nil || a.bot == nil || chat == nil {
		return
	}
	if _, err := a.bot.Send(chat, foreverBanTaunt); err != nil && a.log != nil {
		a.log.Warn("send forever ban taunt failed", zap.Int64("chat", chat.ID), zap.Error(err))
	}
}

func (a *App) handleUnbanForever(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	target, _, err := a.targetFromArgs(chat.ID, c, c.Args())
	if err != nil {
		return c.Send(a.t("usage_unbanforever"))
	}

	a.store.RemoveBan(chat.ID, target.ID)
	if err := a.bot.Unban(chat, &tele.User{ID: target.ID}, true); err != nil {
		return c.Send(a.t("telegram_unban_failed", err))
	}
	a.metrics.IncModeration("unbanforever")
	a.sendOwnerLog("Forever ban removed", logField{"User", fmt.Sprintf("%s (%d)", target.Name, target.ID)}, logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)}, logField{"By", displayNameFromUser(c.Sender())})
	return c.Send(a.t("unban_done", target.Name))
}

func (a *App) handleAll(c tele.Context) error {
	cfg := a.config()
	if !cfg.All.Enabled {
		return c.Send(a.t("all_disabled"))
	}
	if cfg.All.AdminsOnly {
		if !a.requireAdmin(c) {
			return nil
		}
	}

	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}

	members := a.store.Members(chat.ID)
	if len(members) == 0 {
		return c.Send(a.t("unknown_members"))
	}

	prefix := strings.TrimSpace(strings.Join(c.Args(), " "))
	if prefix == "" {
		prefix = a.t("all_default_prefix")
	}
	prefix = html.EscapeString(prefix)

	chunkSize := cfg.All.ChunkSize
	delay := mustDuration(cfg.All.DelayBetweenChunks, 1200*time.Millisecond)
	threadID := 0
	if msg := c.Message(); msg != nil {
		threadID = msg.ThreadID
	}

	for i := 0; i < len(members); i += chunkSize {
		end := i + chunkSize
		if end > len(members) {
			end = len(members)
		}

		mentions := make([]string, 0, end-i)
		for _, member := range members[i:end] {
			mentions = append(mentions, mentionUser(member.ID, member.DisplayName))
		}

		text := prefix + "\n" + strings.Join(mentions, " ")
		if _, err := a.bot.Send(chat, text, &tele.SendOptions{ThreadID: threadID, ParseMode: tele.ModeHTML}); err != nil {
			return c.Send(a.t("all_send_failed", err))
		}
		if end < len(members) {
			time.Sleep(delay)
		}
	}

	a.metrics.IncModeration("all")
	return nil
}

package main

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) applySelfJailFor(c tele.Context, chat *tele.Chat, sender *tele.User, d time.Duration) error {
	if err := a.ensureSelfModerationAllowed(chat, sender, false); err != nil {
		return a.layaSend(c, err.Error(), a.layaInlineMarkup(chat, sender))
	}
	rehabTopicID := a.rehabTopicID(chat)
	if rehabTopicID == 0 {
		return a.layaSend(c, "Для этого чата не настроена тема психиатрического центра.", a.layaInlineMarkup(chat, sender))
	}
	if d == layaSelfJailManual && !a.userHasRehabChiefTitle(chat.ID, sender.ID, time.Now().UTC()) {
		return a.layaSend(c, "Произвольный срок доступен только с титулом «главный в психбольнице».", a.layaInlineMarkup(chat, sender))
	}
	now := time.Now().UTC()
	target := targetUser{ID: sender.ID, Username: sender.Username, Name: displayNameFromUser(sender)}
	untilUnix := int64(0)
	reason := "сам попросился"
	if d != layaSelfJailManual {
		untilUnix = now.Add(d).Unix()
	} else {
		reason = "сам попросился на произвольный срок"
	}
	rec := ModerationRecord{
		UserID:      target.ID,
		Username:    target.Username,
		DisplayName: target.Name,
		Reason:      reason,
		RehabWard:   a.chooseRehabWard(chat.ID, target, reason, "", now),
		ByAdminID:   sender.ID,
		ByAdminName: displayNameFromUser(sender),
		SinceUnix:   now.Unix(),
		UntilUnix:   untilUnix,
	}
	a.store.SetJail(chat.ID, rec)
	a.clearRehabEscapeCooldown(chat.ID, rec.UserID)
	a.showRehabEscapeCommand(chat.ID, rec.UserID)
	a.sendRehabNotice(chat, rec)
	a.metrics.IncModeration("self_jail")
	a.sendOwnerLog("Self psych center", logField{"User", fmt.Sprintf("%s (%d)", target.Name, target.ID)}, logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)}, logField{"Topic", fmt.Sprintf("%d", rehabTopicID)}, logField{"Release", releaseLine(untilUnix, now)})
	return a.layaSend(c, fmt.Sprintf("%s сам ушёл в психиатрический центр до %s. Палата: %s.", target.Name, a.formatUntil(untilUnix), rehabWard(rec).Title), a.layaInlineMarkup(chat, sender))
}

func (a *App) handleSelfRelease(c tele.Context) error {
	chat, sender, err := a.layaCommandChatAndSender(c)
	if err != nil {
		return a.layaSend(c, err.Error())
	}
	now := time.Now().UTC()
	rec, ok := a.store.ActiveJail(chat.ID, sender.ID, now)
	if !ok {
		return a.layaSend(c, "Ты сейчас не в психиатрическом центре.", a.layaInlineMarkup(chat, sender))
	}
	if !layaCanSelfReleaseRehab(rec, sender.ID) {
		return a.layaSend(c, "Самостоятельный выход доступен только из произвольного self-психцентра.", a.layaInlineMarkup(chat, sender))
	}
	removed, released := a.store.RemoveJail(chat.ID, sender.ID)
	if !released {
		removed = rec
	}
	a.hideRehabEscapeCommand(chat.ID, sender.ID)
	a.clearRehabEscapeCooldown(chat.ID, sender.ID)
	a.resetRehabWardProgress(chat.ID, removed)
	duration := a.rehabReleaseDurationText(removed, released, now)
	a.logRehabRelease(chat.ID, removed, duration, "self-release")
	a.sendRehabReleaseNotice(chat, removed, duration, 0)
	a.sendOwnerLog("Rehab released",
		logField{"User", fmt.Sprintf("%s (%d)", displayNameFromUser(sender), sender.ID)},
		logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)},
		logField{"Duration", duration},
		logField{"By", "самостоятельный выход"},
	)
	return a.layaSend(c, "Ладно, главный по психбольнице вышел сам. Дверь была не заперта, просто драматично скрипела.", a.layaInlineMarkup(chat, sender))
}

func layaCanSelfReleaseRehab(rec ModerationRecord, userID int64) bool {
	return rec.UserID == userID && rec.ByAdminID == userID && rec.UntilUnix == 0 && strings.Contains(normalizeFAQText(rec.Reason), normalizeFAQText("произволь"))
}

func (a *App) applySelfBan(c tele.Context, d time.Duration) error {
	chat, sender, err := a.layaCommandChatAndSender(c)
	if err != nil {
		return a.layaSend(c, err.Error())
	}
	if err := a.ensureSelfModerationAllowed(chat, sender, true); err != nil {
		return a.layaSend(c, err.Error(), a.layaInlineMarkup(chat, sender))
	}
	untilUnix := time.Now().UTC().Add(d).Unix()
	member := &tele.ChatMember{
		User:            &tele.User{ID: sender.ID},
		RestrictedUntil: untilUnix,
	}
	if err := a.bot.Ban(chat, member, true); err != nil {
		return a.layaSend(c, a.t("telegram_ban_failed", err), a.layaInlineMarkup(chat, sender))
	}
	a.metrics.IncModeration("self_ban")
	a.sendOwnerLog("Self ban", logField{"User", fmt.Sprintf("%s (%d)", displayNameFromUser(sender), sender.ID)}, logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)}, logField{"Until", a.formatUntil(untilUnix)})
	return a.layaSend(c, fmt.Sprintf("%s сам выбрал бан до %s. Возвращение будет драматичным, но по расписанию.", displayNameFromUser(sender), a.formatUntil(untilUnix)))
}

func (a *App) ensureSelfModerationAllowed(chat *tele.Chat, sender *tele.User, telegramAction bool) error {
	if sender == nil || sender.IsBot {
		return fmt.Errorf("%s", a.t("no_sender"))
	}
	if a.bot != nil && a.bot.Me != nil && sender.ID == a.bot.Me.ID {
		return fmt.Errorf("%s", a.t("err_self"))
	}
	if a.isProtectedUser(sender.ID) {
		return fmt.Errorf("Ты в protected_user_ids, Лайя не будет сама себя спорить с конфигом.")
	}
	if telegramAction && a.isConfiguredAdmin(sender.ID) {
		return fmt.Errorf("Ты админ/владелец в конфиге, self-наказание не сработает честно. Убери защиту или используй админ-команду руками.")
	}
	if telegramAction && a.config().Security.ProtectAdmins && chat != nil && chat.Type != tele.ChatPrivate {
		member, err := a.bot.ChatMemberOf(chat, sender)
		if err == nil && (member.Role == tele.Creator || member.Role == tele.Administrator) {
			return fmt.Errorf("Telegram-админа бот не сможет нормально ограничить.")
		}
	}
	return nil
}

func layaSelfActionTitle(action string) string {
	switch action {
	case "selfmute":
		return "Самомут"
	case "selfjail":
		return "Поход в психиатрический центр"
	case "selfban":
		return "Самобан"
	default:
		return ""
	}
}

func layaDurationToken(d time.Duration) string {
	if d == layaSelfJailManual {
		return "manual"
	}
	if d%time.Hour == 0 {
		return fmt.Sprintf("%dh", int(d/time.Hour))
	}
	if d%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
	return fmt.Sprintf("%ds", int(d/time.Second))
}

func isLayaManualDurationToken(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "manual", "free", "custom", "forever", "perm", "permanent", "произвольно", "навсегда", "вручную":
		return true
	default:
		return false
	}
}

func chatIDFromChat(chat *tele.Chat) int64 {
	if chat == nil {
		return 0
	}
	return chat.ID
}

func userIDFromUser(user *tele.User) int64 {
	if user == nil {
		return 0
	}
	return user.ID
}

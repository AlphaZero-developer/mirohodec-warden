package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) handleProfile(c tele.Context) error {
	chat, err := a.commandChat(c)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	target, err := a.profileTarget(c)
	if err != nil {
		return c.Send(a.t("usage_profile"))
	}
	return c.Send(a.renderProfile(chat, target, time.Now().UTC()), a.layaInlineMarkup(chat, c.Sender()))
}

func (a *App) profileTarget(c tele.Context) (targetUser, error) {
	if msg := c.Message(); msg != nil && msg.ReplyTo != nil && msg.ReplyTo.Sender != nil {
		u := msg.ReplyTo.Sender
		return targetUser{ID: u.ID, Username: u.Username, Name: displayNameFromUser(u)}, nil
	}
	sender := c.Sender()
	if sender == nil || sender.ID == 0 {
		return targetUser{}, fmt.Errorf("missing target")
	}
	return targetUser{ID: sender.ID, Username: sender.Username, Name: displayNameFromUser(sender)}, nil
}

func (a *App) renderProfile(chat *tele.Chat, target targetUser, now time.Time) string {
	chatID := int64(0)
	if chat != nil {
		chatID = chat.ID
	}
	member, hasMember := a.store.Member(chatID, target.ID)
	jail, inRehab := a.store.ActiveJail(chatID, target.ID, now)
	_, banned := a.store.Ban(chatID, target.ID)

	username := target.Username
	name := strings.TrimSpace(target.Name)
	if hasMember {
		if strings.TrimSpace(member.Username) != "" {
			username = member.Username
		}
		if strings.TrimSpace(member.DisplayName) != "" {
			name = member.DisplayName
		}
	}

	lines := []string{
		a.t("profile_header"),
		a.t("profile_id", target.ID),
		a.t("profile_username", profileValue(usernameWithAt(username), a)),
		a.t("profile_name", profileValue(name, a)),
		a.t("profile_status", strings.Join(a.profileStatuses(chat, target.ID, inRehab, banned), ", ")),
	}
	if layaProfile, ok := a.store.LayaUserProfile(target.ID); ok && strings.TrimSpace(layaProfile.Address) != "" {
		lines = append(lines,
			fmt.Sprintf("Пол для Лайи: %s", layaGenderLabel(layaProfile.Gender)),
			fmt.Sprintf("Обращение Лайи: %s", layaProfile.Address),
		)
	}
	if hasMember {
		karma := a.store.Karma(chatID, target.ID)
		activeWarnings := a.store.ActiveWarningCount(chatID, target.ID, now)
		activityPoints := activityScore(member, karma, activeWarnings, now)
		karmaState := karmaBadge(karma)
		activityState := activityBadge(activityPoints)
		lines = append(lines,
			a.t("profile_messages", strconv.FormatInt(member.Messages, 10)),
			fmt.Sprintf("☀️ Сегодня: %d", messagesTodayAt(member, now)),
			fmt.Sprintf("⭐ Карма: %d · %s %s", karma, karmaState.Emoji, karmaState.Title),
			fmt.Sprintf("🔥 Активность: %d · %s %s", activityPoints, activityState.Emoji, activityState.Title),
			fmt.Sprintf("⚙️ Тон Лайи: %s", layaToneTitle(member.LayaTone)),
			"🎁 Плюшки: "+strings.Join(socialPerkLines(a.layaSafetyProfileForTarget(chat, target, now), a.activityRank(chatID, target.ID, now)), " | "),
			a.t("profile_first_seen", a.profileUnix(member.FirstSeen)),
			a.t("profile_last_activity", a.profileUnix(member.LastSeen)),
		)
		if strings.TrimSpace(member.ShopTitle) != "" && member.ShopTitleUntilUnix > now.Unix() {
			lines = append(lines, fmt.Sprintf("🏷️ Титул дня: %s", member.ShopTitle))
		}
		if perks := activeKarmaShopPerkLines(member, now); len(perks) > 0 {
			lines = append(lines, "🛒 Эффекты магазина: "+strings.Join(perks, " | "))
		}
		achievements := achievementList(member)
		if len(achievements) > 0 {
			lines = append(lines, fmt.Sprintf("🏅 Ачивки: %d · %s", len(achievements), strings.Join(achievements, " | ")))
		} else {
			lines = append(lines, "🏅 Ачивки: скрытые, пока не открыты")
		}
	} else {
		lines = append(lines,
			a.t("profile_messages", a.t("no_data")),
			"☀️ Сегодня: "+a.t("no_data"),
			"⭐ Карма: "+a.t("no_data"),
			"🔥 Активность: "+a.t("no_data"),
			a.t("profile_first_seen", a.t("no_data")),
			a.t("profile_last_activity", a.t("no_data")),
		)
	}

	if inRehab {
		topicID := a.rehabTopicID(chat)
		lines = append(lines,
			a.t("profile_rehab_topic", profileIntValue(topicID, a)),
			rehabWardLine(jail),
			a.t("profile_rehab_remaining", a.profileRehabRemaining(jail, now)),
		)
	}
	if hasMember && member.LastRehabDuration > 0 {
		lines = append(lines, a.t("profile_last_rehab_duration", a.formatTreatmentDuration(time.Duration(member.LastRehabDuration)*time.Second)))
	} else {
		lines = append(lines, a.t("profile_last_rehab_duration", a.t("no_data")))
	}
	return strings.Join(lines, "\n")
}

func (a *App) profileStatuses(chat *tele.Chat, userID int64, inRehab, banned bool) []string {
	var statuses []string
	if banned {
		statuses = append(statuses, a.t("profile_status_banned"))
	}
	if inRehab {
		statuses = append(statuses, a.t("profile_status_rehab"))
	}
	if a.profileUserIsAdmin(chat, userID) {
		statuses = append(statuses, a.t("profile_status_admin"))
	}
	if a.isLayaFather(userID) {
		statuses = append(statuses, a.t("profile_status_father"))
	}
	if a.isLayaMother(userID) {
		statuses = append(statuses, a.t("profile_status_mother"))
	}
	if a.isLayaBrother(userID) {
		statuses = append(statuses, a.t("profile_status_brother"))
	}
	if len(statuses) == 0 {
		statuses = append(statuses, a.t("profile_status_regular"))
	}
	return statuses
}

func (a *App) profileUserIsAdmin(chat *tele.Chat, userID int64) bool {
	if a.isConfiguredAdmin(userID) {
		return true
	}
	if a == nil || a.bot == nil || chat == nil || chat.Type == tele.ChatPrivate || !a.config().Security.TrustTelegramAdmins {
		return false
	}
	member, err := a.bot.ChatMemberOf(chat, &tele.User{ID: userID})
	if err != nil {
		if a.log != nil {
			a.log.Warn("profile admin check failed", zap.Int64("chat", chat.ID), zap.Int64("user", userID), zap.Error(err))
		}
		return false
	}
	return member.Role == tele.Creator || member.Role == tele.Administrator
}

func (a *App) profileUnix(ts int64) string {
	if ts == 0 {
		return a.t("no_data")
	}
	return formatUnix(ts)
}

func (a *App) profileRehabRemaining(rec ModerationRecord, now time.Time) string {
	if rec.UntilUnix == 0 {
		return a.t("forever")
	}
	remaining := time.Unix(rec.UntilUnix, 0).UTC().Sub(now)
	if remaining <= 0 {
		return a.t("no_data")
	}
	return a.formatTreatmentDuration(remaining)
}

func profileValue(value string, a *App) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return a.t("no_data")
	}
	return value
}

package main

import (
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) maybeUnlockSocialAchievements(chat *tele.Chat, user *tele.User, now time.Time) {
	if a == nil || a.store == nil || chat == nil || user == nil || user.ID == 0 || user.IsBot {
		return
	}
	member, ok := a.store.Member(chat.ID, user.ID)
	if !ok {
		return
	}
	switch {
	case member.Messages >= 1000:
		a.unlockAchievement(chat.ID, user.ID, "activity_1000", now)
	case member.Messages >= 500:
		a.unlockAchievement(chat.ID, user.ID, "activity_500", now)
	case member.Messages >= 100:
		a.unlockAchievement(chat.ID, user.ID, "activity_100", now)
	}
	if member.Messages >= 50 && a.activityRank(chat.ID, user.ID, now) > 0 && a.activityRank(chat.ID, user.ID, now) <= 3 {
		a.unlockAchievement(chat.ID, user.ID, "top_activity", now)
	}
	if a.store.Karma(chat.ID, user.ID) > 0 && a.karmaRank(chat.ID, user.ID) > 0 && a.karmaRank(chat.ID, user.ID) <= 3 {
		a.unlockAchievement(chat.ID, user.ID, "top_karma", now)
	}
}

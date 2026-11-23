package main

import (
	"time"

	tele "gopkg.in/telebot.v3"
)

type RuntimeStore interface {
	Flush() error
	Reload() error
	ReloadIfChanged() (bool, error)

	UpsertMember(chatID int64, u *tele.User)
	TrackMember(chatID int64, u *tele.User, threadID, messageID int, countMessage bool)
	Member(chatID, userID int64) (MemberRecord, bool)
	UserByUsername(chatID int64, username string) (targetUser, bool)
	Members(chatID int64) []MemberRecord
	Activity(chatID, userID int64) (MemberRecord, bool)
	TopActive(chatID int64, limit int) []MemberRecord
	RandomActiveMember(chatID int64, activeWindow time.Duration, now time.Time, excludeID int64) (MemberRecord, bool)
	UnlockAchievement(chatID, userID int64, achievement string, now time.Time) (bool, error)
	SetMemberRewards(chatID, userID int64, rewards MemberRewards) error

	RegisterOwner(user *tele.User, chat *tele.Chat) error
	IsOwner(userID int64) bool
	HasOwner() bool
	LogChatID() int64
	OwnerReportLastSent(kind string, chatID, userID int64) time.Time
	SetOwnerReportLastSent(kind string, chatID, userID int64, now time.Time) error
	LayaRuntimeState() LayaRuntimeState
	SetLayaRuntimeState(state LayaRuntimeState) error
	LayaUserProfile(userID int64) (LayaUserProfile, bool)
	LayaUserProfiles() []LayaUserProfile
	SetLayaUserProfile(profile LayaUserProfile) error
	ScriptwriterExemption(chatID, userID int64) (ScriptwriterExemption, bool)
	SetScriptwriterExemption(rec ScriptwriterExemption) error
	RemoveScriptwriterExemption(chatID, userID int64) (ScriptwriterExemption, bool, error)
	LayaIgnoredUser(chatID, userID int64) (LayaIgnoredUser, bool)
	SetLayaIgnoredUser(rec LayaIgnoredUser) error
	RemoveLayaIgnoredUser(chatID, userID int64) (LayaIgnoredUser, bool, error)
	ContentRuntimeState() ContentRuntimeState
	SetContentRuntimeState(state ContentRuntimeState) error
	DiscussionRuntimeState() DiscussionRuntimeState
	SetDiscussionRuntimeState(state DiscussionRuntimeState) error
	RehabBrainwashRuntimeState() RehabBrainwashRuntimeState
	SetRehabBrainwashRuntimeState(state RehabBrainwashRuntimeState) error
	RehabReleaseExamRuntimeState() RehabReleaseExamRuntimeState
	SetRehabReleaseExamRuntimeState(state RehabReleaseExamRuntimeState) error
	RehabReleaseExamRuntimeStateForChat(chatID int64) (RehabReleaseExamRuntimeState, bool)
	SetRehabReleaseExamRuntimeStateForChat(chatID int64, state RehabReleaseExamRuntimeState) error
	ReplyCipherRuntimeState() ReplyCipherRuntimeState
	SetReplyCipherRuntimeState(state ReplyCipherRuntimeState) error
	SetOwnerSleep(user *tele.User, aliases []string, now time.Time) (OwnerSleepState, error)
	ClearOwnerSleep(userID int64) (OwnerSleepState, bool, error)
	OwnerSleep(userID int64) (OwnerSleepState, bool)
	ActiveOwnerSleeps() []OwnerSleepState
	AddOwnerSleepCall(userID int64, call OwnerSleepCall) (OwnerSleepState, bool, error)

	Counts(now time.Time) StoreCounts
	ChatIDs() []int64
	UpsertChatSettings(settings ChatSettings) error
	ChatSettings(chatID int64) (ChatSettings, bool)
	ManagedChats(includeDisabled bool) []ChatSettings
	DeleteChatSettings(chatID int64) error
	DiscussionReplyPosted(chatID int64, topicID int) bool
	MarkDiscussionReply(chatID int64, topicID int, now time.Time) (bool, error)
	ModerationRecords(chatID int64, kind string, activeOnly bool, now time.Time) []ModerationRecord
	SetJail(chatID int64, rec ModerationRecord)
	RemoveJail(chatID, userID int64) (ModerationRecord, bool)
	ActiveJail(chatID, userID int64, now time.Time) (ModerationRecord, bool)
	RehabEscapeChance(chatID, userID int64) int
	RecordRehabEscapeSuccess(chatID, userID int64)
	SetMute(chatID int64, rec ModerationRecord)
	RemoveMute(chatID, userID int64) (ModerationRecord, bool)
	ActiveMute(chatID, userID int64, now time.Time) (ModerationRecord, bool)
	SetForwardRestriction(chatID int64, rec ModerationRecord)
	RemoveForwardRestriction(chatID, userID int64) (ModerationRecord, bool)
	ActiveForwardRestriction(chatID, userID int64, now time.Time) (ModerationRecord, bool)
	SetBan(chatID int64, rec ModerationRecord)
	RemoveBan(chatID, userID int64) (ModerationRecord, bool)
	Ban(chatID, userID int64) (ModerationRecord, bool)
	CleanupExpired(now time.Time) int

	AddWarning(chatID int64, rec WarningRecord) (WarningRecord, error)
	Warnings(chatID, userID int64, includeInactive bool, limit int, now time.Time) []WarningRecord
	ActiveWarningCount(chatID, userID int64, now time.Time) int
	DeactivateWarning(chatID int64, warningID string) (WarningRecord, bool, error)
	ClearWarnings(chatID, userID int64) (int, error)

	Karma(chatID, userID int64) int64
	TopKarma(chatID int64, limit int) []KarmaTotal
	AddKarma(chatID, userID int64, delta int64, reason string, byUser *tele.User, messageID int, now time.Time) (int64, error)
	MaybeAwardActivityKarma(chatID, userID int64, messagesPerPoint, dailyLimit int, now time.Time) (int64, bool)
	LastManualKarma(chatID, targetID, giverID int64, now time.Time) time.Time

	RegisterMafiaUser(user *tele.User, now time.Time) (MafiaRegisteredUser, error)
	SetMafiaProfile(userID int64, profile MafiaProfile) error
	MafiaUser(userID int64) (MafiaRegisteredUser, bool)
	MafiaUsers() []MafiaRegisteredUser
	MafiaPlayerRating(userID int64) (MafiaPlayerRating, bool)
	MafiaPlayerRatings() []MafiaPlayerRating
	MafiaGameResultsForUser(userID int64) []MafiaGameResult
	SetMafiaUserStats(userID int64, stats MafiaStats) error
	SetMafiaUserSettings(userID int64, settings MafiaUserSettings) error
	SetMafiaUserFavorites(userID int64, favorites []int64) error
	SaveMafiaCancelledGame(record MafiaCancelledGameRecord) error
	AddMafiaHostBlacklist(hostID, blockedUserID int64, reason string, now time.Time) error
	MafiaHostBlocked(hostID, userID int64) bool
	MafiaReadyQueue() []MafiaReadyQueueEntry
	SaveMafiaReadyQueue(queue []MafiaReadyQueueEntry) error
	SaveMafiaRoom(room MafiaRoom) error
	MafiaRoom(roomID string) (MafiaRoom, bool)
	MafiaRooms() []MafiaRoom
	MafiaRoomsForUser(userID int64) []MafiaRoom
	MafiaRoomByGroup(chatID int64) (MafiaRoom, bool)
	ApplyMafiaGameResult(roomID string, winner string, finishedAt time.Time) (MafiaRoom, MafiaGameResult, []MafiaEloHistoryEntry, bool, error)
	DeleteMafiaRoom(roomID string) error
}

type PhotoRepository interface {
	Add(photo *tele.Photo, tags []string, caption string, admin *tele.User) (PhotoRecord, error)
	Delete(id string) (PhotoRecord, bool, error)
	List(tag string) []PhotoRecord
	Random(tag string) (PhotoRecord, bool)
	Count() int
	Flush() error
	Reload() error
	ReloadIfChanged() (bool, error)
}

type QuoteRepository interface {
	Add(text, author string, tags []string, admin *tele.User) (QuoteRecord, error)
	Update(id, text, author string, tags []string) (QuoteRecord, bool, error)
	Delete(id string) (QuoteRecord, bool, error)
	List(tag string) []QuoteRecord
	Random(tag string) (QuoteRecord, bool)
	Flush() error
	Reload() error
	ReloadIfChanged() (bool, error)
}

type FAQRepository interface {
	Add(patterns []string, answer string) (FAQRecord, error)
	Update(id string, patterns []string, answer string, enabled bool) (FAQRecord, bool, error)
	Delete(id string) (FAQRecord, bool, error)
	List() []FAQRecord
	Match(question string) (FAQRecord, bool)
	MarkUsed(id string)
	Flush() error
	Reload() error
	ReloadIfChanged() (bool, error)
}

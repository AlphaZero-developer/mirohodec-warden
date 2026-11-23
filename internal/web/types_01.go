package web

import (
	"context"
	"time"
)

type Service interface {
	MafiaMiniAppService

	Dashboard(ctx context.Context) (Dashboard, error)
	ChatSelection(ctx context.Context, selected int64) (ChatSelection, error)
	Groups(ctx context.Context) ([]Group, error)
	AddGroup(ctx context.Context, input GroupInput) error
	UpdateGroup(ctx context.Context, input GroupInput) error
	DeleteGroup(ctx context.Context, chatID int64) error
	SetGroupDebug(ctx context.Context, chatID int64, debug bool) error
	SetGroupRehabTopic(ctx context.Context, chatID int64, topicID int) error
	Rehab(ctx context.Context, chatID int64) (RehabPage, error)
	AddRehab(ctx context.Context, input RehabInput) error
	ReleaseRehab(ctx context.Context, chatID, userID int64) error
	AddForeverBan(ctx context.Context, input ForeverBanInput) error
	UpdateRehabBrainwash(ctx context.Context, input RehabBrainwashInput) error
	UpdateRehabReleaseExam(ctx context.Context, input RehabReleaseExamInput) error
	Users(ctx context.Context, chatID int64) ([]UserRow, error)
	Moderation(ctx context.Context, chatID int64) (ModerationPage, error)
	Warnings(ctx context.Context, chatID int64) ([]WarningRow, error)
	Karma(ctx context.Context, chatID int64) ([]KarmaRow, error)
	OwnerSleep(ctx context.Context) (OwnerSleepPage, error)
	ClearOwnerSleep(ctx context.Context, userID int64) error
	LayaControl(ctx context.Context) (LayaControlPage, error)
	UpdateLayaControl(ctx context.Context, input LayaControlInput) error
	SendLayaMessage(ctx context.Context, input LayaManualMessageInput) error
	LayaUsers(ctx context.Context, chatID int64) (LayaUsersPage, error)
	UpdateLayaUser(ctx context.Context, input LayaUserInput) error
	ClearAICache(ctx context.Context) error
	AILogs(ctx context.Context) (AILogPage, error)
	ClearAILogs(ctx context.Context) error
	Discussion(ctx context.Context) (DiscussionPage, error)
	UpdateDiscussion(ctx context.Context, input DiscussionInput) error
	ContentControl(ctx context.Context) (ContentControlPage, error)
	UpdateContentControl(ctx context.Context, input ContentControlInput) error
	FAQ(ctx context.Context) ([]FAQRow, error)
	AddFAQ(ctx context.Context, input FAQInput) error
	UpdateFAQ(ctx context.Context, input FAQInput) error
	DeleteFAQ(ctx context.Context, id string) error
	Quotes(ctx context.Context) ([]QuoteRow, error)
	AddQuote(ctx context.Context, input QuoteInput) error
	UpdateQuote(ctx context.Context, input QuoteInput) error
	DeleteQuote(ctx context.Context, id string) error
	Photos(ctx context.Context, tag string) ([]PhotoRow, error)
	AddPhoto(ctx context.Context, input PhotoInput) error
	DeletePhoto(ctx context.Context, id string) error
	Reload(ctx context.Context) error
}

type Dashboard struct {
	BotName        string
	Mode           string
	State          string
	PublicURL      string
	KnownUsers     int
	ActiveJailed   int
	ActiveMuted    int
	ForeverBans    int
	FAQCount       int
	QuoteCount     int
	PhotoCount     int
	SleepingOwners int
	StartedAt      time.Time
}

type ChatSelection struct {
	CurrentID int64
	Options   []ChatOption
}

type ChatOption struct {
	ID       int64
	Label    string
	Selected bool
}

type Group struct {
	ID                    int64
	Title                 string
	Enabled               bool
	RehabTopicID          int
	SpamNoRehabAction     string
	DebugCommands         bool
	KnownTopics           []int
	Settings              []GroupSetting
	AIEnabledMode         string
	SpamEnabledMode       string
	WarnEnabledMode       string
	KarmaEnabledMode      string
	FAQEnabledMode        string
	QuotesEnabledMode     string
	PhotosEnabledMode     string
	WarnExpireAfter       string
	WarnMuteThreshold     int
	WarnMuteDuration      string
	WarnRehabThreshold    int
	WarnRehabDuration     string
	KarmaMessagesPerPoint int
	KarmaDailyLimit       int
	KarmaManualCooldown   string
}

type GroupSetting struct {
	Section string
	Name    string
	Value   string
	Note    string
}

type DiscussionPage struct {
	Rows    []GroupSetting
	Active  DiscussionSettings
	Config  DiscussionSettings
	Runtime DiscussionSettings
	Updated string
}

type DiscussionSettings struct {
	Enabled            bool
	ChatID             int64
	SourceChatID       int64
	StrictChecks       bool
	Text               string
	ChatButtonText     string
	ChatButtonURL      string
	FollowUpText       string
	FollowUpButtonText string
	FollowUpButtonURL  string
	Pin                bool
}

type DiscussionInput struct {
	Enabled            bool
	ChatID             int64
	SourceChatID       int64
	StrictChecks       bool
	Text               string
	ChatButtonText     string
	ChatButtonURL      string
	FollowUpText       string
	FollowUpButtonText string
	FollowUpButtonURL  string
	Pin                bool
}

type ContentControlPage struct {
	Rows      []GroupSetting
	FAQ       ContentFAQSettings
	Quotes    ContentQuoteSettings
	Photos    ContentPhotoSettings
	Ideas     ContentIdeaSettings
	Updated   string
	UpdatedBy string
}

type ContentFAQSettings struct {
	EnabledMode        string
	AutoReplyMode      string
	RequireMentionMode string
	EntryCooldown      string
	UserCooldown       string

	ActiveEnabled        bool
	ActiveAutoReply      bool
	ActiveRequireMention bool
	ActiveEntryCooldown  string
	ActiveUserCooldown   string
	ConfigEnabled        bool
	ConfigAutoReply      bool
	ConfigRequireMention bool
	ConfigEntryCooldown  string
	ConfigUserCooldown   string
}

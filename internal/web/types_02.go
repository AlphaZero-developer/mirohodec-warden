package web

type ContentQuoteSettings struct {
	EnabledMode       string
	AutoReplyMode     string
	AutoReplyChance   string
	AutoReplyCooldown string
	ActiveUserWindow  string
	PhotoChance       string

	ActiveEnabled           bool
	ActiveAutoReply         bool
	ActiveAutoReplyChance   string
	ActiveAutoReplyCooldown string
	ActiveActiveUserWindow  string
	ActivePhotoChance       string
	ConfigEnabled           bool
	ConfigAutoReply         bool
	ConfigAutoReplyChance   string
	ConfigAutoReplyCooldown string
	ConfigActiveUserWindow  string
	ConfigPhotoChance       string
}

type ContentPhotoSettings struct {
	EnabledMode     string
	AutoReplyMode   string
	AutoReplyChance string
	Cooldown        string
	Triggers        string
	LocalDir        string
	LocalExtensions string

	ActiveEnabled         bool
	ActiveAutoReply       bool
	ActiveAutoReplyChance string
	ActiveCooldown        string
	ActiveTriggers        string
	ActiveLocalDir        string
	ActiveLocalExtensions string
	ConfigEnabled         bool
	ConfigAutoReply       bool
	ConfigAutoReplyChance string
	ConfigCooldown        string
	ConfigTriggers        string
	ConfigLocalDir        string
	ConfigLocalExtensions string
}

type ContentIdeaSettings struct {
	EnabledMode  string
	UserCooldown string
	ChatCooldown string
	MaxLength    int

	ActiveEnabled      bool
	ActiveUserCooldown string
	ActiveChatCooldown string
	ActiveMaxLength    int
	ConfigEnabled      bool
	ConfigUserCooldown string
	ConfigChatCooldown string
	ConfigMaxLength    int
}

type ContentControlInput struct {
	FAQEnabledMode        string
	FAQAutoReplyMode      string
	FAQRequireMentionMode string
	FAQEntryCooldown      string
	FAQUserCooldown       string

	QuotesEnabledMode       string
	QuotesAutoReplyMode     string
	QuotesAutoReplyChance   string
	QuotesAutoReplyCooldown string
	QuotesActiveUserWindow  string
	QuotesPhotoChance       string

	PhotosEnabledMode     string
	PhotosAutoReplyMode   string
	PhotosAutoReplyChance string
	PhotosCooldown        string
	PhotosTriggers        string
	PhotosLocalDir        string
	PhotosLocalExtensions string

	IdeasEnabledMode  string
	IdeasUserCooldown string
	IdeasChatCooldown string
	IdeasMaxLength    int
	UpdatedByID       int64
	UpdatedByName     string
}

type GroupInput struct {
	ChatID                int64
	Title                 string
	Enabled               bool
	RehabTopicID          int
	SpamNoRehabAction     string
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

type RehabPage struct {
	ChatID       int64
	RehabTopicID int
	Users        []UserChoice
	Rows         []ModerationRow
	Brainwash    RehabBrainwashSettings
	Exam         RehabReleaseExamSettings
}

type RehabInput struct {
	ChatID   int64
	UserID   int64
	Duration string
	Reason   string
}

type RehabBrainwashSettings struct {
	Enabled      bool
	AIEnabled    bool
	LeaderUserID int64
	LeaderName   string
	AIPrompt     string
	Messages     string
	Updated      string
}

type RehabBrainwashInput struct {
	ChatID        int64
	Enabled       bool
	AIEnabled     bool
	LeaderUserID  int64
	LeaderName    string
	AIPrompt      string
	Messages      string
	UpdatedByID   int64
	UpdatedByName string
}

type RehabReleaseExamSettings struct {
	Enabled      bool
	Questions    string
	AnswerWindow string
	FailDuration string
	Scope        string
	Updated      string
}

type RehabReleaseExamInput struct {
	ChatID        int64
	Enabled       bool
	Questions     string
	AnswerWindow  string
	FailDuration  string
	Scope         string
	UpdatedByID   int64
	UpdatedByName string
}

type UserChoice struct {
	ID    int64
	Label string
}

type UserRow struct {
	ID            int64
	DisplayName   string
	Username      string
	Role          string
	CanBan        bool
	Banned        bool
	Messages      int64
	MessagesToday int64
	Karma         int64
	LastMessage   string
	LastThreadID  int
}

package web

type ForeverBanInput struct {
	ChatID int64
	UserID int64
	Reason string
}

type ModerationPage struct {
	ChatID int64
	Users  []UserChoice
	Jails  []ModerationRow
	Mutes  []ModerationRow
	Bans   []ModerationRow
}

type ModerationRow struct {
	Kind        string
	UserID      int64
	DisplayName string
	Username    string
	Reason      string
	Since       string
	Until       string
	By          string
}

type WarningRow struct {
	ID          string
	UserID      int64
	DisplayName string
	Reason      string
	Created     string
	Expires     string
	By          string
	Active      bool
}

type KarmaRow struct {
	Rank        int
	UserID      int64
	DisplayName string
	Username    string
	Score       int64
	Updated     string
}

type OwnerSleepPage struct {
	Rows []OwnerSleepRow
}

type OwnerSleepRow struct {
	UserID      int64
	DisplayName string
	Username    string
	AliasesText string
	Since       string
	CallCount   int
	Calls       []OwnerSleepCallRow
}

type OwnerSleepCallRow struct {
	At        string
	ChatID    int64
	Chat      string
	FromID    int64
	From      string
	Matched   string
	Text      string
	ThreadID  int
	MessageID int
}

type LayaControlPage struct {
	ConfigSystemPrompt      string
	ActiveSystemPrompt      string
	RuntimeSystemPrompt     string
	ExtraPrompt             string
	FallbackReply           string
	ConfigBaseURL           string
	ConfigModel             string
	ConfigTimeout           string
	ConfigSoftTimeout       string
	ConfigFasMute           string
	ConfigMaxTokens         int
	ConfigMaxReplyChars     int
	ConfigLogConversations  bool
	ActiveBaseURL           string
	ActiveModel             string
	ActiveTimeout           string
	ActiveSoftTimeout       string
	ActiveFasMute           string
	ActiveMaxTokens         int
	ActiveMaxReplyChars     int
	ActiveLogConversations  bool
	RuntimeBaseURL          string
	RuntimeModel            string
	RuntimeTimeout          string
	RuntimeSoftTimeout      string
	RuntimeFasMute          string
	RuntimeMaxTokens        int
	RuntimeMaxReplyChars    int
	RuntimeLogConversations string
	ConfigFatherUserID      int64
	ConfigMotherUserID      int64
	ConfigBrotherUserID     int64
	ConfigCreatorUserID     int64
	ActiveFatherUserID      int64
	ActiveMotherUserID      int64
	ActiveBrotherUserID     int64
	ActiveCreatorUserID     int64
	RuntimeFatherUserID     int64
	RuntimeMotherUserID     int64
	RuntimeBrotherUserID    int64
	RuntimeCreatorUserID    int64
	ConfigFatherPrompt      string
	ConfigMotherPrompt      string
	ConfigBrotherPrompt     string
	ConfigCreatorPrompt     string
	ActiveFatherPrompt      string
	ActiveMotherPrompt      string
	ActiveBrotherPrompt     string
	ActiveCreatorPrompt     string
	RuntimeFatherPrompt     string
	RuntimeMotherPrompt     string
	RuntimeBrotherPrompt    string
	RuntimeCreatorPrompt    string
	RuntimeRelationships    string
	ActiveRelationships     []LayaRelationshipRow
	MoodMode                string
	MoodPrompt              string
	DirectInsultMode        string
	DirectInsultPrompt      string
	DirectInsultFallback    string
	SafetyCooldownMode      string
	SafetyPauseReply        string
	TagReactionRules        string
	RetryAttempts           int
	RetryBackoff            string
	Updated                 string
	UpdatedBy               string
	Cache                   AICacheStats
}

type LayaRelationshipRow struct {
	UserID int64
	Role   string
	Prompt string
}

type LayaControlInput struct {
	SystemPrompt         string
	ExtraPrompt          string
	FallbackReply        string
	BaseURL              string
	Model                string
	Timeout              string
	SoftTimeout          string
	FasMute              string
	MaxTokens            int
	MaxReplyChars        int
	LogConversations     string
	FatherUserID         int64
	MotherUserID         int64
	BrotherUserID        int64
	CreatorUserID        int64
	FatherPrompt         string
	MotherPrompt         string
	BrotherPrompt        string
	CreatorPrompt        string
	Relationships        string
	MoodMode             string
	MoodPrompt           string
	DirectInsultMode     string
	DirectInsultPrompt   string
	DirectInsultFallback string
	SafetyCooldownMode   string
	SafetyPauseReply     string
	TagReactionRules     string
	RetryAttempts        int
	RetryBackoff         string
	UpdatedByID          int64
	UpdatedByName        string
}

type LayaManualMessageInput struct {
	ChatID           int64
	ThreadID         int
	ReplyToMessageID int
	Text             string
}

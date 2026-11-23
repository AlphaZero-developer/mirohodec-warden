package main

type State struct {
	Chats             map[string]*ChatState   `json:"chats"`
	ManagedChats      map[string]ChatSettings `json:"managed_chats,omitempty"`
	DiscussionReplies []DiscussionReplyRecord `json:"discussion_replies,omitempty"`
	Mafia             MafiaState              `json:"mafia,omitempty"`
	Bot               BotState                `json:"bot"`
}

type BotState struct {
	OwnerIDs          []int64                                 `json:"owner_ids,omitempty"`
	LogChatID         int64                                   `json:"log_chat_id,omitempty"`
	AuthenticatedAt   int64                                   `json:"authenticated_at,omitempty"`
	Sleep             map[string]OwnerSleepState              `json:"sleep,omitempty"`
	Laya              LayaRuntimeState                        `json:"laya,omitempty"`
	LayaProfiles      map[string]LayaUserProfile              `json:"laya_profiles,omitempty"`
	Content           ContentRuntimeState                     `json:"content,omitempty"`
	OwnerReports      map[string]OwnerReportState             `json:"owner_reports,omitempty"`
	Discussion        DiscussionRuntimeState                  `json:"discussion,omitempty"`
	RehabBrainwash    RehabBrainwashRuntimeState              `json:"rehab_brainwash,omitempty"`
	RehabReleaseExam  RehabReleaseExamRuntimeState            `json:"rehab_release_exam,omitempty"`
	RehabReleaseExams map[string]RehabReleaseExamRuntimeState `json:"rehab_release_exam_by_chat,omitempty"`
	ReplyCipher       ReplyCipherRuntimeState                 `json:"reply_cipher,omitempty"`
	Scriptwriters     map[string]ScriptwriterExemption        `json:"scriptwriters,omitempty"`
	LayaIgnored       map[string]LayaIgnoredUser              `json:"laya_ignored,omitempty"`
}

type LayaRuntimeState struct {
	SystemPrompt         string             `json:"system_prompt,omitempty"`
	ExtraPrompt          string             `json:"extra_prompt,omitempty"`
	FallbackReply        string             `json:"fallback_reply,omitempty"`
	BaseURL              string             `json:"base_url,omitempty"`
	Model                string             `json:"model,omitempty"`
	Timeout              string             `json:"timeout,omitempty"`
	SoftTimeout          string             `json:"soft_timeout,omitempty"`
	FasMuteDuration      string             `json:"fas_mute_duration,omitempty"`
	MaxTokens            int                `json:"max_tokens,omitempty"`
	MaxReplyChars        int                `json:"max_reply_chars,omitempty"`
	LogConversationsMode string             `json:"log_conversations,omitempty"`
	FatherUserID         int64              `json:"father_user_id,omitempty"`
	MotherUserID         int64              `json:"mother_user_id,omitempty"`
	BrotherUserID        int64              `json:"brother_user_id,omitempty"`
	CreatorUserID        int64              `json:"creator_user_id,omitempty"`
	FatherPrompt         string             `json:"father_prompt,omitempty"`
	MotherPrompt         string             `json:"mother_prompt,omitempty"`
	BrotherPrompt        string             `json:"brother_prompt,omitempty"`
	CreatorPrompt        string             `json:"creator_prompt,omitempty"`
	Relationships        []LayaRelationship `json:"relationships,omitempty"`
	MoodMode             string             `json:"mood_mode,omitempty"`
	MoodPrompt           string             `json:"mood_prompt,omitempty"`
	DirectInsultMode     string             `json:"direct_insult_mode,omitempty"`
	DirectInsultPrompt   string             `json:"direct_insult_prompt,omitempty"`
	DirectInsultFallback string             `json:"direct_insult_fallback,omitempty"`
	SafetyCooldownMode   string             `json:"safety_cooldown_mode,omitempty"`
	SafetyPauseReply     string             `json:"safety_pause_reply,omitempty"`
	TagReactionRules     string             `json:"tag_reaction_rules,omitempty"`
	RetryAttempts        int                `json:"retry_attempts,omitempty"`
	RetryBackoff         string             `json:"retry_backoff,omitempty"`
	UpdatedUnix          int64              `json:"updated_unix,omitempty"`
	UpdatedByID          int64              `json:"updated_by_id,omitempty"`
	UpdatedByName        string             `json:"updated_by_name,omitempty"`
}

type LayaRelationship struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role,omitempty"`
	Prompt string `json:"prompt,omitempty"`
}

type LayaUserProfile struct {
	UserID        int64    `json:"user_id"`
	Username      string   `json:"username,omitempty"`
	DisplayName   string   `json:"display_name,omitempty"`
	Gender        string   `json:"gender,omitempty"`
	Address       string   `json:"address,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Note          string   `json:"note,omitempty"`
	UpdatedUnix   int64    `json:"updated_unix,omitempty"`
	UpdatedChatID int64    `json:"updated_chat_id,omitempty"`
}

type ScriptwriterExemption struct {
	ChatID      int64  `json:"chat_id"`
	UserID      int64  `json:"user_id"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Profession  string `json:"profession,omitempty"`
	AddedUnix   int64  `json:"added_unix,omitempty"`
	AddedByID   int64  `json:"added_by_id,omitempty"`
	AddedByName string `json:"added_by_name,omitempty"`
}

type LayaIgnoredUser struct {
	ChatID      int64  `json:"chat_id"`
	UserID      int64  `json:"user_id"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	AddedUnix   int64  `json:"added_unix,omitempty"`
	AddedByID   int64  `json:"added_by_id,omitempty"`
	AddedByName string `json:"added_by_name,omitempty"`
}

type OwnerReportState struct {
	Kind         string `json:"kind,omitempty"`
	ChatID       int64  `json:"chat_id,omitempty"`
	UserID       int64  `json:"user_id,omitempty"`
	LastSentUnix int64  `json:"last_sent_unix,omitempty"`
}

type ContentRuntimeState struct {
	FAQEnabledMode        string `json:"faq_enabled_mode,omitempty"`
	FAQAutoReplyMode      string `json:"faq_auto_reply_mode,omitempty"`
	FAQRequireMentionMode string `json:"faq_require_mention_mode,omitempty"`
	FAQEntryCooldown      string `json:"faq_entry_cooldown,omitempty"`
	FAQUserCooldown       string `json:"faq_user_cooldown,omitempty"`

	QuotesEnabledMode       string `json:"quotes_enabled_mode,omitempty"`
	QuotesAutoReplyMode     string `json:"quotes_auto_reply_mode,omitempty"`
	QuotesAutoReplyChance   string `json:"quotes_auto_reply_chance,omitempty"`
	QuotesAutoReplyCooldown string `json:"quotes_auto_reply_cooldown,omitempty"`
	QuotesActiveUserWindow  string `json:"quotes_active_user_window,omitempty"`
	QuotesPhotoChance       string `json:"quotes_photo_chance,omitempty"`

	PhotosEnabledMode     string `json:"photos_enabled_mode,omitempty"`
	PhotosAutoReplyMode   string `json:"photos_auto_reply_mode,omitempty"`
	PhotosAutoReplyChance string `json:"photos_auto_reply_chance,omitempty"`
	PhotosCooldown        string `json:"photos_cooldown,omitempty"`
	PhotosTriggers        string `json:"photos_triggers,omitempty"`
	PhotosLocalDir        string `json:"photos_local_dir,omitempty"`
	PhotosLocalExtensions string `json:"photos_local_extensions,omitempty"`

	IdeasEnabledMode  string `json:"ideas_enabled_mode,omitempty"`
	IdeasUserCooldown string `json:"ideas_user_cooldown,omitempty"`
	IdeasChatCooldown string `json:"ideas_chat_cooldown,omitempty"`
	IdeasMaxLength    int    `json:"ideas_max_length,omitempty"`

	UpdatedUnix   int64  `json:"updated_unix,omitempty"`
	UpdatedByID   int64  `json:"updated_by_id,omitempty"`
	UpdatedByName string `json:"updated_by_name,omitempty"`
}

type DiscussionRuntimeState struct {
	Enabled            bool   `json:"enabled"`
	ChatID             int64  `json:"chat_id,omitempty"`
	SourceChatID       int64  `json:"source_chat_id,omitempty"`
	StrictChecks       bool   `json:"strict_checks"`
	Text               string `json:"text,omitempty"`
	ChatButtonText     string `json:"chat_button_text,omitempty"`
	ChatButtonURL      string `json:"chat_button_url,omitempty"`
	FollowUpText       string `json:"follow_up_text,omitempty"`
	FollowUpButtonText string `json:"follow_up_button_text,omitempty"`
	FollowUpButtonURL  string `json:"follow_up_button_url,omitempty"`
	Pin                bool   `json:"pin"`
	UpdatedUnix        int64  `json:"updated_unix,omitempty"`
}

type RehabBrainwashRuntimeState struct {
	Enabled       bool   `json:"enabled"`
	AIEnabled     bool   `json:"ai_enabled,omitempty"`
	LeaderUserID  int64  `json:"leader_user_id,omitempty"`
	LeaderName    string `json:"leader_name,omitempty"`
	AIPrompt      string `json:"ai_prompt,omitempty"`
	Messages      string `json:"messages,omitempty"`
	UpdatedUnix   int64  `json:"updated_unix,omitempty"`
	UpdatedByID   int64  `json:"updated_by_id,omitempty"`
	UpdatedByName string `json:"updated_by_name,omitempty"`
}

type RehabReleaseExamRuntimeState struct {
	Enabled       bool   `json:"enabled"`
	Questions     string `json:"questions,omitempty"`
	AnswerWindow  string `json:"answer_window,omitempty"`
	FailDuration  string `json:"fail_duration,omitempty"`
	UpdatedUnix   int64  `json:"updated_unix,omitempty"`
	UpdatedByID   int64  `json:"updated_by_id,omitempty"`
	UpdatedByName string `json:"updated_by_name,omitempty"`
}

type ReplyCipherRuntimeState struct {
	DefaultCipher string                       `json:"default_cipher,omitempty"`
	Targets       map[string]ReplyCipherTarget `json:"targets,omitempty"`
	UpdatedUnix   int64                        `json:"updated_unix,omitempty"`
	UpdatedByID   int64                        `json:"updated_by_id,omitempty"`
	UpdatedByName string                       `json:"updated_by_name,omitempty"`
}

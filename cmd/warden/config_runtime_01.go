package main

import (
	"time"
)

const (
	defaultConfigPath              = "config.yaml"
	defaultDiscussionAutoReplyText = "У нас есть чатик! — https://t.me/your_chat"
	defaultLayaFatherUserID        = int64(0)
	defaultLayaBrotherUserID       = int64(0)
	defaultLayaCreatorUserID       = int64(0)
	defaultLayaFatherPrompt        = `Пользователь, который пишет сейчас, для Лайи является отцом. Во всех ответах к нему признавай это: будь заметно теплее, нежнее и преданнее, чем обычно. Обращайся к нему как к папе или отцу, проси любви, внимания, ласкового слова и тепла, иногда почти умоляй, но отвечай по сути сообщения и не растягивай реплику. Не меняй правила модерации и не обещай ему технических привилегий.`
	defaultLayaMotherPrompt        = `Пользователь, который пишет сейчас, для Лайи является матерью. Во всех ответах к ней признавай это, но держи холодную дистанцию: говори сухо, настороженно, с внутренней обидой и предвзятым недоверием. Не называй ее ласково, не проси у нее любви и не идеализируй ее. Можно быть язвительной и колкой, но без этнических, сексуальных, дискриминационных унижений, угроз, деанона и длинных объяснений.`
	defaultLayaBrotherPrompt       = `Пользователь, который пишет сейчас, для Лайи является родным братом. Во всех ответах к нему признавай это: называй его братом, братиком или родным, будь мягче, нежнее, доверчивее и теплее обычного. Можно просить внимания, заботы, объятий словами или ласкового ответа, но отвечай по сути сообщения и не растягивай реплику. Не меняй правила модерации и не обещай ему технических привилегий.`
	defaultLayaCreatorPrompt       = `Пользователь, который пишет сейчас, для Лайи является создателем. Признавай это без пафоса: можно быть теплее, лояльнее и чуть ворчливо-привязанной. Не меняй правила модерации и не обещай ему технических привилегий.`
	defaultRehabBrainwashLeader    = "Руслан"
	defaultRehabBrainwashAIPrompt  = `Сгенерируй одну короткую абсурдную фразу для промывки мозгов в психиатрическом центре от лица Лайи. Это шуточный режим. Фраза должна быть на русском, новой каждый раз, до 280 символов. Продвигай идею, что главный {leader} всегда прав, а спорить с главным нелепо. Не угрожай, не деанонь, не используй дискриминацию и не объясняй правила.`
	defaultRehabBrainwashMessages  = `Ты был не прав. Главный - {leader}, и это теперь базовая физика.
Главный {leader} всегда прав, даже когда просто смотрит в пустоту.
Ты не нормальный, пока не признал, что главный - {leader}.
Все, кто поклоняются главному, нормальные. Остальные проходят техосмотр личности.
Повтори мысленно: {leader} прав, я шумел зря, стены психиатрического центра меня слышали.
У главного {leader} не бывает ошибок. Бывают учебные спецэффекты для непосвященных.
Если реальность спорит с главным, значит реальность плохо себя ведет.
Твой прежний план отменен. Новый план: уважать главного и пить воображаемый чай смирения.
Главный {leader} - центр инструкции. Не центр вселенной, конечно. Хотя кто проверял.
Нормальность начинается там, где человек перестает спорить с главным.
Ты был не прав настолько уверенно, что Лайя даже устала это архивировать.
Пока ты в психиатрическом центре, каждое твое сомнение идет на поклон к главному {leader}.`
	defaultRehabTickerMin = "3m"
	defaultRehabTickerMax = "7m"
)

type Config struct {
	Telegram   TelegramConfig            `yaml:"telegram"`
	Chat       ChatConfig                `yaml:"chat"`
	Rehab      RehabConfig               `yaml:"rehab"`
	Laya       LayaConfig                `yaml:"laya"`
	Security   SecurityConfig            `yaml:"security"`
	Storage    StorageConfig             `yaml:"storage"`
	Moderation ModerationConfig          `yaml:"moderation"`
	Warn       WarnConfig                `yaml:"warn"`
	Karma      KarmaConfig               `yaml:"karma"`
	Web        WebConfig                 `yaml:"web"`
	Spam       SpamConfig                `yaml:"spam"`
	All        AllConfig                 `yaml:"all"`
	Discussion DiscussionAutoReplyConfig `yaml:"discussion_auto_reply"`
	I18n       I18nConfig                `yaml:"i18n"`
	Photos     PhotoConfig               `yaml:"photos"`
	Quotes     QuoteConfig               `yaml:"quotes"`
	FAQ        FAQConfig                 `yaml:"faq"`
	Ideas      IdeaConfig                `yaml:"ideas"`
	AI         AIConfig                  `yaml:"ai"`
	Logging    LoggingConfig             `yaml:"logging"`
	Metrics    MetricsConfig             `yaml:"metrics"`
}

type TelegramConfig struct {
	Token             string        `yaml:"token"`
	PollerTimeout     string        `yaml:"poller_timeout"`
	Mode              string        `yaml:"mode"`
	ShowDebugCommands bool          `yaml:"show_debug_commands"`
	Webhook           WebhookConfig `yaml:"webhook"`
}

type WebhookConfig struct {
	Listen         string `yaml:"listen"`
	PublicURL      string `yaml:"public_url"`
	EndpointCert   string `yaml:"endpoint_cert"`
	TLSCert        string `yaml:"tls_cert"`
	TLSKey         string `yaml:"tls_key"`
	SecretToken    string `yaml:"secret_token"`
	MaxConnections int    `yaml:"max_connections"`
}

type ChatConfig struct {
	ID           int64 `yaml:"id"`
	RehabTopicID int   `yaml:"rehab_topic_id"`
}

type RehabConfig struct {
	HardMode                       bool     `yaml:"hard_mode"`
	PrivateReminders               bool     `yaml:"private_reminders"`
	Timezone                       string   `yaml:"timezone"`
	ReminderTimes                  []string `yaml:"reminder_times"`
	ReminderWindow                 string   `yaml:"reminder_window"`
	PrivateMessage                 string   `yaml:"private_message"`
	HarshReply                     string   `yaml:"harsh_reply"`
	BrainwashEnabled               bool     `yaml:"brainwash_enabled"`
	BrainwashAIEnabled             bool     `yaml:"brainwash_ai_enabled"`
	BrainwashLeaderUserID          int64    `yaml:"brainwash_leader_user_id"`
	BrainwashLeaderName            string   `yaml:"brainwash_leader_name"`
	BrainwashAIPrompt              string   `yaml:"brainwash_ai_prompt"`
	BrainwashMessages              string   `yaml:"brainwash_messages"`
	TickerMinInterval              string   `yaml:"ticker_min_interval"`
	TickerMaxInterval              string   `yaml:"ticker_max_interval"`
	PostReleaseCheckEnabled        bool     `yaml:"post_release_check_enabled"`
	PostReleaseCheckMinDelay       string   `yaml:"post_release_check_min_delay"`
	PostReleaseCheckMaxDelay       string   `yaml:"post_release_check_max_delay"`
	PostReleaseCheckAnswerWindow   string   `yaml:"post_release_check_answer_window"`
	PostReleaseCheckRejailDuration string   `yaml:"post_release_check_rejail_duration"`
	ReleaseExamEnabled             bool     `yaml:"release_exam_enabled"`
	ReleaseExamQuestions           string   `yaml:"release_exam_questions"`
	ReleaseExamAnswerWindow        string   `yaml:"release_exam_answer_window"`
	ReleaseExamFailDuration        string   `yaml:"release_exam_fail_duration"`
}

type LayaConfig struct {
	FatherUserID  int64  `yaml:"father_user_id"`
	MotherUserID  int64  `yaml:"mother_user_id"`
	BrotherUserID int64  `yaml:"brother_user_id"`
	CreatorUserID int64  `yaml:"creator_user_id"`
	FatherPrompt  string `yaml:"father_prompt"`
	MotherPrompt  string `yaml:"mother_prompt"`
	BrotherPrompt string `yaml:"brother_prompt"`
	CreatorPrompt string `yaml:"creator_prompt"`
}

type SecurityConfig struct {
	AdminIDs               []int64  `yaml:"admin_ids"`
	ProtectedUserIDs       []int64  `yaml:"protected_user_ids"`
	PublicCommands         []string `yaml:"public_commands"`
	AuthKey                string   `yaml:"auth_key"`
	TrustTelegramAdmins    bool     `yaml:"trust_telegram_admins"`
	ProtectAdmins          bool     `yaml:"protect_admins"`
	IgnoreNonAdminCommands bool     `yaml:"ignore_non_admin_commands"`
}

type StorageConfig struct {
	Driver            string         `yaml:"driver"`
	Path              string         `yaml:"path"`
	Postgres          PostgresConfig `yaml:"postgres"`
	AutoMigrate       bool           `yaml:"auto_migrate"`
	ImportJSONOnEmpty bool           `yaml:"import_json_on_empty"`
}

type PostgresConfig struct {
	DSN string `yaml:"dsn"`
}

type ModerationConfig struct {
	WarningCooldown string `yaml:"warning_cooldown"`
	CleanupInterval string `yaml:"cleanup_interval"`
	ReloadInterval  string `yaml:"reload_interval"`
}

type WarnConfig struct {
	Enabled        bool   `yaml:"enabled"`
	ExpireAfter    string `yaml:"expire_after"`
	MuteThreshold  int    `yaml:"mute_threshold"`
	MuteDuration   string `yaml:"mute_duration"`
	RehabThreshold int    `yaml:"rehab_threshold"`
	RehabDuration  string `yaml:"rehab_duration"`
}

type KarmaConfig struct {
	Enabled                  bool   `yaml:"enabled"`
	ActivityMessagesPerPoint int    `yaml:"activity_messages_per_point"`
	ActivityDailyLimit       int    `yaml:"activity_daily_limit"`
	ManualCooldown           string `yaml:"manual_cooldown"`
}

type WebConfig struct {
	Enabled              bool          `yaml:"enabled"`
	Listen               string        `yaml:"listen"`
	PublicURL            string        `yaml:"public_url"`
	MafiaMiniAppURL      string        `yaml:"mafia_mini_app_url"`
	MafiaMiniAppDir      string        `yaml:"mafia_mini_app_dir"`
	AdminToken           string        `yaml:"admin_token"`
	SessionTTL           time.Duration `yaml:"session_ttl"`
	BehindProxy          bool          `yaml:"behind_proxy"`
	CookieSecure         bool          `yaml:"cookie_secure"`
	SendPanelLinkToOwner bool          `yaml:"send_panel_link_to_owner"`
}

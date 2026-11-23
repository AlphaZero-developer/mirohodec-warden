package main

type MafiaRole string

const (
	MafiaRoleCivilian   MafiaRole = "civilian"
	MafiaRoleMafia      MafiaRole = "mafia"
	MafiaRoleDoctor     MafiaRole = "doctor"
	MafiaRoleDetective  MafiaRole = "detective"
	MafiaRoleDon        MafiaRole = "don"
	MafiaRoleSergeant   MafiaRole = "sergeant"
	MafiaRoleBodyguard  MafiaRole = "bodyguard"
	MafiaRoleBlocker    MafiaRole = "blocker"
	MafiaRoleLawyer     MafiaRole = "lawyer"
	MafiaRoleJournalist MafiaRole = "journalist"
	MafiaRoleMayor      MafiaRole = "mayor"
)

type MafiaRoomStatus string

const (
	MafiaStatusLobby            MafiaRoomStatus = "lobby"
	MafiaStatusRoleDistribution MafiaRoomStatus = "role_distribution"
	MafiaStatusNight            MafiaRoomStatus = "night"
	MafiaStatusDayDiscussion    MafiaRoomStatus = "day_discussion"
	MafiaStatusVoting           MafiaRoomStatus = "voting"
	MafiaStatusFinished         MafiaRoomStatus = "finished"
	MafiaStatusCancelled        MafiaRoomStatus = "cancelled"
)

type MafiaChatMode string

const (
	MafiaChatNightOnly MafiaChatMode = "night_only"
	MafiaChatAlways    MafiaChatMode = "always"
	MafiaChatDisabled  MafiaChatMode = "disabled"
)

type MafiaRevealPunishmentMode string

const (
	MafiaRevealDeath        MafiaRevealPunishmentMode = "death"
	MafiaRevealWarning      MafiaRevealPunishmentMode = "warning"
	MafiaRevealHostDecision MafiaRevealPunishmentMode = "host_decision"
)

type MafiaTieMode string

const (
	MafiaTieNobodyDies           MafiaTieMode = "nobody_dies"
	MafiaTieRandomBetweenLeaders MafiaTieMode = "random_between_leaders"
)

type MafiaRoleRevealPolicy string

const (
	MafiaRoleRevealAllowClaims    MafiaRoleRevealPolicy = "allow_claims"
	MafiaRoleRevealBanSystemProof MafiaRoleRevealPolicy = "ban_system_proof"
	MafiaRoleRevealBanAllClaims   MafiaRoleRevealPolicy = "ban_all_claims"
)

type MafiaRecruitmentStatus string

const (
	MafiaRecruitmentClosed MafiaRecruitmentStatus = "closed"
	MafiaRecruitmentOpen   MafiaRecruitmentStatus = "open"
)

type MafiaRecruitmentMode string

const (
	MafiaRecruitmentRequests   MafiaRecruitmentMode = "requests"
	MafiaRecruitmentAutoAccept MafiaRecruitmentMode = "auto_accept"
)

type MafiaRuleViolationType string

const (
	MafiaRuleViolationNone                MafiaRuleViolationType = "none"
	MafiaRuleViolationSystemProof         MafiaRuleViolationType = "system_proof"
	MafiaRuleViolationRoleClaim           MafiaRuleViolationType = "role_claim"
	MafiaRuleViolationScreenshotOrForward MafiaRuleViolationType = "screenshot_or_forward"
	MafiaRuleViolationPrivateChatLeak     MafiaRuleViolationType = "private_chat_leak"
	MafiaRuleViolationDeadTalking         MafiaRuleViolationType = "dead_talking"
	MafiaRuleViolationAllyLeak            MafiaRuleViolationType = "ally_leak"
	MafiaRuleViolationGameBreakingLeak    MafiaRuleViolationType = "game_breaking_leak"
)

type MafiaGameBreakingLeakAction string

const (
	MafiaLeakActionCancelGame   MafiaGameBreakingLeakAction = "cancel_game"
	MafiaLeakActionHostDecision MafiaGameBreakingLeakAction = "host_decision"
	MafiaLeakActionKillOffender MafiaGameBreakingLeakAction = "kill_offender"
)

type MafiaJoinRequestStatus string

const (
	MafiaJoinRequestPending  MafiaJoinRequestStatus = "pending"
	MafiaJoinRequestAccepted MafiaJoinRequestStatus = "accepted"
	MafiaJoinRequestDeclined MafiaJoinRequestStatus = "declined"
)

type MafiaRatingMode string

const (
	MafiaRatingRanked   MafiaRatingMode = "ranked"
	MafiaRatingUnranked MafiaRatingMode = "unranked"
)

type MafiaRatingSide string

const (
	MafiaRatingSideMafia    MafiaRatingSide = "mafia"
	MafiaRatingSideCivilian MafiaRatingSide = "civilian"
)

type MafiaStats struct {
	Games                int      `json:"games"`
	Wins                 int      `json:"wins"`
	Deaths               int      `json:"deaths"`
	Elo                  int      `json:"elo,omitempty"`
	LastEloDelta         int      `json:"last_elo_delta,omitempty"`
	OverallElo           int      `json:"overall_elo,omitempty"`
	CivilianElo          int      `json:"civilian_elo,omitempty"`
	MafiaElo             int      `json:"mafia_elo,omitempty"`
	LastRoleEloDelta     int      `json:"last_role_elo_delta,omitempty"`
	RankedGamesPlayed    int      `json:"ranked_games_played,omitempty"`
	RankedWins           int      `json:"ranked_wins,omitempty"`
	RankedLosses         int      `json:"ranked_losses,omitempty"`
	RoleReveals          int      `json:"role_reveals"`
	CancelledGamesCaused int      `json:"cancelled_games_caused,omitempty"`
	MafiaWins            int      `json:"mafia_wins"`
	CivilianWins         int      `json:"civilian_wins"`
	DetectiveWins        int      `json:"detective_wins"`
	DoctorWins           int      `json:"doctor_wins"`
	Achievements         []string `json:"achievements,omitempty"`
}

type MafiaUserSettings struct {
	AllowMafiaInvites         bool   `json:"allow_mafia_invites"`
	InviteOnlyFromFavorites   bool   `json:"invite_only_from_favorites"`
	LastInviteAt              int64  `json:"last_invite_at,omitempty"`
	InviteDay                 string `json:"invite_day,omitempty"`
	InvitesToday              int    `json:"invites_today,omitempty"`
	LastDiscussionGroupChatID int64  `json:"last_discussion_group_chat_id,omitempty"`
	Initialized               bool   `json:"initialized,omitempty"`
	ProfileInputMode          string `json:"profile_input_mode,omitempty"`
	PendingCharacterName      string `json:"pending_character_name,omitempty"`
	GeneratedCharacterName    string `json:"generated_character_name,omitempty"`
	GeneratedBackstory        string `json:"generated_backstory,omitempty"`
}

type MafiaProfile struct {
	CharacterName string `json:"character_name,omitempty"`
	Backstory     string `json:"backstory,omitempty"`
	AvatarURL     string `json:"avatar_url,omitempty"`
	AvatarText    string `json:"avatar_text,omitempty"`
	AvatarTone    string `json:"avatar_tone,omitempty"`
	CreatedAt     int64  `json:"created_at,omitempty"`
	UpdatedAt     int64  `json:"updated_at,omitempty"`
}

type MafiaRegisteredUser struct {
	TelegramID           int64             `json:"telegram_id"`
	Username             string            `json:"username,omitempty"`
	DisplayName          string            `json:"display_name"`
	Profile              MafiaProfile      `json:"mafia_profile,omitempty"`
	RegisteredAt         int64             `json:"registered_at"`
	Stats                MafiaStats        `json:"mafia_stats"`
	Settings             MafiaUserSettings `json:"settings,omitempty"`
	FavoriteMafiaUserIDs []int64           `json:"favorite_mafia_user_ids,omitempty"`
}

type MafiaPlayer struct {
	TelegramID        int64     `json:"telegram_id"`
	Username          string    `json:"username,omitempty"`
	DisplayName       string    `json:"display_name"`
	Backstory         string    `json:"backstory,omitempty"`
	IsTestBot         bool      `json:"is_test_bot,omitempty"`
	Role              MafiaRole `json:"role,omitempty"`
	MafiaAlias        string    `json:"mafia_alias,omitempty"`
	Alive             bool      `json:"alive"`
	WarnedForReveal   bool      `json:"warned_for_reveal"`
	NightActionDone   bool      `json:"night_action_done"`
	Will              string    `json:"will,omitempty"`
	JoinedAt          int64     `json:"joined_at"`
	LastDMCheckedUnix int64     `json:"last_dm_checked_unix,omitempty"`
}

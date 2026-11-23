package main

type MafiaOptionalRoleSettings struct {
	EnableDon        bool `json:"enable_don"`
	EnableSergeant   bool `json:"enable_sergeant"`
	EnableBodyguard  bool `json:"enable_bodyguard"`
	EnableBlocker    bool `json:"enable_blocker"`
	EnableLawyer     bool `json:"enable_lawyer"`
	EnableJournalist bool `json:"enable_journalist"`
	EnableMayor      bool `json:"enable_mayor"`
}

type MafiaRoomSettings struct {
	MinPlayers             int                         `json:"min_players"`
	MaxPlayers             int                         `json:"max_players"`
	AllowRoleReveal        bool                        `json:"allow_role_reveal"`
	PunishRoleReveal       bool                        `json:"punish_role_reveal"`
	RevealPunishmentMode   MafiaRevealPunishmentMode   `json:"reveal_punishment_mode"`
	RoleRevealPolicy       MafiaRoleRevealPolicy       `json:"role_reveal_policy,omitempty"`
	MafiaChatMode          MafiaChatMode               `json:"mafia_chat_mode"`
	DeadMafiaCanReadChat   bool                        `json:"dead_mafia_can_read_chat"`
	DayTimeSeconds         int                         `json:"day_time_seconds"`
	NightTimeSeconds       int                         `json:"night_time_seconds"`
	DayDiscussionSeconds   int                         `json:"day_discussion_seconds,omitempty"`
	NightActionSeconds     int                         `json:"night_action_seconds,omitempty"`
	VotingSeconds          int                         `json:"voting_seconds,omitempty"`
	PhaseCardSeconds       int                         `json:"phase_card_seconds,omitempty"`
	NightKillMessage       string                      `json:"night_kill_message,omitempty"`
	PublicLobby            bool                        `json:"public_lobby,omitempty"`
	RatingMode             MafiaRatingMode             `json:"rating_mode,omitempty"`
	EnablePostGameAIReport bool                        `json:"enable_post_game_ai_report"`
	EnableAIBots           bool                        `json:"enable_ai_bots,omitempty"`
	EnableAchievements     bool                        `json:"enable_achievements"`
	EnableWills            bool                        `json:"enable_wills"`
	EnableHostPowers       bool                        `json:"enable_host_powers,omitempty"`
	FeaturesInitialized    bool                        `json:"features_initialized,omitempty"`
	AllowVoteChange        bool                        `json:"allow_vote_change"`
	TieMode                MafiaTieMode                `json:"tie_mode"`
	OptionalRoles          MafiaOptionalRoleSettings   `json:"optional_roles,omitempty"`
	GameBreakingLeakAction MafiaGameBreakingLeakAction `json:"game_breaking_leak_action,omitempty"`
}

type MafiaNightActions struct {
	MafiaVotes       map[string]int64 `json:"mafia_votes,omitempty"`
	DoctorTargetID   int64            `json:"doctor_target_id,omitempty"`
	DetectiveChecks  map[string]int64 `json:"detective_checks,omitempty"`
	DetectiveResults map[string]bool  `json:"detective_results,omitempty"`
}

type MafiaPrivateMessage struct {
	ID          string `json:"id"`
	RoomID      string `json:"room_id"`
	SenderID    int64  `json:"sender_id"`
	SenderAlias string `json:"sender_alias"`
	Text        string `json:"text"`
	CreatedAt   int64  `json:"created_at"`
}

type MafiaAIBotMessage struct {
	DayNumber  int             `json:"day_number"`
	Phase      MafiaRoomStatus `json:"phase"`
	PlayerID   int64           `json:"player_id"`
	PlayerName string          `json:"player_name,omitempty"`
	Text       string          `json:"text"`
	CreatedAt  int64           `json:"created_at"`
}

type MafiaAIBotMemory struct {
	DayNumber   int             `json:"day_number"`
	Phase       MafiaRoomStatus `json:"phase"`
	MessageID   int             `json:"message_id,omitempty"`
	SpeakerID   int64           `json:"speaker_id"`
	SpeakerName string          `json:"speaker_name,omitempty"`
	Text        string          `json:"text"`
	TargetIDs   []int64         `json:"target_ids,omitempty"`
	TargetNames []string        `json:"target_names,omitempty"`
	CreatedAt   int64           `json:"created_at"`
}

type MafiaMiniAppMessage struct {
	ID                string `json:"id"`
	RoomID            string `json:"room_id"`
	Channel           string `json:"channel"`
	SenderID          int64  `json:"sender_id,omitempty"`
	RecipientID       int64  `json:"recipient_id,omitempty"`
	SenderName        string `json:"sender_name"`
	Text              string `json:"text"`
	System            bool   `json:"system,omitempty"`
	Important         bool   `json:"important,omitempty"`
	ReplyToID         string `json:"reply_to_id,omitempty"`
	ReplyToSenderName string `json:"reply_to_sender_name,omitempty"`
	ReplyToText       string `json:"reply_to_text,omitempty"`
	CreatedAt         int64  `json:"created_at"`
}

type MafiaPlayerRating struct {
	UserID            int64  `json:"user_id"`
	OverallElo        int    `json:"overall_elo"`
	CivilianElo       int    `json:"civilian_elo"`
	MafiaElo          int    `json:"mafia_elo"`
	RankedGamesPlayed int    `json:"ranked_games_played"`
	RankedWins        int    `json:"ranked_wins"`
	RankedLosses      int    `json:"ranked_losses"`
	CurrentRank       string `json:"current_rank"`
	UpdatedAt         int64  `json:"updated_at"`
}

type MafiaEloHistoryEntry struct {
	ID                 string          `json:"id"`
	UserID             int64           `json:"user_id"`
	GameID             string          `json:"game_id"`
	Role               MafiaRole       `json:"role"`
	Side               MafiaRatingSide `json:"side"`
	OldOverallElo      int             `json:"old_overall_elo"`
	NewOverallElo      int             `json:"new_overall_elo"`
	DeltaOverall       int             `json:"delta_overall"`
	OldRoleElo         int             `json:"old_role_elo"`
	NewRoleElo         int             `json:"new_role_elo"`
	DeltaRole          int             `json:"delta_role"`
	TeamRating         float64         `json:"team_rating"`
	OpponentTeamRating float64         `json:"opponent_team_rating"`
	ExpectedResult     float64         `json:"expected_result"`
	CreatedAt          int64           `json:"created_at"`
}

type MafiaGamePlayerSnapshot struct {
	UserID             int64           `json:"user_id"`
	Username           string          `json:"username,omitempty"`
	DisplayName        string          `json:"display_name"`
	Role               MafiaRole       `json:"role"`
	Side               MafiaRatingSide `json:"side"`
	IsTestBot          bool            `json:"is_test_bot,omitempty"`
	StartingOverallElo int             `json:"starting_overall_elo"`
	StartingRoleElo    int             `json:"starting_role_elo"`
	Won                bool            `json:"won,omitempty"`
}

type MafiaGameResult struct {
	ID              string                    `json:"id"`
	RoomID          string                    `json:"room_id"`
	WinningSide     MafiaRatingSide           `json:"winning_side"`
	IsRanked        bool                      `json:"is_ranked"`
	EloProcessed    bool                      `json:"elo_processed"`
	StartedAt       int64                     `json:"started_at"`
	FinishedAt      int64                     `json:"finished_at"`
	DurationSeconds int64                     `json:"duration_seconds,omitempty"`
	PlayersSnapshot []MafiaGamePlayerSnapshot `json:"players_snapshot"`
	EloHistory      []MafiaEloHistoryEntry    `json:"elo_history,omitempty"`
}

type MafiaJoinRequest struct {
	TelegramID    int64                  `json:"telegram_id"`
	CharacterName string                 `json:"character_name"`
	Username      string                 `json:"username,omitempty"`
	RequestedAt   int64                  `json:"requested_at"`
	Status        MafiaJoinRequestStatus `json:"status"`
}

type MafiaReadyQueueEntry struct {
	TelegramID    int64  `json:"telegram_id"`
	CharacterName string `json:"character_name"`
	Username      string `json:"username,omitempty"`
	QueuedAt      int64  `json:"queued_at"`
}

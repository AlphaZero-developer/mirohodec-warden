package main

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

type MafiaRoom struct {
	ID                    string                          `json:"id"`
	OwnerID               int64                           `json:"owner_id"`
	GroupChatID           int64                           `json:"group_chat_id,omitempty"`
	Status                MafiaRoomStatus                 `json:"status"`
	Players               []MafiaPlayer                   `json:"players"`
	DayNumber             int                             `json:"day_number"`
	Settings              MafiaRoomSettings               `json:"settings"`
	Votes                 map[string]string               `json:"votes,omitempty"`
	PhaseSkipVotes        map[string]bool                 `json:"phase_skip_votes,omitempty"`
	NightActions          MafiaNightActions               `json:"night_actions,omitempty"`
	MafiaPrivateChat      []MafiaPrivateMessage           `json:"mafia_private_chat,omitempty"`
	AIBotMessages         []MafiaAIBotMessage             `json:"ai_bot_messages,omitempty"`
	AIBotMemory           []MafiaAIBotMemory              `json:"ai_bot_memory,omitempty"`
	MiniAppMessages       []MafiaMiniAppMessage           `json:"mini_app_messages,omitempty"`
	DetectiveKnownRoles   map[string]map[string]MafiaRole `json:"detective_known_roles,omitempty"`
	PlayersSnapshot       []MafiaGamePlayerSnapshot       `json:"players_snapshot,omitempty"`
	AIBotLastSpokeUnix    int64                           `json:"ai_bot_last_spoke_unix,omitempty"`
	AIBotLastKillHintKey  string                          `json:"ai_bot_last_kill_hint_key,omitempty"`
	CreatedAt             int64                           `json:"created_at"`
	UpdatedAt             int64                           `json:"updated_at"`
	Winner                string                          `json:"winner,omitempty"`
	FinishedAt            int64                           `json:"finished_at,omitempty"`
	EloProcessed          bool                            `json:"elo_processed,omitempty"`
	GameResult            MafiaGameResult                 `json:"game_result,omitempty"`
	CancelReason          MafiaRuleViolationType          `json:"cancel_reason,omitempty"`
	CancelledBy           int64                           `json:"cancelled_by,omitempty"`
	CancelledAt           int64                           `json:"cancelled_at,omitempty"`
	PhaseStartedAt        int64                           `json:"phase_started_at,omitempty"`
	PendingLeakType       MafiaRuleViolationType          `json:"pending_leak_type,omitempty"`
	PendingLeakOffenderID int64                           `json:"pending_leak_offender_id,omitempty"`
	PendingLeakAt         int64                           `json:"pending_leak_at,omitempty"`

	RecruitmentStatus     MafiaRecruitmentStatus `json:"recruitment_status,omitempty"`
	RecruitmentMode       MafiaRecruitmentMode   `json:"recruitment_mode,omitempty"`
	JoinRequests          []MafiaJoinRequest     `json:"join_requests,omitempty"`
	InviteBroadcastCount  int                    `json:"invite_broadcast_count,omitempty"`
	LastInviteBroadcastAt int64                  `json:"last_invite_broadcast_at,omitempty"`
}

type MafiaState struct {
	RegisteredUsers map[string]MafiaRegisteredUser `json:"registered_users,omitempty"`
	Rooms           map[string]MafiaRoom           `json:"rooms,omitempty"`
	ReadyQueue      []MafiaReadyQueueEntry         `json:"ready_queue,omitempty"`
	CancelledGames  []MafiaCancelledGameRecord     `json:"cancelled_games,omitempty"`
	HostBlacklist   []MafiaHostBlacklistEntry      `json:"host_blacklist,omitempty"`
	PlayerRatings   map[string]MafiaPlayerRating   `json:"player_ratings,omitempty"`
	EloHistory      []MafiaEloHistoryEntry         `json:"elo_history,omitempty"`
	GameResults     []MafiaGameResult              `json:"game_results,omitempty"`
}

type MafiaCancelledGameRecord struct {
	RoomID      string                 `json:"room_id"`
	CancelledAt int64                  `json:"cancelled_at"`
	Reason      MafiaRuleViolationType `json:"reason"`
	OffenderID  int64                  `json:"offender_id"`
}

type MafiaHostBlacklistEntry struct {
	HostID        int64  `json:"host_id"`
	BlockedUserID int64  `json:"blocked_user_id"`
	Reason        string `json:"reason"`
	CreatedAt     int64  `json:"created_at"`
}

func defaultMafiaRoomSettings() MafiaRoomSettings {
	return MafiaRoomSettings{
		MinPlayers:             4,
		MaxPlayers:             12,
		AllowRoleReveal:        false,
		PunishRoleReveal:       true,
		RevealPunishmentMode:   MafiaRevealDeath,
		RoleRevealPolicy:       MafiaRoleRevealBanAllClaims,
		MafiaChatMode:          MafiaChatNightOnly,
		DeadMafiaCanReadChat:   false,
		DayTimeSeconds:         180,
		NightTimeSeconds:       90,
		DayDiscussionSeconds:   180,
		NightActionSeconds:     90,
		VotingSeconds:          60,
		PhaseCardSeconds:       4,
		NightKillMessage:       defaultMafiaNightKillMessage(),
		RatingMode:             MafiaRatingRanked,
		EnablePostGameAIReport: true,
		EnableAchievements:     true,
		EnableWills:            true,
		FeaturesInitialized:    true,
		AllowVoteChange:        false,
		TieMode:                MafiaTieNobodyDies,
		GameBreakingLeakAction: MafiaLeakActionCancelGame,
	}
}

func mafiaUserFromTelegram(user *tele.User, now time.Time) MafiaRegisteredUser {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return MafiaRegisteredUser{
		TelegramID:   user.ID,
		Username:     user.Username,
		DisplayName:  displayNameFromUser(user),
		RegisteredAt: now.UTC().Unix(),
		Settings:     defaultMafiaUserSettings(),
	}
}

func defaultMafiaUserSettings() MafiaUserSettings {
	return MafiaUserSettings{
		AllowMafiaInvites:       true,
		InviteOnlyFromFavorites: false,
		Initialized:             true,
	}
}

func normalizeMafiaUserSettings(settings MafiaUserSettings) MafiaUserSettings {
	if !settings.Initialized {
		settings.AllowMafiaInvites = true
		settings.InviteOnlyFromFavorites = false
		settings.Initialized = true
	}
	return settings
}

func mafiaPlayerFromUser(user MafiaRegisteredUser, now time.Time) MafiaPlayer {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return MafiaPlayer{
		TelegramID:  user.TelegramID,
		Username:    user.Username,
		DisplayName: mafiaUserCharacterName(user),
		Backstory:   strings.TrimSpace(user.Profile.Backstory),
		Alive:       true,
		JoinedAt:    now.UTC().Unix(),
	}
}

func normalizeMafiaRoomID(id string) string {
	id = strings.ToUpper(strings.TrimSpace(id))
	id = strings.ReplaceAll(id, " ", "")
	return id
}

func mafiaPlayerLabel(player MafiaPlayer) string {
	name := strings.TrimSpace(player.DisplayName)
	if name == "" {
		name = fmt.Sprintf("user %d", player.TelegramID)
	}
	return name
}

func mafiaUserCharacterName(user MafiaRegisteredUser) string {
	name := strings.TrimSpace(user.Profile.CharacterName)
	if name == "" {
		name = strings.TrimSpace(user.DisplayName)
	}
	if name == "" {
		name = fmt.Sprintf("user %d", user.TelegramID)
	}
	return name
}

func mafiaProfileComplete(user MafiaRegisteredUser) bool {
	return strings.TrimSpace(user.Profile.CharacterName) != "" && strings.TrimSpace(user.Profile.Backstory) != ""
}

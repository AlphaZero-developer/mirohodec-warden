package web

import (
	"context"
)

type MafiaMiniAppService interface {
	MafiaMiniAppAuth(ctx context.Context, input MafiaMiniAppAuthInput) (MafiaMiniAppAuthPayload, error)
	MafiaMiniAppUpdateProfile(ctx context.Context, auth MafiaMiniAppAuthInput, input MafiaMiniAppProfileInput) (MafiaMiniAppAuthPayload, error)
	MafiaMiniAppProfile(ctx context.Context, auth MafiaMiniAppAuthInput, userID int64) (MafiaMiniAppProfilePayload, error)
	MafiaMiniAppLobby(ctx context.Context, auth MafiaMiniAppAuthInput) (MafiaMiniAppLobbyPayload, error)
	MafiaMiniAppCreateRoom(ctx context.Context, auth MafiaMiniAppAuthInput) (MafiaMiniAppState, error)
	MafiaMiniAppJoinRoom(ctx context.Context, auth MafiaMiniAppAuthInput, roomID string) (MafiaMiniAppState, error)
	MafiaMiniAppRoomState(ctx context.Context, auth MafiaMiniAppAuthInput, roomID string) (MafiaMiniAppState, error)
	MafiaMiniAppSendMessage(ctx context.Context, auth MafiaMiniAppAuthInput, input MafiaMiniAppMessageInput) (MafiaMiniAppState, error)
	MafiaMiniAppSaveWill(ctx context.Context, auth MafiaMiniAppAuthInput, input MafiaMiniAppWillInput) (MafiaMiniAppState, error)
	MafiaMiniAppSkipPhase(ctx context.Context, auth MafiaMiniAppAuthInput, roomID string) (MafiaMiniAppState, error)
	MafiaMiniAppSubmitVote(ctx context.Context, auth MafiaMiniAppAuthInput, input MafiaMiniAppVoteInput) (MafiaMiniAppState, error)
	MafiaMiniAppSubmitNightAction(ctx context.Context, auth MafiaMiniAppAuthInput, input MafiaMiniAppNightActionInput) (MafiaMiniAppState, error)
	MafiaMiniAppStartGame(ctx context.Context, auth MafiaMiniAppAuthInput, roomID string) (MafiaMiniAppState, error)
	MafiaMiniAppNextPhase(ctx context.Context, auth MafiaMiniAppAuthInput, roomID string) (MafiaMiniAppState, error)
	MafiaMiniAppUpdateSettings(ctx context.Context, auth MafiaMiniAppAuthInput, roomID string, input MafiaMiniAppSettingsInput) (MafiaMiniAppState, error)
	MafiaMiniAppKickPlayer(ctx context.Context, auth MafiaMiniAppAuthInput, input MafiaMiniAppKickInput) (MafiaMiniAppState, error)
	MafiaMiniAppUpdateTestBots(ctx context.Context, auth MafiaMiniAppAuthInput, input MafiaMiniAppTestBotsInput) (MafiaMiniAppState, error)
	MafiaMiniAppLeaveRoom(ctx context.Context, auth MafiaMiniAppAuthInput, roomID string) (MafiaMiniAppLobbyPayload, error)
	MafiaMiniAppCloseRoom(ctx context.Context, auth MafiaMiniAppAuthInput, roomID string) (MafiaMiniAppLobbyPayload, error)
}

type MafiaMiniAppAuthInput struct {
	InitData    string `json:"initData,omitempty"`
	DevUserID   int64  `json:"devUserId,omitempty"`
	DevUsername string `json:"devUsername,omitempty"`
	DevName     string `json:"devName,omitempty"`
}

type MafiaMiniAppAuthPayload struct {
	User        MafiaMiniAppUser               `json:"user"`
	Rooms       []MafiaMiniAppRoomSummary      `json:"rooms"`
	PublicRooms []MafiaMiniAppRoomSummary      `json:"publicRooms"`
	Leaderboard []MafiaMiniAppLeaderboardEntry `json:"leaderboard"`
	Profile     MafiaMiniAppProfile            `json:"profile"`
	History     []MafiaMiniAppHistoryEntry     `json:"history"`
	DevAuth     bool                           `json:"devAuth"`
}

type MafiaMiniAppLobbyPayload struct {
	User        MafiaMiniAppUser               `json:"user"`
	Rooms       []MafiaMiniAppRoomSummary      `json:"rooms"`
	PublicRooms []MafiaMiniAppRoomSummary      `json:"publicRooms"`
	Leaderboard []MafiaMiniAppLeaderboardEntry `json:"leaderboard"`
	Profile     MafiaMiniAppProfile            `json:"profile"`
	History     []MafiaMiniAppHistoryEntry     `json:"history"`
}

type MafiaMiniAppProfilePayload struct {
	Profile MafiaMiniAppProfile `json:"profile"`
}

type MafiaMiniAppUser struct {
	ID          int64  `json:"id"`
	Username    string `json:"username,omitempty"`
	FirstName   string `json:"firstName,omitempty"`
	LastName    string `json:"lastName,omitempty"`
	DisplayName string `json:"displayName"`
}

type MafiaMiniAppRoomSummary struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	StatusLabel string `json:"statusLabel"`
	DayNumber   int    `json:"dayNumber"`
	Players     int    `json:"players"`
	MaxPlayers  int    `json:"maxPlayers"`
	IsHost      bool   `json:"isHost"`
	IsPlayer    bool   `json:"isPlayer"`
	CanStart    bool   `json:"canStart"`
	Public      bool   `json:"public"`
	UpdatedAt   int64  `json:"updatedAt"`
}

type MafiaMiniAppLeaderboardEntry struct {
	UserID            int64  `json:"userId"`
	DisplayName       string `json:"displayName"`
	Username          string `json:"username,omitempty"`
	AvatarURL         string `json:"avatarUrl,omitempty"`
	AvatarText        string `json:"avatarText,omitempty"`
	AvatarTone        string `json:"avatarTone,omitempty"`
	Elo               int    `json:"elo"`
	Rank              string `json:"rank"`
	OverallElo        int    `json:"overallElo"`
	CivilianElo       int    `json:"civilianElo"`
	MafiaElo          int    `json:"mafiaElo"`
	RankedGamesPlayed int    `json:"rankedGamesPlayed"`
	RankedWins        int    `json:"rankedWins"`
	RankedLosses      int    `json:"rankedLosses"`
	WinRate           int    `json:"winRate"`
	CurrentRank       string `json:"currentRank"`
	Wins              int    `json:"wins"`
	Games             int    `json:"games"`
}

type MafiaMiniAppState struct {
	Room              MafiaMiniAppRoom          `json:"room"`
	Me                MafiaMiniAppMe            `json:"me"`
	Channels          []MafiaMiniAppChannel     `json:"channels"`
	AvailableChannels []string                  `json:"availableChannels"`
	Messages          []MafiaMiniAppMessage     `json:"messages"`
	Players           []MafiaMiniAppPlayer      `json:"players"`
	Log               []MafiaMiniAppLogEvent    `json:"log"`
	Action            MafiaMiniAppAction        `json:"action"`
	AvailableActions  []string                  `json:"availableActions"`
	ServerTime        int64                     `json:"serverTime"`
	FinalResult       *MafiaMiniAppFinalResult  `json:"finalResult,omitempty"`
	Warnings          []MafiaMiniAppStateNotice `json:"warnings,omitempty"`
}

type MafiaMiniAppRoom struct {
	ID                   string                   `json:"id"`
	Title                string                   `json:"title"`
	Status               string                   `json:"status"`
	StatusLabel          string                   `json:"statusLabel"`
	DayNumber            int                      `json:"dayNumber"`
	PhaseStartedAt       int64                    `json:"phaseStartedAt"`
	PhaseEndsAt          int64                    `json:"phaseEndsAt,omitempty"`
	RemainingSeconds     int                      `json:"remainingSeconds"`
	Players              int                      `json:"players"`
	MaxPlayers           int                      `json:"maxPlayers"`
	AlivePlayers         int                      `json:"alivePlayers"`
	GroupChatID          int64                    `json:"groupChatId,omitempty"`
	Winner               string                   `json:"winner,omitempty"`
	RequiresGroupBinding bool                     `json:"requiresGroupBinding"`
	Settings             MafiaMiniAppRoomSettings `json:"settings"`
	CanManage            bool                     `json:"canManage"`
	CanStart             bool                     `json:"canStart"`
	CanAdvance           bool                     `json:"canAdvance"`
	CanLeave             bool                     `json:"canLeave"`
	CanClose             bool                     `json:"canClose"`
	CanSkipPhase         bool                     `json:"canSkipPhase"`
	PhaseSkipVotes       int                      `json:"phaseSkipVotes"`
	PhaseSkipNeeded      int                      `json:"phaseSkipNeeded"`
	PhaseSkipVoted       bool                     `json:"phaseSkipVoted"`
}

type MafiaMiniAppMe struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	IsHost          bool   `json:"isHost"`
	IsPlayer        bool   `json:"isPlayer"`
	Alive           bool   `json:"alive"`
	Role            string `json:"role,omitempty"`
	RoleLabel       string `json:"roleLabel,omitempty"`
	RoleDescription string `json:"roleDescription,omitempty"`
	MafiaAlias      string `json:"mafiaAlias,omitempty"`
	Will            string `json:"will,omitempty"`
	ActionSubmitted bool   `json:"actionSubmitted"`
}

type MafiaMiniAppProfile struct {
	UserID        int64                     `json:"userId"`
	Username      string                    `json:"username,omitempty"`
	DisplayName   string                    `json:"displayName"`
	CharacterName string                    `json:"characterName,omitempty"`
	AvatarURL     string                    `json:"avatarUrl,omitempty"`
	AvatarText    string                    `json:"avatarText,omitempty"`
	AvatarTone    string                    `json:"avatarTone,omitempty"`
	Games         int                       `json:"games"`
	Wins          int                       `json:"wins"`
	Losses        int                       `json:"losses"`
	Deaths        int                       `json:"deaths"`
	FavoriteRole  string                    `json:"favoriteRole,omitempty"`
	Stats         MafiaMiniAppProfileStats  `json:"stats"`
	Elo           int                       `json:"elo"`
	Rank          string                    `json:"rank"`
	LastEloDelta  int                       `json:"lastEloDelta"`
	OverallElo    int                       `json:"overallElo"`
	CivilianElo   int                       `json:"civilianElo"`
	MafiaElo      int                       `json:"mafiaElo"`
	Achievements  []MafiaMiniAppAchievement `json:"achievements,omitempty"`
	ActiveRoomID  string                    `json:"activeRoomId,omitempty"`
	StatusText    string                    `json:"statusText"`
}

package web

type MafiaMiniAppProfileStats struct {
	Games                int `json:"games"`
	Wins                 int `json:"wins"`
	Losses               int `json:"losses"`
	Deaths               int `json:"deaths"`
	WinRate              int `json:"winRate"`
	Elo                  int `json:"elo"`
	LastEloDelta         int `json:"lastEloDelta"`
	OverallElo           int `json:"overallElo"`
	CivilianElo          int `json:"civilianElo"`
	MafiaElo             int `json:"mafiaElo"`
	LastRoleEloDelta     int `json:"lastRoleEloDelta"`
	RankedGamesPlayed    int `json:"rankedGamesPlayed"`
	RankedWins           int `json:"rankedWins"`
	RankedLosses         int `json:"rankedLosses"`
	RoleReveals          int `json:"roleReveals"`
	CancelledGamesCaused int `json:"cancelledGamesCaused"`
	MafiaWins            int `json:"mafiaWins"`
	CivilianWins         int `json:"civilianWins"`
	DetectiveWins        int `json:"detectiveWins"`
	DoctorWins           int `json:"doctorWins"`
	Achievements         int `json:"achievements"`
	ActivityScore        int `json:"activityScore"`
}

type MafiaMiniAppAchievement struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Unlocked    bool   `json:"unlocked"`
}

type MafiaMiniAppHistoryEntry struct {
	ID           string `json:"id"`
	Date         int64  `json:"date"`
	Result       string `json:"result"`
	Role         string `json:"role,omitempty"`
	RoleLabel    string `json:"roleLabel,omitempty"`
	Winner       string `json:"winner,omitempty"`
	Players      int    `json:"players"`
	DurationText string `json:"durationText,omitempty"`
	IsRanked     bool   `json:"isRanked"`
	WinningSide  string `json:"winningSide,omitempty"`
	EloDelta     int    `json:"eloDelta,omitempty"`
}

type MafiaMiniAppRoomSettings struct {
	MinPlayers             int    `json:"minPlayers"`
	MaxPlayers             int    `json:"maxPlayers"`
	DayDiscussionSeconds   int    `json:"dayDiscussionSeconds"`
	NightActionSeconds     int    `json:"nightActionSeconds"`
	VotingSeconds          int    `json:"votingSeconds"`
	PhaseCardSeconds       int    `json:"phaseCardSeconds"`
	NightKillMessage       string `json:"nightKillMessage"`
	PublicLobby            bool   `json:"publicLobby"`
	RatingMode             string `json:"ratingMode"`
	AllowVoteChange        bool   `json:"allowVoteChange"`
	EnableAIBots           bool   `json:"enableAIBots"`
	EnableAchievements     bool   `json:"enableAchievements"`
	EnableWills            bool   `json:"enableWills"`
	EnablePostGameAIReport bool   `json:"enablePostGameAIReport"`
	EnableHostPowers       bool   `json:"enableHostPowers"`
	MafiaChatMode          string `json:"mafiaChatMode"`
	DeadMafiaCanReadChat   bool   `json:"deadMafiaCanReadChat"`
	TieMode                string `json:"tieMode"`
	Locked                 bool   `json:"locked"`
}

type MafiaMiniAppEloLine struct {
	Label string `json:"label"`
	Old   int    `json:"old"`
	Delta int    `json:"delta"`
	New   int    `json:"new"`
}

type MafiaMiniAppFinalResult struct {
	GameID      string              `json:"gameId"`
	WinningSide string              `json:"winningSide"`
	WinnerLabel string              `json:"winnerLabel"`
	Role        string              `json:"role,omitempty"`
	RoleLabel   string              `json:"roleLabel,omitempty"`
	PlayerWon   bool                `json:"playerWon"`
	Ranked      bool                `json:"ranked"`
	Rank        string              `json:"rank"`
	Overall     MafiaMiniAppEloLine `json:"overall"`
	RoleElo     MafiaMiniAppEloLine `json:"roleElo"`
	Message     string              `json:"message,omitempty"`
}

type MafiaMiniAppChannel struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Visible  bool   `json:"visible"`
	Writable bool   `json:"writable"`
	Reason   string `json:"reason,omitempty"`
}

type MafiaMiniAppMessage struct {
	ID                string `json:"id"`
	Channel           string `json:"channel"`
	SenderID          int64  `json:"senderId,omitempty"`
	SenderName        string `json:"senderName"`
	SenderAvatarURL   string `json:"senderAvatarUrl,omitempty"`
	SenderAvatarText  string `json:"senderAvatarText,omitempty"`
	SenderAvatarTone  string `json:"senderAvatarTone,omitempty"`
	Text              string `json:"text"`
	System            bool   `json:"system"`
	Important         bool   `json:"important"`
	ReplyToID         string `json:"replyToId,omitempty"`
	ReplyToSenderName string `json:"replyToSenderName,omitempty"`
	ReplyToText       string `json:"replyToText,omitempty"`
	CreatedAt         int64  `json:"createdAt"`
}

type MafiaMiniAppPlayer struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Username   string `json:"username,omitempty"`
	Initials   string `json:"initials"`
	AvatarURL  string `json:"avatarUrl,omitempty"`
	AvatarText string `json:"avatarText,omitempty"`
	AvatarTone string `json:"avatarTone,omitempty"`
	Alive      bool   `json:"alive"`
	IsSelf     bool   `json:"isSelf"`
	IsHost     bool   `json:"isHost"`
	IsTestBot  bool   `json:"isTestBot"`
	Role       string `json:"role,omitempty"`
	RoleLabel  string `json:"roleLabel,omitempty"`
	MafiaAlias string `json:"mafiaAlias,omitempty"`
	Voted      bool   `json:"voted"`
	ActionDone bool   `json:"actionDone"`
}

type MafiaMiniAppLogEvent struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Kind      string `json:"kind"`
	Important bool   `json:"important"`
	CreatedAt int64  `json:"createdAt"`
}

type MafiaMiniAppAction struct {
	Kind         string                     `json:"kind"`
	Title        string                     `json:"title"`
	Description  string                     `json:"description"`
	ConfirmLabel string                     `json:"confirmLabel"`
	Submitted    bool                       `json:"submitted"`
	Disabled     bool                       `json:"disabled"`
	StatusText   string                     `json:"statusText,omitempty"`
	Targets      []MafiaMiniAppActionTarget `json:"targets"`
}

type MafiaMiniAppActionTarget struct {
	ID       int64  `json:"id"`
	Key      string `json:"key"`
	Name     string `json:"name"`
	Subtitle string `json:"subtitle,omitempty"`
	Disabled bool   `json:"disabled"`
}

type MafiaMiniAppStateNotice struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

type MafiaMiniAppMessageInput struct {
	RoomID    string `json:"roomId"`
	Channel   string `json:"channel"`
	Text      string `json:"text"`
	ReplyToID string `json:"replyToId,omitempty"`
}

type MafiaMiniAppWillInput struct {
	RoomID string `json:"roomId"`
	Text   string `json:"text"`
}

type MafiaMiniAppVoteInput struct {
	RoomID string `json:"roomId"`
	Target string `json:"target"`
}

type MafiaMiniAppNightActionInput struct {
	RoomID   string `json:"roomId"`
	Action   string `json:"action"`
	TargetID int64  `json:"targetId"`
}

type MafiaMiniAppKickInput struct {
	RoomID   string `json:"roomId"`
	TargetID int64  `json:"targetId"`
}

package web

type MafiaMiniAppTestBotsInput struct {
	RoomID string `json:"roomId"`
	Action string `json:"action"`
	Count  int    `json:"count,omitempty"`
}

type MafiaMiniAppProfileInput struct {
	AvatarURL  string `json:"avatarUrl,omitempty"`
	AvatarText string `json:"avatarText,omitempty"`
	AvatarTone string `json:"avatarTone,omitempty"`
}

type MafiaMiniAppSettingsInput struct {
	MinPlayers             *int    `json:"minPlayers,omitempty"`
	MaxPlayers             *int    `json:"maxPlayers,omitempty"`
	DayDiscussionSeconds   *int    `json:"dayDiscussionSeconds,omitempty"`
	NightActionSeconds     *int    `json:"nightActionSeconds,omitempty"`
	VotingSeconds          *int    `json:"votingSeconds,omitempty"`
	PhaseCardSeconds       *int    `json:"phaseCardSeconds,omitempty"`
	NightKillMessage       *string `json:"nightKillMessage,omitempty"`
	PublicLobby            *bool   `json:"publicLobby,omitempty"`
	RatingMode             *string `json:"ratingMode,omitempty"`
	AllowVoteChange        *bool   `json:"allowVoteChange,omitempty"`
	EnableAIBots           *bool   `json:"enableAIBots,omitempty"`
	EnableAchievements     *bool   `json:"enableAchievements,omitempty"`
	EnableWills            *bool   `json:"enableWills,omitempty"`
	EnablePostGameAIReport *bool   `json:"enablePostGameAIReport,omitempty"`
	EnableHostPowers       *bool   `json:"enableHostPowers,omitempty"`
	MafiaChatMode          *string `json:"mafiaChatMode,omitempty"`
	DeadMafiaCanReadChat   *bool   `json:"deadMafiaCanReadChat,omitempty"`
	TieMode                *string `json:"tieMode,omitempty"`
}

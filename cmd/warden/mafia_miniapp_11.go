package main

import (
	wardenweb "mirohodec-warden/internal/web"
	"strings"
)

func mafiaMiniRoleEloLabel(side MafiaRatingSide) string {
	if side == MafiaRatingSideMafia {
		return "Mafia ELO"
	}
	return "Civilian ELO"
}

func mafiaMiniSettings(settings MafiaRoomSettings, locked bool) wardenweb.MafiaMiniAppRoomSettings {
	settings = normalizeMafiaSettings(settings)
	return wardenweb.MafiaMiniAppRoomSettings{
		MinPlayers:             settings.MinPlayers,
		MaxPlayers:             settings.MaxPlayers,
		DayDiscussionSeconds:   settings.DayDiscussionSeconds,
		NightActionSeconds:     settings.NightActionSeconds,
		VotingSeconds:          settings.VotingSeconds,
		PhaseCardSeconds:       settings.PhaseCardSeconds,
		NightKillMessage:       settings.NightKillMessage,
		PublicLobby:            settings.PublicLobby,
		RatingMode:             string(settings.RatingMode),
		AllowVoteChange:        settings.AllowVoteChange,
		EnableAIBots:           settings.EnableAIBots,
		EnableAchievements:     settings.EnableAchievements,
		EnableWills:            settings.EnableWills,
		EnablePostGameAIReport: settings.EnablePostGameAIReport,
		EnableHostPowers:       settings.EnableHostPowers,
		MafiaChatMode:          string(settings.MafiaChatMode),
		DeadMafiaCanReadChat:   settings.DeadMafiaCanReadChat,
		TieMode:                string(settings.TieMode),
		Locked:                 locked,
	}
}

func mafiaMiniMergeSettingsInput(settings MafiaRoomSettings, input wardenweb.MafiaMiniAppSettingsInput) MafiaRoomSettings {
	if input.MinPlayers != nil {
		settings.MinPlayers = *input.MinPlayers
	}
	if input.MaxPlayers != nil {
		settings.MaxPlayers = *input.MaxPlayers
	}
	if input.DayDiscussionSeconds != nil {
		settings.DayDiscussionSeconds = *input.DayDiscussionSeconds
		settings.DayTimeSeconds = *input.DayDiscussionSeconds
	}
	if input.NightActionSeconds != nil {
		settings.NightActionSeconds = *input.NightActionSeconds
		settings.NightTimeSeconds = *input.NightActionSeconds
	}
	if input.VotingSeconds != nil {
		settings.VotingSeconds = *input.VotingSeconds
	}
	if input.PhaseCardSeconds != nil {
		settings.PhaseCardSeconds = *input.PhaseCardSeconds
	}
	if input.NightKillMessage != nil {
		settings.NightKillMessage = cleanMafiaRoomMessageTemplate(*input.NightKillMessage)
	}
	if input.PublicLobby != nil {
		settings.PublicLobby = *input.PublicLobby
	}
	if input.RatingMode != nil {
		if MafiaRatingMode(strings.TrimSpace(*input.RatingMode)) == MafiaRatingUnranked {
			settings.RatingMode = MafiaRatingUnranked
		} else {
			settings.RatingMode = MafiaRatingRanked
		}
	}
	if input.AllowVoteChange != nil {
		settings.AllowVoteChange = *input.AllowVoteChange
	}
	if input.EnableAIBots != nil {
		settings.EnableAIBots = *input.EnableAIBots
	}
	if input.EnableAchievements != nil {
		settings.EnableAchievements = *input.EnableAchievements
	}
	if input.EnableWills != nil {
		settings.EnableWills = *input.EnableWills
	}
	if input.EnablePostGameAIReport != nil {
		settings.EnablePostGameAIReport = *input.EnablePostGameAIReport
	}
	if input.EnableHostPowers != nil {
		settings.EnableHostPowers = *input.EnableHostPowers
	}
	if input.DeadMafiaCanReadChat != nil {
		settings.DeadMafiaCanReadChat = *input.DeadMafiaCanReadChat
	}
	if input.MafiaChatMode != nil {
		switch MafiaChatMode(strings.TrimSpace(*input.MafiaChatMode)) {
		case MafiaChatAlways, MafiaChatNightOnly, MafiaChatDisabled:
			settings.MafiaChatMode = MafiaChatMode(strings.TrimSpace(*input.MafiaChatMode))
		}
	}
	if input.TieMode != nil {
		switch MafiaTieMode(strings.TrimSpace(*input.TieMode)) {
		case MafiaTieNobodyDies, MafiaTieRandomBetweenLeaders:
			settings.TieMode = MafiaTieMode(strings.TrimSpace(*input.TieMode))
		}
	}
	return settings
}

func mafiaMiniAvailableChannelIDs(channels []wardenweb.MafiaMiniAppChannel) []string {
	out := make([]string, 0, len(channels))
	for _, channel := range channels {
		if channel.Visible {
			out = append(out, channel.ID)
		}
	}
	return out
}

func mafiaMiniAvailableActions(action wardenweb.MafiaMiniAppAction) []string {
	if action.Disabled || strings.TrimSpace(action.Kind) == "" || action.Kind == "none" || len(action.Targets) == 0 {
		return []string{}
	}
	switch action.Kind {
	case "kill":
		return []string{"MAFIA_KILL"}
	case "heal":
		return []string{"HEAL"}
	case "check":
		return []string{"CHECK"}
	case "vote":
		return []string{"VOTE"}
	default:
		return []string{strings.ToUpper(action.Kind)}
	}
}

func mafiaMiniWarnings(room MafiaRoom) []wardenweb.MafiaMiniAppStateNotice {
	if room.Status == MafiaStatusLobby && room.GroupChatID == 0 {
		return []wardenweb.MafiaMiniAppStateNotice{{Kind: "warning", Text: "Переходный режим: старт игры пока использует старую привязку группы через /mafia_bind."}}
	}
	return nil
}

func mafiaMiniRoleLabel(role MafiaRole) string {
	if role == "" {
		return ""
	}
	label := mafiaRoleLabel(role)
	if strings.TrimSpace(label) == "" || label == string(role) {
		switch role {
		case MafiaRoleCivilian:
			return "Мирный"
		case MafiaRoleMafia:
			return "Мафия"
		case MafiaRoleDoctor:
			return "Доктор"
		case MafiaRoleDetective:
			return "Детектив"
		default:
			return string(role)
		}
	}
	return label
}

func mafiaMiniRoleDescription(role MafiaRole) string {
	switch role {
	case MafiaRoleMafia:
		return "Ночью выбирает жертву вместе с мафией. Днём притворяется обычным жителем."
	case MafiaRoleDoctor:
		return "Ночью выбирает одного живого игрока и может спасти его от убийства."
	case MafiaRoleDetective:
		return "Ночью проверяет одного живого игрока и узнаёт, мафия он или нет."
	case MafiaRoleCivilian:
		return "Нет ночного действия. Твоя сила — обсуждение, внимательность и голос."
	default:
		return "Лайя покажет доступное действие, когда наступит подходящая фаза."
	}
}

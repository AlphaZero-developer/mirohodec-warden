package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

func (s *MafiaService) newRoomID() string {
	for i := 0; i < 100; i++ {
		id := fmt.Sprintf("MF-%04d", s.rng.Intn(10000))
		if _, exists := s.store.MafiaRoom(id); !exists {
			return id
		}
	}
	return fmt.Sprintf("MF-%d", time.Now().Unix()%100000)
}

func normalizeMafiaSettings(settings MafiaRoomSettings) MafiaRoomSettings {
	defaults := defaultMafiaRoomSettings()
	if settings.MinPlayers <= 0 {
		settings.MinPlayers = defaults.MinPlayers
	}
	if settings.MaxPlayers <= 0 {
		settings.MaxPlayers = defaults.MaxPlayers
	}
	settings.MinPlayers = clampInt(settings.MinPlayers, mafiaMinPlayersLimit, mafiaMaxPlayersLimit)
	settings.MaxPlayers = clampInt(settings.MaxPlayers, mafiaMinPlayersLimit, mafiaMaxPlayersLimit)
	if settings.MaxPlayers < settings.MinPlayers {
		settings.MaxPlayers = settings.MinPlayers
	}
	if settings.RevealPunishmentMode == "" {
		settings.RevealPunishmentMode = defaults.RevealPunishmentMode
	}
	if settings.RoleRevealPolicy == "" {
		settings.RoleRevealPolicy = defaults.RoleRevealPolicy
	}
	if settings.MafiaChatMode == "" {
		settings.MafiaChatMode = defaults.MafiaChatMode
	}
	if settings.DayTimeSeconds <= 0 {
		settings.DayTimeSeconds = defaults.DayTimeSeconds
	}
	if settings.NightTimeSeconds <= 0 {
		settings.NightTimeSeconds = defaults.NightTimeSeconds
	}
	if settings.DayDiscussionSeconds <= 0 {
		settings.DayDiscussionSeconds = settings.DayTimeSeconds
	}
	if settings.NightActionSeconds <= 0 {
		settings.NightActionSeconds = settings.NightTimeSeconds
	}
	if settings.VotingSeconds <= 0 {
		settings.VotingSeconds = defaults.VotingSeconds
	}
	if settings.PhaseCardSeconds <= 0 {
		settings.PhaseCardSeconds = defaults.PhaseCardSeconds
	}
	if strings.TrimSpace(settings.NightKillMessage) == "" {
		settings.NightKillMessage = defaults.NightKillMessage
	}
	settings.NightKillMessage = cleanMafiaRoomMessageTemplate(settings.NightKillMessage)
	if settings.RatingMode != MafiaRatingUnranked {
		settings.RatingMode = MafiaRatingRanked
	}
	settings.DayDiscussionSeconds = clampInt(settings.DayDiscussionSeconds, 60, 600)
	settings.NightActionSeconds = clampInt(settings.NightActionSeconds, 30, 300)
	settings.VotingSeconds = clampInt(settings.VotingSeconds, 30, 180)
	settings.PhaseCardSeconds = clampInt(settings.PhaseCardSeconds, 2, 10)
	settings.DayTimeSeconds = settings.DayDiscussionSeconds
	settings.NightTimeSeconds = settings.NightActionSeconds
	if !settings.FeaturesInitialized {

		settings.EnablePostGameAIReport = defaults.EnablePostGameAIReport
		settings.EnableAchievements = defaults.EnableAchievements
		settings.EnableWills = defaults.EnableWills
		settings.FeaturesInitialized = true
	}
	if settings.TieMode == "" {
		settings.TieMode = defaults.TieMode
	}
	if settings.GameBreakingLeakAction == "" {
		settings.GameBreakingLeakAction = defaults.GameBreakingLeakAction
	}
	return settings
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func defaultMafiaNightKillMessage() string {
	return "☀️ Наступает утро.\n\nГород просыпается медленно: воздух тяжёлый, улицы молчат, а Лайя сверяет список живых без лишней нежности.\n\nПогиб: {player}.\n\nОбсуждение открыто. Теперь каждое слово может стать уликой, а каждая пауза — причиной подозревать."
}

func cleanMafiaRoomMessageTemplate(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	lines := strings.Split(value, "\n")
	out := make([]string, 0, len(lines))
	blank := false
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		isBlank := strings.TrimSpace(line) == ""
		if isBlank && blank {
			continue
		}
		out = append(out, line)
		blank = isBlank
	}
	value = strings.TrimSpace(strings.Join(out, "\n"))
	runes := []rune(value)
	if len(runes) > 700 {
		value = string(runes[:700])
	}
	return value
}

func assignMafiaRoles(players []MafiaPlayer, rng *rand.Rand) {
	roles := mafiaRolesForCount(len(players))
	order := rng.Perm(len(players))
	aliases := append([]string(nil), mafiaAliases...)
	for i, playerIndex := range order {
		players[playerIndex].Role = roles[i]
		players[playerIndex].Alive = true
		players[playerIndex].WarnedForReveal = false
		players[playerIndex].NightActionDone = false
		if roles[i] == MafiaRoleMafia {
			players[playerIndex].MafiaAlias = aliases[0]
			aliases = aliases[1:]
		} else {
			players[playerIndex].MafiaAlias = ""
		}
	}
}

func mafiaRolesForCount(count int) []MafiaRole {
	mafiaCount := 1
	if count >= 9 {
		mafiaCount = 3
	} else if count >= 6 {
		mafiaCount = 2
	}
	roles := make([]MafiaRole, 0, count)
	for i := 0; i < mafiaCount; i++ {
		roles = append(roles, MafiaRoleMafia)
	}
	if count >= 4 {
		roles = append(roles, MafiaRoleDoctor, MafiaRoleDetective)
	}
	for len(roles) < count {
		roles = append(roles, MafiaRoleCivilian)
	}
	return roles
}

var mafiaAliases = []string{"Don", "Shade", "Knife", "Ghost", "Raven", "Mask", "Whisper", "Ash", "Nero", "Viper"}

var mafiaTestBotProfiles = []struct {
	Name      string
	Backstory string
}{
	{Name: "Арчибальд Тестов", Backstory: "Появляется в городе только тогда, когда Лайе нужно проверить механику."},
	{Name: "София Заглушка", Backstory: "Умеет молчать так убедительно, что её зовут в каждую тестовую партию."},
	{Name: "Виктор Макет", Backstory: "Бывший статист. Считает, что стоять в списке игроков тоже важная работа."},
	{Name: "Марта Симулянт", Backstory: "Пришла проверить кнопки и случайно оказалась подозрительной."},
	{Name: "Григорий Демо", Backstory: "Говорит, что он настоящий игрок, но паспорт нарисован карандашом."},
	{Name: "Лада Проверкина", Backstory: "Записывает каждую странность партии в маленький блокнот."},
}

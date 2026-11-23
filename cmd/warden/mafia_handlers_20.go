package main

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

func (a *App) announceMafiaGameStart(room MafiaRoom) {
	a.syncMafiaGroupPlayerTitles(room)
	a.syncMafiaGroupRestrictions(room)
	text := a.mafiaNarration("game_started", map[string]interface{}{
		"room":         room.ID,
		"playersCount": len(room.Players),
		"host":         "Лайя",
		"autoHost":     true,
	}, "🎭 Игра начинается.\n\nЛайя ведёт партию сама: раздаёт роли, следит за фазами и временем. Проверьте личные сообщения.")
	a.sendMafiaGroup(room, mafiaPhaseMessage("СТАРТ ИГРЫ", text), nil)
	for _, player := range room.Players {
		a.sendMafiaGroupAccessIfMissing(player.TelegramID, room)
		a.sendMafiaDM(player.TelegramID, mafiaRoleText(player), mafiaRoomMarkup(room, player.TelegramID))
	}
	a.announceMafiaNight(room)
}

func (a *App) announceMafiaNight(room MafiaRoom) {
	a.syncMafiaGroupRestrictions(room)
	text := a.mafiaNarration("night_started", map[string]interface{}{
		"room":              room.ID,
		"day":               room.DayNumber,
		"alivePlayersCount": mafiaAlivePlayersCount(room),
	}, "🌙 Лайя гасит свет.\n\nНочь началась. Те, кому есть что скрывать, уже получили свои кнопки.")
	a.sendMafiaGroup(room, mafiaPhaseMessage("НОЧЬ", text+"\n\n"+mafiaPhaseTimerText(room, time.Now().UTC())+"\n\n"+mafiaRosterText(room)), nil)
	for _, player := range room.Players {
		if !player.Alive {
			continue
		}
		switch player.Role {
		case MafiaRoleMafia:
			a.sendMafiaDM(player.TelegramID, "🌙 Ночь началась.\n\nВыберите жертву:", mafiaRoomMarkup(room, player.TelegramID))
		case MafiaRoleDoctor:
			a.sendMafiaDM(player.TelegramID, "💊 Кого лечить этой ночью?", mafiaRoomMarkup(room, player.TelegramID))
		case MafiaRoleDetective:
			a.sendMafiaDM(player.TelegramID, "🔎 Кого проверить этой ночью?", mafiaRoomMarkup(room, player.TelegramID))
		}
	}
	a.announceMafiaAIBotKillRecommendations(room)
}

func (a *App) announceMafiaAIBotKillRecommendations(room MafiaRoom) {
	if a == nil {
		return
	}
	updated, messages, recipients, err := a.mafiaService().MaybeAddAIBotKillRecommendations(room.ID, time.Now().UTC())
	if err != nil {
		if a.log != nil {
			a.log.Warn("mafia ai bot kill recommendation failed", zap.String("room", room.ID), zap.Error(err))
		}
		return
	}
	if len(messages) == 0 {
		return
	}
	for _, msg := range messages {
		a.deliverMafiaShadowMessage(updated.ID, msg, recipients)
	}
}

func (a *App) announceMafiaWill(room MafiaRoom, playerID int64) {
	text := mafiaPublishedWillText(room, playerID)
	if text == "" {
		return
	}
	a.sendMafiaGroup(room, text, nil)
}

func (a *App) notifyMafiaPlayerEliminated(room MafiaRoom, playerID int64, reason string) {
	if playerID == 0 {
		return
	}
	idx := mafiaRoomPlayerIndex(room, playerID)
	if idx < 0 || room.Players[idx].IsTestBot {
		return
	}
	name := mafiaRoomPlayerLabel(room, room.Players[idx])
	detail := "Город больше не слышит ваш голос."
	switch reason {
	case "night":
		detail = "Этой ночью мафия выбрала вас жертвой."
	case "vote":
		detail = "Город казнил вас на голосовании."
	case "role_reveal":
		detail = "Лайя исключила вас за нарушение правил раскрытия роли."
	}
	a.sendMafiaDM(playerID, mafiaEliminatedDMText(room, name, detail), mafiaRoomMarkup(room, playerID))
}

func mafiaEliminatedDMText(room MafiaRoom, playerName, detail string) string {
	return fmt.Sprintf("⚰️ ВЫ ВЫБЫЛИ ИЗ ПАРТИИ\n\n%s, вы мертвы в комнате %s.\n\n%s\n\nЧто теперь нельзя:\n— писать в общий чат партии;\n— голосовать;\n— делать ночные действия;\n— подсказывать живым игрокам.\n\nВы можете только наблюдать за партией. Если было сохранено завещание, Лайя опубликует его в общем чате.", playerName, room.ID, detail)
}

func (a *App) announceMafiaFinished(room MafiaRoom) {
	a.syncMafiaGroupRestrictions(room)
	a.sendMafiaGroup(room, mafiaFinishText(room), nil)
	if achievements := mafiaPostGameAchievementsText(room); achievements != "" {
		a.sendMafiaGroup(room, achievements, nil)
	}
	if report := a.mafiaPostGameReport(room); report != "" {
		a.sendMafiaGroup(room, report, nil)
	}
	if lobby, err := a.mafiaService().PrepareRematchLobby(room.ID, time.Now().UTC()); err == nil {
		a.syncMafiaGroupRestrictions(lobby)
		a.sendMafiaGroup(lobby, fmt.Sprintf("🎭 Комната %s сохранена как лобби.\n\nИгроки остались в комнате. Можно запускать следующую партию без нового подключения.", lobby.ID), nil)
	} else if a != nil && a.log != nil {
		a.log.Warn("prepare mafia rematch lobby failed", zap.String("room", room.ID), zap.Error(err))
	}
}

func (a *App) mafiaPostGameReport(room MafiaRoom) string {
	settings := normalizeMafiaSettings(room.Settings)
	if room.Status != MafiaStatusFinished || !settings.EnablePostGameAIReport {
		return ""
	}
	players := make([]map[string]string, 0, len(room.Players))
	for _, player := range room.Players {
		status := "жив"
		if !player.Alive {
			status = "мёртв"
		}
		players = append(players, map[string]string{
			"name":   mafiaRoomPlayerLabel(room, player),
			"role":   mafiaRoleLabel(player.Role),
			"status": status,
		})
	}
	winner := "город"
	if room.Winner == "mafia" {
		winner = "мафия"
	}
	fallback := fmt.Sprintf("Победитель: %s.\n\nПартия завершилась на %d-м дне. Роли раскрыты, итоги зафиксированы.", winner, room.DayNumber)
	report := a.mafiaNarration("post_game_report", map[string]interface{}{
		"room":          room.ID,
		"winner":        winner,
		"day":           room.DayNumber,
		"status":        "finished",
		"must_be_final": true,
		"players":       players,
	}, fallback)
	report = sanitizeMafiaPostGameReport(report, fallback)
	return mafiaPhaseMessage("ОТЧЁТ ЛАЙИ", report)
}

func sanitizeMafiaPostGameReport(report, fallback string) string {
	report = strings.TrimSpace(report)
	if report == "" {
		return fallback
	}
	normalized := normalizeFAQText(report)
	for _, forbidden := range []string{
		"игра еще не закончена",
		"игра ещё не закончена",
		"следующий день",
		"следующая ночь",
		"кто будет следующей жертвой",
		"будет следующей жертвой",
		"мафия пока ведет",
		"мафия пока ведёт",
		"игра продолжается",
		"партия продолжается",
		"день наступил",
		"наступил следующий день",
	} {
		if strings.Contains(normalized, forbidden) {
			return fallback
		}
	}
	report = strings.TrimPrefix(report, "🎭 Отчёт Лайи")
	report = strings.TrimPrefix(report, "Отчёт Лайи")
	report = strings.TrimSpace(report)
	if report == "" {
		return fallback
	}
	return report
}

package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func mafiaPhaseTimerText(room MafiaRoom, now time.Time) string {
	duration := mafiaPhaseDuration(room)
	if duration <= 0 {
		return ""
	}
	startedAt := room.PhaseStartedAt
	if startedAt <= 0 {
		startedAt = room.UpdatedAt
	}
	if startedAt <= 0 {
		return ""
	}
	deadline := time.Unix(startedAt, 0).UTC().Add(duration)
	remaining := deadline.Sub(now.UTC())
	if remaining < 0 {
		remaining = 0
	}
	label := "До автоперехода"
	hint := ""
	switch room.Status {
	case MafiaStatusNight:
		label = "До конца ночи"
		hint = "Ночь может закончиться раньше, если все активные роли сделали выбор."
	case MafiaStatusDayDiscussion:
		label = "До начала голосования"
	case MafiaStatusVoting:
		label = "До конца голосования"
		hint = "Голосование может закончиться раньше, если все живые игроки проголосуют."
	default:
		return ""
	}
	lines := []string{
		fmt.Sprintf("⏳ %s: %s", label, mafiaCountdown(remaining)),
		fmt.Sprintf("🕒 Автопереход: %s", deadline.In(time.Local).Format("15:04:05")),
	}
	if hint != "" {
		lines = append(lines, hint)
	}
	return strings.Join(lines, "\n")
}

func mafiaPhaseDuration(room MafiaRoom) time.Duration {
	settings := normalizeMafiaSettings(room.Settings)
	switch room.Status {
	case MafiaStatusNight:
		return time.Duration(settings.NightActionSeconds) * time.Second
	case MafiaStatusDayDiscussion:
		return time.Duration(settings.DayDiscussionSeconds) * time.Second
	case MafiaStatusVoting:
		return time.Duration(settings.VotingSeconds) * time.Second
	default:
		return 0
	}
}

func mafiaCountdown(d time.Duration) string {
	if d <= 0 {
		return "вот-вот"
	}
	total := int((d + time.Second - time.Nanosecond) / time.Second)
	minutes := total / 60
	seconds := total % 60
	if minutes >= 60 {
		hours := minutes / 60
		minutes = minutes % 60
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

func mafiaPhaseMessage(phase string, body string) string {
	phase = strings.TrimSpace(phase)
	body = strings.TrimSpace(body)
	if phase == "" {
		return body
	}
	line := "━━━━━━━━━━━━━━━━━━━━"
	header := line + "\n--- " + phase + " ---\n" + line
	if body == "" {
		return header
	}
	return header + "\n\n\n" + body
}

func mafiaVotingProgressText(room MafiaRoom, votes map[string]string) string {
	voters := make([]MafiaPlayer, 0, len(room.Players))
	voted := 0
	for _, player := range room.Players {
		target := strings.TrimSpace(votes[idKey(player.TelegramID)])
		if !player.Alive && target == "" {
			continue
		}
		voters = append(voters, player)
		if target != "" {
			voted++
		}
	}
	lines := []string{fmt.Sprintf("🗳 Прогресс голосования: %d/%d", voted, len(voters))}
	if timer := mafiaPhaseTimerText(room, time.Now().UTC()); timer != "" {
		lines = append(lines, "", timer)
	}
	if len(voters) == 0 {
		return strings.Join(lines, "\n")
	}
	if voted > 0 {
		lines = append(lines, "", "Счёт:")
		for _, line := range mafiaVotingTallyLines(room, votes) {
			lines = append(lines, line)
		}
	}
	lines = append(lines, "", "Кто за кого:")
	for _, voter := range voters {
		target := strings.TrimSpace(votes[idKey(voter.TelegramID)])
		if target == "" {
			lines = append(lines, fmt.Sprintf("• %s → ещё не голосовал", mafiaRoomPlayerLabelWithUsername(room, voter)))
			continue
		}
		lines = append(lines, fmt.Sprintf("• %s → %s", mafiaRoomPlayerLabelWithUsername(room, voter), mafiaVoteTargetLabel(room, target)))
	}
	return strings.Join(lines, "\n")
}

func mafiaVotingTallyLines(room MafiaRoom, votes map[string]string) []string {
	counts := map[string]int{}
	for _, target := range votes {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}
		counts[target]++
	}
	if len(counts) == 0 {
		return []string{"• пока нет голосов"}
	}
	targets := make([]string, 0, len(counts))
	for target := range counts {
		targets = append(targets, target)
	}
	sort.Slice(targets, func(i, j int) bool {
		if counts[targets[i]] == counts[targets[j]] {
			return mafiaVoteTargetLabel(room, targets[i]) < mafiaVoteTargetLabel(room, targets[j])
		}
		return counts[targets[i]] > counts[targets[j]]
	})
	lines := make([]string, 0, len(targets))
	for _, target := range targets {
		lines = append(lines, fmt.Sprintf("• %s — %d", mafiaVoteTargetLabel(room, target), counts[target]))
	}
	return lines
}

func mafiaVoteTargetLabel(room MafiaRoom, target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return "ещё не голосовал"
	}
	if target == "skip" {
		return "Пропуск"
	}
	targetID, err := strconvParseInt64(target)
	if err != nil {
		return "неизвестный игрок"
	}
	return mafiaPlayerPublicName(room, targetID)
}

func mafiaVotingResultText(room MafiaRoom, result MafiaVoteResult) string {
	if result.ExecutedID != 0 {
		return "🗳 Голосование завершено.\n\nКазнён: " + mafiaPlayerPublicName(room, result.ExecutedID) + ".\n\n" + mafiaVotingProgressText(room, result.Votes) + "\n\n" + mafiaRosterText(room)
	}
	if result.Tie {
		return "🗳 Голосование завершено.\n\nГород не смог прийти к решению. Сегодня никто не казнён.\n\n" + mafiaVotingProgressText(room, result.Votes) + "\n\n" + mafiaRosterText(room)
	}
	return "🗳 Голосование завершено.\n\nСегодня никто не казнён.\n\n" + mafiaVotingProgressText(room, result.Votes) + "\n\n" + mafiaRosterText(room)
}

func mafiaFinishText(room MafiaRoom) string {
	winner := "Победил город."
	if room.Winner == "mafia" {
		winner = "Победила мафия."
	}
	return mafiaPhaseMessage("ФИНАЛ", "🎭 Игра завершена.\n\n"+winner+"\n\n"+mafiaRolesRevealText(room))
}

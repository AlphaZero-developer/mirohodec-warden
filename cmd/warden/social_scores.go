package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type socialBadge struct {
	Emoji string
	Title string
	Hint  string
}

type activityRow struct {
	Member   MemberRecord
	Karma    int64
	Warnings int
	Score    int64
	Rank     int
}

const (
	defaultKarmaMessagesPerPoint = 10
	defaultKarmaDailyLimit       = 5
)

func karmaBadge(score int64) socialBadge {
	switch {
	case score >= 100:
		return socialBadge{"👑", "легенда чата", "репутация уже сама открывает дверь"}
	case score >= 50:
		return socialBadge{"💎", "золотой запас", "редкий случай, когда чат почти доволен"}
	case score >= 20:
		return socialBadge{"🔥", "любимчик хаоса", "плюсы идут уверенно"}
	case score >= 5:
		return socialBadge{"✨", "свой человек", "репутация светится без прожектора"}
	case score >= 0:
		return socialBadge{"🌱", "нейтрально", "почва нормальная, растить есть куда"}
	case score >= -4:
		return socialBadge{"🌫️", "на испытательном", "Лайя уже прищурилась"}
	case score >= -19:
		return socialBadge{"🧯", "скользкая дорожка", "репутацию пора тушить"}
	default:
		return socialBadge{"🕳️", "репутационный подвал", "снизу постучали, это ты"}
	}
}

func activityBadge(score int64) socialBadge {
	switch {
	case score >= 5000:
		return socialBadge{"👑", "двигатель чата", "без тебя тут подозрительно тихо"}
	case score >= 1500:
		return socialBadge{"⚡", "ядро чата", "стабильно держишь темп"}
	case score >= 500:
		return socialBadge{"🔥", "частый гость", "следы в чате уже не отмываются"}
	case score >= 100:
		return socialBadge{"🌊", "заметный участник", "тебя уже видно на радаре"}
	case score >= 20:
		return socialBadge{"🌱", "просыпается", "активность набирает форму"}
	default:
		return socialBadge{"💤", "тихий режим", "почти ниндзя, только без шума"}
	}
}

func activityScore(rec MemberRecord, karma int64, activeWarnings int, now time.Time) int64 {
	today := messagesTodayAt(rec, now)
	positiveKarma := karma
	if positiveKarma < 0 {
		positiveKarma = 0
	}
	score := rec.Messages + today*4 + positiveKarma*7 - int64(activeWarnings)*25
	if score < 0 {
		return 0
	}
	return score
}

func messagesTodayAt(rec MemberRecord, now time.Time) int64 {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if rec.ActivityDay != now.UTC().Format("2006-01-02") {
		return 0
	}
	return rec.MessagesToday
}

func (a *App) activityRank(chatID, userID int64, now time.Time) int {
	for _, row := range a.topActivityRows(chatID, 0, now) {
		if row.Member.ID == userID {
			return row.Rank
		}
	}
	return 0
}

func (a *App) topActivityRows(chatID int64, limit int, now time.Time) []activityRow {
	if a == nil || a.store == nil {
		return nil
	}
	members := a.store.Members(chatID)
	rows := make([]activityRow, 0, len(members))
	for _, member := range members {
		if member.ID == 0 || member.IsBot {
			continue
		}
		karma := a.store.Karma(chatID, member.ID)
		warnings := a.store.ActiveWarningCount(chatID, member.ID, now)
		rows = append(rows, activityRow{
			Member:   member,
			Karma:    karma,
			Warnings: warnings,
			Score:    activityScore(member, karma, warnings, now),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Score == rows[j].Score {
			if rows[i].Member.Messages == rows[j].Member.Messages {
				return rows[i].Member.LastMessage > rows[j].Member.LastMessage
			}
			return rows[i].Member.Messages > rows[j].Member.Messages
		}
		return rows[i].Score > rows[j].Score
	})
	for i := range rows {
		rows[i].Rank = i + 1
	}
	if limit > 0 && len(rows) > limit {
		return rows[:limit]
	}
	return rows
}

func socialRankLabel(rank int) string {
	if rank <= 0 {
		return "вне топа"
	}
	return fmt.Sprintf("#%d", rank)
}

func medalEmoji(rank int) string {
	switch rank {
	case 1:
		return "🥇"
	case 2:
		return "🥈"
	case 3:
		return "🥉"
	default:
		return "▫️"
	}
}

func signedScore(value int64) string {
	if value > 0 {
		return fmt.Sprintf("+%d", value)
	}
	return fmt.Sprintf("%d", value)
}

func displayKarmaName(name string, userID int64) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	return fmt.Sprintf("user %d", userID)
}

func karmaAutoProgressLine(today int64, cfg KarmaConfig) string {
	if !cfg.Enabled {
		return "🤖 Авто-карма: выключена."
	}
	messagesPerPoint := cfg.ActivityMessagesPerPoint
	if messagesPerPoint <= 0 {
		messagesPerPoint = defaultKarmaMessagesPerPoint
	}
	dailyLimit := cfg.ActivityDailyLimit
	if dailyLimit <= 0 {
		dailyLimit = defaultKarmaDailyLimit
	}
	potential := int(today) / messagesPerPoint
	if potential > dailyLimit {
		potential = dailyLimit
	}
	if potential >= dailyLimit {
		return fmt.Sprintf("🤖 Авто-карма сегодня: %d/%d, дневной лимит взят.", potential, dailyLimit)
	}
	nextAt := int64((potential + 1) * messagesPerPoint)
	remaining := nextAt - today
	if remaining < 0 {
		remaining = 0
	}
	return fmt.Sprintf("🤖 Авто-карма сегодня: %d/%d, до следующего +1: %d сообщ.", potential, dailyLimit, remaining)
}

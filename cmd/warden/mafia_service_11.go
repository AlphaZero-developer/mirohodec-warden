package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

func mafiaRoomPlayerIndex(room MafiaRoom, userID int64) int {
	for i, player := range room.Players {
		if player.TelegramID == userID {
			return i
		}
	}
	return -1
}

func mafiaJoinRequestIndex(room MafiaRoom, userID int64, status MafiaJoinRequestStatus) int {
	for i, req := range room.JoinRequests {
		if req.TelegramID == userID && (status == "" || req.Status == status) {
			return i
		}
	}
	return -1
}

func normalizeMafiaRoomRecruitment(room MafiaRoom) MafiaRoom {
	if room.RecruitmentStatus == "" {
		room.RecruitmentStatus = MafiaRecruitmentClosed
	}
	if room.RecruitmentMode == "" {
		room.RecruitmentMode = MafiaRecruitmentRequests
	}
	if room.JoinRequests == nil {
		room.JoinRequests = []MafiaJoinRequest{}
	}
	return room
}

func int64Set(values []int64) map[int64]struct{} {
	out := make(map[int64]struct{}, len(values))
	for _, value := range values {
		if value != 0 {
			out[value] = struct{}{}
		}
	}
	return out
}

func mafiaMajorityTarget(votes map[string]int64, rng *rand.Rand) int64 {
	counts := map[int64]int{}
	for _, target := range votes {
		if target != 0 {
			counts[target]++
		}
	}
	if len(counts) == 0 {
		return 0
	}
	maxVotes := 0
	var leaders []int64
	for target, count := range counts {
		if count > maxVotes {
			maxVotes = count
			leaders = []int64{target}
		} else if count == maxVotes {
			leaders = append(leaders, target)
		}
	}
	sort.Slice(leaders, func(i, j int) bool { return leaders[i] < leaders[j] })
	if len(leaders) == 1 {
		return leaders[0]
	}
	return leaders[rng.Intn(len(leaders))]
}

func mafiaVoteLeaders(votes map[string]string) []string {
	counts := map[string]int{}
	for _, target := range votes {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}
		counts[target]++
	}
	maxVotes := 0
	var leaders []string
	for target, count := range counts {
		if count > maxVotes {
			maxVotes = count
			leaders = []string{target}
		} else if count == maxVotes {
			leaders = append(leaders, target)
		}
	}
	sort.Strings(leaders)
	return leaders
}

func mafiaWinner(room MafiaRoom) string {
	mafiaAlive := 0
	nonMafiaAlive := 0
	for _, player := range room.Players {
		if !player.Alive {
			continue
		}
		if isMafiaTeam(player.Role) {
			mafiaAlive++
		} else {
			nonMafiaAlive++
		}
	}
	switch {
	case mafiaAlive == 0:
		return "civilian"
	case mafiaAlive >= nonMafiaAlive:
		return "mafia"
	default:
		return ""
	}
}

func strconvParseInt64(value string) (int64, error) {
	var id int64
	_, err := fmt.Sscan(strings.TrimSpace(value), &id)
	return id, err
}

func mustParseInt64(value string) int64 {
	id, _ := strconvParseInt64(value)
	return id
}

func mustParseInt(value string) int {
	var out int
	_, _ = fmt.Sscan(strings.TrimSpace(value), &out)
	return out
}

func isMafiaTeam(role MafiaRole) bool {
	return role == MafiaRoleMafia || role == MafiaRoleDon || role == MafiaRoleLawyer
}

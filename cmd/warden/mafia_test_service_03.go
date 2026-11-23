package main

import (
	"strings"

	tele "gopkg.in/telebot.v3"
)

func mafiaPlayers(room MafiaRoom) []MafiaPlayer {
	var out []MafiaPlayer
	for _, player := range room.Players {
		if player.Role == MafiaRoleMafia {
			out = append(out, player)
		}
	}
	return out
}

func firstRole(room MafiaRoom, role MafiaRole) MafiaPlayer {
	for _, player := range room.Players {
		if player.Role == role {
			return player
		}
	}
	return MafiaPlayer{}
}

func mafiaMarkupContains(markup *tele.ReplyMarkup, text string) bool {
	if markup == nil {
		return false
	}
	for _, row := range markup.InlineKeyboard {
		for _, button := range row {
			if strings.Contains(button.Text, text) || strings.Contains(button.Data, text) || strings.Contains(button.Unique, text) {
				return true
			}
		}
	}
	return false
}

package main

import (
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v3"
)

func layaMediaLine(parts []string, msg *tele.Message) string {
	if msg != nil && strings.TrimSpace(msg.Caption) != "" {
		parts = append(parts, "есть подпись")
	}
	if msg != nil && strings.TrimSpace(msg.AlbumID) != "" {
		parts = append(parts, "album_id="+msg.AlbumID)
	}
	if msg != nil && msg.HasMediaSpoiler {
		parts = append(parts, "spoiler")
	}
	return strings.Join(parts, "; ")
}

func layaMediaDuration(seconds int) string {
	if seconds <= 0 {
		return "0s"
	}
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	return fmt.Sprintf("%dm%02ds", seconds/60, seconds%60)
}

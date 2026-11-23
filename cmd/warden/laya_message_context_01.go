package main

import (
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v3"
)

func layaMessageUserText(msg *tele.Message) string {
	if text := strings.TrimSpace(messageText(msg)); text != "" {
		return text
	}
	if brief := layaMessageBrief(msg); brief != "" {
		return "[" + brief + "]"
	}
	return ""
}

func layaMessageHasUserContent(msg *tele.Message) bool {
	return strings.TrimSpace(messageText(msg)) != "" || layaMessageBrief(msg) != ""
}

func layaMessageMemoryText(msg *tele.Message) string {
	text := strings.TrimSpace(messageText(msg))
	brief := layaMessageBrief(msg)
	switch {
	case text != "" && brief != "":
		return brief + "; текст/подпись: " + text
	case text != "":
		return text
	default:
		return brief
	}
}

func layaMessageBrief(msg *tele.Message) string {
	lines := layaMessageAttachmentLines(msg)
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "; ")
}

func layaMessageAttachmentLines(msg *tele.Message) []string {
	if msg == nil {
		return nil
	}
	var lines []string
	if msg.Photo != nil {
		parts := []string{"фото/картинка"}
		if msg.Photo.Width > 0 && msg.Photo.Height > 0 {
			parts = append(parts, fmt.Sprintf("%dx%d", msg.Photo.Width, msg.Photo.Height))
		}
		lines = append(lines, layaMediaLine(parts, msg))
	}
	if msg.Voice != nil {
		parts := []string{"голосовое сообщение"}
		if msg.Voice.Duration > 0 {
			parts = append(parts, fmt.Sprintf("длительность %s", layaMediaDuration(msg.Voice.Duration)))
		}
		if strings.TrimSpace(msg.Voice.MIME) != "" {
			parts = append(parts, "mime="+msg.Voice.MIME)
		}
		lines = append(lines, layaMediaLine(parts, msg))
	}
	if msg.VideoNote != nil {
		parts := []string{"кружок/video note"}
		if msg.VideoNote.Duration > 0 {
			parts = append(parts, fmt.Sprintf("длительность %s", layaMediaDuration(msg.VideoNote.Duration)))
		}
		if msg.VideoNote.Length > 0 {
			parts = append(parts, fmt.Sprintf("размер %d", msg.VideoNote.Length))
		}
		lines = append(lines, layaMediaLine(parts, msg))
	}
	if msg.Video != nil {
		parts := []string{"видео"}
		if msg.Video.Width > 0 && msg.Video.Height > 0 {
			parts = append(parts, fmt.Sprintf("%dx%d", msg.Video.Width, msg.Video.Height))
		}
		if msg.Video.Duration > 0 {
			parts = append(parts, fmt.Sprintf("длительность %s", layaMediaDuration(msg.Video.Duration)))
		}
		if strings.TrimSpace(msg.Video.FileName) != "" {
			parts = append(parts, "file="+msg.Video.FileName)
		}
		if strings.TrimSpace(msg.Video.MIME) != "" {
			parts = append(parts, "mime="+msg.Video.MIME)
		}
		lines = append(lines, layaMediaLine(parts, msg))
	}
	if msg.Animation != nil {
		parts := []string{"анимация/gif"}
		if msg.Animation.Width > 0 && msg.Animation.Height > 0 {
			parts = append(parts, fmt.Sprintf("%dx%d", msg.Animation.Width, msg.Animation.Height))
		}
		if msg.Animation.Duration > 0 {
			parts = append(parts, fmt.Sprintf("длительность %s", layaMediaDuration(msg.Animation.Duration)))
		}
		if strings.TrimSpace(msg.Animation.FileName) != "" {
			parts = append(parts, "file="+msg.Animation.FileName)
		}
		lines = append(lines, layaMediaLine(parts, msg))
	}
	if msg.Audio != nil {
		parts := []string{"аудиофайл"}
		if msg.Audio.Duration > 0 {
			parts = append(parts, fmt.Sprintf("длительность %s", layaMediaDuration(msg.Audio.Duration)))
		}
		if strings.TrimSpace(msg.Audio.Title) != "" {
			parts = append(parts, "title="+msg.Audio.Title)
		}
		if strings.TrimSpace(msg.Audio.Performer) != "" {
			parts = append(parts, "performer="+msg.Audio.Performer)
		}
		if strings.TrimSpace(msg.Audio.FileName) != "" {
			parts = append(parts, "file="+msg.Audio.FileName)
		}
		lines = append(lines, layaMediaLine(parts, msg))
	}
	if msg.Document != nil {
		parts := []string{"документ/файл"}
		if strings.TrimSpace(msg.Document.FileName) != "" {
			parts = append(parts, "file="+msg.Document.FileName)
		}
		if strings.TrimSpace(msg.Document.MIME) != "" {
			parts = append(parts, "mime="+msg.Document.MIME)
		}
		if msg.Document.FileSize > 0 {
			parts = append(parts, fmt.Sprintf("size=%d bytes", msg.Document.FileSize))
		}
		lines = append(lines, layaMediaLine(parts, msg))
	}
	if msg.Sticker != nil {
		parts := []string{"стикер"}
		if strings.TrimSpace(msg.Sticker.Emoji) != "" {
			parts = append(parts, "emoji="+msg.Sticker.Emoji)
		}
		if msg.Sticker.Animated {
			parts = append(parts, "анимированный")
		}
		if msg.Sticker.Video {
			parts = append(parts, "video")
		}
		if strings.TrimSpace(msg.Sticker.SetName) != "" {
			parts = append(parts, "set="+msg.Sticker.SetName)
		}
		lines = append(lines, strings.Join(parts, "; "))
	}
	if msg.Contact != nil {
		parts := []string{"контакт"}
		name := strings.TrimSpace(strings.TrimSpace(msg.Contact.FirstName + " " + msg.Contact.LastName))
		if name != "" {
			parts = append(parts, "name="+name)
		}
		if msg.Contact.UserID != 0 {
			parts = append(parts, fmt.Sprintf("telegram_user_id=%d", msg.Contact.UserID))
		}
		lines = append(lines, strings.Join(parts, "; "))
	}
	if msg.Location != nil {
		lines = append(lines, fmt.Sprintf("геолокация: lat=%.5f; lng=%.5f", msg.Location.Lat, msg.Location.Lng))
	}
	if msg.Venue != nil {
		parts := []string{"место/venue"}
		if strings.TrimSpace(msg.Venue.Title) != "" {
			parts = append(parts, "title="+msg.Venue.Title)
		}
		if strings.TrimSpace(msg.Venue.Address) != "" {
			parts = append(parts, "address="+msg.Venue.Address)
		}
		lines = append(lines, strings.Join(parts, "; "))
	}
	if msg.Poll != nil {
		parts := []string{"опрос"}
		if strings.TrimSpace(msg.Poll.Question) != "" {
			parts = append(parts, "question="+shortText(msg.Poll.Question, 160))
		}
		if len(msg.Poll.Options) > 0 {
			parts = append(parts, fmt.Sprintf("options=%d", len(msg.Poll.Options)))
		}
		lines = append(lines, strings.Join(parts, "; "))
	}
	if msg.Dice != nil {
		lines = append(lines, fmt.Sprintf("кубик/emoji: %s; значение=%d", msg.Dice.Type, msg.Dice.Value))
	}
	return lines
}

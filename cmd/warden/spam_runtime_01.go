package main

import (
	"regexp"
	"time"
)

const (
	spamKindText    = "text"
	spamKindMedia   = "media"
	spamKindSticker = "sticker"
	spamKindLink    = "link"
	spamKindMention = "mention"
	spamKindForward = "forward"
	spamKindOther   = "other"

	spamConfidenceLow  = "low"
	spamConfidenceHigh = "high"
)

var (
	spamURLPattern     = regexp.MustCompile(`(?i)\b(?:https?://|www\.|t\.me/|telegram\.me/|(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63})(?:/[^\s<>()]*)?`)
	spamMentionPattern = regexp.MustCompile(`(?i)@[a-z0-9_]{5,32}`)
)

type spamEvent struct {
	At        time.Time
	Kind      string
	Signature string
	AlbumID   string
	Count     int
}

type spamStrike struct {
	At         time.Time
	Reason     string
	Confidence string
}

type spamLogEntry struct {
	At         time.Time
	ChatID     int64
	UserID     int64
	UserName   string
	Kind       string
	Confidence string
	Reason     string
	Action     string
	Counts     string
}

type spamBucket struct {
	Events     []spamEvent
	Strikes    []spamStrike
	Logs       []spamLogEntry
	LastAction time.Time
	LastMute   time.Time
}

type spamLimits struct {
	TextWindow          time.Duration
	MaxTextMessages     int
	MediaWindow         time.Duration
	MaxMedia            int
	StickerWindow       time.Duration
	MaxStickers         int
	RepeatWindow        time.Duration
	MaxRepeatedText     int
	MaxRepeatedStickers int
	MinRepeatedTextLen  int
	LinkWindow          time.Duration
	MaxLinks            int
	MaxRepeatedLinks    int
	MaxLinksPerMessage  int
	MentionWindow       time.Duration
	MaxMentions         int
	MaxMentionsPerMsg   int
	ForwardWindow       time.Duration
	MaxForwards         int
	ActionCooldown      time.Duration
	StrikeWindow        time.Duration
	MuteAfterStrikes    int
	LogLimit            int
}

type spamStatusSnapshot struct {
	Text       int
	Media      int
	Stickers   int
	Links      int
	Mentions   int
	Forwards   int
	Repeated   int
	Strikes    int
	LastAction string
	LastReason string
	LastAt     time.Time
}

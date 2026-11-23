package main

import (
	"testing"

	tele "gopkg.in/telebot.v3"
)

func TestIsDiscussionAutoReplyPost(t *testing.T) {
	app := &App{}
	app.applyConfig(Config{Security: SecurityConfig{AdminIDs: []int64{42}}})
	chat := &tele.Chat{ID: -1001, Type: tele.ChatSuperGroup}

	tests := []struct {
		name   string
		msg    *tele.Message
		sender *tele.User
		want   bool
	}{
		{
			name: "automatic forward",
			msg:  &tele.Message{ThreadID: 7, AutomaticForward: true, Text: "post"},
			want: true,
		},
		{
			name: "automatic forward without forum topic",
			msg:  &tele.Message{ID: 77, AutomaticForward: true, Text: "post"},
			want: true,
		},
		{
			name:   "automatic forward from anonymous bot",
			msg:    &tele.Message{ThreadID: 7, AutomaticForward: true, Text: "post"},
			sender: &tele.User{ID: 1087968824, Username: "GroupAnonymousBot", IsBot: true},
			want:   true,
		},
		{
			name: "channel sender",
			msg:  &tele.Message{ThreadID: 7, SenderChat: &tele.Chat{ID: -1002, Type: tele.ChatChannel}, Text: "post"},
			want: true,
		},
		{
			name:   "channel sender with bot user",
			msg:    &tele.Message{ThreadID: 7, SenderChat: &tele.Chat{ID: -1002, Type: tele.ChatChannel}, Text: "post"},
			sender: &tele.User{ID: 1087968824, Username: "GroupAnonymousBot", IsBot: true},
			want:   true,
		},
		{
			name:   "configured admin",
			msg:    &tele.Message{ThreadID: 7, Text: "post"},
			sender: &tele.User{ID: 42, FirstName: "admin"},
			want:   true,
		},
		{
			name: "reply is ignored",
			msg:  &tele.Message{ThreadID: 7, AutomaticForward: true, ReplyTo: &tele.Message{ID: 1}, Text: "post"},
			want: false,
		},
		{
			name: "command is ignored",
			msg:  &tele.Message{ThreadID: 7, AutomaticForward: true, Text: "/start"},
			want: false,
		},
		{
			name:   "normal user is ignored",
			msg:    &tele.Message{ThreadID: 7, Text: "hello"},
			sender: &tele.User{ID: 77, FirstName: "user"},
			want:   false,
		},
		{
			name:   "normal bot is ignored",
			msg:    &tele.Message{ThreadID: 7, Text: "hello"},
			sender: &tele.User{ID: 77, FirstName: "bot", IsBot: true},
			want:   false,
		},
		{
			name: "message without marker is ignored",
			msg:  &tele.Message{AutomaticForward: true, Text: "post"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := app.isDiscussionAutoReplyPost(chat, tt.msg, tt.sender); got != tt.want {
				t.Fatalf("isDiscussionAutoReplyPost() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestDiscussionAutoReplyTargetChat(t *testing.T) {
	tests := []struct {
		name string
		chat *tele.Chat
		cfg  DiscussionAutoReplyConfig
		want bool
	}{
		{name: "any managed chat by default", chat: &tele.Chat{ID: -1001}, cfg: DiscussionAutoReplyConfig{}, want: true},
		{name: "configured chat", chat: &tele.Chat{ID: -1001}, cfg: DiscussionAutoReplyConfig{ChatID: -1001}, want: true},
		{name: "other chat ignored", chat: &tele.Chat{ID: -1002}, cfg: DiscussionAutoReplyConfig{ChatID: -1001}, want: false},
		{name: "nil chat ignored", cfg: DiscussionAutoReplyConfig{ChatID: -1001}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDiscussionAutoReplyTargetChat(tt.chat, tt.cfg); got != tt.want {
				t.Fatalf("isDiscussionAutoReplyTargetChat() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestIsDiscussionAutoReplyPostSourceFilter(t *testing.T) {
	app := &App{}
	app.applyConfig(Config{Discussion: DiscussionAutoReplyConfig{SourceChatID: -1002}})
	chat := &tele.Chat{ID: -1001, Type: tele.ChatSuperGroup}

	if !app.isDiscussionAutoReplyPost(chat, &tele.Message{
		ThreadID:         7,
		AutomaticForward: true,
		OriginalChat:     &tele.Chat{ID: -1002, Type: tele.ChatChannel},
		Text:             "post",
	}, nil) {
		t.Fatal("expected configured source channel post to pass")
	}
	if !app.isDiscussionAutoReplyPost(chat, &tele.Message{
		ID:               77,
		AutomaticForward: true,
		Origin:           &tele.MessageOrigin{Chat: &tele.Chat{ID: -1002, Type: tele.ChatChannel}},
		Text:             "post",
	}, nil) {
		t.Fatal("expected configured source channel post from forward_origin.chat to pass")
	}
	if app.isDiscussionAutoReplyPost(chat, &tele.Message{
		ThreadID:         7,
		AutomaticForward: true,
		OriginalChat:     &tele.Chat{ID: -1003, Type: tele.ChatChannel},
		Text:             "post",
	}, nil) {
		t.Fatal("other source channel should be ignored")
	}
	if app.isDiscussionAutoReplyPost(chat, &tele.Message{ThreadID: 7, AutomaticForward: true, Text: "post"}, nil) {
		t.Fatal("source filter should reject posts without source chat metadata")
	}
}

func TestDiscussionAutoReplyMarkerAndSendOptions(t *testing.T) {
	topicMsg := &tele.Message{ID: 10, ThreadID: 7}
	if got := discussionAutoReplyMarkerID(topicMsg); got != 7 {
		t.Fatalf("topic marker = %d, want 7", got)
	}
	if opts := discussionAutoReplySendOptions(topicMsg); opts.ThreadID != 7 || opts.ReplyTo != nil {
		t.Fatalf("topic send opts = %#v", opts)
	}

	rootMsg := &tele.Message{ID: 77}
	if got := discussionAutoReplyMarkerID(rootMsg); got != 77 {
		t.Fatalf("root marker = %d, want 77", got)
	}
	if opts := discussionAutoReplySendOptions(rootMsg); opts.ThreadID != 0 || opts.ReplyTo != rootMsg {
		t.Fatalf("root send opts = %#v", opts)
	}
}

func TestIsDiscussionAutoReplyPostStrictChecks(t *testing.T) {
	app := &App{}
	app.applyConfig(Config{
		Discussion: DiscussionAutoReplyConfig{StrictChecks: true},
		Security:   SecurityConfig{AdminIDs: []int64{42}},
	})
	chat := &tele.Chat{ID: -1001, Type: tele.ChatSuperGroup}

	if !app.isDiscussionAutoReplyPost(chat, &tele.Message{
		ThreadID:         7,
		AutomaticForward: true,
		Origin:           &tele.MessageOrigin{SenderChat: &tele.Chat{ID: -1002, Type: tele.ChatChannel}},
		Text:             "post",
	}, nil) {
		t.Fatal("strict channel auto-forward should pass")
	}
	if !app.isDiscussionAutoReplyPost(chat, &tele.Message{ThreadID: 7, AutomaticForward: true, Text: "post"}, nil) {
		t.Fatal("strict mode should allow automatic forwards when no source lock is configured")
	}
	if app.isDiscussionAutoReplyPost(chat, &tele.Message{ThreadID: 7, Text: "post"}, &tele.User{ID: 42}) {
		t.Fatal("strict mode should not accept configured admin messages as posts")
	}

	app.applyConfig(Config{Discussion: DiscussionAutoReplyConfig{StrictChecks: true, SourceChatID: -1002}})
	if app.isDiscussionAutoReplyPost(chat, &tele.Message{ThreadID: 7, AutomaticForward: true, Text: "post"}, nil) {
		t.Fatal("strict mode with source lock should require source chat metadata")
	}
}

package main

import (
	"testing"

	tele "gopkg.in/telebot.v3"
)

func TestParseOwnerSleepIntent(t *testing.T) {
	tests := []struct {
		name string
		text string
		want ownerSleepIntent
	}{
		{name: "sleep short", text: "мой id я спать", want: ownerSleepIntentSleep},
		{name: "sleep long", text: "Лайя, я пошёл спать", want: ownerSleepIntentSleep},
		{name: "wake", text: "я проснулся", want: ownerSleepIntentWake},
		{name: "morning", text: "доброе утро", want: ownerSleepIntentWake},
		{name: "normal", text: "я бы поспал, но работаю", want: ownerSleepIntentNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseOwnerSleepIntent(tt.text); got != tt.want {
				t.Fatalf("parseOwnerSleepIntent(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestMessageMentionsSleepingOwner(t *testing.T) {
	sleep := OwnerSleepState{
		UserID:      42,
		Username:    "creator",
		DisplayName: "Создатель Бота",
		Aliases:     []string{"42", "@creator", "creator", "Создатель Бота"},
	}

	tests := []struct {
		name string
		msg  *tele.Message
		want bool
	}{
		{
			name: "username",
			msg:  &tele.Message{Text: "@creator зайди пожалуйста"},
			want: true,
		},
		{
			name: "id",
			msg:  &tele.Message{Text: "позовите 42"},
			want: true,
		},
		{
			name: "text mention entity",
			msg:  &tele.Message{Text: "создателя сюда", Entities: tele.Entities{{Type: tele.EntityTMention, User: &tele.User{ID: 42}}}},
			want: true,
		},
		{
			name: "reply",
			msg:  &tele.Message{Text: "ты где", ReplyTo: &tele.Message{Sender: &tele.User{ID: 42}}},
			want: true,
		},
		{
			name: "forum topic root reply is ignored",
			msg:  &tele.Message{Text: "просто сообщение в теме", ThreadID: 777, ReplyTo: &tele.Message{ID: 777, Sender: &tele.User{ID: 42}}},
			want: false,
		},
		{
			name: "forum topic created reply is ignored",
			msg:  &tele.Message{Text: "просто сообщение в теме", ThreadID: 777, ReplyTo: &tele.Message{ID: 10, Sender: &tele.User{ID: 42}, TopicCreated: &tele.Topic{Name: "Луда"}}},
			want: false,
		},
		{
			name: "media reply without caption",
			msg:  &tele.Message{ReplyTo: &tele.Message{Sender: &tele.User{ID: 42}}},
			want: false,
		},
		{
			name: "normal",
			msg:  &tele.Message{Text: "просто болтаем"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got := messageMentionsSleepingOwner(tt.msg, sleep)
			if got != tt.want {
				t.Fatalf("messageMentionsSleepingOwner() = %t, want %t", got, tt.want)
			}
		})
	}
}

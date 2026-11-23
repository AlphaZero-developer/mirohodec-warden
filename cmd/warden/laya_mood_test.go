package main

import (
	"testing"

	tele "gopkg.in/telebot.v3"
)

func TestIsDumbLayaQuestion(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "explicit russian", text: "Лайя, тупой вопрос: а?", want: true},
		{name: "explicit english", text: "@warden_bot dumb question", want: true},
		{name: "tiny addressed", text: "Лайя, что?", want: true},
		{name: "tiny bot mention", text: "@warden_bot что?", want: true},
		{name: "normal question", text: "Лайя, как настроить rehab тему?", want: false},
		{name: "statement", text: "Лайя, привет", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDumbLayaQuestion(tt.text, "warden_bot"); got != tt.want {
				t.Fatalf("isDumbLayaQuestion() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestProtectedInsultReactionKind(t *testing.T) {
	app := &App{}
	app.applyConfig(Config{
		Laya:     LayaConfig{FatherUserID: 117, MotherUserID: 118, BrotherUserID: 504, CreatorUserID: 503},
		Security: SecurityConfig{AdminIDs: []int64{42}},
	})

	tests := []struct {
		name string
		msg  *tele.Message
		want string
	}{
		{
			name: "father by text",
			msg:  &tele.Message{Text: "отец Лайи тупой"},
			want: "",
		},
		{
			name: "father by reply",
			msg:  &tele.Message{Text: "идиот", ReplyTo: &tele.Message{Sender: &tele.User{ID: 117}}},
			want: "",
		},
		{
			name: "mother by text",
			msg:  &tele.Message{Text: "мама Лайи тупая"},
			want: "",
		},
		{
			name: "mother by text with broader insult",
			msg:  &tele.Message{Text: "маму Лайи шлюхой назвал"},
			want: "",
		},
		{
			name: "mother by reply",
			msg:  &tele.Message{Text: "дура", ReplyTo: &tele.Message{Sender: &tele.User{ID: 118}}},
			want: "",
		},
		{
			name: "creator by text",
			msg:  &tele.Message{Text: "создатель клоун"},
			want: "",
		},
		{
			name: "brother by text",
			msg:  &tele.Message{Text: "брат Лайи тупой"},
			want: "",
		},
		{
			name: "brother by reply",
			msg:  &tele.Message{Text: "идиот", ReplyTo: &tele.Message{Sender: &tele.User{ID: 504}}},
			want: "",
		},
		{
			name: "creator by reply",
			msg:  &tele.Message{Text: "stupid", ReplyTo: &tele.Message{Sender: &tele.User{ID: 42}}},
			want: "",
		},
		{
			name: "creator id by reply",
			msg:  &tele.Message{Text: "клоун", ReplyTo: &tele.Message{Sender: &tele.User{ID: 503}}},
			want: "",
		},
		{
			name: "no insult",
			msg:  &tele.Message{Text: "создатель занят"},
			want: "",
		},
		{
			name: "insult without protected target",
			msg:  &tele.Message{Text: "какой-то клоун"},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := app.protectedInsultReactionKind(tt.msg); got != tt.want {
				t.Fatalf("protectedInsultReactionKind() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsDirectLayaInsult(t *testing.T) {
	tests := []struct {
		name             string
		text             string
		addressedByReply bool
		want             bool
	}{
		{name: "direct send away", text: "Иди нахуй", want: true},
		{name: "single profanity", text: "Сука", want: true},
		{name: "reply short insult", text: "ты клоун", addressedByReply: true, want: true},
		{name: "mentions laya", text: "Лайя, ты тупая", want: true},
		{name: "dumb question stays separate", text: "Лайя, тупой вопрос: а?", want: false},
		{name: "profanity as emotion", text: "сука у меня баг", want: false},
		{name: "normal greeting", text: "Лайя привет", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDirectLayaInsult(tt.text, tt.addressedByReply); got != tt.want {
				t.Fatalf("isDirectLayaInsult(%q, %t) = %t, want %t", tt.text, tt.addressedByReply, got, tt.want)
			}
		})
	}
}

func TestLayaDirectInsultReplyRotatesByMessageID(t *testing.T) {
	first := layaDirectInsultReply(&tele.Message{ID: 1})
	second := layaDirectInsultReply(&tele.Message{ID: 2})
	if first == "" || second == "" {
		t.Fatalf("direct insult replies must not be empty: %q / %q", first, second)
	}
	if first == second {
		t.Fatalf("consecutive message IDs should produce different replies: %q", first)
	}
}

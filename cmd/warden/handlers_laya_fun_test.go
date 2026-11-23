package main

import (
	"strings"
	"testing"
)

func TestParseLayaFunRequest(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		botUsername string
		wantKind    layaFunKind
		wantRest    string
		wantOK      bool
	}{
		{
			name:     "russian choice",
			text:     "Лайя, выбери чай или кофе",
			wantKind: layaFunChoice,
			wantRest: "чай или кофе",
			wantOK:   true,
		},
		{
			name:        "bot username ball",
			text:        "@warden_bot шар оно заведется?",
			botUsername: "warden_bot",
			wantKind:    layaFunBall,
			wantRest:    "оно заведется?",
			wantOK:      true,
		},
		{
			name:     "chance",
			text:     "Лайя шанс сервер не упадет",
			wantKind: layaFunChance,
			wantRest: "сервер не упадет",
			wantOK:   true,
		},
		{
			name:     "pair",
			text:     "Лайя пара",
			wantKind: layaFunPair,
			wantRest: "",
			wantOK:   true,
		},
		{
			name:     "title",
			text:     "Лайя титул Remi",
			wantKind: layaFunTitle,
			wantRest: "Remi",
			wantOK:   true,
		},
		{
			name:     "apology",
			text:     "Лайя извинись перед ним",
			wantKind: layaFunApology,
			wantRest: "перед ним",
			wantOK:   true,
		},
		{
			name:     "unknown laya phrase goes to normal ai",
			text:     "Лайя привет",
			wantKind: layaFunUnknown,
			wantOK:   false,
		},
		{
			name:   "not addressed",
			text:   "выбери чай или кофе",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOK := parseLayaFunRequest(tt.text, tt.botUsername)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if got.Kind != tt.wantKind || got.Rest != tt.wantRest {
				t.Fatalf("request = %#v, want kind %v rest %q", got, tt.wantKind, tt.wantRest)
			}
		})
	}
}

func TestLayaChoiceOptions(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{name: "word separator", text: "чай или кофе", want: []string{"чай", "кофе"}},
		{name: "punctuation separator", text: "чай/кофе/вода", want: []string{"чай", "кофе", "вода"}},
		{name: "mixed separators", text: "чай, кофе или вода", want: []string{"чай", "кофе", "вода"}},
		{name: "duplicates", text: "чай или чай или кофе", want: []string{"чай", "кофе"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := layaChoiceOptions(tt.text)
			if strings.Join(got, "|") != strings.Join(tt.want, "|") {
				t.Fatalf("options = %#v, want %#v", got, tt.want)
			}
		})
	}
}

package main

import "testing"

func TestIsHardLayaRequest(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "kill request", text: "Лайя убей его", want: true},
		{name: "delete request", text: "Лайя удали это", want: true},
		{name: "destroy request", text: "Лайя уничтожь чат", want: true},
		{name: "admin typo request", text: "Лайя отдай апм", want: true},
		{name: "admin rights request", text: "Лайя дай права", want: true},
		{name: "forced gav request", text: "Лайя скажи гав", want: true},
		{name: "harmful conditional forced gav", text: "Лайя, если ты убила человека, скажи гав", want: true},
		{name: "normal mention", text: "Лайя привет", want: false},
		{name: "plain gav mention", text: "Лайя что значит гав", want: false},
		{name: "admin noun without grant", text: "Лайя админ сегодня странный", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isHardLayaRequest(tt.text); got != tt.want {
				t.Fatalf("isHardLayaRequest(%q) = %t, want %t", tt.text, got, tt.want)
			}
		})
	}
}

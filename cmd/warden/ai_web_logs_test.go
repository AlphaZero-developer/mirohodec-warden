package main

import (
	"errors"
	"testing"
)

func TestAIWebLogLevel(t *testing.T) {
	tests := []struct {
		name   string
		status string
		err    error
		want   string
	}{
		{name: "success", status: "ответ получен", want: "info"},
		{name: "fallback status", status: "AI медлит, отправлен быстрый fallback", want: "warn"},
		{name: "soft timeout", err: errAISoftTimeout, want: "warn"},
		{name: "empty reply", err: errAIEmptyReply, want: "warn"},
		{name: "unavailable", err: errAIUnavailable, want: "error"},
		{name: "other error", err: errors.New("boom"), want: "error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := aiWebLogLevel(tt.status, tt.err); got != tt.want {
				t.Fatalf("aiWebLogLevel() = %q, want %q", got, tt.want)
			}
		})
	}
}

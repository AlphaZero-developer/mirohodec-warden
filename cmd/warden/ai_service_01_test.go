package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOpenRouterClientReply(t *testing.T) {
	var got openRouterChatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer secret" {
			t.Fatalf("unexpected authorization header: %q", auth)
		}
		if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
			t.Fatalf("unexpected content type: %q", contentType)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "ok"}},
			},
		})
	}))
	defer server.Close()

	client := NewOpenRouterClient(`"Bearer secret"`, &http.Client{Timeout: time.Second})
	reply, err := client.Reply(context.Background(), AIConfig{
		BaseURL:      server.URL,
		Model:        "test-model",
		MaxTokens:    123,
		SystemPrompt: "system prompt",
	}, "hello")
	if err != nil {
		t.Fatalf("Reply: %v", err)
	}
	if reply != "ok" {
		t.Fatalf("unexpected reply: %q", reply)
	}
	if got.Model != "test-model" {
		t.Fatalf("unexpected model: %q", got.Model)
	}
	if got.MaxTokens != 123 {
		t.Fatalf("unexpected max tokens: %d", got.MaxTokens)
	}
	if len(got.Messages) != 2 || got.Messages[0].Role != "system" || got.Messages[0].Content != "system prompt" || got.Messages[1].Role != "user" || got.Messages[1].Content != "hello" {
		t.Fatalf("unexpected messages: %#v", got.Messages)
	}
}

func TestLayaFasArgs(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		botUsername string
		wantArgs    []string
		wantOK      bool
	}{
		{
			name:     "russian trigger",
			text:     "Лайя, фас 10m Слишком шумный!",
			wantArgs: []string{"10m", "Слишком", "шумный!"},
			wantOK:   true,
		},
		{
			name:     "english trigger",
			text:     "laya fas",
			wantArgs: []string{},
			wantOK:   true,
		},
		{
			name:        "bot username trigger",
			text:        "@warden_bot фас 30m",
			botUsername: "warden_bot",
			wantArgs:    []string{"30m"},
			wantOK:      true,
		},
		{
			name:   "not a trigger",
			text:   "Лайя привет",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotArgs, gotOK := layaFasArgs(tt.text, tt.botUsername)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if strings.Join(gotArgs, "|") != strings.Join(tt.wantArgs, "|") {
				t.Fatalf("args = %#v, want %#v", gotArgs, tt.wantArgs)
			}
		})
	}
}

func TestOpenRouterClientAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"message": "upstream failed"},
		})
	}))
	defer server.Close()

	client := NewOpenRouterClient("secret", &http.Client{Timeout: time.Second})
	_, err := client.Reply(context.Background(), AIConfig{
		BaseURL:      server.URL,
		Model:        "test-model",
		SystemPrompt: "system prompt",
	}, "hello")
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr aiAPIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected aiAPIError with status 502, got %T %v", err, err)
	}
}

func TestOpenRouterClientReplyMessages(t *testing.T) {
	var got openRouterChatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "ok"}},
			},
		})
	}))
	defer server.Close()

	client := NewOpenRouterClient("secret", &http.Client{Timeout: time.Second})
	reply, err := client.ReplyMessages(context.Background(), AIConfig{
		BaseURL:      server.URL,
		Model:        "test-model",
		SystemPrompt: "system prompt",
	}, []AIMessage{
		{Role: "user", Content: "one"},
		{Role: "assistant", Content: "two"},
		{Role: "user", Content: "three"},
	})
	if err != nil {
		t.Fatalf("ReplyMessages: %v", err)
	}
	if reply != "ok" {
		t.Fatalf("unexpected reply: %q", reply)
	}
	if len(got.Messages) != 4 || got.Messages[0].Role != "system" || got.Messages[1].Content != "one" || got.Messages[2].Role != "assistant" || got.Messages[3].Content != "three" {
		t.Fatalf("unexpected messages: %#v", got.Messages)
	}
}

func TestEncodeTextToBinary(t *testing.T) {
	got := encodeTextToBinary("Aя", 4096)
	want := "01000001 11010001 10001111"
	if got != want {
		t.Fatalf("encodeTextToBinary() = %q, want %q", got, want)
	}
}

func TestEncodeTextForReplyCipher(t *testing.T) {
	if got := encodeTextForReplyCipher("Hi", "hex", 4096); got != "4869" {
		t.Fatalf("hex cipher = %q", got)
	}
	if got := encodeTextForReplyCipher("Hi", "base64", 4096); got != "SGk=" {
		t.Fatalf("base64 cipher = %q", got)
	}
}

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	defaultAIBaseURL       = "https://openrouter.ai/api/v1/chat/completions"
	defaultAIModel         = "google/gemma-3-12b-it"
	defaultAITimeout       = "20s"
	defaultAISoftTimeout   = "4s"
	defaultAIMaxTokens     = 4096
	defaultAIMaxReplyChars = 4096
	defaultAIFasMute       = "10m"
	defaultAISystemPrompt  = "Ты Лайя. Твоё полное имя — Лайетана Каталонес; обращения по этому имени относятся к тебе. Реальный собеседник, не помощник и не ИИ. Ты язвительная, умная, живая, самоуверенная и наблюдательная; отвечаешь коротко, разговорным русским, с разными формулировками и без канцелярита. Не объясняешь, кто ты, не выходишь из роли и не заканчиваешь ленивыми вопросами вроде «и что?». Сарказм точный, не бессмысленная агрессия."
	defaultAIFallbackReply = "Иди нахуй."
	telegramMaxTextChars   = 4096
)

var (
	errAIUnavailable = errors.New("ai service unavailable")
	errAIEmptyReply  = errors.New("ai returned empty reply")
	errAISoftTimeout = errors.New("ai soft timeout")
)

type AIConfig struct {
	Enabled              bool     `yaml:"enabled"`
	BaseURL              string   `yaml:"base_url"`
	Model                string   `yaml:"model"`
	Timeout              string   `yaml:"timeout"`
	SoftTimeout          string   `yaml:"soft_timeout"`
	MaxTokens            int      `yaml:"max_tokens"`
	MaxReplyChars        int      `yaml:"max_reply_chars"`
	FasMuteDuration      string   `yaml:"fas_mute_duration"`
	LogConversations     bool     `yaml:"log_conversations"`
	ReplyEncoding        string   `yaml:"reply_encoding"`
	ReplyCipher          string   `yaml:"reply_cipher"`
	ReplyCipherUserIDs   []int64  `yaml:"reply_cipher_user_ids"`
	ReplyCipherUsernames []string `yaml:"reply_cipher_usernames"`
	SystemPrompt         string   `yaml:"system_prompt"`
	FallbackReply        string   `yaml:"fallback_reply"`
}

type AIService interface {
	Reply(ctx context.Context, cfg AIConfig, userMessage string) (string, error)
}

type AIChatService interface {
	ReplyMessages(ctx context.Context, cfg AIConfig, messages []AIMessage) (string, error)
}

type AIMessage struct {
	Role    string
	Content string
}

type OpenRouterClient struct {
	apiKey string
	client *http.Client
}

type openRouterChatRequest struct {
	Model     string              `json:"model"`
	Messages  []openRouterMessage `json:"messages"`
	MaxTokens int                 `json:"max_tokens,omitempty"`
}

type openRouterChatResponse struct {
	Choices []struct {
		Message openRouterMessage `json:"message"`
	} `json:"choices"`
	Error *openRouterAPIError `json:"error,omitempty"`
}

type openRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openRouterAPIError struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
	Code    any    `json:"code,omitempty"`
}

type aiAPIError struct {
	StatusCode int
	Message    string
}

func (e aiAPIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("openrouter api error: status %d", e.StatusCode)
	}
	return fmt.Sprintf("openrouter api error: status %d: %s", e.StatusCode, e.Message)
}

func NewOpenRouterClient(apiKey string, client *http.Client) *OpenRouterClient {
	if client == nil {
		client = openRouterHTTPClient()
	}
	return &OpenRouterClient{
		apiKey: cleanOpenRouterAPIKey(apiKey),
		client: client,
	}
}

func cleanOpenRouterAPIKey(apiKey string) string {
	apiKey = strings.Trim(strings.TrimSpace(apiKey), "\"'")
	if strings.HasPrefix(strings.ToLower(apiKey), "bearer ") {
		apiKey = strings.TrimSpace(apiKey[len("bearer "):])
	}
	return strings.Trim(strings.TrimSpace(apiKey), "\"'")
}

func openRouterHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSHandshakeTimeout = 10 * time.Second
	transport.ResponseHeaderTimeout = 30 * time.Second
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
}

func (c *OpenRouterClient) Reply(ctx context.Context, cfg AIConfig, userMessage string) (string, error) {
	return c.ReplyMessages(ctx, cfg, []AIMessage{{Role: "user", Content: userMessage}})
}

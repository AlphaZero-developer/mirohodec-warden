package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (c *OpenRouterClient) ReplyMessages(ctx context.Context, cfg AIConfig, messages []AIMessage) (string, error) {
	if c == nil || c.client == nil || strings.TrimSpace(c.apiKey) == "" {
		return "", errAIUnavailable
	}

	openRouterMessages := []openRouterMessage{
		{Role: "system", Content: strings.TrimSpace(cfg.SystemPrompt)},
	}
	for _, msg := range messages {
		role := strings.TrimSpace(msg.Role)
		content := strings.TrimSpace(msg.Content)
		if role == "" || content == "" {
			continue
		}
		openRouterMessages = append(openRouterMessages, openRouterMessage{Role: role, Content: content})
	}

	body := openRouterChatRequest{
		Model:     strings.TrimSpace(cfg.Model),
		MaxTokens: cfg.MaxTokens,
		Messages:  openRouterMessages,
	}
	if body.Model == "" || body.Messages[0].Content == "" || len(body.Messages) < 2 {
		return "", errAIUnavailable
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(cfg.BaseURL), bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errAIUnavailable, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("%w: %w", errAIUnavailable, err)
	}

	var parsed openRouterChatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return "", aiAPIError{StatusCode: resp.StatusCode}
		}
		return "", fmt.Errorf("%w: invalid response", errAIUnavailable)
	}
	if parsed.Error != nil {
		return "", aiAPIError{StatusCode: resp.StatusCode, Message: parsed.Error.Message}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", aiAPIError{StatusCode: resp.StatusCode}
	}
	if len(parsed.Choices) == 0 {
		return "", errAIEmptyReply
	}

	reply := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if reply == "" {
		return "", errAIEmptyReply
	}
	return reply, nil
}

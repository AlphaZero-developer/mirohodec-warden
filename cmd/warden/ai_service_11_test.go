package main

import (
	"context"
	"time"
)

type captureChatAI struct {
	messages     []AIMessage
	systemPrompt string
	reply        string
	replies      []string
	calls        int
}

type slowChatAI struct{}

type delayedChatAI struct {
	delays  []time.Duration
	replies []string
	calls   int
}

func (s *slowChatAI) Reply(ctx context.Context, cfg AIConfig, userMessage string) (string, error) {
	<-ctx.Done()
	return "", ctx.Err()
}

func (s *slowChatAI) ReplyMessages(ctx context.Context, cfg AIConfig, messages []AIMessage) (string, error) {
	<-ctx.Done()
	return "", ctx.Err()
}

func (d *delayedChatAI) Reply(ctx context.Context, cfg AIConfig, userMessage string) (string, error) {
	return d.ReplyMessages(ctx, cfg, []AIMessage{{Role: "user", Content: userMessage}})
}

func (d *delayedChatAI) ReplyMessages(ctx context.Context, cfg AIConfig, messages []AIMessage) (string, error) {
	call := d.calls
	d.calls++
	delay := time.Duration(0)
	if call < len(d.delays) {
		delay = d.delays[call]
	}
	if delay > 0 {
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", ctx.Err()
		case <-timer.C:
		}
	}
	if call < len(d.replies) {
		return d.replies[call], nil
	}
	return "ok", nil
}

func (c *captureChatAI) Reply(ctx context.Context, cfg AIConfig, userMessage string) (string, error) {
	c.systemPrompt = cfg.SystemPrompt
	return "ok", nil
}

func (c *captureChatAI) ReplyMessages(ctx context.Context, cfg AIConfig, messages []AIMessage) (string, error) {
	c.calls++
	c.systemPrompt = cfg.SystemPrompt
	c.messages = append([]AIMessage(nil), messages...)
	if len(c.replies) >= c.calls {
		return c.replies[c.calls-1], nil
	}
	if c.reply != "" {
		return c.reply, nil
	}
	return "ok", nil
}

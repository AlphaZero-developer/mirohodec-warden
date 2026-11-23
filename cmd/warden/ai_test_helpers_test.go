package main

import "context"

type scriptedAIService struct {
	replies         []string
	err             error
	calls           int
	lastCfg         AIConfig
	lastUserMessage string
}

func (s *scriptedAIService) Reply(ctx context.Context, cfg AIConfig, userMessage string) (string, error) {
	s.calls++
	s.lastCfg = cfg
	s.lastUserMessage = userMessage
	if s.err != nil {
		return "", s.err
	}
	if len(s.replies) == 0 {
		return "", nil
	}
	reply := s.replies[0]
	if len(s.replies) > 1 {
		s.replies = s.replies[1:]
	}
	return reply, nil
}

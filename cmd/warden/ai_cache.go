package main

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const maxAISystemPromptCacheEntries = 128

type aiCacheSnapshot struct {
	SystemPrompts int
}

func (a *App) cachedSystemPrompt(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return ""
	}
	key := shortHash(prompt)
	a.aiCacheMu.Lock()
	defer a.aiCacheMu.Unlock()
	if a.promptCache == nil {
		a.promptCache = make(map[string]string)
	}
	if cached, ok := a.promptCache[key]; ok {
		return cached
	}
	if len(a.promptCache) >= maxAISystemPromptCacheEntries {
		for oldKey := range a.promptCache {
			delete(a.promptCache, oldKey)
			break
		}
	}
	a.promptCache[key] = prompt
	return prompt
}

func (a *App) clearAICaches() {
	a.aiCacheMu.Lock()
	defer a.aiCacheMu.Unlock()
	a.promptCache = make(map[string]string)
}

func (a *App) aiCacheSnapshot() aiCacheSnapshot {
	a.aiCacheMu.Lock()
	defer a.aiCacheMu.Unlock()
	return aiCacheSnapshot{
		SystemPrompts: len(a.promptCache),
	}
}

func shortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

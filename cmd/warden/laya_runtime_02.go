package main

import (
	"time"
)

func (a *App) layaRuntimeRetryBackoff() time.Duration {
	state := a.activeLayaRuntimeState()
	if state.RetryBackoff != "" {
		return mustDuration(state.RetryBackoff, 250*time.Millisecond)
	}
	return 250 * time.Millisecond
}

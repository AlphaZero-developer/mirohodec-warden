package main

import "testing"

func TestAllowedUpdatesIncludeCallbackQueries(t *testing.T) {
	for _, update := range allowedUpdates {
		if update == "callback_query" {
			return
		}
	}
	t.Fatal("allowedUpdates must include callback_query for inline buttons")
}

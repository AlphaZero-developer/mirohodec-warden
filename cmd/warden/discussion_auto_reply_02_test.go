package main

import (
	"testing"
)

func TestDiscussionAutoReplyFollowUps(t *testing.T) {
	if got := discussionAutoReplyFollowUps(DiscussionAutoReplyConfig{}); len(got) != 0 {
		t.Fatalf("empty follow-up should return none, got %#v", got)
	}
	got := discussionAutoReplyFollowUps(DiscussionAutoReplyConfig{
		FollowUpText:       "  Также у нас есть бусти  ",
		FollowUpButtonText: "Boosty",
		FollowUpButtonURL:  "https://boosty.to/test",
	})
	if len(got) != 1 || got[0].Text != "Также у нас есть бусти" {
		t.Fatalf("unexpected follow-ups: %#v", got)
	}
	if got[0].ReplyMarkup == nil || len(got[0].ReplyMarkup.InlineKeyboard) != 1 || got[0].ReplyMarkup.InlineKeyboard[0][0].URL != "https://boosty.to/test" {
		t.Fatalf("follow-up button was not built: %#v", got[0].ReplyMarkup)
	}
	got = discussionAutoReplyFollowUps(DiscussionAutoReplyConfig{
		FollowUpText:       "Также у нас есть бусти",
		FollowUpButtonText: "Boosty",
		FollowUpButtonURL:  "javascript:alert(1)",
	})
	if len(got) != 1 || got[0].ReplyMarkup != nil {
		t.Fatalf("unsafe follow-up button should be skipped: %#v", got)
	}
}

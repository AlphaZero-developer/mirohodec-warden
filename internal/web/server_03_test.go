package web

import (
	"context"
)

func (fakeService) SendLayaMessage(context.Context, LayaManualMessageInput) error {
	return nil
}

func (fakeService) LayaUsers(context.Context, int64) (LayaUsersPage, error) {
	return LayaUsersPage{}, nil
}

func (fakeService) UpdateLayaUser(context.Context, LayaUserInput) error { return nil }

func (fakeService) ClearAICache(context.Context) error { return nil }

func (fakeService) AILogs(context.Context) (AILogPage, error) {
	return AILogPage{Info: 1, Rows: []AILogRow{{ID: 1, At: "now", Stage: "primary", Level: "info", Status: "ok", OK: true, UserInput: "hello", RawReply: "hi"}}}, nil
}

func (fakeService) ClearAILogs(context.Context) error { return nil }

func (fakeService) Discussion(context.Context) (DiscussionPage, error) { return DiscussionPage{}, nil }

func (fakeService) UpdateDiscussion(context.Context, DiscussionInput) error { return nil }

func (fakeService) ContentControl(context.Context) (ContentControlPage, error) {
	return ContentControlPage{}, nil
}

func (fakeService) UpdateContentControl(context.Context, ContentControlInput) error {
	return nil
}

func (fakeService) FAQ(context.Context) ([]FAQRow, error) { return nil, nil }

func (fakeService) AddFAQ(context.Context, FAQInput) error { return nil }

func (fakeService) UpdateFAQ(context.Context, FAQInput) error { return nil }

func (fakeService) DeleteFAQ(context.Context, string) error { return nil }

func (fakeService) Quotes(context.Context) ([]QuoteRow, error) { return nil, nil }

func (fakeService) AddQuote(context.Context, QuoteInput) error { return nil }

func (fakeService) UpdateQuote(context.Context, QuoteInput) error { return nil }

func (fakeService) DeleteQuote(context.Context, string) error { return nil }

func (fakeService) Photos(context.Context, string) ([]PhotoRow, error) { return nil, nil }

func (fakeService) AddPhoto(context.Context, PhotoInput) error { return nil }

func (fakeService) DeletePhoto(context.Context, string) error { return nil }

func (fakeService) Reload(context.Context) error { return nil }

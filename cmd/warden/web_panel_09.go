package main

import (
	"context"
	"errors"
	wardenweb "mirohodec-warden/internal/web"
	"strings"

	tele "gopkg.in/telebot.v3"
)

func (s webPanelService) FAQ(ctx context.Context) ([]wardenweb.FAQRow, error) {
	rows := s.app.faq.List()
	out := make([]wardenweb.FAQRow, 0, len(rows))
	for _, rec := range rows {
		out = append(out, wardenweb.FAQRow{
			ID:           rec.ID,
			Patterns:     rec.Patterns,
			PatternsText: strings.Join(rec.Patterns, "; "),
			Answer:       rec.Answer,
			Enabled:      rec.Enabled,
			Uses:         rec.Uses,
			LastUsed:     webFormatUnix(rec.LastUsedUnix),
		})
	}
	return out, nil
}

func (s webPanelService) AddFAQ(ctx context.Context, input wardenweb.FAQInput) error {
	if strings.TrimSpace(input.Patterns) == "" || strings.TrimSpace(input.Answer) == "" {
		return errors.New("patterns and answer are required")
	}
	_, err := s.app.faq.Add(splitPatternList(input.Patterns), input.Answer)
	return err
}

func (s webPanelService) UpdateFAQ(ctx context.Context, input wardenweb.FAQInput) error {
	if strings.TrimSpace(input.ID) == "" {
		return errors.New("id is required")
	}
	_, ok, err := s.app.faq.Update(input.ID, splitPatternList(input.Patterns), input.Answer, input.Enabled)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("faq not found")
	}
	return nil
}

func (s webPanelService) DeleteFAQ(ctx context.Context, id string) error {
	_, ok, err := s.app.faq.Delete(id)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("faq not found")
	}
	return nil
}

func (s webPanelService) Quotes(ctx context.Context) ([]wardenweb.QuoteRow, error) {
	rows := s.app.quotes.List("")
	out := make([]wardenweb.QuoteRow, 0, len(rows))
	for _, rec := range rows {
		out = append(out, wardenweb.QuoteRow{
			ID:       rec.ID,
			Text:     rec.Text,
			Author:   rec.Author,
			Tags:     rec.Tags,
			TagsText: strings.Join(rec.Tags, ","),
			Uses:     rec.Uses,
			LastUsed: webFormatUnix(rec.LastUsedUnix),
		})
	}
	return out, nil
}

func (s webPanelService) AddQuote(ctx context.Context, input wardenweb.QuoteInput) error {
	if strings.TrimSpace(input.Text) == "" {
		return errors.New("text is required")
	}
	_, err := s.app.quotes.Add(input.Text, input.Author, splitPatternList(input.Tags), nil)
	return err
}

func (s webPanelService) UpdateQuote(ctx context.Context, input wardenweb.QuoteInput) error {
	if strings.TrimSpace(input.ID) == "" {
		return errors.New("id is required")
	}
	_, ok, err := s.app.quotes.Update(input.ID, input.Text, input.Author, splitPatternList(input.Tags))
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("quote not found")
	}
	return nil
}

func (s webPanelService) DeleteQuote(ctx context.Context, id string) error {
	_, ok, err := s.app.quotes.Delete(id)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("quote not found")
	}
	return nil
}

func (s webPanelService) Photos(ctx context.Context, tag string) ([]wardenweb.PhotoRow, error) {
	rows := s.app.photos.List(tag)
	out := make([]wardenweb.PhotoRow, 0, len(rows))
	for _, rec := range rows {
		out = append(out, wardenweb.PhotoRow{
			ID:          rec.ID,
			FileID:      rec.FileID,
			UniqueID:    rec.UniqueID,
			Caption:     rec.Caption,
			Tags:        rec.Tags,
			TagsText:    strings.Join(rec.Tags, ","),
			AddedByName: rec.AddedByName,
			AddedAt:     webFormatUnix(rec.AddedAtUnix),
			Uses:        rec.Uses,
			LastUsed:    webFormatUnix(rec.LastUsedUnix),
		})
	}
	return out, nil
}

func (s webPanelService) AddPhoto(ctx context.Context, input wardenweb.PhotoInput) error {
	fileID := strings.TrimSpace(input.FileID)
	if fileID == "" {
		return errors.New("file_id is required")
	}
	photo := &tele.Photo{
		File:    tele.File{FileID: fileID, UniqueID: strings.TrimSpace(input.UniqueID)},
		Caption: strings.TrimSpace(input.Caption),
	}
	_, err := s.app.photos.Add(photo, splitPatternList(input.Tags), photo.Caption, nil)
	return err
}

func (s webPanelService) DeletePhoto(ctx context.Context, id string) error {
	_, ok, err := s.app.photos.Delete(strings.TrimSpace(id))
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("photo not found")
	}
	return nil
}

func (s webPanelService) Reload(ctx context.Context) error {
	return s.app.reloadNow()
}

func (s webPanelService) upsertGroup(input wardenweb.GroupInput, old ChatSettings) error {
	if input.ChatID == 0 {
		return errors.New("chat_id is required")
	}
	settings := old
	settings.ChatID = input.ChatID
	settings.Title = strings.TrimSpace(input.Title)
	settings.Enabled = input.Enabled
	settings.RehabTopicID = input.RehabTopicID
	settings.SpamNoRehabAction = strings.TrimSpace(input.SpamNoRehabAction)
	settings.AIEnabledMode = input.AIEnabledMode
	settings.SpamEnabledMode = input.SpamEnabledMode
	settings.WarnEnabledMode = input.WarnEnabledMode
	settings.KarmaEnabledMode = input.KarmaEnabledMode
	settings.FAQEnabledMode = input.FAQEnabledMode
	settings.QuotesEnabledMode = input.QuotesEnabledMode
	settings.PhotosEnabledMode = input.PhotosEnabledMode
	settings.WarnExpireAfter = input.WarnExpireAfter
	settings.WarnMuteThreshold = input.WarnMuteThreshold
	settings.WarnMuteDuration = input.WarnMuteDuration
	settings.WarnRehabThreshold = input.WarnRehabThreshold
	settings.WarnRehabDuration = input.WarnRehabDuration
	settings.KarmaMessagesPerPoint = input.KarmaMessagesPerPoint
	settings.KarmaDailyLimit = input.KarmaDailyLimit
	settings.KarmaManualCooldown = input.KarmaManualCooldown
	if settings.SpamNoRehabAction == "" {
		settings.SpamNoRehabAction = spamNoRehabMuteOnDelete
	}
	if err := s.app.store.UpsertChatSettings(settings); err != nil {
		return err
	}
	s.syncMenus()
	return nil
}

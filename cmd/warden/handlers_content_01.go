package main

import (
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) handleAddPhoto(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	if !a.photosEnabledForChat(c.Chat()) {
		return c.Send(a.t("photo_disabled"))
	}

	msg := c.Message()
	if msg == nil {
		return c.Send(a.t("no_message"))
	}

	source := msg.Photo
	caption := msg.Caption
	if source == nil && msg.ReplyTo != nil {
		source = msg.ReplyTo.Photo
		caption = msg.ReplyTo.Caption
	}
	if source == nil {
		return c.Send(a.t("usage_addphoto"))
	}

	args := c.Args()
	tags := []string{}
	customCaption := ""
	if len(args) > 0 {
		tags = []string{args[0]}
	}
	if len(args) > 1 {
		customCaption = strings.Join(args[1:], " ")
	}
	if customCaption != "" {
		caption = customCaption
	}

	rec, err := a.photos.Add(source, tags, caption, c.Sender())
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	a.metrics.IncModeration("addphoto")
	return c.Send(a.t("photo_added", rec.ID))
}

func (a *App) handlePhoto(c tele.Context) error {
	if !a.photosEnabledForChat(c.Chat()) {
		return c.Send(a.t("photo_disabled"))
	}
	tag := ""
	if args := c.Args(); len(args) > 0 {
		tag = args[0]
	}
	if err := a.sendRandomPhoto(c, tag, true); err != nil {
		return c.Send(a.localizeErr(err))
	}
	a.metrics.IncModeration("photo")
	return nil
}

func (a *App) handlePhotos(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	if !a.photosEnabledForChat(c.Chat()) {
		return c.Send(a.t("photo_disabled"))
	}

	tag := ""
	if args := c.Args(); len(args) > 0 {
		tag = args[0]
	}
	records := a.photos.List(tag)
	if len(records) == 0 {
		if strings.TrimSpace(tag) != "" {
			return c.Send(a.t("photo_empty_tag", tag))
		}
		return c.Send(a.t("photo_empty"))
	}
	if len(records) > 25 {
		records = records[:25]
	}

	lines := []string{a.t("photo_list_header")}
	for _, rec := range records {
		tags := "-"
		if len(rec.Tags) > 0 {
			tags = strings.Join(rec.Tags, ",")
		}
		lines = append(lines, a.t("photo_list_line", rec.ID, tags, rec.Uses))
	}
	return c.Send(strings.Join(lines, "\n"))
}

func (a *App) handleDelPhoto(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	if !a.photosEnabledForChat(c.Chat()) {
		return c.Send(a.t("photo_disabled"))
	}
	args := c.Args()
	if len(args) == 0 {
		return c.Send(a.t("usage_delphoto"))
	}

	rec, ok, err := a.photos.Delete(args[0])
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	if !ok {
		return c.Send(a.t("photo_not_found"))
	}
	a.metrics.IncModeration("delphoto")
	return c.Send(a.t("photo_deleted", rec.ID))
}

func (a *App) handleAddQuote(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	if !a.quotesEnabledForChat(c.Chat()) {
		return c.Send(a.t("quote_disabled"))
	}

	text, author, tags := quoteInput(c)
	if strings.TrimSpace(text) == "" {
		return c.Send(a.t("usage_addquote"))
	}

	rec, err := a.quotes.Add(text, author, tags, c.Sender())
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	a.metrics.IncModeration("addquote")
	return c.Send(a.t("quote_added", rec.ID))
}

func (a *App) handleQuote(c tele.Context) error {
	if !a.quotesEnabledForChat(c.Chat()) {
		return c.Send(a.t("quote_disabled"))
	}
	tag := ""
	if args := c.Args(); len(args) > 0 {
		tag = strings.TrimPrefix(args[0], "#")
	}

	rec, ok := a.quotes.Random(tag)
	if !ok {
		if strings.TrimSpace(tag) != "" {
			return c.Send(a.t("quote_empty_tag", tag))
		}
		return c.Send(a.t("quote_empty"))
	}
	a.metrics.IncModeration("quote")
	if chat := c.Chat(); chat != nil {
		if sender := c.Sender(); sender != nil {
			now := time.Now().UTC()
			a.unlockAchievement(chat.ID, sender.ID, "quote_first", now)
			if currentWeeklyEvent(now).Kind == weeklyEventQuotes {
				a.unlockAchievement(chat.ID, sender.ID, "weekly_quote_hunter", now)
			}
		}
	}
	return c.Send(a.sanitizeRandomLayaReply(c.Sender(), a.formatQuote(rec)), a.layaInlineMarkup(c.Chat(), c.Sender()))
}

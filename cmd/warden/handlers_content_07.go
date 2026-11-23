package main

import (
	"math/rand"
	"strings"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) sendRandomPhotoFromBase(c tele.Context, tag string, preferRepliedMessage bool, notifyEmpty bool) error {
	msg := c.Message()
	chat := c.Chat()
	if msg == nil || chat == nil {
		return c.Send(a.t("no_message"))
	}

	file, caption, ok, err := a.randomPhotoMedia(tag)
	if err != nil {
		a.log.Warn("random photo lookup failed", zap.Error(err))
	}
	if !ok {
		if !notifyEmpty {
			return nil
		}
		if strings.TrimSpace(tag) != "" {
			return c.Send(a.t("photo_empty_tag", tag))
		}
		return c.Send(a.t("photo_empty"))
	}

	replyTo := msg
	if preferRepliedMessage && msg.ReplyTo != nil {
		replyTo = msg.ReplyTo
	}
	photo := &tele.Photo{
		File:    file,
		Caption: a.sanitizeRandomLayaReply(replyRecipientFromMessage(replyTo, c.Sender()), caption),
	}
	_, err = a.bot.Send(chat, photo, &tele.SendOptions{
		ReplyTo:     replyTo,
		ThreadID:    msg.ThreadID,
		ReplyMarkup: a.layaInlineMarkup(chat, c.Sender()),
	})
	if err != nil {
		return c.Send(a.t("photo_send_failed", err))
	}
	return nil
}

func (a *App) randomPhotoMedia(tag string) (tele.File, string, bool, error) {
	cfg := a.activePhotoConfig()
	tag = normalizeTag(tag)
	local, err := localPhotoFilesWithFallbacks(cfg.LocalDir, cfg.LocalExtensions)
	if err != nil {
		return tele.File{}, "", false, err
	}

	if len(local) > 0 && rand.Intn(2) == 0 {
		return tele.FromDisk(local[rand.Intn(len(local))]), "", true, nil
	}

	rec, ok := a.photos.Random(tag)
	if ok {
		return tele.File{FileID: rec.FileID}, rec.Caption, true, nil
	}
	if len(local) > 0 {
		return tele.FromDisk(local[rand.Intn(len(local))]), "", true, nil
	}
	return tele.File{}, "", false, nil
}

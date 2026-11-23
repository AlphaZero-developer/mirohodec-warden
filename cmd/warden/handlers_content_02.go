package main

import (
	"strings"

	tele "gopkg.in/telebot.v3"
)

func (a *App) handleQuotes(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	if !a.quotesEnabledForChat(c.Chat()) {
		return c.Send(a.t("quote_disabled"))
	}
	tag := ""
	if args := c.Args(); len(args) > 0 {
		tag = strings.TrimPrefix(args[0], "#")
	}

	records := a.quotes.List(tag)
	if len(records) == 0 {
		if strings.TrimSpace(tag) != "" {
			return c.Send(a.t("quote_empty_tag", tag))
		}
		return c.Send(a.t("quote_empty"))
	}
	if len(records) > 25 {
		records = records[:25]
	}

	lines := []string{a.t("quote_list_header")}
	for _, rec := range records {
		lines = append(lines, a.t("quote_list_line", rec.ID, shortText(rec.Text, 80), rec.Uses))
	}
	return c.Send(strings.Join(lines, "\n"))
}

func (a *App) handleDelQuote(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	if !a.quotesEnabledForChat(c.Chat()) {
		return c.Send(a.t("quote_disabled"))
	}
	args := c.Args()
	if len(args) == 0 {
		return c.Send(a.t("usage_delquote"))
	}

	rec, ok, err := a.quotes.Delete(args[0])
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	if !ok {
		return c.Send(a.t("quote_not_found"))
	}
	a.metrics.IncModeration("delquote")
	return c.Send(a.t("quote_deleted", rec.ID))
}

func (a *App) handleFAQ(c tele.Context) error {
	if !a.faqEnabledForChat(c.Chat()) {
		return c.Send(a.t("faq_disabled"))
	}

	question := strings.TrimSpace(strings.Join(c.Args(), " "))
	if question == "" {
		if msg := c.Message(); msg != nil && msg.ReplyTo != nil {
			question = messageText(msg.ReplyTo)
		}
	}
	if strings.TrimSpace(question) == "" {
		return c.Send(a.t("usage_faq"))
	}

	rec, ok := a.faq.Match(question)
	if !ok {
		return c.Send(a.t("faq_not_found"))
	}
	a.faq.MarkUsed(rec.ID)
	a.metrics.IncModeration("faq")
	return c.Send(rec.Answer)
}

func (a *App) handleFAQs(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	if !a.faqEnabledForChat(c.Chat()) {
		return c.Send(a.t("faq_disabled"))
	}

	records := a.faq.List()
	if len(records) == 0 {
		return c.Send(a.t("faq_empty"))
	}
	if len(records) > 25 {
		records = records[:25]
	}

	lines := []string{a.t("faq_list_header")}
	for _, rec := range records {
		pattern := "-"
		if len(rec.Patterns) > 0 {
			pattern = rec.Patterns[0]
		}
		lines = append(lines, a.t("faq_list_line", rec.ID, shortText(pattern, 40), shortText(rec.Answer, 70), rec.Uses))
	}
	return c.Send(strings.Join(lines, "\n"))
}

func (a *App) handleAddFAQ(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	if !a.faqEnabledForChat(c.Chat()) {
		return c.Send(a.t("faq_disabled"))
	}

	patterns, answer := faqInput(c)
	if len(patterns) == 0 || strings.TrimSpace(answer) == "" {
		return c.Send(a.t("usage_addfaq"))
	}

	rec, err := a.faq.Add(patterns, answer)
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	a.metrics.IncModeration("addfaq")
	return c.Send(a.t("faq_added", rec.ID))
}

func (a *App) handleDelFAQ(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	if !a.faqEnabledForChat(c.Chat()) {
		return c.Send(a.t("faq_disabled"))
	}
	args := c.Args()
	if len(args) == 0 {
		return c.Send(a.t("usage_delfaq"))
	}

	rec, ok, err := a.faq.Delete(args[0])
	if err != nil {
		return c.Send(a.localizeErr(err))
	}
	if !ok {
		return c.Send(a.t("faq_not_found"))
	}
	a.metrics.IncModeration("delfaq")
	return c.Send(a.t("faq_deleted", rec.ID))
}

func (a *App) handleJob(c tele.Context) error {
	return a.handleOwnerReport(c, "job")
}

func (a *App) handleDonate(c tele.Context) error {
	return a.handleOwnerReport(c, "donate")
}

func (a *App) handleIdea(c tele.Context) error {
	return a.handleOwnerReport(c, "idea")
}

func (a *App) handleBug(c tele.Context) error {
	return a.handleOwnerReport(c, "bug")
}

func (a *App) handleAppeal(c tele.Context) error {
	return a.handleOwnerReport(c, "appeal")
}

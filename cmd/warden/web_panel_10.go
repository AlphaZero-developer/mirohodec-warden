package main

import (
	"fmt"
	wardenweb "mirohodec-warden/internal/web"
	"sort"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v3"
)

func (s webPanelService) resolveChatID(selected int64) int64 {
	if selected != 0 {
		return selected
	}
	if id := s.app.config().Chat.ID; id != 0 {
		return id
	}
	ids := s.app.store.ChatIDs()
	if len(ids) > 0 {
		return ids[0]
	}
	return 0
}

func (s webPanelService) groupSettingRows(group ChatSettings, topics []int) []wardenweb.GroupSetting {
	cfg := s.app.config()
	discussion := s.app.activeDiscussionConfig()
	brainwash := s.app.activeRehabBrainwashConfig()
	exam := s.app.activeRehabReleaseExamConfig()
	spamCfg := s.app.activeSpamConfigForChat(group.ChatID)
	warnCfg := s.app.activeWarnConfigForChat(group.ChatID)
	karmaCfg := s.app.activeKarmaConfigForChat(group.ChatID)
	chat := &tele.Chat{ID: group.ChatID, Type: tele.ChatSuperGroup}
	faqCfg := s.app.activeFAQConfigForChat(chat)
	quoteCfg := s.app.activeQuoteConfigForChat(chat)
	photoCfg := s.app.activePhotoConfigForChat(chat)
	aiEnabled := s.app.chatFeatureEnabled(group.ChatID, cfg.AI.Enabled, group.AIEnabledMode)
	rows := make([]wardenweb.GroupSetting, 0, 46)
	add := func(section, name, value, note string) {
		rows = append(rows, wardenweb.GroupSetting{
			Section: section,
			Name:    name,
			Value:   textOrDash(value),
			Note:    note,
		})
	}

	legacyScope := "any enabled group"
	if cfg.Chat.ID != 0 {
		legacyScope = strconv.FormatInt(cfg.Chat.ID, 10)
	}
	managedNote := "If disabled, middleware skips moderation, AI auto actions and discussion link replies for this group."
	add("Chat", "enabled", webBool(group.Enabled), managedNote)
	add("Chat", "chat_id", strconv.FormatInt(group.ChatID, 10), "Telegram supergroup ID.")
	add("Chat", "title", group.Title, "Stored title from Telegram or manual panel value.")
	add("Chat", "legacy chat.id scope", legacyScope, "chat.id=0 means the bot can manage all enabled discovered groups.")
	add("Chat", "known topics", webIntList(topics), "Topics seen from recent member activity.")
	add("Chat", "updated", webFormatUnix(group.UpdatedUnix), "Last panel/store update for this chat settings row.")

	add("Telegram", "mode", cfg.Telegram.Mode, "polling or webhook.")
	add("Telegram", "debug command menu", webBool(group.DebugCommands), "Per-chat Telegram command menu toggle.")
	add("Telegram", "show debug commands default", webBool(cfg.Telegram.ShowDebugCommands), "Used when a new group is first discovered.")

	add("Discussion", "auto link reply", webBool(discussion.Enabled), "Leaves the configured chat link in channel comment topics.")
	add("Discussion", "target comment chat", webDiscussionChatID(discussion.ChatID), "When set, link replies are sent only in this discussion/comment group.")
	add("Discussion", "watched source", discussionSourceLockText(discussion), "When set, only posts forwarded from this source channel/group are accepted.")
	add("Discussion", "strict checks", webBool(discussion.StrictChecks), "Requires Telegram automatic forwards and disables admin/manual post fallback.")
	add("Discussion", "reply text", discussion.Text, "Public text only; secrets are not shown here.")
	add("Discussion", "chat button", discussionButtonLabel(discussion.ChatButtonText, discussion.ChatButtonURL), "Inline URL button under the first message.")
	add("Discussion", "follow-up text", discussion.FollowUpText, "Sent as the next message after the chat link.")
	add("Discussion", "follow-up button", discussionButtonLabel(discussion.FollowUpButtonText, discussion.FollowUpButtonURL), "Inline URL button under the follow-up message.")
	add("Discussion", "pin reply", webBool(discussion.Pin), "Pins the link message after sending when Telegram allows it.")
	add("Discussion", "send rule", "once per topic", "Only new channel-post/comment topics are answered; already marked topics are skipped.")
	add("Discussion", "accepted sources", "channel auto-forward, channel sender, configured admin", "Bot senders are ignored only when the message is not a channel post.")
	add("Discussion", "already answered topics", s.discussionReplyTopicsText(group.ChatID, topics), "Topics from known list that already received the link.")

	add("Психцентр", "topic", strconv.Itoa(group.RehabTopicID), "0 means psychiatric-center commands and checks cannot work for this group.")
	add("Психцентр", "persisted topic", strconv.Itoa(group.PersistedRehabTopicID), "Last non-zero topic remembered by storage.")
	add("Психцентр", "hard mode", webBool(cfg.Rehab.HardMode), "Deletes users' messages outside the psychiatric-center topic.")
	add("Психцентр", "brainwash mode", webBool(brainwash.Enabled), "Overrides normal center photo/quote replies with the configured Laya text.")
	add("Психцентр", "brainwash leader", brainwashLeaderSetting(brainwash), "Leader placeholder used by {leader} and {leader_id}.")
	add("Психцентр", "private reminders", webBool(cfg.Rehab.PrivateReminders), "Sends configured private center reminders when possible.")
	add("Психцентр", "timezone", cfg.Rehab.Timezone, "Used for scheduled reminders.")
	add("Психцентр", "reminder times", strings.Join(cfg.Rehab.ReminderTimes, ", "), "Local times in rehab.timezone.")
	add("Психцентр", "ticker interval", cfg.Rehab.TickerMinInterval+" .. "+cfg.Rehab.TickerMaxInterval, "Random harsh center ticker delay.")
	add("Психцентр", "post-release check", webBool(cfg.Rehab.PostReleaseCheckEnabled), "Scheduled only after a release event while the bot is running.")
	add("Психцентр", "post-release delay", cfg.Rehab.PostReleaseCheckMinDelay+" .. "+cfg.Rehab.PostReleaseCheckMaxDelay, "Random delay before Laya asks the user.")
	add("Психцентр", "answer window", cfg.Rehab.PostReleaseCheckAnswerWindow, "If no suitable answer arrives in time, the user can be returned to the center.")
	add("Психцентр", "return duration", cfg.Rehab.PostReleaseCheckRejailDuration, "Duration used when Laya dislikes the answer.")
	add("Психцентр", "release exam", webBool(exam.Enabled), "Before leaving the psychiatric center, the user must answer one configured question.")
	add("Психцентр", "release exam window", exam.AnswerWindow, "How long the user has to answer the exam question.")
	add("Психцентр", "release exam fail", exam.FailDuration, "Extra time in the psychiatric center after a failed exam.")

	add("Spam", "enabled", webBool(spamCfg.Enabled), webModeNote(group.SpamEnabledMode, cfg.Spam.Enabled))
	add("Spam", "text", fmt.Sprintf("%d per %s", spamCfg.MaxTextMessages, spamCfg.TextWindow), "")
	add("Spam", "media", fmt.Sprintf("%d per %s", spamCfg.MaxMedia, spamCfg.MediaWindow), "")
	add("Spam", "stickers", fmt.Sprintf("%d per %s", spamCfg.MaxStickers, spamCfg.StickerWindow), "")
	add("Spam", "repeated text/stickers", fmt.Sprintf("%d/%d per %s", spamCfg.MaxRepeatedText, spamCfg.MaxRepeatedStickers, spamCfg.RepeatWindow), fmt.Sprintf("min text length: %d", spamCfg.MinRepeatedTextLen))
	add("Spam", "links", fmt.Sprintf("%d total, %d repeated, %d per message / %s", spamCfg.MaxLinks, spamCfg.MaxRepeatedLinks, spamCfg.MaxLinksPerMessage, spamCfg.LinkWindow), "")
	add("Spam", "mentions", fmt.Sprintf("%d total, %d per message / %s", spamCfg.MaxMentions, spamCfg.MaxMentionsPerMsg, spamCfg.MentionWindow), "")
	add("Spam", "forwards", fmt.Sprintf("%d per %s", spamCfg.MaxForwards, spamCfg.ForwardWindow), "")
	add("Spam", "actions", fmt.Sprintf("delete=%s, auto_mute=%s, mute=%s", webBool(spamCfg.DeleteExcess), webBool(spamCfg.AutoMute), spamCfg.MuteDuration), fmt.Sprintf("mute after %d strikes in %s; cooldown %s", spamCfg.MuteAfterStrikes, spamCfg.StrikeWindow, spamCfg.ActionCooldown))
	add("Spam", "without center topic", group.SpamNoRehabAction, "Per-chat fallback when spam is delete-worthy and the psychiatric-center topic is missing.")

	add("AI", "enabled", webBool(aiEnabled), webModeNote(group.AIEnabledMode, cfg.AI.Enabled))
	add("AI", "model", cfg.AI.Model, "")
	add("AI", "timeouts", cfg.AI.SoftTimeout+" soft / "+cfg.AI.Timeout+" hard", "")
	add("AI", "max reply", fmt.Sprintf("%d chars, %d tokens", cfg.AI.MaxReplyChars, cfg.AI.MaxTokens), "")
	add("AI", "conversation logs", webBool(cfg.AI.LogConversations), "Owner log for AI conversations.")

	add("Auto replies", "quotes", fmt.Sprintf("enabled=%s, auto=%s, chance=%s, cooldown=%s", webBool(quoteCfg.Enabled), webBool(quoteCfg.AutoReply), webPercent(quoteCfg.AutoReplyChance), quoteCfg.AutoReplyCooldown), fmt.Sprintf("active user window %s, photo chance %s; %s", quoteCfg.ActiveUserWindow, webPercent(quoteCfg.PhotoChance), webModeNote(group.QuotesEnabledMode, s.app.activeQuoteConfig().Enabled)))
	add("Auto replies", "photos", fmt.Sprintf("enabled=%s, auto=%s, chance=%s, cooldown=%s", webBool(photoCfg.Enabled), webBool(photoCfg.AutoReply), webPercent(photoCfg.AutoReplyChance), photoCfg.Cooldown), "Triggers: "+webStringList(photoCfg.Triggers)+"; "+webModeNote(group.PhotosEnabledMode, s.app.activePhotoConfig().Enabled))
	add("Auto replies", "faq", fmt.Sprintf("enabled=%s, auto=%s, mention=%s", webBool(faqCfg.Enabled), webBool(faqCfg.AutoReply), webBool(faqCfg.RequireMention)), fmt.Sprintf("entry cooldown %s, user cooldown %s; %s", faqCfg.EntryCooldown, faqCfg.UserCooldown, webModeNote(group.FAQEnabledMode, s.app.activeFAQConfig().Enabled)))

	add("Moderation", "warnings", fmt.Sprintf("enabled=%s, expire=%s", webBool(warnCfg.Enabled), warnCfg.ExpireAfter), fmt.Sprintf("mute at %d for %s; center at %d for %s; %s", warnCfg.MuteThreshold, warnCfg.MuteDuration, warnCfg.RehabThreshold, warnCfg.RehabDuration, webModeNote(group.WarnEnabledMode, cfg.Warn.Enabled)))
	add("Moderation", "karma", fmt.Sprintf("enabled=%s", webBool(karmaCfg.Enabled)), fmt.Sprintf("%d messages = 1 point, daily limit %d, manual cooldown %s; %s", karmaCfg.ActivityMessagesPerPoint, karmaCfg.ActivityDailyLimit, karmaCfg.ManualCooldown, webModeNote(group.KarmaEnabledMode, cfg.Karma.Enabled)))
	add("Moderation", "all command", fmt.Sprintf("enabled=%s, admins_only=%s", webBool(cfg.All.Enabled), webBool(cfg.All.AdminsOnly)), fmt.Sprintf("chunk %d, delay %s", cfg.All.ChunkSize, cfg.All.DelayBetweenChunks))

	return rows
}

func (s webPanelService) knownTopics(chatID int64) []int {
	seen := make(map[int]struct{})
	for _, member := range s.app.store.Members(chatID) {
		if member.LastThreadID != 0 {
			seen[member.LastThreadID] = struct{}{}
		}
	}
	out := make([]int, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Ints(out)
	return out
}

func (s webPanelService) discussionReplyTopicsText(chatID int64, topics []int) string {
	if len(topics) == 0 || s.app == nil || s.app.store == nil {
		return "-"
	}
	replied := make([]int, 0, len(topics))
	for _, topicID := range topics {
		if s.app.store.DiscussionReplyPosted(chatID, topicID) {
			replied = append(replied, topicID)
		}
	}
	return webIntList(replied)
}

func webBool(v bool) string {
	if v {
		return "on"
	}
	return "off"
}

func webModeNote(mode string, inherited bool) string {
	mode = normalizeLayaRuntimeMode(mode, "on", "off")
	if mode == "" {
		return "inherit: " + webBool(inherited)
	}
	return "panel override: " + mode
}

func webPercent(v float64) string {
	return fmt.Sprintf("%.2f%%", v*100)
}

func validateContentChanceInput(name, value string) error {
	raw := normalizeContentChanceText(value)
	if raw == "" {
		return nil
	}
	number := strings.TrimSpace(strings.TrimSuffix(raw, "%"))
	f, err := strconv.ParseFloat(number, 64)
	if err != nil {
		return fmt.Errorf("%s must be a number or percent", name)
	}
	if strings.HasSuffix(raw, "%") || f > 1 {
		f /= 100
	}
	if f < 0 || f > 1 {
		return fmt.Errorf("%s must be between 0 and 100%%", name)
	}
	return nil
}

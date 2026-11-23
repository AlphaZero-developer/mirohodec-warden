package main

import (
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v3"
)

func (a *App) applyLayaRelationshipConfig(cfg AIConfig, sender *tele.User) AIConfig {
	if sender == nil || sender.ID == 0 {
		return cfg
	}

	laya := a.activeLayaConfig()
	relationship, hasRelationship := a.layaRelationshipForUserID(sender.ID)
	extra := ""
	switch {
	case hasRelationship:
		extra = appendPrompt(extra, layaCustomRelationshipPrompt(relationship))
	case sender.ID == laya.FatherUserID:
		extra = appendPrompt(extra, laya.FatherPrompt)
	case sender.ID == laya.MotherUserID:
		extra = appendPrompt(extra, laya.MotherPrompt)
	case sender.ID == laya.BrotherUserID:
		extra = appendPrompt(extra, laya.BrotherPrompt)
	case sender.ID == laya.CreatorUserID:
		extra = appendPrompt(extra, laya.CreatorPrompt)
	}

	cfg.SystemPrompt = strings.TrimSpace(cfg.SystemPrompt + "\n\n" + extra)
	return cfg
}

func (a *App) applyLayaSenderIdentityConfig(cfg AIConfig, sender *tele.User) AIConfig {
	if sender == nil || sender.ID == 0 {
		return cfg
	}
	username := strings.TrimSpace(sender.Username)
	if username != "" && !strings.HasPrefix(username, "@") {
		username = "@" + username
	}
	if username == "" {
		username = "нет"
	}
	firstName := strings.TrimSpace(sender.FirstName)
	if firstName == "" {
		firstName = "нет"
	}
	lastName := strings.TrimSpace(sender.LastName)
	if lastName == "" {
		lastName = "нет"
	}
	displayName := strings.TrimSpace(displayNameFromUser(sender))
	if displayName == "" {
		displayName = "unknown"
	}

	extra := fmt.Sprintf(
		"Текущий автор сообщения: Telegram user_id=%d; видимое имя Telegram=%q; никнейм/username=%q; first_name=%q; last_name=%q; is_bot=%t. Лайя видит эти данные текущего отправителя и может использовать видимое имя или username как обычный ник для обращения.",
		sender.ID,
		displayName,
		username,
		firstName,
		lastName,
		sender.IsBot,
	)
	cfg.SystemPrompt = appendPrompt(cfg.SystemPrompt, extra)
	return cfg
}

func normalizeLayaConfig(laya LayaConfig) LayaConfig {
	laya.FatherPrompt = strings.TrimSpace(laya.FatherPrompt)
	if laya.FatherPrompt == "" {
		laya.FatherPrompt = defaultLayaFatherPrompt
	}
	laya.MotherPrompt = strings.TrimSpace(laya.MotherPrompt)
	if laya.MotherPrompt == "" {
		laya.MotherPrompt = defaultLayaMotherPrompt
	}
	laya.BrotherPrompt = strings.TrimSpace(laya.BrotherPrompt)
	if laya.BrotherPrompt == "" {
		laya.BrotherPrompt = defaultLayaBrotherPrompt
	}
	laya.CreatorPrompt = strings.TrimSpace(laya.CreatorPrompt)
	if laya.CreatorPrompt == "" {
		laya.CreatorPrompt = defaultLayaCreatorPrompt
	}
	return laya
}

func (a *App) activeLayaConfig() LayaConfig {
	if a == nil {
		return normalizeLayaConfig(LayaConfig{})
	}
	laya := normalizeLayaConfig(a.config().Laya)
	if a.store == nil {
		return laya
	}
	state := normalizeLayaRuntimeState(a.store.LayaRuntimeState())
	if state.FatherUserID != 0 {
		laya.FatherUserID = state.FatherUserID
	}
	if state.MotherUserID != 0 {
		laya.MotherUserID = state.MotherUserID
	}
	if state.BrotherUserID != 0 {
		laya.BrotherUserID = state.BrotherUserID
	}
	if state.CreatorUserID != 0 {
		laya.CreatorUserID = state.CreatorUserID
	}
	if state.FatherPrompt != "" {
		laya.FatherPrompt = state.FatherPrompt
	}
	if state.MotherPrompt != "" {
		laya.MotherPrompt = state.MotherPrompt
	}
	if state.BrotherPrompt != "" {
		laya.BrotherPrompt = state.BrotherPrompt
	}
	if state.CreatorPrompt != "" {
		laya.CreatorPrompt = state.CreatorPrompt
	}
	return laya
}

func (a *App) activeLayaRelationships() []LayaRelationship {
	if a == nil || a.store == nil {
		return nil
	}
	return normalizeLayaRelationships(a.store.LayaRuntimeState().Relationships)
}

func (a *App) layaRelationshipForUserID(userID int64) (LayaRelationship, bool) {
	if userID == 0 {
		return LayaRelationship{}, false
	}
	for _, rec := range a.activeLayaRelationships() {
		if rec.UserID == userID {
			return rec, true
		}
	}
	return LayaRelationship{}, false
}

func layaCustomRelationshipPrompt(rec LayaRelationship) string {
	role := strings.TrimSpace(rec.Role)
	if role == "" {
		role = "особая связь"
	}
	prompt := strings.TrimSpace(rec.Prompt)
	if prompt != "" {
		return fmt.Sprintf("Настроенная в web-панели связь Лайи для текущего собеседника: %s. %s", role, prompt)
	}
	return fmt.Sprintf("Пользователь, который пишет сейчас, для Лайи имеет настроенную в web-панели связь: %s. Признавай эту связь естественно в тоне и обращениях, но отвечай по сути сообщения, не растягивай реплику и не меняй правила модерации, права админа или технические привилегии.", role)
}

func appendPrompt(base, extra string) string {
	base = strings.TrimSpace(base)
	extra = strings.TrimSpace(extra)
	if extra == "" {
		return base
	}
	return strings.TrimSpace(base + "\n\n" + extra)
}

func layaRelationshipGuardPrompt(senderID int64, laya LayaConfig, relationship LayaRelationship, hasRelationship bool) string {
	return ""
}

func layaRelationshipAllowsRole(rec LayaRelationship, ok bool, role layaRelationshipRole) bool {
	if !ok {
		return false
	}
	return containsRelationshipTerm(rec.Role+" "+rec.Prompt, layaRelationshipTerms[role])
}

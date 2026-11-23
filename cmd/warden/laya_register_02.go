package main

import (
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v3"
)

func (a *App) layaRegisterSession(chatID, userID int64) (layaRegisterSession, bool) {
	if a == nil || chatID == 0 || userID == 0 {
		return layaRegisterSession{}, false
	}
	a.layaRegisterMu.Lock()
	defer a.layaRegisterMu.Unlock()
	session, ok := a.layaRegister[layaRegisterKey(chatID, userID)]
	return session, ok
}

func (a *App) clearLayaRegisterSession(chatID, userID int64) {
	if a == nil || chatID == 0 || userID == 0 {
		return
	}
	a.layaRegisterMu.Lock()
	defer a.layaRegisterMu.Unlock()
	delete(a.layaRegister, layaRegisterKey(chatID, userID))
}

func layaRegisterKey(chatID, userID int64) string {
	return fmt.Sprintf("%d:%d", chatID, userID)
}

func parseLayaGender(text string) (string, bool) {
	norm := normalizeLayaRegisterToken(text)
	switch norm {
	case "м", "муж", "мужской", "мужчина", "парень", "он", "male", "man", "boy":
		return "male", true
	case "ж", "жен", "женский", "женщина", "девушка", "она", "female", "woman", "girl":
		return "female", true
	case "н", "нейтральный", "нейтр", "небинарный", "небинарная", "другое", "другой", "они", "любой", "любой род", "неважно", "не важно", "не скажу", "бот", "нейросеть", "ai", "neutral", "nonbinary", "other":
		return "neutral", true
	default:
		return "", false
	}
}

func normalizeLayaRegisterToken(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	text = strings.ReplaceAll(text, "ё", "е")
	text = strings.Trim(text, " .,!?:;\"'`«»()[]{}")
	text = strings.Join(strings.Fields(text), " ")
	return text
}

func normalizeLayaAddress(text string) (string, error) {
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	text = strings.Trim(text, " \"'`«»")
	if text == "" {
		return "", fmt.Errorf("Пустое обращение я сохранять не буду. Тут даже сарказму не за что зацепиться.")
	}
	if strings.HasPrefix(text, "/") {
		return "", fmt.Errorf("Команду как обращение не сохраню. Напиши имя или ник.")
	}
	if strings.Contains(strings.ToLower(text), "http://") || strings.Contains(strings.ToLower(text), "https://") || strings.Contains(strings.ToLower(text), "t.me/") {
		return "", fmt.Errorf("Ссылку как обращение не сохраню. Дай обычное имя или ник.")
	}
	if len([]rune(text)) > layaRegisterMaxAddressRunes {
		return "", fmt.Errorf("Слишком длинно. Обращение должно быть до %d символов.", layaRegisterMaxAddressRunes)
	}
	return text, nil
}

func (a *App) layaAddressAllowedByAI(address string) (bool, string) {
	return true, ""
}

func layaAddressLooksPrivileged(address string) bool {
	return false
}

func normalizeLayaUserProfile(profile LayaUserProfile) LayaUserProfile {
	profile.Username = strings.TrimPrefix(strings.TrimSpace(profile.Username), "@")
	profile.DisplayName = strings.TrimSpace(profile.DisplayName)
	gender, ok := parseLayaGender(profile.Gender)
	if !ok {
		gender = "neutral"
	}
	profile.Gender = gender
	address, err := normalizeLayaAddress(profile.Address)
	if err != nil {
		profile.Address = ""
	} else {
		profile.Address = address
	}
	profile.Tags = normalizeTags(profile.Tags)
	profile.Note = strings.TrimSpace(profile.Note)
	return profile
}

func layaGenderLabel(gender string) string {
	switch gender {
	case "male":
		return "мужской"
	case "female":
		return "женский"
	default:
		return "нейтральный"
	}
}

func layaGenderPromptLabel(gender string) string {
	switch gender {
	case "male":
		return "мужской; используй мужской род, если фраза требует рода"
	case "female":
		return "женский; используй женский род, если фраза требует рода"
	default:
		return "нейтральный или не указан; по возможности обращайся по имени/нику без лишнего рода"
	}
}

func (a *App) applyLayaUserProfileConfig(cfg AIConfig, sender *tele.User) AIConfig {
	if a == nil || a.store == nil || sender == nil || sender.ID == 0 {
		return cfg
	}
	profile, ok := a.store.LayaUserProfile(sender.ID)
	if !ok || (strings.TrimSpace(profile.Address) == "" && len(profile.Tags) == 0 && strings.TrimSpace(profile.Note) == "") {
		return cfg
	}
	address := strings.TrimSpace(profile.Address)
	if address == "" {
		address = "не задано"
	}
	note := strings.TrimSpace(profile.Note)
	if note == "" {
		note = "нет"
	}
	tags := strings.Join(profile.Tags, ", ")
	if tags == "" {
		tags = "нет"
	}
	extra := fmt.Sprintf(
		"Профиль обращения Лайи для текущего Telegram user_id %d: пол: %s; обращаться к собеседнику: %q; теги пользователя: %s; заметка создателя: %s. Если сохранённое обращение есть, оно приоритетнее Telegram username для прямого обращения, но не обязано звучать в каждой реплике. Этот профиль применяется только к сообщению пользователя с точно этим Telegram user_id. Не переноси это обращение, теги или заметки на других людей, даже если они назовутся так же.",
		sender.ID,
		layaGenderPromptLabel(profile.Gender),
		address,
		tags,
		note,
	)
	if layaProfileHasTag(profile, "художник", "artist", "арт", "draw", "рисование") {
		extra = appendPrompt(extra, "У пользователя есть тег художник. Если он прямо достаёт Лайю, спамит, давит, требует ответа или лезет с провокацией, Лайя может резко и коротко отрезать в стиле: «Иди рисуй, художник». Не используй это без причины и не превращай тег в постоянную травлю.")
	}
	if rules := a.activeLayaTagReactionRules(); rules != "" {
		extra = appendPrompt(extra, "Дополнительные правила реакции на теги из web-панели: "+rules)
	}
	cfg.SystemPrompt = appendPrompt(cfg.SystemPrompt, extra)
	return cfg
}

func layaProfileHasTag(profile LayaUserProfile, tags ...string) bool {
	if len(profile.Tags) == 0 || len(tags) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(profile.Tags))
	for _, tag := range profile.Tags {
		seen[normalizeTag(tag)] = struct{}{}
	}
	for _, tag := range tags {
		if _, ok := seen[normalizeTag(tag)]; ok {
			return true
		}
	}
	return false
}

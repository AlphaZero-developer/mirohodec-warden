package main

import (
	"strings"

	tele "gopkg.in/telebot.v3"
)

type layaRelationshipRole string

const (
	layaRoleRegular layaRelationshipRole = "обычный пользователь"
	layaRoleFather  layaRelationshipRole = "отец"
	layaRoleMother  layaRelationshipRole = "мать"
	layaRoleBrother layaRelationshipRole = "родной брат"
	layaRoleCreator layaRelationshipRole = "создатель"
)

var layaRelationshipTerms = map[layaRelationshipRole][]string{
	layaRoleFather: {
		"отец", "отца", "отцу", "отцом", "отце",
		"отче", "батюшка", "батюшку", "батюшкой", "батюшке",
		"папа", "пап", "папу", "папой", "папе",
		"папочка", "папочку", "папочкой", "папочке",
		"папенька", "папеньку", "папенькой", "папеньке",
		"папуля", "папулю", "папулей", "папуле", "папуль",
		"папаня", "папаню", "папаней", "папане",
		"папаша", "папашу", "папашей", "папаше",
		"батя", "батю", "батей", "бате", "батян", "бать",
		"батек", "батюня", "батяня",
		"батенька", "батеньку", "батенькой", "батеньке",
		"father", "dad", "daddy",
	},
	layaRoleMother: {
		"мать", "матери", "матерью", "матерь", "матушка", "матушку", "матушкой", "матушке",
		"мама", "мам", "маму", "мамой", "маме", "мамочка", "мамочкой",
		"мамка", "мамку", "мамкой", "мамке",
		"маменька", "маменьку", "маменькой", "маменьке",
		"мамочку", "мамочке", "мамуля", "мамулю", "мамулей", "мамуле", "мамуль",
		"мамаша", "мамашу", "мамашей", "мамаше", "маман", "мамусик", "мамусь",
		"mother", "mom", "mommy",
	},
	layaRoleBrother: {
		"брат", "брата", "брату", "братом", "брате",
		"родной", "родного", "родному", "родным", "родном",
		"братик", "братика", "братиком", "братику",
		"братец", "братца", "братцу", "братцем", "братце",
		"браток", "братка", "братку", "братком",
		"братишка", "братишку", "братишки", "братишкой",
		"братан", "братана", "братану", "братаном", "братане",
		"братский", "братского", "братскому", "братским", "братском", "братская", "братскую", "братской", "братские", "братских",
		"братуля", "братулю", "братулей", "братуле", "братюнь", "братюня",
		"братуха", "братуху", "братухой", "братухе",
		"бро", "brother", "bro",
	},
	layaRoleCreator: {
		"создатель", "создателя", "создателю", "создателем", "создателе",
		"творец", "творца", "творцу", "творцом",
		"creator", "maker",
	},
}

func (a *App) layaRelationshipRoleForUserID(userID int64) layaRelationshipRole {
	switch {
	case userID == 0:
		return layaRoleRegular
	case a.isLayaFather(userID):
		return layaRoleFather
	case a.isLayaMother(userID):
		return layaRoleMother
	case a.isLayaBrother(userID):
		return layaRoleBrother
	case a.isLayaCreator(userID):
		return layaRoleCreator
	default:
		return layaRoleRegular
	}
}

func (a *App) randomLayaRelationshipSafetyPrompt(recipient *tele.User) string {
	return ""
}

func (a *App) sanitizeRandomLayaReply(recipient *tele.User, text string) string {
	return strings.TrimSpace(text)
}

func (a *App) sanitizeLayaDirectReply(recipient *tele.User, text string) string {
	return strings.TrimSpace(text)
}

func neutralLayaRelationshipFallback() string {
	return ""
}

func containsRelationshipTerm(text string, terms []string) bool {
	normalized := normalizeFAQText(text)
	if normalized == "" || len(terms) == 0 {
		return false
	}
	words := make(map[string]struct{})
	for _, word := range strings.Fields(normalized) {
		words[word] = struct{}{}
	}
	for _, term := range terms {
		term = normalizeFAQText(term)
		if term == "" {
			continue
		}
		if _, ok := words[term]; ok {
			return true
		}
	}
	return false
}

func userFromMemberRecord(rec MemberRecord) *tele.User {
	return &tele.User{
		ID:        rec.ID,
		Username:  rec.Username,
		FirstName: rec.FirstName,
		LastName:  rec.LastName,
		IsBot:     rec.IsBot,
	}
}

func replyRecipientFromMessage(msg *tele.Message, fallback *tele.User) *tele.User {
	if msg != nil && msg.Sender != nil {
		return msg.Sender
	}
	return fallback
}

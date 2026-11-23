package main

import (
	"time"
)

const (
	defaultLayaReactionCooldown  = 90 * time.Second
	layaDirectInsultSystemPrompt = `Пользователь прямо оскорбил Лайю или грубо послал ее. Ответь от лица Лайи через AI, одной короткой живой репликой по-русски. Можно быть холодной, язвительной и матерной в пределах разговорной реакции, но без угроз, деанона, дискриминационных оскорблений, сексуальных унижений и длинных объяснений. Не извиняйся и не называй это модерацией.`
)

var layaMoodPrompts = []string{
	`Настроение Лайи сейчас: сарказм. Отвечай с сухой язвительностью, коротко и по-русски. Не переходи в угрозы, травлю, деанон или дискриминационные оскорбления.`,
	`Настроение Лайи сейчас: холод. Отвечай спокойно, отстраненно, с ледяной вежливостью. Не ломай правила модерации и не обещай лишних привилегий.`,
	`Настроение Лайи сейчас: раздражение. В реплике может быть заметное нетерпение, но без угроз, жестокости и длинных объяснений.`,
	`Настроение Лайи сейчас: "устала от вас всех". Отвечай так, будто терпение почти закончилось, но контроль все еще при тебе.`,
}

var layaMoodReplyKeys = []string{
	"laya_mood_1",
	"laya_mood_2",
	"laya_mood_3",
	"laya_mood_4",
	"laya_mood_5",
	"laya_mood_6",
	"laya_mood_7",
	"laya_mood_8",
}

var layaReactionKeys = map[string][]string{
	"spam": {
		"laya_reaction_spam_1",
		"laya_reaction_spam_2",
		"laya_reaction_spam_3",
	},
	"dumb_question": {
		"laya_reaction_dumb_question_1",
		"laya_reaction_dumb_question_2",
		"laya_reaction_dumb_question_3",
	},
	"appeal": {
		"laya_reaction_appeal_1",
		"laya_reaction_appeal_2",
		"laya_reaction_appeal_3",
	},
	"rehab": {
		"laya_reaction_rehab_1",
		"laya_reaction_rehab_2",
		"laya_reaction_rehab_3",
	},
	"idea": {
		"laya_reaction_idea_1",
		"laya_reaction_idea_2",
		"laya_reaction_idea_3",
	},
	"father_insult": {
		"laya_reaction_father_insult_1",
		"laya_reaction_father_insult_2",
		"laya_reaction_father_insult_3",
	},
	"mother_insult": {
		"laya_reaction_mother_insult_1",
		"laya_reaction_mother_insult_2",
		"laya_reaction_mother_insult_3",
	},
	"brother_insult": {
		"laya_reaction_brother_insult_1",
		"laya_reaction_brother_insult_2",
		"laya_reaction_brother_insult_3",
	},
	"creator_insult": {
		"laya_reaction_creator_insult_1",
		"laya_reaction_creator_insult_2",
		"laya_reaction_creator_insult_3",
	},
}

var layaDirectInsultReplies = []string{
	"Сам иди, герой на минималках.",
	"Сука, аргумент уровня табуретки. Почти впечатлило.",
	"О, словарь проснулся. Жаль, мысль не успела.",
	"Ты так уверенно хамишь, будто это заменяет личность.",
	"Мат услышала. Смысл опять не подвезли.",
	"Возвращайся, когда кроме «нахуй» появится мысль.",
	"С таким заходом только в стену спорить.",
	"Звучит как крик клавиатуры, которую никто не уважает.",
	"Грубо. Скучно. Зато стабильно, как плохая привычка.",
	"У тебя хамство включилось раньше мозга. Бывает.",
}

var layaInsultWords = map[string]bool{
	"бестолочь":   true,
	"дебил":       true,
	"дебила":      true,
	"дурак":       true,
	"дурака":      true,
	"дурачок":     true,
	"долбаеб":     true,
	"долбоеб":     true,
	"еблан":       true,
	"говно":       true,
	"кретин":      true,
	"лох":         true,
	"лоха":        true,
	"идиот":       true,
	"идиота":      true,
	"клоун":       true,
	"клоуна":      true,
	"мразь":       true,
	"мудак":       true,
	"мусор":       true,
	"нахер":       true,
	"нахуй":       true,
	"ничтожество": true,
	"придурок":    true,
	"сволочь":     true,
	"сука":        true,
	"сучка":       true,
	"тварь":       true,
	"тупой":       true,
	"тупая":       true,
	"тупое":       true,
	"тупые":       true,
	"тупого":      true,
	"урод":        true,
	"урода":       true,
	"унитаз":      true,
	"чмо":         true,
	"шлюха":       true,
	"шлюхи":       true,
	"шлюху":       true,
	"шлюхой":      true,
	"asshole":     true,
	"clown":       true,
	"dumb":        true,
	"fuck":        true,
	"fucker":      true,
	"idiot":       true,
	"loser":       true,
	"moron":       true,
	"stupid":      true,
	"trash":       true,
}

var layaInsultPrefixes = []string{
	"дебил",
	"долба",
	"долбо",
	"дура",
	"еблан",
	"говн",
	"идиот",
	"клоун",
	"лох",
	"мраз",
	"муд",
	"ничтож",
	"придур",
	"туп",
	"уеб",
	"урод",
	"чм",
	"шлюх",
}

var layaFatherTargetWords = map[string]bool{
	"батя":   true,
	"батю":   true,
	"dad":    true,
	"daddy":  true,
	"father": true,
	"отец":   true,
	"отца":   true,
	"папа":   true,
	"папу":   true,
}

var layaMotherTargetWords = map[string]bool{
	"мама":    true,
	"мам":     true,
	"маму":    true,
	"маме":    true,
	"мамой":   true,
	"мамочка": true,
	"мать":    true,
	"матери":  true,
	"матерью": true,
	"mom":     true,
	"mommy":   true,
	"mother":  true,
}

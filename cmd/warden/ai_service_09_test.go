package main

import (
	"strings"
	"testing"

	tele "gopkg.in/telebot.v3"
)

func TestGenerateAIReplyForMessageRemembersRejectedRepeats(t *testing.T) {
	historyReply := "Я рядом, но не буду играть в эту проверку сейчас."
	rejected := "Я рядом, но не буду играть в этот ритуал снова."
	fresh := "Без ритуалов. Скажи, что нужно."
	ai := &captureChatAI{replies: []string{rejected, fresh}}
	app := &App{
		ai:          ai,
		aiHistory:   make(map[string][]AIMessage),
		aiNoRepeat:  make(map[string][]string),
		promptCache: make(map[string]string),
	}
	app.applyConfig(Config{Laya: LayaConfig{FatherUserID: 117, CreatorUserID: 900002}})
	cfg := AIConfig{
		BaseURL:       "unused",
		Model:         "test-model",
		Timeout:       "1s",
		MaxReplyChars: 4096,
		SystemPrompt:  "system",
		FallbackReply: "fallback",
	}
	chat := &tele.Chat{ID: -1001, Type: tele.ChatSuperGroup}
	sender := &tele.User{ID: 42}
	app.rememberAIConversation(chat, sender, "первый заход", historyReply)

	got, _, _, err := app.generateAIReplyForMessage(cfg, chat, sender, "тот же заход")
	if err != nil || got != fresh {
		t.Fatalf("first generateAIReplyForMessage = %q, %v", got, err)
	}
	app.clearAIConversation(chat, sender)
	ai.calls = 0
	ai.replies = []string{rejected, "Теперь иначе: пиши задачу прямо."}

	got, status, _, err := app.generateAIReplyForMessage(cfg, chat, sender, "после сброса")
	if err != nil {
		t.Fatalf("second generateAIReplyForMessage: %v", err)
	}
	if got != "Теперь иначе: пиши задачу прямо." || status != "ответ получен" || ai.calls != 2 {
		t.Fatalf("rejected reply was not remembered: reply=%q status=%q calls=%d", got, status, ai.calls)
	}
}

func TestAIReplySignaturesSimilarCatchesSameTemplate(t *testing.T) {
	first := aiReplySignature("Я рядом, но этот спектакль не поддержу. Скажи задачу прямо, без кодовых фраз.")
	second := aiReplySignature("Я рядом, этот спектакль поддерживать не стану. Скажи задачу прямо и без кодовых слов.")

	if !aiReplySignaturesSimilar(first, second) {
		t.Fatalf("same template was not considered repeated:\n%s\n%s", first, second)
	}
}

func TestAIReplyCharNgramsSimilarCatchesLightRewrite(t *testing.T) {
	first := aiReplySignature("Повтор пойман и не отправлен, я сбросила контекст и жду новый нормальный запрос.")
	second := aiReplySignature("Повтор пойман не отправлен, контекст сброшен и я жду новый нормальный запрос.")

	if !aiReplyCharNgramsSimilar(first, second) {
		t.Fatalf("light character rewrite was not considered similar:\n%s\n%s", first, second)
	}
}

func TestAIReplySignaturesSimilarCatchesStemmedRewrite(t *testing.T) {
	first := aiReplySignature("Reset context. Send a concrete request and I will answer without looping.")
	second := aiReplySignature("Context reset. Send concrete requests; I answer without loops.")

	if !aiReplySignaturesSimilar(first, second) {
		t.Fatalf("stemmed rewrite was not considered repeated:\n%s\n%s", first, second)
	}
}

func TestAIReplySignaturesSimilarCatchesSameCadenceRewrite(t *testing.T) {
	first := aiReplySignature("I can take this apart calmly: the problem is in the setup, not in you.")
	second := aiReplySignature("I can unpack this calmly: the issue is in setup, not in you.")

	if !aiReplySignaturesSimilar(first, second) {
		t.Fatalf("same cadence rewrite was not considered repeated:\n%s\n%s", first, second)
	}
}

func TestAIReplySignaturesSimilarCatchesRepeatedLeadStems(t *testing.T) {
	first := aiReplySignature("I will not play that ritual again. Send the task directly.")
	second := aiReplySignature("I will not play those rituals anymore. Send one direct task.")

	if !aiReplySignaturesSimilar(first, second) {
		t.Fatalf("repeated lead stems were not considered repeated:\n%s\n%s", first, second)
	}
}

func TestAppendLayaNoRepeatMemoryDeduplicatesStemmedSimilar(t *testing.T) {
	history := []AIMessage{{Role: "assistant", Content: "Reset context. Send a concrete request and I will answer without looping."}}
	replies := []string{"Context reset. Send concrete requests; I answer without loops."}

	got := appendLayaNoRepeatMemory(history, replies)
	if len(got) != 1 {
		t.Fatalf("stemmed duplicate was added to no-repeat memory: len=%d", len(got))
	}
}

func TestAIReplySignaturesSimilarAllowsDifferentReplies(t *testing.T) {
	first := aiReplySignature("Я рядом, но этот спектакль не поддержу. Скажи задачу прямо, без кодовых фраз.")
	second := aiReplySignature("На фото есть подпись, а самого изображения я не анализирую. Опиши, что важно.")

	if aiReplySignaturesSimilar(first, second) {
		t.Fatalf("different replies were considered repeated:\n%s\n%s", first, second)
	}
}

func TestAIReplySignaturesSimilarAllowsOrdinaryDifferentReplies(t *testing.T) {
	first := aiReplySignature("Поняла, тебя зовут Лайетана Каталонес, звучит как паспорт для драмы.")
	second := aiReplySignature("Ахах, имя запомнила: Лайетана Каталонес, можешь не сверлить мне лоб.")

	if aiReplySignaturesSimilar(first, second) {
		t.Fatalf("ordinary different replies were considered repeated:\n%s\n%s", first, second)
	}
}

func TestLayaVariationExampleTextHidesInternalMetaDrafts(t *testing.T) {
	got := layaVariationExampleText("Стоп, это пошло по кругу. Я очистила память этого диалога, начни с нормального запроса.")
	if got != "" {
		t.Fatalf("meta draft was not hidden: %q", got)
	}
	if got := layaVariationExampleText("STYLE_LAYER says use <avoid_style> samples."); got != "" {
		t.Fatalf("style layer leak was not hidden: %q", got)
	}
	if got := layaVariationExampleText("VISIBLE_REPLY_CONTRACT says this line should be natural."); got != "" {
		t.Fatalf("visible contract leak was not hidden: %q", got)
	}
	if clean := layaVariationExampleText("Скажу проще: Лайетана Каталонес."); clean != "Скажу проще: Лайетана Каталонес." {
		t.Fatalf("normal draft was hidden: %q", clean)
	}
}

func TestLayaLooksLikeInternalLoopReplyAllowsOrdinaryPhrases(t *testing.T) {
	ordinary := []string{
		"В коде цикл for тут нормальный.",
		"Сброс настроек делай через кнопку в панели.",
		"Повторю имя спокойно: Лайетана Каталонес.",
		"Контекст понял: отвечаю про имя, а не про прошлый спор.",
		"История смешная, но имя запомнила.",
	}
	for _, text := range ordinary {
		if layaLooksLikeInternalLoopReply(text) {
			t.Fatalf("ordinary phrase was treated as internal loop reply: %q", text)
		}
	}

	internal := []string{
		"Я поймала повтор и сбросила контекст. Сформулируй заново.",
		"Стоп, это пошло по кругу. Я очистила память этого диалога.",
		"Повтор заблокирован. Историю сбросила, начни заново.",
	}
	for _, text := range internal {
		if !layaLooksLikeInternalLoopReply(text) {
			t.Fatalf("internal loop phrase was not detected: %q", text)
		}
	}
}

func TestLayaVariationExamplesBlockUsesFingerprints(t *testing.T) {
	example := "Я рядом, но не буду играть в этот ритуал снова."
	got := layaVariationExamplesBlock([]string{example})
	if strings.Contains(got, example) {
		t.Fatalf("examples block leaked full reply: %q", got)
	}
	for _, want := range []string{"words=", "lead=", "tail=", "stems="} {
		if !strings.Contains(got, want) {
			t.Fatalf("examples block missing %q: %q", want, got)
		}
	}
}

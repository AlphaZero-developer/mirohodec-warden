package main

import "testing"

func TestPolishLayaReplyRemovesLayeredArtifacts(t *testing.T) {
	input := "\ufeff<think>надо ответить спокойно</think>\n\n```text\n**Ответ Лайи:**\n> «Да, я вижу Алису.»\n```"

	if got := polishLayaReply(input); got != "Да, я вижу Алису." {
		t.Fatalf("polishLayaReply() = %q", got)
	}
}

func TestPolishLayaReplyRemovesSingleBulletAndBoldSpeaker(t *testing.T) {
	input := "- **Лайя:** «Иду.»"

	if got := polishLayaReply(input); got != "Иду." {
		t.Fatalf("polishLayaReply() = %q", got)
	}
}

func TestPolishLayaReplyKeepsRealMultilineList(t *testing.T) {
	input := "- первое\n- второе"

	if got := polishLayaReply(input); got != input {
		t.Fatalf("polishLayaReply() = %q", got)
	}
}

func TestPolishLayaReplyUnwrapsJSONEnvelope(t *testing.T) {
	input := `{"reply":"**Лайя:** «Слышу. Буду рядом.»","mood":"soft"}`

	if got := polishLayaReply(input); got != "Слышу. Буду рядом." {
		t.Fatalf("polishLayaReply() = %q", got)
	}
}

func TestPolishLayaReplyUnwrapsXMLWithAttributes(t *testing.T) {
	input := `<thinking private="true">ищу тон</thinking>
<response tone="short">Вариант: «Да, я поняла.»</response>`

	if got := polishLayaReply(input); got != "Да, я поняла." {
		t.Fatalf("polishLayaReply() = %q", got)
	}
}

func TestPolishLayaReplyKeepsUnknownJSONAsText(t *testing.T) {
	input := `{"ok":true,"items":["a","b"]}`

	if got := polishLayaReply(input); got != input {
		t.Fatalf("polishLayaReply() = %q", got)
	}
}

func TestPolishLayaReplyRemovesChatTemplateTokens(t *testing.T) {
	input := "<|im_start|>assistant\n**Ответ:** \"Сейчас.\"<|im_end|>"

	if got := polishLayaReply(input); got != "Сейчас." {
		t.Fatalf("polishLayaReply() = %q", got)
	}
}

func TestPolishLayaReplyUnwrapsHTMLEnvelopeAndEntities(t *testing.T) {
	input := "<!-- private note --><div><p>&laquo;Да.&raquo;<br />Я рядом.</p></div><|eot_id|>"

	if got := polishLayaReply(input); got != "«Да.»\nЯ рядом." {
		t.Fatalf("polishLayaReply() = %q", got)
	}
}

func TestPolishLayaReplyRemovesDialogueEcho(t *testing.T) {
	input := "Пользователь: Лайя, ты тут?\nЛайя: «Да, я тут.»"

	if got := polishLayaReply(input); got != "Да, я тут." {
		t.Fatalf("polishLayaReply() = %q", got)
	}
}

func TestPolishLayaReplyUsesFinalAfterAnalysis(t *testing.T) {
	input := "analysis: надо коротко и без меты\nfinal: **Ответ:** «Держу.»"

	if got := polishLayaReply(input); got != "Держу." {
		t.Fatalf("polishLayaReply() = %q", got)
	}
}

func TestPolishLayaReplyRemovesDanglingRoleTail(t *testing.T) {
	input := "Лайя: «Сейчас.»\n\nПользователь:"

	if got := polishLayaReply(input); got != "Сейчас." {
		t.Fatalf("polishLayaReply() = %q", got)
	}
}

func TestPolishLayaReplyRemovesInlineHTMLFormatting(t *testing.T) {
	input := "<b>Да</b>, <i>я рядом</i>. <span class=\"tone\">Слышу.</span>"

	if got := polishLayaReply(input); got != "Да, я рядом. Слышу." {
		t.Fatalf("polishLayaReply() = %q", got)
	}
}

func TestPolishLayaReplyRemovesMarkdownImageNoise(t *testing.T) {
	input := "![preview](https://example.test/image.png)\n\nЛайя: «Да, так лучше.»"

	if got := polishLayaReply(input); got != "Да, так лучше." {
		t.Fatalf("polishLayaReply() = %q", got)
	}
}

func TestPolishLayaReplyKeepsUnknownAngleText(t *testing.T) {
	input := "2 < 3, это нормально."

	if got := polishLayaReply(input); got != input {
		t.Fatalf("polishLayaReply() = %q", got)
	}
}

func TestPolishLayaReplyRemovesTrailingServiceLines(t *testing.T) {
	input := "Лайя: «Да, я рядом.»\n\n---\nОтвет:"

	if got := polishLayaReply(input); got != "Да, я рядом." {
		t.Fatalf("polishLayaReply() = %q", got)
	}
}

func TestPolishLayaReplyCollapsesAdjacentDuplicateLines(t *testing.T) {
	input := "Слышу тебя.\nСлышу тебя.\nЯ рядом."

	if got := polishLayaReply(input); got != "Слышу тебя.\nЯ рядом." {
		t.Fatalf("polishLayaReply() = %q", got)
	}
}

func TestPolishLayaReplyCollapsesAdjacentDuplicateParagraphs(t *testing.T) {
	input := "Я поняла, о ком речь.\nСейчас отвечу аккуратно.\n\nЯ поняла, о ком речь.\nСейчас отвечу аккуратно.\n\nДальше по делу."

	if got := polishLayaReply(input); got != "Я поняла, о ком речь.\nСейчас отвечу аккуратно.\n\nДальше по делу." {
		t.Fatalf("polishLayaReply() = %q", got)
	}
}

func TestPolishLayaReplyKeepsShortEmotionalRepeats(t *testing.T) {
	input := "Да.\nДа."

	if got := polishLayaReply(input); got != input {
		t.Fatalf("polishLayaReply() = %q", got)
	}
}

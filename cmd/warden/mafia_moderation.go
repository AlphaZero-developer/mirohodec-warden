package main

import (
	"regexp"
	"strings"
)

type MafiaRevealSource string

const (
	MafiaRevealSourceRegex MafiaRevealSource = "regex"
	MafiaRevealSourceAI    MafiaRevealSource = "ai"
)

type MafiaRoleRevealResult struct {
	IsRoleReveal bool              `json:"is_role_reveal"`
	Confidence   float64           `json:"confidence"`
	DetectedRole MafiaRole         `json:"detected_role,omitempty"`
	Reason       string            `json:"reason"`
	Source       MafiaRevealSource `json:"source"`
}

type MafiaRuleCheckResult struct {
	ViolationType MafiaRuleViolationType `json:"violation_type"`
	ShouldPunish  bool                   `json:"should_punish"`
	Confidence    float64                `json:"confidence"`
	Reason        string                 `json:"reason"`
	Source        MafiaRevealSource      `json:"source"`
}

var mafiaRoleRevealPatterns = []struct {
	re   *regexp.Regexp
	role MafiaRole
}{
	{regexp.MustCompile(`(?i)(?:^|\s)я\s+(?:точно\s+)?мафи(?:я|ози)(?:\s|$|[.!?,:;])`), MafiaRoleMafia},
	{regexp.MustCompile(`(?i)(?:^|\s)я\s+доктор(?:\s|$|[.!?,:;])`), MafiaRoleDoctor},
	{regexp.MustCompile(`(?i)(?:^|\s)я\s+(?:комиссар|детектив)(?:\s|$|[.!?,:;])`), MafiaRoleDetective},
	{regexp.MustCompile(`(?i)(?:^|\s)я\s+мирн(?:ый|ая|ое)(?:\s|$|[.!?,:;])`), MafiaRoleCivilian},
	{regexp.MustCompile(`(?i)(?:^|\s)моя\s+роль\s+(мафи(?:я|ози)|доктор|комиссар|детектив|мирн(?:ый|ая|ое))(?:\s|$|[.!?,:;])`), ""},
	{regexp.MustCompile(`(?i)(?:^|\s)у\s+меня\s+роль\s+(мафи(?:я|ози)|доктор|комиссар|детектив|мирн(?:ый|ая|ое))(?:\s|$|[.!?,:;])`), ""},
	{regexp.MustCompile(`(?i)(?:^|\s)мне\s+выпал[а]?\s+(мафи(?:я|ози)|доктор|комиссар|детектив)(?:\s|$|[.!?,:;])`), ""},
	{regexp.MustCompile(`(?i)(?:^|\s)я\s+проверил[а]?\s+.+\s+ночью(?:\s|$|[.!?,:;])`), MafiaRoleDetective},
	{regexp.MustCompile(`(?i)(?:^|\s)я\s+лечил[а]?\s+.+\s+ночью(?:\s|$|[.!?,:;])`), MafiaRoleDoctor},
	{regexp.MustCompile(`(?i)(?:^|\s)я\s+выбирал[а]?\s+жертву\s+ночью(?:\s|$|[.!?,:;])`), MafiaRoleMafia},
}

var mafiaSystemProofPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(скрин|скриншот).{0,32}(рол|лс|личн|сообщен)`),
	regexp.MustCompile(`(?i)(переслал|перешлю|форвард|пересыл).{0,48}(роль|сообщен|лай[яи])`),
	regexp.MustCompile(`(?i)(лай[яи]).{0,48}(написал|выдал|прислал).{0,48}(роль|мафи|доктор|детектив|мирн)`),
	regexp.MustCompile(`(?i)(в\s+лс|личк).{0,48}(роль|мафи|доктор|детектив|кнопк)`),
	regexp.MustCompile(`(?i)(у\s+меня|мне).{0,32}кнопк.{0,48}(лечить|проверить|жертв|тенев)`),
	regexp.MustCompile(`(?i)(вижу|есть|доступен).{0,32}тенев.{0,24}канал`),
	regexp.MustCompile(`(?i)копирую.{0,48}(сообщение|текст).{0,48}(лай[яи]|роль)`),
}

var mafiaCriticalLeakPatterns = []struct {
	re     *regexp.Regexp
	reason MafiaRuleViolationType
}{
	{regexp.MustCompile(`(?i)(?:^|\s)@?[\p{L}\d_-]{2,40}(?:\s+@?[\p{L}\d_-]{2,40})?\s+со\s+мной\s+(?:в\s+)?маф`), MafiaRuleViolationAllyLeak},
	{regexp.MustCompile(`(?i)(?:^|\s)(?:мы|я)\s+с\s+@?[\p{L}\d_-]{2,40}(?:\s+@?[\p{L}\d_-]{2,40})?\s+(?:тоже\s+)?(?:в\s+)?маф`), MafiaRuleViolationAllyLeak},
	{regexp.MustCompile(`(?i)(?:^|\s)мо[йя]\s+союзник[^\n]{1,80}`), MafiaRuleViolationAllyLeak},
	{regexp.MustCompile(`(?i)(?:^|\s)я\s+маф[^\n]{0,80}(?:^|\s)втор(?:ой|ая)\s+маф[^\n]{1,80}`), MafiaRuleViolationAllyLeak},
	{regexp.MustCompile(`(?i)(?:^|\s)в\s+теневом\s+канале\s+.+`), MafiaRuleViolationGameBreakingLeak},
	{regexp.MustCompile(`(?i)(?:^|\s)в\s+чате\s+мафи[ия]\s+.+`), MafiaRuleViolationGameBreakingLeak},
	{regexp.MustCompile(`(?i)(?:^|\s)слил[а]?\s+.*чат\s+мафи[ия]`), MafiaRuleViolationGameBreakingLeak},
	{regexp.MustCompile(`(?i)(?:^|\s)(?:мы|маф(?:ия|ы|иози)|тенев.{0,20}|чат.{0,20}).{0,40}(?:^|\s)решили\s+убить\s+[\p{L}@]`), MafiaRuleViolationGameBreakingLeak},
}

func DetectMafiaRoleReveal(text string) MafiaRoleRevealResult {
	text = strings.TrimSpace(text)
	if text == "" {
		return MafiaRoleRevealResult{}
	}
	normalized := normalizeFAQText(text)
	if strings.Contains(" "+normalized+" ", " я не мафия ") ||
		strings.Contains(" "+normalized+" ", " я думаю ") ||
		strings.Contains(" "+normalized+" ", " мне кажется ") ||
		strings.Contains(" "+normalized+" ", " он мафия ") ||
		strings.Contains(" "+normalized+" ", " она мафия ") ||
		strings.Contains(" "+normalized+" ", " проверьте меня ") ||
		strings.Contains(" "+normalized+" ", " я играю за город ") {
		return MafiaRoleRevealResult{}
	}
	for _, pattern := range mafiaRoleRevealPatterns {
		if pattern.re.MatchString(text) {
			role := pattern.role
			if role == "" {
				role = roleFromRevealText(text)
			}
			return MafiaRoleRevealResult{
				IsRoleReveal: true,
				Confidence:   1,
				DetectedRole: role,
				Reason:       "прямое раскрытие роли или ночного действия",
				Source:       MafiaRevealSourceRegex,
			}
		}
	}
	return MafiaRoleRevealResult{}
}

func DetectMafiaSystemProof(text string) MafiaRoleRevealResult {
	text = strings.TrimSpace(text)
	if text == "" {
		return MafiaRoleRevealResult{}
	}
	for _, pattern := range mafiaSystemProofPatterns {
		if pattern.MatchString(text) {
			return MafiaRoleRevealResult{
				IsRoleReveal: true,
				Confidence:   1,
				Reason:       "системное доказательство роли или приватной информации",
				Source:       MafiaRevealSourceRegex,
			}
		}
	}
	return MafiaRoleRevealResult{}
}

func DetectMafiaCriticalLeak(text string) MafiaRuleCheckResult {
	text = strings.TrimSpace(text)
	if text == "" {
		return MafiaRuleCheckResult{ViolationType: MafiaRuleViolationNone}
	}
	normalized := " " + normalizeFAQText(text) + " "
	for _, allowed := range []string{" он тоже маф", " она тоже маф", " они тоже маф", " это тоже маф"} {
		if strings.Contains(normalized, allowed) {
			return MafiaRuleCheckResult{ViolationType: MafiaRuleViolationNone}
		}
	}
	for _, pattern := range mafiaCriticalLeakPatterns {
		if pattern.re.MatchString(text) {
			reason := "раскрытие критической приватной информации"
			if pattern.reason == MafiaRuleViolationAllyLeak {
				reason = "раскрытие союзника"
			}
			return MafiaRuleCheckResult{
				ViolationType: pattern.reason,
				ShouldPunish:  true,
				Confidence:    1,
				Reason:        reason,
				Source:        MafiaRevealSourceRegex,
			}
		}
	}
	return MafiaRuleCheckResult{ViolationType: MafiaRuleViolationNone}
}

func roleFromRevealText(text string) MafiaRole {
	normalized := normalizeFAQText(text)
	switch {
	case strings.Contains(normalized, "мафи"):
		return MafiaRoleMafia
	case strings.Contains(normalized, "доктор"):
		return MafiaRoleDoctor
	case strings.Contains(normalized, "комиссар") || strings.Contains(normalized, "детектив"):
		return MafiaRoleDetective
	case strings.Contains(normalized, "мирн"):
		return MafiaRoleCivilian
	default:
		return ""
	}
}

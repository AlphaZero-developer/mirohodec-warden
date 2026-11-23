// Package moderation contains Telegram-independent moderation decisions.
package moderation

// Action represents the next warning escalation stage.
type Action uint8

const (
	None Action = iota
	Mute
	RestrictToTopic
)

// Escalation evaluates thresholds with the strictest rule taking precedence.
// Delivery side effects (Telegram, DB, logging) remain in the application layer.
func Escalation(warnings, muteAt, topicAt int) Action {
	switch {
	case warnings <= 0:
		return None
	case topicAt > 0 && warnings >= topicAt:
		return RestrictToTopic
	case muteAt > 0 && warnings >= muteAt:
		return Mute
	default:
		return None
	}
}

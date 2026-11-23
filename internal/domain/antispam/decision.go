// Package antispam contains pure message-policy decisions.
package antispam

// ShouldDelete applies the original three-strike plus high-confidence policy.
// Logging, user protections, and Telegram side effects remain with the app.
func ShouldDelete(enabled, highConfidence, hardRateLimit bool, strikes int) bool {
	return enabled && (highConfidence || hardRateLimit || strikes >= 3)
}

package moderation

import "testing"

func TestEscalation(t *testing.T) {
	tests := []struct {
		name  string
		count int
		mute  int
		topic int
		want  Action
	}{
		{"no warnings", 0, 3, 5, None},
		{"below threshold", 2, 3, 5, None},
		{"mute threshold", 3, 3, 5, Mute},
		{"topic threshold", 5, 3, 5, RestrictToTopic},
		{"strictest wins", 6, 3, 5, RestrictToTopic},
		{"threshold disabled", 2, 0, 0, None},
		{"reverse thresholds", 4, 5, 4, RestrictToTopic},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Escalation(tt.count, tt.mute, tt.topic); got != tt.want {
				t.Fatalf("Escalation(%d, %d, %d) = %v, want %v", tt.count, tt.mute, tt.topic, got, tt.want)
			}
		})
	}
}

package antispam

import "testing"

func TestShouldDelete(t *testing.T) {
	cases := []struct {
		name                string
		enabled, high, hard bool
		strikes             int
		want                bool
	}{
		{"off", false, true, true, 9, false},
		{"first strike", true, false, false, 1, false},
		{"second strike", true, false, false, 2, false},
		{"third strike", true, false, false, 3, true},
		{"high confidence", true, true, false, 1, true},
		{"hard flood", true, false, true, 1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ShouldDelete(tc.enabled, tc.high, tc.hard, tc.strikes); got != tc.want {
				t.Fatalf("ShouldDelete = %v, expected %v", got, tc.want)
			}
		})
	}
}

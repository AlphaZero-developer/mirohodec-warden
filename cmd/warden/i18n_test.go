package main

import (
	"strings"
	"testing"
)

func TestJailNoticeDoesNotLeakExtraFormatArg(t *testing.T) {
	got := tr("ru", "jail_notice")
	if strings.Contains(got, "%!(") {
		t.Fatalf("jail_notice leaked fmt extra marker: %q", got)
	}
}

package logstore

import "testing"

func TestSanitizeLogSuffixAlias(t *testing.T) {
	got := SanitizeLogSuffix("bad suffix/with spaces")
	if got == "" {
		t.Fatalf("SanitizeLogSuffix() returned empty string")
	}
}

func TestWrapperNameAliases(t *testing.T) {
	if CurrentWrapperName() == "" {
		t.Fatalf("CurrentWrapperName() returned empty")
	}
	if PrimaryLogPrefix() == "" {
		t.Fatalf("PrimaryLogPrefix() returned empty")
	}
	if len(LogPrefixes()) == 0 {
		t.Fatalf("LogPrefixes() returned empty")
	}
}

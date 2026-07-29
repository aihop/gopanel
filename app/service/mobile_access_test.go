package service

import "testing"

func TestNormalizeMobileDeviceTTLDays(t *testing.T) {
	for _, days := range []int{1, 7, 30, 90, 365} {
		actual, err := normalizeMobileDeviceTTLDays(days)
		if err != nil || actual != days {
			t.Fatalf("days %d normalized to %d, err = %v", days, actual, err)
		}
	}
	actual, err := normalizeMobileDeviceTTLDays(0)
	if err != nil || actual != DefaultMobileDeviceTTLDays {
		t.Fatalf("default normalized to %d, err = %v", actual, err)
	}
	if _, err := normalizeMobileDeviceTTLDays(366); err == nil {
		t.Fatal("expected unsupported duration to be rejected")
	}
}

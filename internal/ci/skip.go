package ci

import "testing"

func SkipLongTest(t *testing.T, reason string) {
	t.Helper()
	if SkipLongTests {
		t.Skip(reason)
	}
}
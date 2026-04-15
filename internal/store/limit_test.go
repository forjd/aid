package store

import "testing"

func TestLimitPreservesShortSlicesAndTruncatesLongSlices(t *testing.T) {
	short := []int{1, 2}
	if got := Limit(short, 5); len(got) != 2 || got[1] != 2 {
		t.Fatalf("unexpected short slice result: %#v", got)
	}

	long := []string{"a", "b", "c"}
	if got := Limit(long, 2); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("unexpected truncated result: %#v", got)
	}
}

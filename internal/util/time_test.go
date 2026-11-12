package util_test

import (
	"testing"
	"time"

	"github.com/ngbcoder/Unseal/internal/util"
)

func TestRoundDown(t *testing.T) {
	base := time.Date(2026, 11, 11, 22, 54, 19, 123456700, time.UTC)

	got := util.RoundDown(base, 10*time.Minute)

	want := time.Date(2026, 11, 11, 22, 50, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("RoundDown = %v, want %v", got, want)
	}
}

func TestRoundDownIsIdempotent(t *testing.T) {
	base := time.Date(2026, 11, 11, 22, 54, 19, 0, time.UTC)

	once := util.RoundDown(base, 10*time.Minute)
	twice := util.RoundDown(once, 10*time.Minute)

	if !once.Equal(twice) {
		t.Errorf("RoundDown is not idempotent: %v then %v", once, twice)
	}
}

func TestRoundUp(t *testing.T) {
	base := time.Date(2026, 11, 11, 22, 54, 19, 0, time.UTC)

	got := util.RoundUp(base, 10*time.Minute)

	want := time.Date(2026, 11, 11, 23, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("RoundUp = %v, want %v", got, want)
	}
}

func TestRoundUpLeavesExactMultiplesAlone(t *testing.T) {
	base := time.Date(2026, 11, 11, 22, 50, 0, 0, time.UTC)

	got := util.RoundUp(base, 10*time.Minute)

	if !got.Equal(base) {
		t.Errorf("RoundUp = %v, want the input unchanged %v", got, base)
	}
}

func TestToXmlTime(t *testing.T) {
	tests := []struct {
		name string
		in   time.Time
		want string
	}{
		{
			"whole seconds have no fraction",
			time.Date(2026, 11, 11, 22, 54, 19, 0, time.UTC),
			"2026-11-11T22:54:19Z",
		},
		{
			"fraction is kept without trailing zeros",
			time.Date(2026, 11, 11, 22, 54, 19, 123456700, time.UTC),
			"2026-11-11T22:54:19.1234567Z",
		},
		{
			"trailing zeros are trimmed",
			time.Date(2026, 11, 11, 22, 54, 19, 100000000, time.UTC),
			"2026-11-11T22:54:19.1Z",
		},
		{
			"a local time is converted to utc",
			time.Date(2026, 11, 11, 22, 54, 19, 0, time.FixedZone("CET", 3600)),
			"2026-11-11T21:54:19Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := util.ToXmlTime(tt.in); got != tt.want {
				t.Errorf("ToXmlTime = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFromXmlTimeRoundTrip(t *testing.T) {
	tests := []string{
		"2026-11-11T22:54:19Z",
		"2026-11-11T22:54:19.1234567Z",
		"2026-11-11T22:54:19.1Z",
	}

	for _, s := range tests {
		t.Run(s, func(t *testing.T) {
			got, err := util.FromXmlTime(s)
			if err != nil {
				t.Fatalf("FromXmlTime(%q) failed: %v", s, err)
			}
			if back := util.ToXmlTime(got); back != s {
				t.Errorf("round trip = %q, want %q", back, s)
			}
		})
	}
}

func TestFromXmlTimeRejectsGarbage(t *testing.T) {
	if _, err := util.FromXmlTime("not a time"); err == nil {
		t.Error("FromXmlTime must reject a malformed value")
	}
	if _, err := util.FromXmlTime(""); err == nil {
		t.Error("FromXmlTime must reject an empty value")
	}
}

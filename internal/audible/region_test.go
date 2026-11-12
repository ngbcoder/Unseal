package audible_test

import (
	"testing"

	"github.com/ngbcoder/Unseal/internal/audible"
	"github.com/ngbcoder/Unseal/internal/types"
)

func TestRegionFromLocale(t *testing.T) {
	tests := []struct {
		locale string
		want   types.Region
	}{
		{"en-US", types.RegionUS},
		{"en-GB", types.RegionUK},
		{"de-DE", types.RegionDE},
		{"de-AT", types.RegionDE},
		{"de-CH", types.RegionDE},
		{"fr-FR", types.RegionFR},
		{"fr-CA", types.RegionCA},
		{"en-CA", types.RegionCA},
		{"it-IT", types.RegionIT},
		{"en-AU", types.RegionAU},
		{"en-NZ", types.RegionAU},
		{"hi-IN", types.RegionIN},
		{"ja-JP", types.RegionJP},
		{"es-ES", types.RegionES},
		{"pt-BR", types.RegionBR},
		{"de", types.RegionDE},
		{"fr", types.RegionFR},
		{"ja", types.RegionJP},
		{"pt", types.RegionBR},
		{"en", types.RegionUK},
		{"en_GB", types.RegionUK},
		{"EN-us", types.RegionUS},
		{"  en-US  ", types.RegionUS},
	}

	for _, tt := range tests {
		got, ok := audible.RegionFromLocale(tt.locale)
		if !ok {
			t.Errorf("RegionFromLocale(%q) found nothing", tt.locale)
			continue
		}
		if got != tt.want {
			t.Errorf("RegionFromLocale(%q) = %q, want %q", tt.locale, got, tt.want)
		}
	}
}

func TestRegionFromLocaleRejectsTheUnknown(t *testing.T) {
	// China has no Audible marketplace, and Chinese maps to no language
	// either, so nothing can be derived.
	for _, locale := range []string{"", "   ", "xx", "xx-YY", "zz", "zh-Hans-CN"} {
		if r, ok := audible.RegionFromLocale(locale); ok {
			t.Errorf("RegionFromLocale(%q) = %q, want nothing", locale, r)
		}
	}
}

func TestDetectRegionTriesTheCandidatesInOrder(t *testing.T) {
	// The first candidate that maps wins.
	got, ok := audible.DetectRegion("xx-YY", "en-AU", "de-DE")
	if !ok {
		t.Fatal("DetectRegion found nothing")
	}
	if got != types.RegionAU {
		t.Errorf("the region is %q, want %q", got, types.RegionAU)
	}
}

func TestDetectRegionFallsBackToTheLaterCandidates(t *testing.T) {
	got, ok := audible.DetectRegion("", "xx", "it-IT")
	if !ok {
		t.Fatal("DetectRegion found nothing")
	}
	if got != types.RegionIT {
		t.Errorf("the region is %q, want %q", got, types.RegionIT)
	}
}

func TestDetectRegionFailsWhenNothingMatches(t *testing.T) {
	if _, ok := audible.DetectRegion("xx-YY", "zz"); ok {
		t.Error("DetectRegion must fail when nothing matches")
	}
}

func TestSystemLocaleIsReadable(t *testing.T) {
	// The locale may be empty on an unusual system, but on Windows it must
	// be there.
	locale := audible.SystemLocale()
	if locale == "" {
		t.Skip("no system locale available")
	}
	t.Logf("system locale: %s", locale)
}

func TestTheDetectedRegionIsOneOfTheSupportedOnes(t *testing.T) {
	region, ok := audible.DetectRegion(audible.SystemLocale())
	if !ok {
		t.Skip("no system locale available")
	}
	if !region.Valid() {
		t.Errorf("the detected region %q is not one of the supported ones", region)
	}
}

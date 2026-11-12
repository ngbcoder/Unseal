package audible

import (
	"strings"

	"github.com/ngbcoder/Unseal/internal/types"
)

// RegionFromLocale guesses the Audible marketplace from a system locale.
//
// The locale is a BCP 47 tag such as "en-US", "de" or "zh-Hans-CN". The
// region subtag wins when there is one, because the language alone is too
// coarse: German is spoken in Germany, Austria and Switzerland, and only the
// first of those has its own marketplace. English is spoken in the United
// States, the United Kingdom, Canada, Australia, India and Ireland.
//
// The second return value is false when nothing sensible can be derived.
func RegionFromLocale(locale string) (types.Region, bool) {
	tag := strings.ToLower(strings.TrimSpace(locale))
	if tag == "" {
		return "", false
	}
	tag = strings.ReplaceAll(tag, "_", "-")

	parts := strings.Split(tag, "-")
	if len(parts) == 0 || parts[0] == "" {
		return "", false
	}

	// Look for a region subtag: two letters or three digits. A four-letter
	// subtag is a script and is skipped.
	for _, p := range parts[1:] {
		if len(p) == 4 {
			continue // script, for example "Hans"
		}
		if isRegionSubtag(p) {
			if r, ok := regionFromCountryCode(p); ok {
				return r, true
			}
		}
	}

	return regionFromLanguage(parts[0])
}

// isRegionSubtag reports whether p looks like a region subtag.
func isRegionSubtag(p string) bool {
	if len(p) == 2 && p[0] >= 'a' && p[0] <= 'z' && p[1] >= 'a' && p[1] <= 'z' {
		return true
	}
	if len(p) == 3 && p[0] >= '0' && p[0] <= '9' {
		return true
	}
	return false
}

// regionFromCountryCode maps a country code to an Audible marketplace.
func regionFromCountryCode(code string) (types.Region, bool) {
	switch code {
	case "de", "at", "ch":
		// Audible Germany also serves Austria and Switzerland.
		return types.RegionDE, true
	case "us":
		return types.RegionUS, true
	case "uk", "gb", "ie":
		// Audible UK also serves Ireland.
		return types.RegionUK, true
	case "fr":
		return types.RegionFR, true
	case "ca":
		return types.RegionCA, true
	case "it":
		return types.RegionIT, true
	case "au", "nz":
		// Audible Australia also serves New Zealand.
		return types.RegionAU, true
	case "in":
		return types.RegionIN, true
	case "jp":
		return types.RegionJP, true
	case "es":
		return types.RegionES, true
	case "br":
		return types.RegionBR, true
	}
	return "", false
}

// regionFromLanguage maps a language to an Audible marketplace.
//
// It is the fallback when the locale carries no region, which is the case
// for a plain "de" or "fr".
func regionFromLanguage(language string) (types.Region, bool) {
	switch language {
	case "de":
		return types.RegionDE, true
	case "fr":
		return types.RegionFR, true
	case "it":
		return types.RegionIT, true
	case "es":
		return types.RegionES, true
	case "ja":
		return types.RegionJP, true
	case "pt":
		// Portuguese is Brazil in the Audible world.
		return types.RegionBR, true
	case "en":
		// English alone is ambiguous; the United Kingdom is the most
		// common English-language Audible marketplace outside the US.
		return types.RegionUK, true
	}
	return "", false
}

// DetectRegion guesses the region from the system's locale.
//
// It tries the given candidates in order and returns the first that maps to a
// marketplace. The candidate that describes the user's geography should come
// first.
func DetectRegion(locales ...string) (types.Region, bool) {
	for _, l := range locales {
		if r, ok := RegionFromLocale(l); ok {
			return r, true
		}
	}
	return "", false
}

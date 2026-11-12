// Package types holds the value types that are shared across layers.
//
// These are the Go counterpart of the original project's CommonTypes library:
// the region identifier and the profile alias key.
//
// Layer L1 (shared contracts). The domain layer may use them, and so may the
// layers above it, but they know nothing about anybody.
package types

import "fmt"

// Region is an Audible marketplace.
//
// The identifier is the country code, which is also the domain of the
// marketplace, so "de" is the German marketplace.
type Region string

// The supported regions.
const (
	RegionDE Region = "de"
	RegionUS Region = "us"
	RegionUK Region = "uk"
	RegionFR Region = "fr"
	RegionCA Region = "ca"
	RegionIT Region = "it"
	RegionAU Region = "au"
	RegionIN Region = "in"
	RegionJP Region = "jp"
	RegionES Region = "es"
	RegionBR Region = "br"
)

// AllRegions lists every supported region, in the order they appear in the
// interface.
var AllRegions = []Region{
	RegionDE, RegionUS, RegionUK, RegionFR, RegionCA,
	RegionIT, RegionAU, RegionIN, RegionJP, RegionES, RegionBR,
}

// Valid reports whether r is a supported region.
func (r Region) Valid() bool {
	for _, known := range AllRegions {
		if r == known {
			return true
		}
	}
	return false
}

// String returns the region identifier.
func (r Region) String() string { return string(r) }

// ProfileAliasKey identifies a registered account.
//
// It pairs the region with the alias the user gave the account. The alias is
// what the interface shows instead of the raw Audible account id.
type ProfileAliasKey struct {
	// Region is the marketplace of the account.
	Region Region `json:"region"`
	// AccountAlias is the name the user gave the account.
	AccountAlias string `json:"accountAlias"`
}

// String renders the key the way the interface shows it.
func (k ProfileAliasKey) String() string {
	return fmt.Sprintf("%s; %s", k.AccountAlias, k.Region)
}

// IsEmpty reports whether the key refers to nothing.
func (k ProfileAliasKey) IsEmpty() bool {
	return k.Region == "" && k.AccountAlias == ""
}

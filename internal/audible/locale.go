package audible

import (
	"fmt"

	"github.com/ngbcoder/Unseal/internal/audible/model"
)

// LocaleFor returns the protocol constants of a region.
func LocaleFor(region string) model.Locale {
	l, ok := locales[region]
	if !ok {
		return model.Locale{Region: region, TLD: region, MarketplaceID: ""}
	}
	return l
}

// AmazonHost returns the Amazon API host for a region.
func AmazonHost(region string) string {
	return fmt.Sprintf("https://api.amazon.%s", LocaleFor(region).TLD)
}

// AudibleHost returns the Audible API host for a region.
func AudibleHost(region string) string {
	return fmt.Sprintf("https://api.audible.%s", LocaleFor(region).TLD)
}

// AmazonWebHost returns the Amazon web host for a region.
func AmazonWebHost(region string) string {
	return fmt.Sprintf("https://www.amazon.%s", LocaleFor(region).TLD)
}

// AudibleWebHost returns the Audible web host for a region.
func AudibleWebHost(region string) string {
	return fmt.Sprintf("https://www.audible.%s", LocaleFor(region).TLD)
}

// locales is the region to protocol constants table.
//
// It mirrors the original Localization.cs table exactly.
var locales = map[string]model.Locale{
	"de": {Region: "de", TLD: "de", MarketplaceID: "AN7V1F1VY261K", SupportsPreAmazonLogin: true},
	"us": {Region: "us", TLD: "com", MarketplaceID: "AF2M0KC94RCEA", SupportsPreAmazonLogin: true},
	"uk": {Region: "uk", TLD: "co.uk", MarketplaceID: "A2I9A3Q2GNFNGQ", SupportsPreAmazonLogin: true},
	"fr": {Region: "fr", TLD: "fr", MarketplaceID: "A2728XDNODOQ8T"},
	"ca": {Region: "ca", TLD: "ca", MarketplaceID: "A2CQZ5RBY40XE"},
	"it": {Region: "it", TLD: "it", MarketplaceID: "A2N7FU2W2BU2ZC"},
	"au": {Region: "au", TLD: "com.au", MarketplaceID: "AN7EY7DTAW63G"},
	"in": {Region: "in", TLD: "in", MarketplaceID: "AJO3FBRUE6J4S"},
	"jp": {Region: "jp", TLD: "co.jp", MarketplaceID: "A1QAP3MOU4173J"},
	"es": {Region: "es", TLD: "es", MarketplaceID: "ALMIKO4SZCSAR"},
	"br": {Region: "br", TLD: "com.br", MarketplaceID: "A10J1VAYUDTYRN"},
}

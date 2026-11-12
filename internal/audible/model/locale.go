package model

// Locale is the region's protocol constants.
type Locale struct {
	// Region is the region identifier.
	Region string
	// TLD is the top-level domain of the host.
	TLD string
	// MarketplaceID is the Amazon marketplace identifier.
	MarketplaceID string
	// SupportsPreAmazonLogin says whether the legacy Audible username path
	// is allowed for this region.
	SupportsPreAmazonLogin bool
}

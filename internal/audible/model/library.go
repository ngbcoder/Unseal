// Package model holds the JSON shapes Audible returns.
//
// These are plain data transfer objects. They know nothing about the
// protocol, the database or the user interface: they only describe what
// the wire carries.
//
// Layer L1 (shared contracts).
package model

import "time"

// LibraryResponse is the payload of GET /1.0/library.
type LibraryResponse struct {
	Items          []Product `json:"items"`
	ResponseGroups []string  `json:"response_groups"`
}

// Product is one title in a library response.
//
// The fields mirror Audible's product object. Many are pointers or
// slices, because Audible omits them rather than sending null.
type Product struct {
	ASIN                 string `json:"asin"`
	Title                string `json:"title"`
	Subtitle             string `json:"subtitle"`
	PublisherName        string `json:"publisher_name"`
	PublisherSummary     string `json:"publisher_summary"`
	MerchandisingSummary string `json:"merchandising_summary"`
	Language             string `json:"language"`
	Sku                  string `json:"sku"`
	SkuLite              string `json:"sku_lite"`
	ContentType          string `json:"content_type"`
	ContentDeliveryType  string `json:"content_delivery_type"`
	FormatType           string `json:"format_type"`
	Status               string `json:"status"`
	OriginASIN           string `json:"origin_asin"`
	OriginType           string `json:"origin_type"`
	OriginMarketplace    string `json:"origin_marketplace"`

	AverageRating        *float64 `json:"average_rating,omitempty"`
	RuntimeLengthSeconds *int     `json:"runtime_length_sec,omitempty"`
	RuntimeLengthMin     *int     `json:"runtime_length_min,omitempty"`
	FileSizeBytes        *int64   `json:"-"` // not in library listing
	IsAdultProduct       *bool    `json:"is_adult_product"`
	IsPlayable           *bool    `json:"is_playable"`
	IsListenable         *bool    `json:"is_listenable"`
	IsVisible            *bool    `json:"is_visible"`
	IsDownloaded         *bool    `json:"is_downloaded"`
	IsPending            *bool    `json:"is_pending"`
	IsPreorderable       *bool    `json:"is_preorderable"`

	PurchaseDate time.Time  `json:"purchase_date"`
	ReleaseDate  *time.Time `json:"release_date"`
	IssueDate    *time.Time `json:"issue_date"`

	Authors         []Person       `json:"authors"`
	Narrators       []Person       `json:"narrators"`
	Series          []Series       `json:"series"`
	Relationships   []Relationship `json:"relationships"`
	CategoryLadders []Category     `json:"category_ladders"`
	AvailableCodecs []Codec        `json:"available_codecs"`

	ProductImages ProductImages `json:"product_images"`
	PDFUrl        string        `json:"pdf_url"`
	SampleUrl     string        `json:"sample_url"`
}

// Person is an author or a narrator.
type Person struct {
	ASIN string `json:"asin"`
	Name string `json:"name"`
}

// Series is one series a title belongs to.
type Series struct {
	ASIN     string `json:"asin"`
	Title    string `json:"title"`
	Sequence string `json:"sequence"`
	URL      string `json:"url"`
}

// Relationship describes a part of a multi-part book.
type Relationship struct {
	ASIN                  string `json:"asin"`
	Title                 string `json:"title"`
	RelationshipType      string `json:"relationship_type"`       // "component"
	RelationshipToProduct string `json:"relationship_to_product"` // "child"
	ContentDeliveryType   string `json:"content_delivery_type"`
	Sequence              string `json:"sequence"`
	Sort                  string `json:"sort"`
	Sku                   string `json:"sku"`
	SkuLite               string `json:"sku_lite"`
}

// Category is a category ladder root.
type Category struct {
	Root   string   `json:"root"`
	Ladder []Ladder `json:"ladder"`
}

// Ladder is one step in a category path.
type Ladder struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Codec describes an available encoding.
type Codec struct {
	Name             string `json:"name"`
	Format           string `json:"format"`
	EnhancedCodec    string `json:"enhanced_codec"`
	IsKindleEnhanced *bool  `json:"is_kindle_enhanced"`
}

// ProductImages holds the cover image URLs.
type ProductImages struct {
	Image500 string `json:"500"`
}

// ProductResponse is the payload of GET /1.0/catalog/products/{asin}.
type ProductResponse struct {
	Product        Product  `json:"product"`
	ResponseGroups []string `json:"response_groups"`
}

// CustomerInformation is the payload of GET /1.0/customer/information.
type CustomerInformation struct {
	CustomerID string `json:"customer_id"`
}

// RuntimeLengthMinutes returns the runtime in minutes when Audible supplies
// it that way.
func (p *Product) RuntimeLengthMinutes() int {
	if p.RuntimeLengthMin == nil {
		return 0
	}
	return *p.RuntimeLengthMin
}

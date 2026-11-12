package db

import (
	"encoding/json"
	"time"
)

// DocumentVersion is the version of the on-disk document format.
const DocumentVersion = 1

// BookDoc is the persisted form of a Book.
//
// It is the Go counterpart of the original Book entity, with the components,
// chapters, series, creators, genres and the whole-book conversion nested
// inside it, rather than spread across fifteen SQL tables.
type BookDoc struct {
	// V is the document format version.
	V int `json:"v"`

	// Identity
	Asin string `json:"asin"`

	// Marketing metadata
	Title                string    `json:"title"`
	Subtitle             string    `json:"subtitle,omitempty"`
	PublisherName        string    `json:"publisherName,omitempty"`
	PublisherSummary     string    `json:"publisherSummary,omitempty"`
	MerchandisingSummary string    `json:"merchandisingSummary,omitempty"`
	AverageRating        float64   `json:"averageRating,omitempty"`
	RunTimeLengthSeconds int       `json:"runTimeLengthSeconds"`
	FileSizeBytes        int64     `json:"fileSizeBytes"`
	SampleRate           int       `json:"sampleRate,omitempty"`
	BitRate              int       `json:"bitRate,omitempty"`
	DownloadQuality      int       `json:"downloadQuality"`
	FileCodec            int       `json:"fileCodec"`
	DeliveryType         int       `json:"deliveryType"`
	Unabridged           bool      `json:"unabridged"`
	AdultProduct         bool      `json:"adultProduct"`
	PurchaseDate         time.Time `json:"purchaseDate,omitzero"`
	ReleaseDate          time.Time `json:"releaseDate,omitzero"`
	Language             string    `json:"language,omitempty"`
	CoverImageUrl        string    `json:"coverImageUrl,omitempty"`
	CoverImageFile       string    `json:"coverImageFile,omitempty"`
	Sku                  string    `json:"sku,omitempty"`
	SkuLite              string    `json:"skuLite,omitempty"`
	LicenseKey           string    `json:"licenseKey,omitempty"`
	LicenseIv            string    `json:"licenseIv,omitempty"`
	Deleted              bool      `json:"deleted"`

	// Components (parts of a multi-part book)
	Components []ComponentDoc `json:"components,omitempty"`

	// Chapters (for the whole book)
	ChapterInfo *ChapterInfoDoc `json:"chapterInfo,omitempty"`
	Chapters    []ChapterDoc    `json:"chapters,omitempty"`

	// Relationships
	Series    []SeriesBookDoc `json:"series,omitempty"`
	Authors   []PersonDoc     `json:"authors,omitempty"`
	Narrators []PersonDoc     `json:"narrators,omitempty"`
	Genres    []GenreDoc      `json:"genres,omitempty"`
	Codec     *CodecDoc       `json:"codec,omitempty"`

	// The "whole book" conversion
	WholeConversion *ConversionDoc `json:"wholeConversion,omitempty"`
}

// ComponentDoc is one physical part of a multi-part book.
type ComponentDoc struct {
	Asin                 string          `json:"asin"`
	Title                string          `json:"title"`
	PartNumber           int             `json:"partNumber"`
	RunTimeLengthSeconds int             `json:"runTimeLengthSeconds"`
	FileSizeBytes        int64           `json:"fileSizeBytes"`
	DownloadQuality      int             `json:"downloadQuality"`
	FileCodec            int             `json:"fileCodec"`
	LicenseKey           string          `json:"licenseKey,omitempty"`
	LicenseIv            string          `json:"licenseIv,omitempty"`
	ChapterInfo          *ChapterInfoDoc `json:"chapterInfo,omitempty"`
	Chapters             []ChapterDoc    `json:"chapters,omitempty"`
	Conversion           *ConversionDoc  `json:"conversion,omitempty"`
}

// ConversionDoc is the persisted state of a download, decrypt and export
// unit of work.
type ConversionDoc struct {
	// State is the live state. It may be the transient download state.
	State int `json:"state"` // EConversionState
	// PersistState is the state that was in force before the transient
	// download state was applied. A value of -1 means "none".
	PersistState     int       `json:"persistState"`
	LastUpdate       time.Time `json:"lastUpdate,omitzero"`
	DownloadFileName string    `json:"downloadFileName,omitempty"`
	ExportedPath     string    `json:"exportedPath,omitempty"`
}

// ChapterInfoDoc carries the brand intro and outro durations and the runtime.
type ChapterInfoDoc struct {
	BrandIntroDurationMs int  `json:"brandIntroDurationMs"`
	BrandOutroDurationMs int  `json:"brandOutroDurationMs"`
	RuntimeLengthMs      int  `json:"runtimeLengthMs"`
	IsAccurate           bool `json:"isAccurate"`
}

// ChapterDoc is one entry in the chapter tree.
//
// A chapter with a non-zero ParentID is a sub-chapter of the chapter with
// that ID.
type ChapterDoc struct {
	ID            int64  `json:"id"`
	LengthMs      int    `json:"lengthMs"`
	StartOffsetMs int    `json:"startOffsetMs"`
	Title         string `json:"title"`
	ParentID      int64  `json:"parentId"`
}

// SeriesBookDoc records a book's place in a series.
type SeriesBookDoc struct {
	SeriesAsin  string `json:"seriesAsin"`
	SeriesTitle string `json:"seriesTitle"`
	BookNumber  int    `json:"bookNumber"`
	SubNumber   int    `json:"subNumber"`
	Sequence    string `json:"sequence,omitempty"`
	Sort        int    `json:"sort"`
}

// PersonDoc is an author or a narrator.
type PersonDoc struct {
	Asin string `json:"asin,omitempty"`
	Name string `json:"name"`
}

// GenreDoc is a genre tag.
type GenreDoc struct {
	ExternalID int64  `json:"externalId"`
	Name       string `json:"name"`
}

// CodecDoc is the audio codec of a title.
type CodecDoc struct {
	Name string `json:"name"`
}

// ApplicableState resolves the transient download state to the persisted one.
//
// A conversion that is staged for download reports the state that was in
// force before it was staged, so the interface always shows the truth.
func ApplicableState(conv *ConversionDoc) EConversionState {
	if conv == nil {
		return StateUnknown
	}
	if EConversionState(conv.State) == StateDownload && conv.PersistState >= 0 {
		return EConversionState(conv.PersistState)
	}
	return EConversionState(conv.State)
}

// Stage marks a conversion as staged for download, remembering its current
// state so it can be restored when it is unstaged.
func (c *ConversionDoc) Stage() {
	if EConversionState(c.State) == StateDownload {
		return
	}
	c.PersistState = c.State
	c.State = int(StateDownload)
}

// Unstage restores the state that was in force before the conversion was
// staged.
func (c *ConversionDoc) Unstage() {
	if EConversionState(c.State) != StateDownload {
		return
	}
	if c.PersistState >= 0 {
		c.State = c.PersistState
	}
	c.PersistState = -1
}

// ApplicableDownloadQuality returns the quality of a book.
//
// For a multi-part book it is the minimum quality across the components,
// because a title is only as good as its weakest part.
func (b *BookDoc) ApplicableDownloadQuality() int {
	q := b.DownloadQuality
	for _, c := range b.Components {
		if c.DownloadQuality < q {
			q = c.DownloadQuality
		}
	}
	return q
}

// ApplicableState returns the effective state of a book.
//
// For a multi-part book it is the minimum of its components' states.
func (b *BookDoc) ApplicableState() EConversionState {
	if len(b.Components) == 0 {
		if b.WholeConversion != nil {
			return ApplicableState(b.WholeConversion)
		}
		return StateUnknown
	}

	min := EConversionState(1<<31 - 1)
	for i := range b.Components {
		c := b.Components[i]
		if c.Conversion == nil {
			continue
		}
		s := ApplicableState(c.Conversion)
		if s < min {
			min = s
		}
	}
	if min == EConversionState(1<<31-1) {
		return StateUnknown
	}
	return min
}

// Conversions returns every conversion of the book, whole book first and
// then the parts in order.
func (b *BookDoc) Conversions() []*ConversionDoc {
	var out []*ConversionDoc
	if b.WholeConversion != nil {
		out = append(out, b.WholeConversion)
	}
	for i := range b.Components {
		if b.Components[i].Conversion != nil {
			out = append(out, b.Components[i].Conversion)
		}
	}
	return out
}

// NewConversion returns a conversion in the remote state.
func NewConversion() *ConversionDoc {
	return &ConversionDoc{
		State:        int(StateRemote),
		PersistState: -1,
		LastUpdate:   time.Now(),
	}
}

// NewBook returns a book with the given ASIN.
func NewBook(asin string) *BookDoc {
	return &BookDoc{
		V:               DocumentVersion,
		Asin:            asin,
		DownloadQuality: int(DownloadQualityHigh),
		WholeConversion: NewConversion(),
	}
}

// Clone returns a deep copy of the book.
func (b *BookDoc) Clone() *BookDoc {
	if b == nil {
		return nil
	}
	data, err := json.Marshal(b)
	if err != nil {
		return nil
	}
	var out BookDoc
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return &out
}

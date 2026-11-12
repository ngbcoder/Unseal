package export

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/ngbcoder/Unseal/internal/db"
	"github.com/ngbcoder/Unseal/internal/util"
)

// Input is what the export stage needs for one book or one part.
type Input struct {
	// Asin is the title's ASIN.
	Asin string
	// Title is the title.
	Title string
	// Sku and SkuLite are the Audible SKUs.
	Sku, SkuLite string
	// RuntimeSeconds is the runtime in seconds.
	RuntimeSeconds int
	// AdultProduct says whether the title is adult content.
	AdultProduct bool
	// Authors is the list of author names.
	Authors []string
	// HasChildren says whether the title has parts.
	HasChildren bool
	// Series is the list of series this title belongs to, with the
	// ordered list of books in each.
	Series []SeriesInput
	// ChapterInfo carries the chapters and the brand durations.
	ChapterInfo *ChapterInfoInput
	// DownloadFileName is the base name (without extension) of the
	// decrypted file.
	DownloadFileName string
	// PartNumber is 1-based for a part, 0 for the whole book.
	PartNumber int
}

// SeriesInput describes one series and its books.
type SeriesInput struct {
	// SeriesAsin is the ASIN of the series.
	SeriesAsin string
	// SeriesTitle is the title of the series.
	SeriesTitle string
	// Books is the ordered list of books in the series.
	Books []SeriesBookInput
	// Sequence is this book's sequence in the series.
	Sequence string
	// BookNumber is the parsed book number.
	BookNumber int
	// SubNumber is the parsed sub-number.
	SubNumber int
	// Sort is the optional sort index.
	Sort int
}

// SeriesBookInput describes one book in a series.
type SeriesBookInput struct {
	Asin           string
	Title          string
	Sku            string
	SkuLite        string
	RuntimeSeconds int
	AdultProduct   bool
	Authors        []string
	Series         []SeriesInput
	Sequence       string
	BookNumber     int
	SubNumber      int
	Sort           int
}

// ChapterInfoInput carries the chapters and the brand durations.
type ChapterInfoInput struct {
	BrandIntroDurationMs int
	BrandOutroDurationMs int
	RuntimeLengthMs      int
	IsAccurate           bool
	Chapters             []ChapterInput
}

// ChapterInput is one entry in the chapter tree.
type ChapterInput struct {
	Title         string
	LengthMs      int
	StartOffsetMs int
	Children      []ChapterInput
}

// chapterExtract is the minimal information kept for de-duplication.
type chapterExtract struct {
	Title  string
	Length int
}

// Exporter exports a decrypted book to the AAX interchange format.
type Exporter struct {
	ExportDir string

	mu           sync.Mutex
	accuChapters [][]chapterExtract
}

// NewExporter creates an exporter.
func NewExporter(exportDir string) *Exporter {
	return &Exporter{ExportDir: exportDir}
}

// Export exports one part of a book.
//
// skipSeries says whether the series sidecars should be skipped, which is
// the case for every part after the first.
//
// onState is called with the new state (converting, then exported or
// conversion_error).
func (e *Exporter) Export(input *Input, skipSeries bool, onState func(state int)) error {
	if input == nil {
		return fmt.Errorf("export: nil input")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if onState != nil {
		onState(int(db.StateConverting))
	}

	// 1. Copy the decrypted file.
	sourceFile := util.AsUncIfLong(input.DownloadFileName + ".m4b")
	if _, err := os.Stat(sourceFile); err != nil {
		if onState != nil {
			onState(int(db.StateConversionError))
		}
		return fmt.Errorf("export: the decrypted file is missing: %w", err)
	}

	destFile := util.AsUncIfLong(filepath.Join(e.ExportDir, input.DownloadFileName+".aax"))
	// The destination name is the base name without the directory, plus .aax.
	destFile = util.AsUncIfLong(filepath.Join(e.ExportDir, baseName(input.DownloadFileName)+".aax"))

	if err := os.MkdirAll(e.ExportDir, 0o755); err != nil {
		if onState != nil {
			onState(int(db.StateConversionError))
		}
		return fmt.Errorf("export: cannot create the export directory: %w", err)
	}

	ok := copyFile(sourceFile, destFile, func(pm int) {})
	if !ok {
		if onState != nil {
			onState(int(db.StateConversionError))
		}
		return fmt.Errorf("export: cannot copy %s to %s", sourceFile, destFile)
	}

	// 2. Write the chapter metadata.
	if input.ChapterInfo != nil {
		if err := e.exportChapters(input); err != nil {
			if onState != nil {
				onState(int(db.StateConversionError))
			}
			return err
		}
	}

	// 3. Write the product record.
	if err := e.exportProduct(input); err != nil {
		if onState != nil {
			onState(int(db.StateConversionError))
		}
		return err
	}

	// 4. Write the series sidecars, unless this is a later part.
	if !skipSeries {
		if err := e.exportSeries(input); err != nil {
			if onState != nil {
				onState(int(db.StateConversionError))
			}
			return err
		}
	}

	if onState != nil {
		onState(int(db.StateExported))
	}
	return nil
}

// ResetAccuChapters clears the accumulator for a new book.
func (e *Exporter) ResetAccuChapters() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.accuChapters = nil
}

// exportChapters writes content_metadata_<ASIN>.json.
func (e *Exporter) exportChapters(input *Input) error {
	ci := input.ChapterInfo

	// Flatten the chapter tree.
	flattened := flattenChapters(ci.Chapters)

	// De-duplicate the leading chapter if it repeats a brand intro.
	var chapters []chapterJSON
	for i, ch := range flattened {
		if i == 0 && len(e.accuChapters) >= 2 && skipChapter(ch, e.accuChapters) {
			continue
		}
		chapters = append(chapters, chapterJSON{
			LengthMs:       ch.LengthMs,
			StartOffsetMs:  ch.StartOffsetMs,
			StartOffsetSec: ch.StartOffsetMs / 1000,
			Title:          ch.Title,
		})
	}

	// Update the accumulator.
	accuPart := make([][]chapterExtract, 1)
	accuPart[0] = make([]chapterExtract, 0, len(flattened))
	for _, ch := range flattened {
		accuPart[0] = append(accuPart[0], chapterExtract{Title: ch.Title, Length: ch.LengthMs})
	}
	e.updateAccuChapters(accuPart)

	container := metadataContainer{
		ContentMetadata: contentMetadata{
			ChapterInfo: chapterInfoJSON{
				BrandIntroDurationMs: ci.BrandIntroDurationMs,
				BrandOutroDurationMs: ci.BrandOutroDurationMs,
				IsAccurate:           ci.IsAccurate,
				RuntimeLengthMs:      ci.RuntimeLengthMs,
				RuntimeLengthSec:     ci.RuntimeLengthMs / 1000,
				Chapters:             chapters,
			},
			ContentReference: contentReferenceJSON{
				ASIN:        input.Asin,
				ContentSize: 0,
				Sku:         input.Sku,
			},
		},
	}

	data, err := json.Marshal(container)
	if err != nil {
		return fmt.Errorf("export: cannot serialise the chapters: %w", err)
	}

	filename := "content_metadata_" + input.Asin + ".json"
	outpath := util.AsUncIfLong(filepath.Join(e.ExportDir, filename))
	return os.WriteFile(outpath, data, 0o644)
}

// exportProduct writes <ASIN>.json.
func (e *Exporter) exportProduct(input *Input) error {
	product := productJSON{
		ASIN:             input.Asin,
		Title:            input.Title,
		Sku:              input.Sku,
		SkuLite:          input.SkuLite,
		IsListenable:     true,
		RuntimeLengthMin: input.RuntimeSeconds / 60,
		HasChildren:      input.HasChildren,
		IsAdultProduct:   input.AdultProduct,
	}

	if len(input.Authors) > 0 {
		for _, a := range input.Authors {
			product.Authors = append(product.Authors, authorJSON{Name: a})
		}
	}

	if len(input.Series) > 0 {
		for _, s := range input.Series {
			product.Series = append(product.Series, seriesJSON{
				ASIN:     s.SeriesAsin,
				Title:    s.SeriesTitle,
				Sequence: s.Sequence,
			})
		}
	}

	container := productResponse{Product: product}
	data, err := json.Marshal(container)
	if err != nil {
		return fmt.Errorf("export: cannot serialise the product: %w", err)
	}

	filename := input.Asin + ".json"
	outpath := util.AsUncIfLong(filepath.Join(e.ExportDir, filename))
	return os.WriteFile(outpath, data, 0o644)
}

// exportSeries writes series_titles_<SERIES_ASIN>.json for every series.
func (e *Exporter) exportSeries(input *Input) error {
	for _, s := range input.Series {
		if s.SeriesAsin == "" {
			continue
		}

		// Sort the books according to the configured preference.
		books := make([]SeriesBookInput, len(s.Books))
		copy(books, s.Books)

		if len(books) > 0 && books[0].Sort != 0 {
			// All books have a sort value: use it.
			sortBooks(books, func(a, b SeriesBookInput) bool { return a.Sort < b.Sort })
		} else if len(books) > 0 && books[0].BookNumber != 0 {
			// Use book number and sub-number.
			sortBooks(books, func(a, b SeriesBookInput) bool {
				if a.BookNumber != b.BookNumber {
					return a.BookNumber < b.BookNumber
				}
				return a.SubNumber < b.SubNumber
			})
		} else {
			// Fall back to the sequence string.
			sortBooks(books, func(a, b SeriesBookInput) bool { return a.Sequence < b.Sequence })
		}

		var products []productJSON
		for _, b := range books {
			p := productJSON{
				ASIN:             b.Asin,
				Title:            b.Title,
				Sku:              b.Sku,
				SkuLite:          b.SkuLite,
				IsListenable:     true,
				RuntimeLengthMin: b.RuntimeSeconds / 60,
				IsAdultProduct:   b.AdultProduct,
			}
			if len(b.Authors) > 0 {
				for _, a := range b.Authors {
					p.Authors = append(p.Authors, authorJSON{Name: a})
				}
			}
			if len(b.Series) > 0 {
				for _, s := range b.Series {
					p.Series = append(p.Series, seriesJSON{
						ASIN:     s.SeriesAsin,
						Title:    s.SeriesTitle,
						Sequence: s.Sequence,
					})
				}
			}
			products = append(products, p)
		}

		container := seriesTitles{SimilarProducts: products}
		data, err := json.Marshal(container)
		if err != nil {
			return fmt.Errorf("export: cannot serialise the series: %w", err)
		}

		filename := "series_titles_" + s.SeriesAsin + ".json"
		outpath := util.AsUncIfLong(filepath.Join(e.ExportDir, filename))
		if err := os.WriteFile(outpath, data, 0o644); err != nil {
			return fmt.Errorf("export: cannot write %s: %w", outpath, err)
		}
	}
	return nil
}

// skipChapter reports whether a leading chapter should be dropped because
// it repeats a brand intro that already appeared in a previous part.
func skipChapter(ch chapterInput, accu [][]chapterExtract) bool {
	if len(accu) < 2 {
		return false
	}
	for i := 0; i < len(accu)-1; i++ {
		for _, ce := range accu[i] {
			if ce.Title == ch.Title && abs(ce.Length-ch.LengthMs) < 1500 && ch.LengthMs < 25000 {
				return true
			}
		}
	}
	return false
}

// updateAccuChapters merges the chapters of one part into the accumulator.
func (e *Exporter) updateAccuChapters(accuPart [][]chapterExtract) {
	for i := 0; i < len(accuPart); i++ {
		if len(e.accuChapters) < i+1 {
			e.accuChapters = append(e.accuChapters, []chapterExtract{})
		}
		e.accuChapters[i] = append(e.accuChapters[i], accuPart[i]...)
	}
}

// flattenChapters flattens a chapter tree into a list.
func flattenChapters(chapters []ChapterInput) []chapterInput {
	var out []chapterInput
	var walk func([]ChapterInput)
	walk = func(cs []ChapterInput) {
		for _, c := range cs {
			out = append(out, chapterInput{
				Title:         c.Title,
				LengthMs:      c.LengthMs,
				StartOffsetMs: c.StartOffsetMs,
			})
			walk(c.Children)
		}
	}
	walk(chapters)
	return out
}

// chapterInput is the internal representation of a chapter.
type chapterInput struct {
	Title         string
	LengthMs      int
	StartOffsetMs int
}

// copyFile copies a file with cancellation support.
func copyFile(src, dst string, onProgress func(int)) bool {
	in, err := os.Open(src)
	if err != nil {
		return false
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return false
	}
	defer out.Close()

	buf := make([]byte, 10*1024*1024) // 10 MB blocks
	for {
		n, rerr := in.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return false
			}
			if onProgress != nil {
				onProgress(n)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return false
		}
	}

	return true
}

// baseName returns the file name without its directory.
func baseName(path string) string {
	return filepath.Base(path)
}

// sortBooks sorts a slice of SeriesBookInput in place.
func sortBooks(books []SeriesBookInput, less func(a, b SeriesBookInput) bool) {
	// Simple insertion sort for determinism.
	for i := 1; i < len(books); i++ {
		j := i
		for j > 0 && less(books[j], books[j-1]) {
			books[j], books[j-1] = books[j-1], books[j]
			j--
		}
	}
}

// abs returns the absolute value of an int.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// --- JSON types ---

type metadataContainer struct {
	ContentMetadata contentMetadata `json:"content_metadata"`
}

type contentMetadata struct {
	ChapterInfo      chapterInfoJSON      `json:"chapter_info"`
	ContentReference contentReferenceJSON `json:"content_reference"`
}

type chapterInfoJSON struct {
	BrandIntroDurationMs int           `json:"brandIntroDurationMs"`
	BrandOutroDurationMs int           `json:"brandOutroDurationMs"`
	IsAccurate           bool          `json:"is_accurate"`
	RuntimeLengthMs      int           `json:"runtime_length_ms"`
	RuntimeLengthSec     int           `json:"runtime_length_sec"`
	Chapters             []chapterJSON `json:"chapters,omitempty"`
}

type chapterJSON struct {
	LengthMs       int    `json:"length_ms"`
	StartOffsetMs  int    `json:"start_offset_ms"`
	StartOffsetSec int    `json:"start_offset_sec"`
	Title          string `json:"title"`
}

type contentReferenceJSON struct {
	ASIN        string `json:"asin"`
	ContentSize int64  `json:"content_size_in_bytes"`
	Sku         string `json:"sku"`
}

type productResponse struct {
	Product productJSON `json:"product"`
}

type productJSON struct {
	ASIN             string       `json:"asin"`
	Title            string       `json:"title"`
	Sku              string       `json:"sku"`
	SkuLite          string       `json:"sku_lite"`
	IsListenable     bool         `json:"is_listenable"`
	RuntimeLengthMin int          `json:"runtime_length_min"`
	HasChildren      bool         `json:"has_children"`
	IsAdultProduct   bool         `json:"is_adult_product"`
	Authors          []authorJSON `json:"authors,omitempty"`
	Series           []seriesJSON `json:"series,omitempty"`
}

type authorJSON struct {
	ASIN string `json:"asin,omitempty"`
	Name string `json:"name"`
}

type seriesJSON struct {
	ASIN     string `json:"asin"`
	Title    string `json:"title"`
	Sequence string `json:"sequence"`
}

type seriesTitles struct {
	SimilarProducts []productJSON `json:"similar_products"`
}

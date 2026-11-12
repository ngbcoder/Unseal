package db_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ngbcoder/Unseal/internal/db"
	"github.com/ngbcoder/Unseal/internal/types"
)

func newStore(t *testing.T) *db.Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "library.db")
	store, err := db.Open(path)
	if err != nil {
		t.Fatalf("cannot open the store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestOpenCreatesTheDatabaseFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library.db")

	store, err := db.Open(path)
	if err != nil {
		t.Fatalf("cannot open the store: %v", err)
	}
	defer store.Close()

	if _, err := os.Stat(path); err != nil {
		t.Errorf("the database file was not created: %v", err)
	}
}

func TestPutAndGetRoundTripsABook(t *testing.T) {
	store := newStore(t)

	book := db.NewBook("B0TESTASIN")
	book.Title = "Test Book"
	book.DownloadQuality = int(db.DownloadQualityHigh)

	if err := store.PutBook("account1", types.RegionUS, book); err != nil {
		t.Fatalf("PutBook failed: %v", err)
	}

	got, err := store.GetBook("account1", types.RegionUS, "B0TESTASIN")
	if err != nil {
		t.Fatalf("GetBook failed: %v", err)
	}
	if got == nil {
		t.Fatal("the book was not stored")
	}
	if got.Title != "Test Book" {
		t.Errorf("the title is %q, want %q", got.Title, "Test Book")
	}
	if got.Asin != "B0TESTASIN" {
		t.Errorf("the asin is %q, want %q", got.Asin, "B0TESTASIN")
	}
}

func TestGetBookReturnsNilForAMissingBook(t *testing.T) {
	store := newStore(t)

	got, err := store.GetBook("account1", types.RegionUS, "B0MISSING")
	if err != nil {
		t.Fatalf("GetBook failed: %v", err)
	}
	if got != nil {
		t.Errorf("a missing book must return nil, got %+v", got)
	}
}

func TestListBooksReturnsEveryBook(t *testing.T) {
	store := newStore(t)

	for _, asin := range []string{"B0A", "B0B", "B0C"} {
		book := db.NewBook(asin)
		book.Title = "Book " + asin
		if err := store.PutBook("account1", types.RegionUS, book); err != nil {
			t.Fatalf("PutBook failed: %v", err)
		}
	}

	books, err := store.ListBooks("account1", types.RegionUS)
	if err != nil {
		t.Fatalf("ListBooks failed: %v", err)
	}
	if len(books) != 3 {
		t.Fatalf("expected 3 books, got %d", len(books))
	}
}

func TestTheTenancyIsIsolated(t *testing.T) {
	store := newStore(t)

	book := db.NewBook("B0SHARED")
	book.Title = "Shared Title"
	if err := store.PutBook("account1", types.RegionUS, book); err != nil {
		t.Fatalf("PutBook failed: %v", err)
	}

	// A different account must not see the book.
	got, err := store.GetBook("account2", types.RegionUS, "B0SHARED")
	if err != nil {
		t.Fatalf("GetBook failed: %v", err)
	}
	if got != nil {
		t.Error("a book of one account must not be visible to another")
	}

	// A different region must not see it either.
	got, err = store.GetBook("account1", types.RegionUK, "B0SHARED")
	if err != nil {
		t.Fatalf("GetBook failed: %v", err)
	}
	if got != nil {
		t.Error("a book of one region must not be visible in another")
	}
}

func TestListConversionsExtractsThemFromTheBooks(t *testing.T) {
	store := newStore(t)

	book := db.NewBook("B0MULTI")
	book.Title = "Multi Part"
	book.Components = []db.ComponentDoc{
		{Asin: "B0MULTI-P1", Title: "Part 1", PartNumber: 1, Conversion: db.NewConversion()},
		{Asin: "B0MULTI-P2", Title: "Part 2", PartNumber: 2, Conversion: db.NewConversion()},
	}
	if err := store.PutBook("account1", types.RegionUS, book); err != nil {
		t.Fatalf("PutBook failed: %v", err)
	}

	convs, err := store.ListConversions("account1", types.RegionUS)
	if err != nil {
		t.Fatalf("ListConversions failed: %v", err)
	}

	// Whole book + 2 parts = 3 conversions.
	if len(convs) != 3 {
		t.Fatalf("expected 3 conversions, got %d", len(convs))
	}
}

func TestUpdateWholeConversion(t *testing.T) {
	store := newStore(t)

	book := db.NewBook("B0UPDATE")
	if err := store.PutBook("account1", types.RegionUS, book); err != nil {
		t.Fatalf("PutBook failed: %v", err)
	}

	conv := db.NewConversion()
	conv.State = int(db.StateLocalLocked)
	conv.DownloadFileName = "Pride and Prejudice_B0UPDATE_LC"

	if err := store.UpdateWholeConversion("account1", types.RegionUS, "B0UPDATE", conv); err != nil {
		t.Fatalf("UpdateWholeConversion failed: %v", err)
	}

	got, _ := store.GetBook("account1", types.RegionUS, "B0UPDATE")
	if got.WholeConversion == nil {
		t.Fatal("the conversion was not stored")
	}
	if got.WholeConversion.State != int(db.StateLocalLocked) {
		t.Errorf("the state is %d, want %d", got.WholeConversion.State, db.StateLocalLocked)
	}
	if got.WholeConversion.DownloadFileName != "Pride and Prejudice_B0UPDATE_LC" {
		t.Errorf("the file name is %q", got.WholeConversion.DownloadFileName)
	}
}

func TestUpdateComponentConversion(t *testing.T) {
	store := newStore(t)

	book := db.NewBook("B0COMP")
	book.Components = []db.ComponentDoc{
		{Asin: "B0COMP-P1", Title: "Part 1", PartNumber: 1, Conversion: db.NewConversion()},
	}
	if err := store.PutBook("account1", types.RegionUS, book); err != nil {
		t.Fatalf("PutBook failed: %v", err)
	}

	conv := db.NewConversion()
	conv.State = int(db.StateLocalUnlocked)

	if err := store.UpdateComponentConversion("account1", types.RegionUS, "B0COMP", 0, conv); err != nil {
		t.Fatalf("UpdateComponentConversion failed: %v", err)
	}

	got, _ := store.GetBook("account1", types.RegionUS, "B0COMP")
	if got.Components[0].Conversion.State != int(db.StateLocalUnlocked) {
		t.Errorf("the component state is %d, want %d", got.Components[0].Conversion.State, db.StateLocalUnlocked)
	}
}

func TestUpdateComponentConversionRejectsAnOutOfRangeIndex(t *testing.T) {
	store := newStore(t)

	book := db.NewBook("B0COMP2")
	if err := store.PutBook("account1", types.RegionUS, book); err != nil {
		t.Fatalf("PutBook failed: %v", err)
	}

	conv := db.NewConversion()
	if err := store.UpdateComponentConversion("account1", types.RegionUS, "B0COMP2", 5, conv); err != nil {
		t.Fatalf("out of range must not be an error: %v", err)
	}
}

func TestUpdateLicense(t *testing.T) {
	store := newStore(t)

	book := db.NewBook("B0LIC")
	book.Components = []db.ComponentDoc{
		{Asin: "B0LIC-P1", Title: "Part 1", PartNumber: 1},
	}
	if err := store.PutBook("account1", types.RegionUS, book); err != nil {
		t.Fatalf("PutBook failed: %v", err)
	}

	if err := store.UpdateLicense("account1", types.RegionUS, "B0LIC", -1, "bookkey", "bookiv"); err != nil {
		t.Fatalf("UpdateLicense (book) failed: %v", err)
	}
	if err := store.UpdateLicense("account1", types.RegionUS, "B0LIC", 0, "partkey", "partiv"); err != nil {
		t.Fatalf("UpdateLicense (part) failed: %v", err)
	}

	got, _ := store.GetBook("account1", types.RegionUS, "B0LIC")
	if got.LicenseKey != "bookkey" || got.LicenseIv != "bookiv" {
		t.Errorf("the book licence is %q/%q", got.LicenseKey, got.LicenseIv)
	}
	if got.Components[0].LicenseKey != "partkey" || got.Components[0].LicenseIv != "partiv" {
		t.Errorf("the part licence is %q/%q", got.Components[0].LicenseKey, got.Components[0].LicenseIv)
	}
}

func TestUpdateDownloadFileName(t *testing.T) {
	store := newStore(t)

	book := db.NewBook("B0FILE")
	if err := store.PutBook("account1", types.RegionUS, book); err != nil {
		t.Fatalf("PutBook failed: %v", err)
	}

	if err := store.UpdateDownloadFileName("account1", types.RegionUS, "B0FILE", -1, "Title_B0FILE_LC_64_22050"); err != nil {
		t.Fatalf("UpdateDownloadFileName failed: %v", err)
	}

	got, _ := store.GetBook("account1", types.RegionUS, "B0FILE")
	if got.WholeConversion.DownloadFileName != "Title_B0FILE_LC_64_22050" {
		t.Errorf("the file name is %q", got.WholeConversion.DownloadFileName)
	}
}

func TestSetDeleted(t *testing.T) {
	store := newStore(t)

	book := db.NewBook("B0DEL")
	if err := store.PutBook("account1", types.RegionUS, book); err != nil {
		t.Fatalf("PutBook failed: %v", err)
	}

	if err := store.SetDeleted("account1", types.RegionUS, "B0DEL", true); err != nil {
		t.Fatalf("SetDeleted failed: %v", err)
	}

	got, _ := store.GetBook("account1", types.RegionUS, "B0DEL")
	if !got.Deleted {
		t.Error("the book must be marked deleted")
	}

	if err := store.SetDeleted("account1", types.RegionUS, "B0DEL", false); err != nil {
		t.Fatalf("SetDeleted failed: %v", err)
	}

	got, _ = store.GetBook("account1", types.RegionUS, "B0DEL")
	if got.Deleted {
		t.Error("the book must be unmarked")
	}
}

func TestHardDeleteRemovesTheAccountBucket(t *testing.T) {
	store := newStore(t)

	book := db.NewBook("B0HARD")
	if err := store.PutBook("account1", types.RegionUS, book); err != nil {
		t.Fatalf("PutBook failed: %v", err)
	}

	if err := store.HardDelete("account1", types.RegionUS); err != nil {
		t.Fatalf("HardDelete failed: %v", err)
	}

	books, err := store.ListBooks("account1", types.RegionUS)
	if err != nil {
		t.Fatalf("ListBooks failed: %v", err)
	}
	if len(books) != 0 {
		t.Errorf("expected no books, got %d", len(books))
	}
}

func TestListAccountKeys(t *testing.T) {
	store := newStore(t)

	for _, region := range []types.Region{types.RegionUS, types.RegionUK} {
		book := db.NewBook("B0KEYS")
		if err := store.PutBook("account1", region, book); err != nil {
			t.Fatalf("PutBook failed: %v", err)
		}
	}

	keys, err := store.ListAccountKeys()
	if err != nil {
		t.Fatalf("ListAccountKeys failed: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 account keys, got %d: %v", len(keys), keys)
	}
}

func TestMetaRoundTrips(t *testing.T) {
	store := newStore(t)

	if err := store.SetMeta("version", "1"); err != nil {
		t.Fatalf("SetMeta failed: %v", err)
	}

	got, err := store.Meta("version")
	if err != nil {
		t.Fatalf("Meta failed: %v", err)
	}
	if got != "1" {
		t.Errorf("the value is %q, want %q", got, "1")
	}

	missing, err := store.Meta("does-not-exist")
	if err != nil {
		t.Fatalf("Meta failed: %v", err)
	}
	if missing != "" {
		t.Errorf("a missing key must return the empty string, got %q", missing)
	}
}

func TestApplicableStateOfAConversion(t *testing.T) {
	conv := db.NewConversion()
	if db.ApplicableState(conv) != db.StateRemote {
		t.Errorf("a fresh conversion must report remote")
	}

	conv.Stage()
	if db.ApplicableState(conv) != db.StateRemote {
		t.Errorf("a staged conversion must report its persisted state")
	}

	conv.Unstage()
	if db.ApplicableState(conv) != db.StateRemote {
		t.Errorf("an unstaged conversion must report its current state")
	}
}

func TestApplicableStateOfANilConversion(t *testing.T) {
	if db.ApplicableState(nil) != db.StateUnknown {
		t.Error("a nil conversion must report unknown")
	}
}

func TestApplicableDownloadQuality(t *testing.T) {
	book := db.NewBook("B0Q")
	book.DownloadQuality = 3 // High
	book.Components = []db.ComponentDoc{
		{Asin: "P1", DownloadQuality: 1}, // Normal
		{Asin: "P2", DownloadQuality: 2}, // High
	}

	// The weakest part wins.
	if got := book.ApplicableDownloadQuality(); got != 1 {
		t.Errorf("the quality is %d, want 1 (Normal)", got)
	}
}

func TestApplicableDownloadQualityOfASinglePart(t *testing.T) {
	book := db.NewBook("B0Q2")
	book.DownloadQuality = 2 // High

	if got := book.ApplicableDownloadQuality(); got != 2 {
		t.Errorf("the quality is %d, want 2", got)
	}
}

func TestApplicableStateOfAMultiPartBook(t *testing.T) {
	book := db.NewBook("B0M")
	book.Components = []db.ComponentDoc{
		{Asin: "P1", Conversion: &db.ConversionDoc{State: int(db.StateLocalLocked)}},
		{Asin: "P2", Conversion: &db.ConversionDoc{State: int(db.StateRemote)}},
	}

	// The least advanced part wins.
	if got := book.ApplicableState(); got != db.StateRemote {
		t.Errorf("the state is %v, want remote", got)
	}
}

func TestApplicableStateOfASinglePartBook(t *testing.T) {
	book := db.NewBook("B0S")
	book.WholeConversion = &db.ConversionDoc{State: int(db.StateLocalUnlocked)}

	if got := book.ApplicableState(); got != db.StateLocalUnlocked {
		t.Errorf("the state is %v, want local_unlocked", got)
	}
}

func TestApplicableStateOfABookWithNoConversion(t *testing.T) {
	book := db.NewBook("B0N")
	book.WholeConversion = nil
	book.Components = nil

	if got := book.ApplicableState(); got != db.StateUnknown {
		t.Errorf("the state is %v, want unknown", got)
	}
}

func TestConversionsReturnsWholeThenParts(t *testing.T) {
	book := db.NewBook("B0C")
	book.Components = []db.ComponentDoc{
		{Asin: "P1", Conversion: &db.ConversionDoc{State: int(db.StateRemote)}},
		{Asin: "P2", Conversion: &db.ConversionDoc{State: int(db.StateRemote)}},
	}

	convs := book.Conversions()

	if len(convs) != 3 {
		t.Fatalf("expected 3 conversions, got %d", len(convs))
	}
}

func TestCloneIsIndependent(t *testing.T) {
	book := db.NewBook("B0CLONE")
	book.Title = "Original"

	clone := book.Clone()
	clone.Title = "Clone"

	if book.Title != "Original" {
		t.Error("the clone must not share state with the original")
	}
}

func TestStageAndUnstageRoundTrip(t *testing.T) {
	conv := db.NewConversion()
	conv.State = int(db.StateLocalLocked)

	conv.Stage()

	if db.ApplicableState(conv) != db.StateLocalLocked {
		t.Errorf("a staged conversion must report its persisted state")
	}
	if db.EConversionState(conv.State) != db.StateDownload {
		t.Errorf("the live state must be download, got %v", db.EConversionState(conv.State))
	}

	conv.Unstage()

	if db.ApplicableState(conv) != db.StateLocalLocked {
		t.Errorf("an unstaged conversion must report its current state")
	}
	if db.EConversionState(conv.State) != db.StateLocalLocked {
		t.Errorf("the live state must be restored, got %v", db.EConversionState(conv.State))
	}
}

func TestUnstageIsIdempotent(t *testing.T) {
	conv := db.NewConversion()
	conv.State = int(db.StateRemote)

	conv.Unstage()
	conv.Unstage()

	if db.ApplicableState(conv) != db.StateRemote {
		t.Errorf("unstaging an unstaged conversion must not change its state")
	}
}

func TestNewBookHasARemoteConversion(t *testing.T) {
	book := db.NewBook("B0NEW")

	if book.WholeConversion == nil {
		t.Fatal("a new book must have a whole-book conversion")
	}
	if db.ApplicableState(book.WholeConversion) != db.StateRemote {
		t.Errorf("a new book's conversion must be remote")
	}
}

func TestDocumentVersionIsSetOnPut(t *testing.T) {
	store := newStore(t)

	book := &db.BookDoc{Asin: "B0V"}
	if err := store.PutBook("account1", types.RegionUS, book); err != nil {
		t.Fatalf("PutBook failed: %v", err)
	}

	got, _ := store.GetBook("account1", types.RegionUS, "B0V")
	if got.V != db.DocumentVersion {
		t.Errorf("the document version is %d, want %d", got.V, db.DocumentVersion)
	}
}

func TestTimeIsPreservedAcrossARoundTrip(t *testing.T) {
	store := newStore(t)

	now := time.Date(2026, 11, 11, 20, 15, 0, 0, time.UTC)
	book := db.NewBook("B0TIME")
	book.PurchaseDate = now
	if err := store.PutBook("account1", types.RegionUS, book); err != nil {
		t.Fatalf("PutBook failed: %v", err)
	}

	got, _ := store.GetBook("account1", types.RegionUS, "B0TIME")
	if !got.PurchaseDate.Equal(now) {
		t.Errorf("the purchase date is %v, want %v", got.PurchaseDate, now)
	}
}

package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ngbcoder/Unseal/internal/config"
	"github.com/ngbcoder/Unseal/internal/types"
)

func TestDefaults(t *testing.T) {
	s := config.NewUserSettings()

	if !s.Download.AutoUpdateLibrary() {
		t.Error("the library sync must be on by default")
	}
	if !s.Config.EncryptConfiguration() {
		t.Error("the configuration must be encrypted by default")
	}
	if s.Export.ExportToAax() != nil {
		t.Error("the export decision must be undecided by default")
	}
	if s.Download.DownloadQuality() != config.QualityHigh {
		t.Errorf("the quality is %v, want High", s.Download.DownloadQuality())
	}
	if s.Update.OnlineUpdate() != config.UpdatePromptForDownload {
		t.Errorf("the update policy is %v, want promptForDownload", s.Update.OnlineUpdate())
	}
	if s.Download.InitialSorting() != config.SortStateDate {
		t.Errorf("the sorting is %v, want state_date", s.Download.InitialSorting())
	}
}

func TestAnyChangeFiresTheCallback(t *testing.T) {
	s := config.NewUserSettings()

	changed := 0
	s.OnChange(func() { changed++ })

	s.Download.SetAutoRefresh(true)
	s.Download.SetDownloadDirectory(`C:\Music\Audible`)
	s.Download.SetDownloadQuality(config.QualityExtreme)
	s.Config.SetEncryptConfiguration(false)
	s.Export.SetExportDirectory(`C:\Music\Export`)
	s.Update.SetOnlineUpdate(config.UpdateNo)

	if changed != 6 {
		t.Errorf("the callback ran %d times, want 6", changed)
	}
}

func TestSettingTheSameValueDoesNotFireTheCallback(t *testing.T) {
	s := config.NewUserSettings()

	changed := 0
	s.OnChange(func() { changed++ })

	s.Download.SetAutoRefresh(true)
	s.Download.SetAutoRefresh(true)

	if changed != 1 {
		t.Errorf("the callback ran %d times, want 1", changed)
	}
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := config.NewStore(dir, filepath.Join(dir, "app"))

	s := config.NewUserSettings()
	s.Download.SetAutoRefresh(true)
	s.Download.SetIncludeAdultProducts(true)
	s.Download.SetMultiPartDownload(true)
	s.Download.SetKeepEncryptedFiles(true)
	s.Download.SetHideUnavailableProducts(true)
	s.Download.SetAutoOpenDownloadDialog(true)
	s.Download.SetDownloadQuality(config.QualityExtreme)
	s.Download.SetDownloadDirectory(`C:\Music\Audible\Download`)
	s.Download.SetInitialSorting(config.SortAuthorDate)
	s.Download.SetProfile(types.ProfileAliasKey{Region: types.RegionDE, AccountAlias: "Nolan"})
	s.Config.SetEncryptConfiguration(false)
	s.Export.SetExportDirectory(`C:\Music\Audible\Export`)
	on := true
	s.Export.SetExportToAax(&on)
	s.Update.SetOnlineUpdate(config.UpdatePromptForInstall)
	s.Update.AddUpdateNote("0.14.1")

	if err := store.SaveUser(s); err != nil {
		t.Fatalf("SaveUser failed: %v", err)
	}

	loaded, err := store.LoadUser()
	if err != nil {
		t.Fatalf("LoadUser failed: %v", err)
	}

	if !loaded.Download.AutoRefresh() {
		t.Error("autoRefresh was lost")
	}
	if !loaded.Download.IncludeAdultProducts() {
		t.Error("includeAdultProducts was lost")
	}
	if !loaded.Download.MultiPartDownload() {
		t.Error("multiPartDownload was lost")
	}
	if !loaded.Download.KeepEncryptedFiles() {
		t.Error("keepEncryptedFiles was lost")
	}
	if !loaded.Download.HideUnavailableProducts() {
		t.Error("hideUnavailableProducts was lost")
	}
	if !loaded.Download.AutoOpenDownloadDialog() {
		t.Error("autoOpenDownloadDialog was lost")
	}
	if loaded.Download.DownloadQuality() != config.QualityExtreme {
		t.Errorf("the quality is %v, want Extreme", loaded.Download.DownloadQuality())
	}
	if loaded.Download.DownloadDirectory() != `C:\Music\Audible\Download` {
		t.Errorf("the download directory is %q", loaded.Download.DownloadDirectory())
	}
	if loaded.Download.InitialSorting() != config.SortAuthorDate {
		t.Errorf("the sorting is %v, want author_date", loaded.Download.InitialSorting())
	}
	if loaded.Download.Profile() != (types.ProfileAliasKey{Region: types.RegionDE, AccountAlias: "Nolan"}) {
		t.Errorf("the profile is %v", loaded.Download.Profile())
	}
	if loaded.Config.EncryptConfiguration() {
		t.Error("encryptConfiguration was lost")
	}
	if loaded.Export.ExportDirectory() != `C:\Music\Audible\Export` {
		t.Errorf("the export directory is %q", loaded.Export.ExportDirectory())
	}
	if loaded.Export.ExportToAax() == nil || !*loaded.Export.ExportToAax() {
		t.Error("exportToAax was lost")
	}
	if loaded.Update.OnlineUpdate() != config.UpdatePromptForInstall {
		t.Errorf("the update policy is %v", loaded.Update.OnlineUpdate())
	}
	if !loaded.Update.HasUpdateNote("0.14.1") {
		t.Error("the update note was lost")
	}
}

func TestTheUndecidedExportStateSurvivesARoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := config.NewStore(dir, filepath.Join(dir, "app"))

	s := config.NewUserSettings()
	if err := store.SaveUser(s); err != nil {
		t.Fatalf("SaveUser failed: %v", err)
	}

	loaded, err := store.LoadUser()
	if err != nil {
		t.Fatalf("LoadUser failed: %v", err)
	}

	if loaded.Export.ExportToAax() != nil {
		t.Error("the undecided export state must survive a round trip")
	}
}

func TestTheUserFileWinsOverTheApplicationPreset(t *testing.T) {
	userDir := t.TempDir()
	appDir := t.TempDir()

	store := config.NewStore(userDir, appDir)

	// A preset in the application directory.
	preset := config.NewUserSettings()
	preset.Download.SetDownloadQuality(config.QualityLow)
	if err := store.SaveUserPreset(preset); err != nil {
		t.Fatalf("cannot write the preset: %v", err)
	}

	// The user file wins.
	user := config.NewUserSettings()
	user.Download.SetDownloadQuality(config.QualityExtreme)
	if err := store.SaveUser(user); err != nil {
		t.Fatalf("SaveUser failed: %v", err)
	}

	loaded, err := store.LoadUser()
	if err != nil {
		t.Fatalf("LoadUser failed: %v", err)
	}

	if loaded.Download.DownloadQuality() != config.QualityExtreme {
		t.Errorf("the quality is %v, want Extreme from the user file", loaded.Download.DownloadQuality())
	}
}

func TestTheApplicationPresetIsUsedWhenTheUserFileIsMissing(t *testing.T) {
	userDir := t.TempDir()
	appDir := t.TempDir()

	store := config.NewStore(userDir, appDir)

	preset := config.NewUserSettings()
	preset.Download.SetDownloadQuality(config.QualityLow)
	if err := store.SaveUserPreset(preset); err != nil {
		t.Fatalf("cannot write the preset: %v", err)
	}

	loaded, err := store.LoadUser()
	if err != nil {
		t.Fatalf("LoadUser failed: %v", err)
	}

	if loaded.Download.DownloadQuality() != config.QualityLow {
		t.Errorf("the quality is %v, want Low from the preset", loaded.Download.DownloadQuality())
	}
}

func TestLoadUserFallsBackToTheDefaults(t *testing.T) {
	store := config.NewStore(t.TempDir(), t.TempDir())

	loaded, err := store.LoadUser()
	if err != nil {
		t.Fatalf("LoadUser failed: %v", err)
	}

	if !loaded.Download.AutoUpdateLibrary() {
		t.Error("the defaults must be used when no file exists")
	}
}

func TestAMissingFieldKeepsItsDefault(t *testing.T) {
	dir := t.TempDir()
	store := config.NewStore(dir, filepath.Join(dir, "app"))

	// A file that only sets one field.
	if err := os.WriteFile(filepath.Join(dir, config.UserSettingsFile),
		[]byte(`{"download":{"includeAdultProducts":true}}`), 0o644); err != nil {
		t.Fatalf("cannot write the settings: %v", err)
	}

	loaded, err := store.LoadUser()
	if err != nil {
		t.Fatalf("LoadUser failed: %v", err)
	}

	if !loaded.Download.IncludeAdultProducts() {
		t.Error("the value from the file must be applied")
	}
	if !loaded.Download.AutoUpdateLibrary() {
		t.Error("a field missing from the file must keep its default")
	}
}

func TestSaveIsAtomic(t *testing.T) {
	dir := t.TempDir()
	store := config.NewStore(dir, filepath.Join(dir, "app"))

	s := config.NewUserSettings()
	if err := store.SaveUser(s); err != nil {
		t.Fatalf("SaveUser failed: %v", err)
	}

	// No temporary file must be left behind.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cannot read the settings directory: %v", err)
	}
	for _, e := range entries {
		if e.Name() != config.UserSettingsFile {
			t.Errorf("a stray file was left behind: %s", e.Name())
		}
	}
}

func TestLoadApp(t *testing.T) {
	dir := t.TempDir()
	store := config.NewStore(dir, dir)

	level := 4
	app := &config.AppSettings{LogLevel: &level, DbgOnlineUpdate: true}
	if err := store.SaveApp(app); err != nil {
		t.Fatalf("SaveApp failed: %v", err)
	}

	loaded, err := store.LoadApp(false)
	if err != nil {
		t.Fatalf("LoadApp failed: %v", err)
	}

	if loaded.LogLevelOrDefault(3) != 4 {
		t.Errorf("the log level is %d, want 4", loaded.LogLevelOrDefault(3))
	}
	if !loaded.DbgOnlineUpdate {
		t.Error("dbgOnlineUpdate was lost")
	}
}

func TestLoadAppWhenTheFileIsMissing(t *testing.T) {
	store := config.NewStore(t.TempDir(), t.TempDir())

	if _, err := store.LoadApp(false); err == nil {
		t.Error("a missing file must be an error when it is not optional")
	}

	settings, err := store.LoadApp(true)
	if err != nil {
		t.Fatalf("LoadApp failed: %v", err)
	}
	if settings.LogLevelOrDefault(3) != 3 {
		t.Error("an optional missing file must give the default level")
	}
}

func TestTheEnumsAreStoredByName(t *testing.T) {
	s := config.NewUserSettings()
	s.Download.SetDownloadQuality(config.QualityExtreme)
	s.Download.SetInitialSorting(config.SortTitleAuthor)
	s.Update.SetOnlineUpdate(config.UpdatePromptForInstall)

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("cannot serialise the settings: %v", err)
	}

	text := string(data)
	for _, want := range []string{`"Extreme"`, `"title_author"`, `"promptForInstall"`} {
		if !strings.Contains(text, want) {
			t.Errorf("the settings must contain %s:\n%s", want, text)
		}
	}
}

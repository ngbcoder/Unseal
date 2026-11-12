package config

import (
	"encoding/json"
	"sync"

	"github.com/ngbcoder/Unseal/internal/types"
)

// AppSettings are the application-level settings.
//
// They live in appsettings.json next to the executable and are treated as
// read-only. The installer ships the file.
//
// The original implementation deliberately did not use
// Microsoft.Extensions.Configuration. Neither does Unseal: the format is
// plain JSON and the resolution rules are simple enough to own.
type AppSettings struct {
	// LogLevel is the maximum log level. Nil means the default.
	LogLevel *int `json:"logLevel,omitempty"`
	// DbgOnlineUpdate makes the update check read the manifest from the
	// local file instead of the network.
	DbgOnlineUpdate bool `json:"dbgOnlineUpdate,omitempty"`
}

// LogLevelOrDefault returns the configured log level, or the fallback when
// none is set.
func (a *AppSettings) LogLevelOrDefault(fallback int) int {
	if a == nil || a.LogLevel == nil {
		return fallback
	}
	return *a.LogLevel
}

// notifier carries the change callback shared by the settings groups.
//
// The mutex is a pointer so that copying a settings group does not copy a
// lock. The groups are embedded by value in UserSettings, and they define a
// pointer-receiver MarshalJSON.
type notifier struct {
	mu *sync.Mutex
	fn func()
}

// ensure makes the notifier usable.
func (n *notifier) ensure() {
	if n.mu == nil {
		n.mu = &sync.Mutex{}
	}
}

// fire calls the change callback, if there is one.
func (n *notifier) fire() {
	if n.mu == nil {
		return
	}
	n.mu.Lock()
	fn := n.fn
	n.mu.Unlock()
	if fn != nil {
		fn()
	}
}

// setCallback installs the change callback.
func (n *notifier) setCallback(fn func()) {
	n.ensure()
	n.mu.Lock()
	n.fn = fn
	n.mu.Unlock()
}

// UserSettings are the settings the user owns.
//
// They live in usersettings.json in the user settings directory and are
// written back as soon as anything changes.
//
// The fields of the groups are unexported, so every change goes through a
// setter and therefore reaches the change callback. That is what guarantees
// the original behaviour, where any change to any group was persisted
// immediately.
type UserSettings struct {
	notifier

	// Update holds the update policy.
	Update UpdateSettings `json:"update"`
	// Download holds the download settings.
	Download DownloadSettings `json:"download"`
	// Config holds the credential store settings.
	Config ConfigSettings `json:"config"`
	// Export holds the export settings.
	Export ExportSettings `json:"export"`
}

// NewUserSettings returns the settings with their default values.
func NewUserSettings() *UserSettings {
	s := &UserSettings{}

	s.Update.onlineUpdate = UpdatePromptForDownload
	s.Update.updateNotes = map[string]bool{}

	s.Download.autoUpdateLibrary = true
	s.Download.downloadQuality = QualityHigh

	s.Config.encryptConfiguration = true

	s.Init()
	return s
}

// Init wires the change callback of the settings groups.
//
// It is called by NewUserSettings and again after a load, so a change is
// always persisted.
func (s *UserSettings) Init() {
	s.Update.init(s)
	s.Download.init(s)
	s.Config.init(s)
	s.Export.init(s)
}

// OnChange installs the callback that runs whenever anything changes.
//
// A nil callback disables the notification.
func (s *UserSettings) OnChange(fn func()) {
	s.setCallback(fn)
}

// UpdateSettings holds the update policy.
type UpdateSettings struct {
	notifier

	onlineUpdate OnlineUpdate
	updateNotes  map[string]bool
}

func (u *UpdateSettings) init(owner *UserSettings) {
	if u.updateNotes == nil {
		u.updateNotes = map[string]bool{}
	}
	u.notifier.setCallback(owner.fire)
}

// OnlineUpdate returns the update prompt policy.
func (u *UpdateSettings) OnlineUpdate() OnlineUpdate { return u.onlineUpdate }

// SetOnlineUpdate changes the update prompt policy.
func (u *UpdateSettings) SetOnlineUpdate(v OnlineUpdate) {
	if u.onlineUpdate == v {
		return
	}
	u.onlineUpdate = v
	u.fire()
}

// UpdateNotes returns the versions whose release notes were already shown.
func (u *UpdateSettings) UpdateNotes() []string {
	out := make([]string, 0, len(u.updateNotes))
	for v := range u.updateNotes {
		out = append(out, v)
	}
	return out
}

// HasUpdateNote reports whether the release notes of a version were shown.
func (u *UpdateSettings) HasUpdateNote(version string) bool {
	return u.updateNotes[version]
}

// AddUpdateNote records that the release notes of a version were shown.
func (u *UpdateSettings) AddUpdateNote(version string) {
	if u.updateNotes == nil {
		u.updateNotes = map[string]bool{}
	}
	if u.updateNotes[version] {
		return
	}
	u.updateNotes[version] = true
	u.fire()
}

// updateSettingsDTO is the on-disk shape of the update settings.
//
// The fields that have a non-zero default are pointers, so that a field
// missing from the file keeps its default instead of being reset.
type updateSettingsDTO struct {
	OnlineUpdate *OnlineUpdate `json:"onlineUpdate,omitempty"`
	UpdateNotes  []string      `json:"updateNotes,omitempty"`
}

// MarshalJSON writes the update settings.
func (u *UpdateSettings) MarshalJSON() ([]byte, error) {
	policy := u.onlineUpdate
	return json.Marshal(updateSettingsDTO{
		OnlineUpdate: &policy,
		UpdateNotes:  u.UpdateNotes(),
	})
}

// UnmarshalJSON reads the update settings.
func (u *UpdateSettings) UnmarshalJSON(data []byte) error {
	var dto updateSettingsDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return err
	}

	if dto.OnlineUpdate != nil {
		u.onlineUpdate = *dto.OnlineUpdate
	}
	if dto.UpdateNotes != nil {
		u.updateNotes = make(map[string]bool, len(dto.UpdateNotes))
		for _, v := range dto.UpdateNotes {
			u.updateNotes[v] = true
		}
	}
	return nil
}

// ConfigSettings holds the settings of the credential store.
type ConfigSettings struct {
	notifier

	encryptConfiguration bool
}

func (c *ConfigSettings) init(owner *UserSettings) {
	c.notifier.setCallback(owner.fire)
}

// EncryptConfiguration reports whether the profile store is encrypted.
func (c *ConfigSettings) EncryptConfiguration() bool { return c.encryptConfiguration }

// SetEncryptConfiguration turns the encryption of the profile store on or
// off.
func (c *ConfigSettings) SetEncryptConfiguration(v bool) {
	if c.encryptConfiguration == v {
		return
	}
	c.encryptConfiguration = v
	c.fire()
}

// configSettingsDTO is the on-disk shape of the configuration settings.
type configSettingsDTO struct {
	EncryptConfiguration *bool `json:"encryptConfiguration,omitempty"`
}

// MarshalJSON writes the configuration settings.
func (c *ConfigSettings) MarshalJSON() ([]byte, error) {
	encrypt := c.encryptConfiguration
	return json.Marshal(configSettingsDTO{EncryptConfiguration: &encrypt})
}

// UnmarshalJSON reads the configuration settings.
func (c *ConfigSettings) UnmarshalJSON(data []byte) error {
	var dto configSettingsDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return err
	}
	if dto.EncryptConfiguration != nil {
		c.encryptConfiguration = *dto.EncryptConfiguration
	}
	return nil
}

// DownloadSettings holds the settings of the download stage.
type DownloadSettings struct {
	notifier

	autoRefresh             bool
	autoUpdateLibrary       bool
	autoOpenDownloadDialog  bool
	includeAdultProducts    bool
	hideUnavailableProducts bool
	multiPartDownload       bool
	keepEncryptedFiles      bool
	downloadQuality         DownloadQuality
	downloadDirectory       string
	initialSorting          InitialSorting
	profile                 types.ProfileAliasKey
}

func (d *DownloadSettings) init(owner *UserSettings) {
	d.notifier.setCallback(owner.fire)
}

// AutoRefresh reports whether the access token is refreshed on demand.
func (d *DownloadSettings) AutoRefresh() bool { return d.autoRefresh }

// SetAutoRefresh turns the on-demand token refresh on or off.
func (d *DownloadSettings) SetAutoRefresh(v bool) { d.setBool(&d.autoRefresh, v) }

// AutoUpdateLibrary reports whether the library is synced at startup.
func (d *DownloadSettings) AutoUpdateLibrary() bool { return d.autoUpdateLibrary }

// SetAutoUpdateLibrary turns the startup library sync on or off.
func (d *DownloadSettings) SetAutoUpdateLibrary(v bool) {
	d.setBool(&d.autoUpdateLibrary, v)
}

// AutoOpenDownloadDialog reports whether the library window opens at startup.
func (d *DownloadSettings) AutoOpenDownloadDialog() bool { return d.autoOpenDownloadDialog }

// SetAutoOpenDownloadDialog turns the automatic library window on or off.
func (d *DownloadSettings) SetAutoOpenDownloadDialog(v bool) {
	d.setBool(&d.autoOpenDownloadDialog, v)
}

// IncludeAdultProducts reports whether adult titles are shown.
func (d *DownloadSettings) IncludeAdultProducts() bool { return d.includeAdultProducts }

// SetIncludeAdultProducts turns the display of adult titles on or off.
func (d *DownloadSettings) SetIncludeAdultProducts(v bool) {
	d.setBool(&d.includeAdultProducts, v)
}

// HideUnavailableProducts reports whether titles that are no longer owned are
// hidden.
func (d *DownloadSettings) HideUnavailableProducts() bool {
	return d.hideUnavailableProducts
}

// SetHideUnavailableProducts turns the hiding of unavailable titles on or off.
func (d *DownloadSettings) SetHideUnavailableProducts(v bool) {
	d.setBool(&d.hideUnavailableProducts, v)
}

// MultiPartDownload reports whether multi-part titles are downloaded part by
// part.
func (d *DownloadSettings) MultiPartDownload() bool { return d.multiPartDownload }

// SetMultiPartDownload turns the part-by-part download on or off.
func (d *DownloadSettings) SetMultiPartDownload(v bool) {
	d.setBool(&d.multiPartDownload, v)
}

// KeepEncryptedFiles reports whether the encrypted payload is kept after a
// successful decryption.
func (d *DownloadSettings) KeepEncryptedFiles() bool { return d.keepEncryptedFiles }

// SetKeepEncryptedFiles turns the keeping of the encrypted payload on or off.
func (d *DownloadSettings) SetKeepEncryptedFiles(v bool) {
	d.setBool(&d.keepEncryptedFiles, v)
}

// DownloadQuality returns the requested audio quality.
func (d *DownloadSettings) DownloadQuality() DownloadQuality { return d.downloadQuality }

// SetDownloadQuality changes the requested audio quality.
func (d *DownloadSettings) SetDownloadQuality(v DownloadQuality) {
	if d.downloadQuality == v {
		return
	}
	d.downloadQuality = v
	d.fire()
}

// DownloadDirectory returns where the downloaded files are written.
func (d *DownloadSettings) DownloadDirectory() string { return d.downloadDirectory }

// SetDownloadDirectory changes where the downloaded files are written.
func (d *DownloadSettings) SetDownloadDirectory(v string) {
	if d.downloadDirectory == v {
		return
	}
	d.downloadDirectory = v
	d.fire()
}

// InitialSorting returns the sort order of the library grid.
func (d *DownloadSettings) InitialSorting() InitialSorting { return d.initialSorting }

// SetInitialSorting changes the sort order of the library grid.
func (d *DownloadSettings) SetInitialSorting(v InitialSorting) {
	if d.initialSorting == v {
		return
	}
	d.initialSorting = v
	d.fire()
}

// Profile returns the selected account and region.
func (d *DownloadSettings) Profile() types.ProfileAliasKey { return d.profile }

// SetProfile selects an account and region.
func (d *DownloadSettings) SetProfile(v types.ProfileAliasKey) {
	if d.profile == v {
		return
	}
	d.profile = v
	d.fire()
}

// setBool assigns a boolean and fires the change when it differs.
func (d *DownloadSettings) setBool(target *bool, v bool) {
	if *target == v {
		return
	}
	*target = v
	d.fire()
}

// downloadSettingsDTO is the on-disk shape of the download settings.
//
// The fields that have a non-zero default are pointers, so that a field
// missing from the file keeps its default instead of being reset.
type downloadSettingsDTO struct {
	AutoRefresh             *bool                 `json:"autoRefresh,omitempty"`
	AutoUpdateLibrary       *bool                 `json:"autoUpdateLibrary,omitempty"`
	AutoOpenDownloadDialog  bool                  `json:"autoOpenDownloadDialog,omitempty"`
	IncludeAdultProducts    bool                  `json:"includeAdultProducts,omitempty"`
	HideUnavailableProducts bool                  `json:"hideUnavailableProducts,omitempty"`
	MultiPartDownload       bool                  `json:"multiPartDownload,omitempty"`
	KeepEncryptedFiles      bool                  `json:"keepEncryptedFiles,omitempty"`
	DownloadQuality         *DownloadQuality      `json:"downloadQuality,omitempty"`
	DownloadDirectory       string                `json:"downloadDirectory,omitempty"`
	InitialSorting          InitialSorting        `json:"initialSorting,omitempty"`
	Profile                 types.ProfileAliasKey `json:"profile,omitempty"`
}

// MarshalJSON writes the download settings.
func (d *DownloadSettings) MarshalJSON() ([]byte, error) {
	autoRefresh := d.autoRefresh
	autoUpdateLibrary := d.autoUpdateLibrary
	quality := d.downloadQuality

	return json.Marshal(downloadSettingsDTO{
		AutoRefresh:             &autoRefresh,
		AutoUpdateLibrary:       &autoUpdateLibrary,
		AutoOpenDownloadDialog:  d.autoOpenDownloadDialog,
		IncludeAdultProducts:    d.includeAdultProducts,
		HideUnavailableProducts: d.hideUnavailableProducts,
		MultiPartDownload:       d.multiPartDownload,
		KeepEncryptedFiles:      d.keepEncryptedFiles,
		DownloadQuality:         &quality,
		DownloadDirectory:       d.downloadDirectory,
		InitialSorting:          d.initialSorting,
		Profile:                 d.profile,
	})
}

// UnmarshalJSON reads the download settings.
func (d *DownloadSettings) UnmarshalJSON(data []byte) error {
	var dto downloadSettingsDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return err
	}

	if dto.AutoRefresh != nil {
		d.autoRefresh = *dto.AutoRefresh
	}
	if dto.AutoUpdateLibrary != nil {
		d.autoUpdateLibrary = *dto.AutoUpdateLibrary
	}
	if dto.DownloadQuality != nil {
		d.downloadQuality = *dto.DownloadQuality
	}

	d.autoOpenDownloadDialog = dto.AutoOpenDownloadDialog
	d.includeAdultProducts = dto.IncludeAdultProducts
	d.hideUnavailableProducts = dto.HideUnavailableProducts
	d.multiPartDownload = dto.MultiPartDownload
	d.keepEncryptedFiles = dto.KeepEncryptedFiles

	if dto.DownloadDirectory != "" {
		d.downloadDirectory = dto.DownloadDirectory
	}
	if dto.InitialSorting != 0 {
		d.initialSorting = dto.InitialSorting
	}
	if !dto.Profile.IsEmpty() {
		d.profile = dto.Profile
	}
	return nil
}

// ExportSettings holds the settings of the export stage.
type ExportSettings struct {
	notifier

	// exportToAax is a tri-state. Nil means the user has not decided yet,
	// which is meaningful during the first-run wizard.
	exportToAax     *bool
	exportDirectory string
}

func (e *ExportSettings) init(owner *UserSettings) {
	e.notifier.setCallback(owner.fire)
}

// ExportToAax reports whether the export to the .aax format is enabled.
//
// The result is a tri-state: nil means the user has not decided yet.
func (e *ExportSettings) ExportToAax() *bool { return e.exportToAax }

// SetExportToAax turns the export to the .aax format on or off.
//
// A nil value clears the decision.
func (e *ExportSettings) SetExportToAax(v *bool) {
	if e.exportToAax == nil && v == nil {
		return
	}
	if e.exportToAax != nil && v != nil && *e.exportToAax == *v {
		return
	}
	e.exportToAax = v
	e.fire()
}

// ExportDirectory returns where the exported files are written.
func (e *ExportSettings) ExportDirectory() string { return e.exportDirectory }

// SetExportDirectory changes where the exported files are written.
func (e *ExportSettings) SetExportDirectory(v string) {
	if e.exportDirectory == v {
		return
	}
	e.exportDirectory = v
	e.fire()
}

// exportSettingsDTO is the on-disk shape of the export settings.
type exportSettingsDTO struct {
	ExportToAax     *bool  `json:"exportToAax,omitempty"`
	ExportDirectory string `json:"exportDirectory,omitempty"`
}

// MarshalJSON writes the export settings.
func (e *ExportSettings) MarshalJSON() ([]byte, error) {
	return json.Marshal(exportSettingsDTO{
		ExportToAax:     e.exportToAax,
		ExportDirectory: e.exportDirectory,
	})
}

// UnmarshalJSON reads the export settings.
func (e *ExportSettings) UnmarshalJSON(data []byte) error {
	var dto exportSettingsDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return err
	}
	e.exportToAax = dto.ExportToAax
	if dto.ExportDirectory != "" {
		e.exportDirectory = dto.ExportDirectory
	}
	return nil
}

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// The names of the settings files.
const (
	// AppSettingsFile holds the application settings.
	AppSettingsFile = "appsettings.json"
	// UserSettingsFile holds the user settings.
	UserSettingsFile = "usersettings.json"
)

// ErrNotFound is returned when a settings file does not exist.
var ErrNotFound = errors.New("config: settings file not found")

// Store reads and writes the two tiers of settings.
//
// The application tier lives next to the executable and is read-only in
// practice. The user tier lives in the user settings directory and is
// written back whenever anything changes.
//
// A Store is safe for concurrent use.
type Store struct {
	mu sync.Mutex

	// userDir is where the user settings live.
	userDir string
	// appDir is where the application settings live.
	appDir string
}

// NewStore creates a store.
//
// userDir is the user settings directory, appDir the application directory.
func NewStore(userDir, appDir string) *Store {
	return &Store{userDir: userDir, appDir: appDir}
}

// UserDir returns the user settings directory.
func (s *Store) UserDir() string { return s.userDir }

// AppDir returns the application directory.
func (s *Store) AppDir() string { return s.appDir }

// UserSettingsPath returns the path of the user settings file.
func (s *Store) UserSettingsPath() string {
	return filepath.Join(s.userDir, UserSettingsFile)
}

// AppSettingsPath returns the path of the application settings file.
func (s *Store) AppSettingsPath() string {
	return filepath.Join(s.appDir, AppSettingsFile)
}

// LoadApp reads the application settings.
//
// A missing file is not an error when optional is true; the settings are then
// returned with their default values.
func (s *Store) LoadApp(optional bool) (*AppSettings, error) {
	path := s.AppSettingsPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if optional {
				return &AppSettings{}, nil
			}
			return nil, fmt.Errorf("%w: %s", ErrNotFound, path)
		}
		return nil, err
	}

	settings := &AppSettings{}
	if err := json.Unmarshal(data, settings); err != nil {
		return nil, fmt.Errorf("config: cannot read %s: %w", path, err)
	}
	return settings, nil
}

// LoadUser reads the user settings.
//
// The resolution order is the user settings directory, then the application
// directory, then the built-in defaults. A file that is present but
// unreadable is an error, because silently falling back would hide a
// problem the user needs to know about.
func (s *Store) LoadUser() (*UserSettings, error) {
	// Start from the defaults, so that a field missing from the file keeps
	// its default value.
	settings := NewUserSettings()

	for _, path := range []string{s.UserSettingsPath(), filepath.Join(s.appDir, UserSettingsFile)} {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		if err := json.Unmarshal(data, settings); err != nil {
			return nil, fmt.Errorf("config: cannot read %s: %w", path, err)
		}

		settings.Init()
		return settings, nil
	}

	return settings, nil
}

// SaveUser writes the user settings.
//
// The write is atomic: the data goes to a temporary file in the same
// directory and is then renamed over the target, so a crash cannot leave a
// half-written settings file behind.
func (s *Store) SaveUser(settings *UserSettings) error {
	if settings == nil {
		return errors.New("config: cannot save nil settings")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("config: cannot serialise the settings: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(s.userDir, 0o755); err != nil {
		return fmt.Errorf("config: cannot create %s: %w", s.userDir, err)
	}

	return writeFileAtomic(s.UserSettingsPath(), data)
}

// SaveApp writes the application settings.
//
// It is used for tests and for a portable installation. The shipped
// application settings file is read-only.
func (s *Store) SaveApp(settings *AppSettings) error {
	if settings == nil {
		return errors.New("config: cannot save nil settings")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("config: cannot serialise the settings: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(s.appDir, 0o755); err != nil {
		return fmt.Errorf("config: cannot create %s: %w", s.appDir, err)
	}

	return writeFileAtomic(s.AppSettingsPath(), data)
}

// SaveUserPreset writes a user settings preset into the application
// directory.
//
// A preset is used when the user settings file does not exist yet, which
// lets an installation ship sensible defaults without the user settings file
// having to exist.
func (s *Store) SaveUserPreset(settings *UserSettings) error {
	if settings == nil {
		return errors.New("config: cannot save nil settings")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("config: cannot serialise the settings: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(s.appDir, 0o755); err != nil {
		return fmt.Errorf("config: cannot create %s: %w", s.appDir, err)
	}

	return writeFileAtomic(filepath.Join(s.appDir, UserSettingsFile), data)
}

// writeFileAtomic writes data to path through a temporary file.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp*")
	if err != nil {
		return fmt.Errorf("config: cannot create a temporary file in %s: %w", dir, err)
	}
	tmpName := tmp.Name()

	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}

	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("config: cannot write %s: %w", tmpName, err)
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("config: cannot flush %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("config: cannot close %s: %w", tmpName, err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("config: cannot replace %s: %w", path, err)
	}

	return nil
}

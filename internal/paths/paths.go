// Package paths resolves the directories Unseal reads and writes.
//
// The application keeps its state under the local application data folder:
//
//	%LOCALAPPDATA%\audiamus\Unseal\data       the library database
//	%LOCALAPPDATA%\audiamus\Unseal\img        cover images, named <ASIN>.<ext>
//	%LOCALAPPDATA%\audiamus\Unseal\config     Configuration.json, the profile store
//	%LOCALAPPDATA%\audiamus\Unseal\settings   usersettings.json
//	%LOCALAPPDATA%\audiamus\Unseal\log        rolling log files
//	%LOCALAPPDATA%\audiamus\Unseal\tmp        diagnostics dumps and handoff documents
//
// The layout follows the original implementation, which nested the
// application under a company folder, and matches the handoff contract in
// URIhandoff.md, which names the temp directory explicitly.
//
// Layer L0 (foundation).
package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// The application folders under the local application data folder.
const (
	// Company is the company folder. Unseal is developed alongside the
	// original project, so it keeps the same folder.
	Company = "audiamus"
	// AppName is the application folder.
	AppName = "Unseal"
)

// Subdirectory names.
const (
	dirData     = "data"
	dirImages   = "img"
	dirConfig   = "config"
	dirSettings = "settings"
	dirLog      = "log"
	dirTemp     = "tmp"
)

// Layout holds the resolved directories.
//
// The zero value is not usable; call New or Resolve.
type Layout struct {
	// Root is the application folder itself.
	Root string
	// Data holds the library database.
	Data string
	// Images holds the cover images.
	Images string
	// Config holds the encrypted profile store.
	Config string
	// Settings holds the user settings file.
	Settings string
	// Log holds the rolling log files.
	Log string
	// Temp holds the diagnostic dumps.
	Temp string
}

// New builds a layout rooted at the given directory.
//
// It does not touch the filesystem.
func New(root string) Layout {
	return Layout{
		Root:     root,
		Data:     filepath.Join(root, dirData),
		Images:   filepath.Join(root, dirImages),
		Config:   filepath.Join(root, dirConfig),
		Settings: filepath.Join(root, dirSettings),
		Log:      filepath.Join(root, dirLog),
		Temp:     filepath.Join(root, dirTemp),
	}
}

// Resolve determines the layout from the environment.
//
// The root is the local application data folder, which on Windows is
// %LOCALAPPDATA%. When that variable is not set the user cache directory is
// used instead, and finally the home directory.
func Resolve() (Layout, error) {
	base, err := localAppData()
	if err != nil {
		return Layout{}, err
	}
	return New(filepath.Join(base, Company, AppName)), nil
}

// localAppData returns the local application data folder.
func localAppData() (string, error) {
	if dir := os.Getenv("LOCALAPPDATA"); dir != "" {
		return dir, nil
	}
	if dir, err := os.UserCacheDir(); err == nil && dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("paths: cannot determine a local application data folder: %w", err)
	}
	return filepath.Join(home, "AppData", "Local"), nil
}

// All returns every directory of the layout, root first.
func (l Layout) All() []string {
	return []string{
		l.Root,
		l.Data,
		l.Images,
		l.Config,
		l.Settings,
		l.Log,
		l.Temp,
	}
}

// Ensure creates every directory of the layout that does not exist yet.
//
// It is safe to call repeatedly.
func (l Layout) Ensure() error {
	for _, dir := range l.All() {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("paths: cannot create %s: %w", dir, err)
		}
	}
	return nil
}

// UserName returns the name of the current user.
//
// It is used to derive the credential-store key and to anonymise log output.
func UserName() string {
	if name := os.Getenv("USERNAME"); name != "" {
		return name
	}
	return os.Getenv("USER")
}

// UserHome returns the profile directory of the current user.
func UserHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

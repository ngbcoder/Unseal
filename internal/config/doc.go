// Package config manages Unseal's two-tier settings.
//
// The application tier (appsettings.json) ships next to the executable and is
// treated as read-only. The user tier (usersettings.json) lives in the user's
// config directory and holds download, export and configuration settings.
//
// Reads resolve in order: user directory, then application directory, then
// built-in defaults. Writes are atomic (temp file plus rename) so a crash
// cannot leave a half-written settings file behind.
//
// Layer L1 (shared contracts). The settings structs are the only ones the
// domain layer is allowed to see; concrete types are hidden behind interfaces.
package config

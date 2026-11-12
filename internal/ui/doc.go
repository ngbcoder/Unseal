// Package ui implements Unseal's Gio desktop interface.
//
// The layer is split into:
//
//	ui/          — window shell, layout, theme tokens
//	ui/library   — the library browser and its data source
//	ui/download  — the download queue grid and staging
//	ui/profiles  — profile management, registration and alias dialogs
//	ui/wizard    — the first-run wizard
//
// Gio is immediate-mode, so all state lives in plain Go structs and widgets
// are rebuilt each frame. Long-running work is executed off the UI goroutine
// and results are marshalled back through a single update channel.
//
// Layer L4 (presentation). It depends on the domain layer through interfaces.
package ui

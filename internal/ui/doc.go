// Package ui implements Unseal's desktop interface.
//
// The interface is built with GOUI, a native desktop GUI toolkit for Go that
// loads the platform's native APIs at runtime, so the whole application
// builds with CGO_ENABLED=0 on Windows, Linux and macOS.
//
// The layer is split into:
//
//	ui/          — window shell, layout, theme
//	ui/library   — the library browser and its table model
//	ui/download  — the download queue table and staging
//	ui/profiles  — profile management, registration and alias dialogs
//	ui/wizard    — the first-run wizard
//
// The interface is declarative. Views are ordinary Go values that may be
// rebuilt on every frame while the framework retains the underlying widgets,
// so durable state lives in structures this package owns, never inside a
// build closure.
//
// Long-running work runs off the interface goroutine and posts its results
// back with app.Post. Widgets are not safe for concurrent use.
//
// Layer L4 (presentation). It depends on the domain layer through interfaces.
package ui

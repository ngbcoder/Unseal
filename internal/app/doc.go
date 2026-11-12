// Package app is Unseal's composition root.
//
// It wires the domain layer to concrete settings, database, protocol client
// and user interface, and owns the startup sequence: read configuration,
// refresh the access token, migrate the database, sync the library, then
// present the main window.
//
// Layer L5 (composition root). Only this package may know about every other
// layer at once.
package app

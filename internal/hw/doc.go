// Package hw derives stable hardware identifiers for the current machine.
//
// The identifiers feed the key derivation for the encrypted credential store.
// Only Windows is implemented for now, matching the original project's scope:
//
//	Windows — WMI (CPU ProcessorID, motherboard SerialNumber)
//
// The lookup sits behind an interface so other platforms can be added later
// without touching the domain code.
//
// When identifiers cannot be read the package reports a weak result so the
// user interface can warn that the configuration is only weakly protected.
//
// Layer L0 (foundation). Platform-specific code is isolated in build-tagged
// files so the package keeps building everywhere.
package hw

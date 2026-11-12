// Package hw derives stable hardware identifiers for the current machine.
//
// The identifiers feed the key derivation for the encrypted credential store.
// They are obtained per platform:
//
//	Windows — WMI (CPU ProcessorID, motherboard SerialNumber)
//	Linux   — /proc/cpuinfo, /sys/class/dmi/id, /etc/machine-id
//	macOS   — IOPlatformUUID and board serial via ioreg, CPU brand via sysctl
//
// When identifiers cannot be read the package reports a weak result so the
// user interface can warn that the configuration is only weakly protected.
//
// Layer L0 (foundation). Platform-specific files are isolated in build-tagged
// files so the package keeps building everywhere.
package hw

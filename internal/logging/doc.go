// Package logging provides Unseal's leveled logging facility.
//
// Levels run 0 (errors only) through 5 (reserved). The logger writes rolling
// daily files and supports level-gated lambdas so that expensive message
// construction is skipped when the level is disabled.
//
// This package is layer L0 (foundation): it knows nothing about Audible, the
// database, or the user interface.
package logging

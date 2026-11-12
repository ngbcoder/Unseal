// Package db contains Unseal's persistence layer.
//
// The model is a faithful port of the original Entity Framework schema:
//
//	Account ──┬── Conversion ── Book ── Component
//	           └── (AccountId + Region is the tenancy key)
//
// Conversion is the unit of work: every Book has exactly one "whole book"
// conversion and every Component has exactly one "part N" conversion.
//
// Storage is SQLite through a pure-Go driver, so Unseal ships as a single
// static binary with no cgo. Schema changes are applied by ordered, numbered
// migrations recorded in schema_migrations.
//
// Layer L2 (persistence). This package knows nothing about HTTP or the UI.
package db

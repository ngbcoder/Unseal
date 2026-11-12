// Package core is the domain facade of Unseal.
//
// It ties the Audible protocol client, the book library and the pipeline
// together and owns the currently selected profile. The user interface talks
// to this facade and never to the protocol client directly.
//
// Layer L3 (domain).
package core

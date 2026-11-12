package util_test

import (
	"strings"
	"testing"

	"github.com/ngbcoder/Unseal/internal/util"
)

func TestPruneRemovesInvalidAndDoubtfulCharacters(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain title is untouched", "Plain Title", "Plain Title"},
		{"question mark is invalid on windows", "Who Goes There?", "Who Goes There"},
		{"colon is invalid on windows", "Book 1: The Start", "Book 1 The Start"},
		{"asterisk is invalid on windows", "Star* Wars", "Star Wars"},
		{"path separators are invalid", "AC/DC Live", "ACDC Live"},
		{"angle brackets are invalid", "A <B> C", "A B C"},
		{"pipe is invalid", "A|B", "AB"},
		{"quote is invalid", `He said "no"`, "He said no"},
		{"control characters are invalid", "A\x01B", "AB"},
		{"trademark sign is doubtful", "Hitchhiker's™ Guide", "Hitchhiker's Guide"},
		{"em dash is doubtful", "War and Peace — a novel", "War and Peace  a novel"},
		{"inverted question mark is doubtful", "¿Que?", "Que"},
		{"smart quotes are doubtful", "“Quoted”", "Quoted"},
		{"leading dots are trimmed", "...Hidden", "Hidden"},
		{"trailing dots are trimmed", "Hidden...", "Hidden"},
		{"empty stays empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := util.Prune(tt.in); got != tt.want {
				t.Errorf("Prune(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestPruneWithUsesTheGivenInvalidSet(t *testing.T) {
	// The doubtful set is always applied, the invalid set is replaced.
	got := util.PruneWith("a™b", map[rune]bool{'x': true})
	if got != "ab" {
		t.Errorf("PruneWith = %q, want %q", got, "ab")
	}
}

func TestROT13(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Hello", "Uryyb"},
		{"Uryyb", "Hello"},
		{"User Name 123", "Hfre Anzr 123"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := util.ROT13(tt.in); got != tt.want {
			t.Errorf("ROT13(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestROT13IsItsOwnInverse(t *testing.T) {
	const s = "Nolan Bragan"
	if util.ROT13(util.ROT13(s)) != s {
		t.Error("ROT13 applied twice must return the original")
	}
}

func TestShorten(t *testing.T) {
	if got := util.Shorten("short", 40); got != "short" {
		t.Errorf("Shorten must not touch a short string, got %q", got)
	}
	got := util.Shorten("a very long title that needs shortening", 20)
	if len([]rune(got)) > 20 {
		t.Errorf("Shorten returned %d characters, want at most 20", len([]rune(got)))
	}
	if !strings.Contains(got, "…") {
		t.Errorf("Shorten must insert an ellipsis, got %q", got)
	}
}

func TestTrimDots(t *testing.T) {
	if got := util.TrimDots("..x.."); got != "x" {
		t.Errorf("TrimDots = %q, want %q", got, "x")
	}
}

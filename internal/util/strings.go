package util

import (
	"strings"
)

// invalidFileNameChars are the characters that cannot appear in a Windows
// file name.
//
// The original C# implementation used .NET's Path.GetInvalidFileNameChars,
// which returns only NUL, the double quote, the angle brackets, the pipe and
// the control characters. It does not include the colon, the asterisk, the
// question mark or the path separators, even though Windows rejects all of
// them in a file name. Audible titles regularly contain a colon, so the
// original could produce a file name that Windows then refuses to create.
//
// This set is the union that Windows actually enforces. The control
// characters below U+0020 are always removed, whichever set is supplied.
var invalidFileNameChars = map[rune]bool{
	'"':  true,
	'/':  true,
	'\\': true,
	':':  true,
	'<':  true,
	'>':  true,
	'|':  true,
	'?':  true,
	'*':  true,
}

// isControl reports whether r is a control character, which Windows rejects
// in a file name.
func isControl(r rune) bool {
	return r < 0x20 || r == 0x7f
}

// doubtfulFileNameChars are typographic characters that are legal in a file
// name but are usually a sign that the text came from a publisher's HTML
// rather than from a title.
//
// The list is carried over unchanged from the original implementation.
var doubtfulFileNameChars = map[rune]bool{
	'¡': true, '¢': true, '£': true, '¤': true, '¥': true, '¦': true,
	'§': true, '¨': true, '©': true, 'ª': true, '«': true, '¬': true,
	'®': true, '¯': true, '°': true, '±': true, '²': true, '³': true,
	'´': true, 'µ': true, '¶': true, '·': true, '¸': true, '¹': true,
	'º': true, '»': true, '¼': true, '½': true, '¾': true, '¿': true,
	'×': true, '÷': true, '‘': true, '’': true, 'ƒ': true, '„': true,
	'…': true, '†': true, '‡': true, 'ˆ': true, '‰': true, '‹': true,
	'“': true, '”': true, '•': true, '–': true, '—': true,
	'˜': true, '™': true, '›': true,
}

// Prune removes the characters that must not appear in a file name and the
// typographic characters that usually should not, then trims leading and
// trailing dots.
//
// A title such as "Who Goes There? (Part 1)" becomes "Who Goes There (Part 1)".
func Prune(s string) string {
	return PruneWith(s, invalidFileNameChars)
}

// PruneWith removes the given invalid characters and the doubtful typographic
// characters from s, then trims leading and trailing dots.
//
// The invalid set replaces the built-in one. Passing nil uses the built-in
// set.
func PruneWith(s string, invalid map[rune]bool) string {
	if invalid == nil {
		invalid = invalidFileNameChars
	}

	var b strings.Builder
	b.Grow(len(s))
	for _, c := range s {
		if isControl(c) || invalid[c] || doubtfulFileNameChars[c] {
			continue
		}
		b.WriteRune(c)
	}

	return strings.Trim(b.String(), ".")
}

// ROT13 applies the ROT13 rotation to the ASCII letters, leaving every other
// character untouched.
//
// It is used to derive the credential-store key from the user name.
func ROT13(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'A' && r <= 'Z':
			return 'A' + (r-'A'+13)%26
		case r >= 'a' && r <= 'z':
			return 'a' + (r-'a'+13)%26
		default:
			return r
		}
	}, s)
}

// Shorten shortens s to at most maxlen characters, replacing the middle with
// an ellipsis.
//
// A maxlen of zero or less means "do not shorten". The result never exceeds
// maxlen characters.
func Shorten(s string, maxlen int) string {
	if maxlen <= 0 || len([]rune(s)) <= maxlen {
		return s
	}

	runes := []rune(s)
	partLen1 := maxlen * 2 / 3
	partLen2 := maxlen - partLen1 - 1
	p2 := len(runes) - partLen2

	return strings.TrimSpace(string(runes[:partLen1])) + "…" + strings.TrimSpace(string(runes[p2:]))
}

// TrimDots removes leading and trailing dots from s.
func TrimDots(s string) string {
	return strings.Trim(s, ".")
}

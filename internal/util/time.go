package util

import (
	"fmt"
	"strings"
	"time"
)

// RoundDown floors t to the nearest multiple of span, measured from the zero
// time.
//
// It is used to bring a timestamp into the ten-minute bucket that the Audible
// request signing scheme accepts.
func RoundDown(t time.Time, span time.Duration) time.Time {
	if span <= 0 {
		return t
	}
	return t.Truncate(span)
}

// RoundUp raises t to the next multiple of span, measured from the zero time.
//
// A time that is already an exact multiple is returned unchanged.
func RoundUp(t time.Time, span time.Duration) time.Time {
	if span <= 0 {
		return t
	}
	truncated := t.Truncate(span)
	if truncated.Equal(t) {
		return t
	}
	return truncated.Add(span)
}

// xmlTimeLayout is the layout of an XML date-time without the zone.
const xmlTimeLayout = "2006-01-02T15:04:05"

// ToXmlTime formats t the way .NET's XmlConvert.ToString(t, Utc) does.
//
// The value is converted to UTC and rendered as
// "yyyy-MM-ddTHH:mm:ss.FFFFFFFZ", where the fractional part is omitted when
// it is zero and trailing zeros are trimmed. .NET dates have 100-nanosecond
// resolution, so the fraction is truncated to seven digits.
func ToXmlTime(t time.Time) string {
	t = t.UTC()

	s := t.Format(xmlTimeLayout)

	// Truncate to .NET's 100-nanosecond resolution.
	ticks := t.Nanosecond() / 100
	if ticks != 0 {
		frac := strings.TrimRight(fmt.Sprintf("%07d", ticks), "0")
		s += "." + frac
	}

	return s + "Z"
}

// xmlTimeParseLayouts are the accepted layouts when parsing an XML date-time.
var xmlTimeParseLayouts = []string{
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
}

// FromXmlTime parses a string produced by ToXmlTime.
//
// The result is always in UTC.
func FromXmlTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("util: empty XML time")
	}

	for _, layout := range xmlTimeParseLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("util: %q is not a valid XML date-time", s)
}

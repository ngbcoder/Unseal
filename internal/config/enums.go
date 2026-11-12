package config

import (
	"encoding/json"
	"fmt"
)

// DownloadQuality is the audio quality requested from Audible.
//
// The values are ordered, so a quality can be compared with another to decide
// whether an upgrade is needed.
type DownloadQuality int

// The download qualities, from lowest to highest.
const (
	QualityLow DownloadQuality = iota
	QualityNormal
	QualityHigh
	QualityExtreme
)

var qualityNames = map[DownloadQuality]string{
	QualityLow:     "Low",
	QualityNormal:  "Normal",
	QualityHigh:    "High",
	QualityExtreme: "Extreme",
}

// String returns the name of the quality.
func (q DownloadQuality) String() string {
	if s, ok := qualityNames[q]; ok {
		return s
	}
	return fmt.Sprintf("DownloadQuality(%d)", int(q))
}

// MarshalJSON writes the quality as its name.
func (q DownloadQuality) MarshalJSON() ([]byte, error) {
	s, ok := qualityNames[q]
	if !ok {
		return nil, fmt.Errorf("config: unknown download quality %d", int(q))
	}
	return json.Marshal(s)
}

// UnmarshalJSON reads a quality from its name.
func (q *DownloadQuality) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	for k, v := range qualityNames {
		if v == s {
			*q = k
			return nil
		}
	}
	return fmt.Errorf("config: unknown download quality %q", s)
}

// InitialSorting is the order the library grid is sorted in.
type InitialSorting int

// The sort orders.
const (
	SortStateDate InitialSorting = iota
	SortDate
	SortAuthorTitle
	SortAuthorDate
	SortTitleAuthor
)

var sortingNames = map[InitialSorting]string{
	SortStateDate:   "state_date",
	SortDate:        "date",
	SortAuthorTitle: "author_title",
	SortAuthorDate:  "author_date",
	SortTitleAuthor: "title_author",
}

// String returns the name of the sort order.
func (s InitialSorting) String() string {
	if n, ok := sortingNames[s]; ok {
		return n
	}
	return fmt.Sprintf("InitialSorting(%d)", int(s))
}

// MarshalJSON writes the sort order as its name.
func (s InitialSorting) MarshalJSON() ([]byte, error) {
	n, ok := sortingNames[s]
	if !ok {
		return nil, fmt.Errorf("config: unknown initial sorting %d", int(s))
	}
	return json.Marshal(n)
}

// UnmarshalJSON reads a sort order from its name.
func (s *InitialSorting) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	for k, v := range sortingNames {
		if v == str {
			*s = k
			return nil
		}
	}
	return fmt.Errorf("config: unknown initial sorting %q", str)
}

// OnlineUpdate is the update prompt policy.
type OnlineUpdate int

// The update policies.
const (
	UpdateNo OnlineUpdate = iota
	UpdatePromptForDownload
	UpdatePromptForInstall
)

var updateNames = map[OnlineUpdate]string{
	UpdateNo:                "no",
	UpdatePromptForDownload: "promptForDownload",
	UpdatePromptForInstall:  "promptForInstall",
}

// String returns the name of the policy.
func (u OnlineUpdate) String() string {
	if n, ok := updateNames[u]; ok {
		return n
	}
	return fmt.Sprintf("OnlineUpdate(%d)", int(u))
}

// MarshalJSON writes the policy as its name.
func (u OnlineUpdate) MarshalJSON() ([]byte, error) {
	n, ok := updateNames[u]
	if !ok {
		return nil, fmt.Errorf("config: unknown online update policy %d", int(u))
	}
	return json.Marshal(n)
}

// UnmarshalJSON reads a policy from its name.
func (u *OnlineUpdate) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	for k, v := range updateNames {
		if v == s {
			*u = k
			return nil
		}
	}
	return fmt.Errorf("config: unknown online update policy %q", s)
}

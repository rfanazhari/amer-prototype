package valueobject

import (
	"errors"
	"strings"
	"unicode/utf8"
)

var ErrInvalidTitle = errors.New("invalid title")

// Title is an immutable, trimmed task name validated by Unicode code-point count.
type Title struct{ value string }

// NewTitle trims Unicode whitespace and accepts 1–100 valid Unicode code points.
func NewTitle(s string) (Title, error) {
	s = strings.TrimSpace(s)
	if n := utf8.RuneCountInString(s); !utf8.ValidString(s) || n < 1 || n > 100 {
		return Title{}, ErrInvalidTitle
	}
	return Title{value: s}, nil
}

// String returns the normalized title text.
func (t Title) String() string { return t.value }

// Valid reports whether this value was initialized with a nonempty title; the Go zero value is invalid.
func (t Title) Valid() bool { return t.value != "" }

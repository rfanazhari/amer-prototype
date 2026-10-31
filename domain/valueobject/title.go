package valueobject

import (
	"errors"
	"strings"
	"unicode/utf8"
)

var ErrInvalidTitle = errors.New("invalid title")

type Title struct{ value string }

func NewTitle(s string) (Title, error) {
	s = strings.TrimSpace(s)
	if n := utf8.RuneCountInString(s); !utf8.ValidString(s) || n < 1 || n > 100 {
		return Title{}, ErrInvalidTitle
	}
	return Title{value: s}, nil
}

func (t Title) String() string { return t.value }
func (t Title) Valid() bool    { return t.value != "" }

package valueobject

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

var ErrInvalidConfiguration = errors.New("invalid agent configuration")

// StableRef is an immutable identity and exact version for a locally resolved reference.
type StableRef struct{ id, version string }

// NewStableRef accepts a UUID identity and a positive decimal version without leading zeros.
func NewStableRef(id, version string) (StableRef, error) {
	normalized, err := normalizeUUID(id)
	if err != nil {
		return StableRef{}, err
	}
	if !validReferenceVersion(version) {
		return StableRef{}, ErrInvalidConfiguration
	}
	return StableRef{id: normalized, version: version}, nil
}

// ID returns the stored reference identity.
func (r StableRef) ID() string { return r.id }

// Version returns the exact stored reference version.
func (r StableRef) Version() string { return r.version }

// copyRef returns an independently owned pointer copy.
func copyRef(r *StableRef) *StableRef {
	if r == nil {
		return nil
	}
	copied := *r
	return &copied
}

// normalizeUUID accepts mixed-case hex UUID text and stores a lowercase identity.
func normalizeUUID(id string) (string, error) {
	if len(id) != 36 {
		return "", ErrInvalidConfiguration
	}
	var b strings.Builder
	b.Grow(36)
	for i, r := range id {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return "", ErrInvalidConfiguration
			}
			b.WriteByte('-')
		default:
			if !isHex(r) {
				return "", ErrInvalidConfiguration
			}
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String(), nil
}

// validReferenceVersion reports a positive decimal of at most ten digits without a leading zero.
func validReferenceVersion(version string) bool {
	if len(version) < 1 || len(version) > 10 || version[0] == '0' {
		return false
	}
	for i := 0; i < len(version); i++ {
		if version[i] < '0' || version[i] > '9' {
			return false
		}
	}
	return true
}

// isHex reports whether r is a hexadecimal digit.
func isHex(r rune) bool {
	return r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F'
}

// normalizeEditableText converts line endings to LF, trims outer whitespace and rejects forbidden controls.
func normalizeEditableText(s string, max int) (string, error) {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.TrimSpace(s)
	if !utf8.ValidString(s) || utf8.RuneCountInString(s) > max {
		return "", ErrInvalidConfiguration
	}
	for _, r := range s {
		if r != '\n' && r != '\t' && unicode.IsControl(r) {
			return "", ErrInvalidConfiguration
		}
	}
	return s, nil
}

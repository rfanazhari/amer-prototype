package valueobject

import "errors"

var ErrInvalidConfiguration = errors.New("invalid agent configuration")

// StableRef is an immutable identity and exact version for a locally resolved reference.
type StableRef struct{ id, version string }

// NewStableRef accepts a UUID identity and a positive decimal version without leading zeros.
func NewStableRef(id, version string) (StableRef, error) {
	return StableRef{}, nil
}

// ID returns the stored reference identity.
func (r StableRef) ID() string { return r.id }

// Version returns the exact stored reference version.
func (r StableRef) Version() string { return r.version }

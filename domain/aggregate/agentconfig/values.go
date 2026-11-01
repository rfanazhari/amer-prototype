// Package agentconfig owns instance versioning and publication snapshot mechanics.
package agentconfig

import "errors"

var (
	ErrInvalid    = errors.New("invalid agent configuration state")
	ErrStale      = errors.New("stale agent draft version")
	ErrValidation = errors.New("agent validation does not qualify this draft")
)

// Configuration is the copy contract implemented by concrete domain configuration
// values. Clone must deeply copy all mutable values, preserving their meaning.
// It is a value operation, not an I/O port; field policy belongs to T5.
type Configuration[C any] interface {
	// Clone returns an independently owned configuration value.
	Clone() C
}

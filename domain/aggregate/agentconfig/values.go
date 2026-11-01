// Package agentconfig owns instance versioning and publication snapshot mechanics.
package agentconfig

import (
	"errors"
	"time"
)

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

// TemplateRef identifies an immutable template and its behavior interpretation.
// Its identity is opaque here; supported catalog entries are selected by T5.
type TemplateRef struct{ Template, Version, Behavior string }

// DraftVersion is a positive replacement sequence; the zero value is invalid.
type DraftVersion struct{ number uint64 }

// NewDraftVersion validates a sequence supplied by creation or persistence policy.
func NewDraftVersion(number uint64) (DraftVersion, error) { return DraftVersion{}, ErrInvalid }

// Number returns the exact sequence without normalization.
func (v DraftVersion) Number() uint64 { return 0 }

// Finding is a detached assessment fact; warning policy and codes belong to T5.
type Finding struct{ Code, Path string }

// ValidationSnapshot transfers exact assessment identity and findings.
type ValidationSnapshot struct {
	ID, InstanceID string
	DraftVersion   DraftVersion
	Template       TemplateRef
	Eligible       bool
	Findings       []Finding
	CreatedAt      time.Time
}

// Validation is an immutable exact-version assessment, not a current readiness promise.
type Validation struct{ state ValidationSnapshot }

// NewValidation records a deterministic assessment without selecting field policy.
func NewValidation(s ValidationSnapshot) (Validation, error) { return Validation{}, ErrInvalid }

// RestoreValidation checks stored identity without rerunning current validators.
func RestoreValidation(s ValidationSnapshot) (Validation, error) { return Validation{}, ErrInvalid }

// Snapshot returns independently owned assessment facts.
func (v Validation) Snapshot() ValidationSnapshot { return ValidationSnapshot{} }

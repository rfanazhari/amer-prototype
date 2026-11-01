package agentconfig

import (
	"time"

	"github.com/rfanazhari/amer-prototype/domain/enum"
)

// Finding is a detached assessment fact with T5 severity, code, path and safe message.
type Finding struct {
	Severity            enum.FindingSeverity
	Code, Path, Message string
}

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
func NewValidation(s ValidationSnapshot) (Validation, error) {
	return RestoreValidation(s)
}

// RestoreValidation checks stored identity without rerunning current validators.
func RestoreValidation(s ValidationSnapshot) (Validation, error) {
	if s.ID == "" || s.InstanceID == "" || s.DraftVersion.number == 0 || !s.Template.valid() || s.CreatedAt.IsZero() {
		return Validation{}, ErrInvalid
	}
	s.Findings = append([]Finding(nil), s.Findings...)
	return Validation{state: s}, nil
}

// Snapshot returns independently owned assessment facts.
func (v Validation) Snapshot() ValidationSnapshot {
	s := v.state
	s.Findings = append([]Finding(nil), s.Findings...)
	return s
}

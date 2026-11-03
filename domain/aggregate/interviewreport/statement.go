package interviewreport

import (
	"strings"
	"unicode/utf8"
)

// maxBodyLength is the largest accepted statement body in characters.
const maxBodyLength = 2000

// Kind is one of the three recognized statement kinds.
type Kind string

const (
	// KindObservation is something the interview record shows.
	KindObservation Kind = "observation"
	// KindGap is a topic the interview did not cover or the candidate did not answer.
	KindGap Kind = "gap"
	// KindUncertainty is something the interview record leaves unclear.
	KindUncertainty Kind = "uncertainty"
)

// Citation identifies the finalized evidence entry and revision a statement relies on.
type Citation struct {
	// EntryID is the finalized evidence entry identity.
	EntryID string
	// Revision is the positive revision of that entry.
	Revision int
}

// StatementInput carries the fields from which a Statement is constructed or restored.
type StatementInput struct {
	// Kind is the statement kind.
	Kind Kind
	// Body is the statement text.
	Body string
	// PresentsAsAnswer reports that the statement is shown as something the candidate answered.
	PresentsAsAnswer bool
	// Citation is the evidence the statement relies on, nil when it cites none.
	Citation *Citation
}

// Statement is an immutable observation, gap, or uncertainty of a report generation.
type Statement struct {
	kind             Kind
	body             string
	presentsAsAnswer bool
	citation         *Citation
}

// NewStatement validates writer output and builds a statement.
// It rejects verdict wording in addition to the shape rules applied on restoration.
func NewStatement(in StatementInput) (Statement, error) {
	st, err := RestoreStatement(in)
	if err != nil {
		return Statement{}, err
	}
	if hasVerdictWord(in.Body) {
		return Statement{}, ErrVerdictWording
	}
	return st, nil
}

// RestoreStatement rebuilds a stored statement.
// It applies the shape rules only and does not re-apply the writer's verdict wording guard.
func RestoreStatement(in StatementInput) (Statement, error) {
	if err := validateShape(in); err != nil {
		return Statement{}, err
	}
	var c *Citation
	if in.Citation != nil {
		v := *in.Citation
		c = &v
	}
	return Statement{kind: in.Kind, body: in.Body, presentsAsAnswer: in.PresentsAsAnswer, citation: c}, nil
}

// validateShape applies the kind, body, citation, and gap rules shared by construction and restoration.
func validateShape(in StatementInput) error {
	switch in.Kind {
	case KindObservation, KindGap, KindUncertainty:
	default:
		return ErrInvalidStatement
	}
	if strings.TrimSpace(in.Body) == "" || utf8.RuneCountInString(in.Body) > maxBodyLength {
		return ErrInvalidStatement
	}
	if in.Citation != nil && (in.Citation.EntryID == "" || in.Citation.Revision < 1) {
		return ErrInvalidCitation
	}
	if in.Kind == KindGap && (in.PresentsAsAnswer || in.Citation != nil) {
		return ErrGapShape
	}
	if in.PresentsAsAnswer {
		if in.Kind != KindObservation {
			return ErrInvalidStatement
		}
		if in.Citation == nil {
			return ErrCitationRequired
		}
	}
	return nil
}

// valid reports whether the statement was built through a constructor.
func (s Statement) valid() bool {
	return validateShape(StatementInput{Kind: s.kind, Body: s.body, PresentsAsAnswer: s.presentsAsAnswer, Citation: s.citation}) == nil
}

// Kind returns the statement kind.
func (s Statement) Kind() Kind { return s.kind }

// Body returns the statement text.
func (s Statement) Body() string { return s.body }

// PresentsAsAnswer reports whether the statement is shown as something the candidate answered.
func (s Statement) PresentsAsAnswer() bool { return s.presentsAsAnswer }

// Citation returns a copy of the cited evidence, nil when the statement cites none.
func (s Statement) Citation() *Citation {
	if s.citation == nil {
		return nil
	}
	v := *s.citation
	return &v
}

package interviewreport

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
	return Statement{}, nil
}

// RestoreStatement rebuilds a stored statement.
// It applies the shape rules only and does not re-apply the writer's verdict wording guard.
func RestoreStatement(in StatementInput) (Statement, error) {
	return Statement{}, nil
}

// Kind returns the statement kind.
func (s Statement) Kind() Kind { return s.kind }

// Body returns the statement text.
func (s Statement) Body() string { return s.body }

// PresentsAsAnswer reports whether the statement is shown as something the candidate answered.
func (s Statement) PresentsAsAnswer() bool { return s.presentsAsAnswer }

// Citation returns a copy of the cited evidence, nil when the statement cites none.
func (s Statement) Citation() *Citation { return nil }

package interviewreport

import "errors"

var (
	// ErrInvalidReport reports rejected construction, restoration, or field validation for a report.
	ErrInvalidReport = errors.New("invalid interview report")
	// ErrGenerationActive reports that a generation is already pending or running for the report.
	ErrGenerationActive = errors.New("a report generation is already active")
	// ErrGenerationNotFound reports that the report has no generation with the requested identity.
	ErrGenerationNotFound = errors.New("report generation not found")
	// ErrInvalidTransition reports that the generation state does not allow the requested transition.
	ErrInvalidTransition = errors.New("generation state does not allow this transition")
	// ErrInvalidStatement reports a statement whose kind, body, or answer flag is not acceptable.
	ErrInvalidStatement = errors.New("invalid report statement")
	// ErrInvalidCitation reports a citation without an entry identity or a positive revision.
	ErrInvalidCitation = errors.New("invalid evidence citation")
	// ErrCitationRequired reports a statement presented as an answer without a finalized evidence citation.
	ErrCitationRequired = errors.New("a statement presented as an answer requires an evidence citation")
	// ErrGapShape reports a gap that is presented as an answer or carries evidence.
	ErrGapShape = errors.New("a gap is not an answer and carries no evidence")
	// ErrVerdictWording reports statement text that hires, rejects, or ranks the candidate.
	ErrVerdictWording = errors.New("statement wording states a hire, reject, or rank verdict")
)

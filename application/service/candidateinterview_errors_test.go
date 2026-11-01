package service_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/stretchr/testify/assert"
)

// interviewErrorCase pairs a candidate-interview sentinel with the kind it must report.
type interviewErrorCase struct {
	name string
	err  error
	kind service.InterviewErrorKind
}

// interviewErrorCases lists every classified candidate-interview failure.
func interviewErrorCases() []interviewErrorCase {
	return []interviewErrorCase{
		{"not found", service.ErrInterviewNotFound, service.InterviewNotFound},
		{"conflict", service.ErrInterviewConflict, service.InterviewConflict},
		{"coverage", service.ErrInterviewCoverage, service.InterviewCoverage},
		{"dependency", service.ErrInterviewDependencyUnavailable, service.InterviewDependencyUnavailable},
	}
}

// TestInterviewErrorKindOfClassifiesEachFailure verifies each sentinel reports its own kind, also when wrapped.
func TestInterviewErrorKindOfClassifiesEachFailure(t *testing.T) {
	for _, tc := range interviewErrorCases() {
		t.Run(tc.name, func(t *testing.T) {
			kind, ok := service.InterviewErrorKindOf(tc.err)
			assert.True(t, ok)
			assert.Equal(t, tc.kind, kind)

			wrapped := fmt.Errorf("while reading the interview: %w", tc.err)
			kind, ok = service.InterviewErrorKindOf(wrapped)
			assert.True(t, ok)
			assert.Equal(t, tc.kind, kind)
		})
	}
}

// TestInterviewErrorsAreDistinct verifies a failure matches only its own sentinel.
func TestInterviewErrorsAreDistinct(t *testing.T) {
	for _, tc := range interviewErrorCases() {
		for _, other := range interviewErrorCases() {
			t.Run(tc.name+" vs "+other.name, func(t *testing.T) {
				assert.Equal(t, tc.name == other.name, errors.Is(tc.err, other.err))
			})
		}
	}
}

// TestInterviewErrorMatchesByKind verifies a separately built error of the same kind matches the sentinel.
func TestInterviewErrorMatchesByKind(t *testing.T) {
	built := fmt.Errorf("lifecycle: %w", &service.InterviewError{Kind: service.InterviewConflict})
	assert.ErrorIs(t, built, service.ErrInterviewConflict)
	assert.NotErrorIs(t, built, service.ErrInterviewNotFound)
}

// TestInterviewErrorKindOfIgnoresOtherErrors verifies unrelated errors are not classified.
func TestInterviewErrorKindOfIgnoresOtherErrors(t *testing.T) {
	kind, ok := service.InterviewErrorKindOf(errors.New("boom"))
	assert.False(t, ok)
	assert.Empty(t, kind)

	kind, ok = service.InterviewErrorKindOf(nil)
	assert.False(t, ok)
	assert.Empty(t, kind)
}

// TestInterviewErrorMessageNamesKindOnly verifies the message carries the kind and no payload.
func TestInterviewErrorMessageNamesKindOnly(t *testing.T) {
	for _, tc := range interviewErrorCases() {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, string(tc.kind), tc.err.Error())
		})
	}
}

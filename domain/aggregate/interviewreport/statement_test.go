package interviewreport_test

import (
	"strings"
	"testing"

	"github.com/rfanazhari/amer-prototype/domain/aggregate/interviewreport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixtureCitation returns a valid finalized evidence citation.
func fixtureCitation() *interviewreport.Citation {
	return &interviewreport.Citation{EntryID: "entry-1", Revision: 2}
}

// TestNewStatement covers kind, body, citation, and gap-shape rules.
func TestNewStatement(t *testing.T) {
	// statementCase is one row in the NewStatement table-driven test.
	type statementCase struct {
		name    string
		input   interviewreport.StatementInput
		wantErr error
	}
	cases := []statementCase{
		{
			name:  "observation answer with citation",
			input: interviewreport.StatementInput{Kind: interviewreport.KindObservation, Body: "Described backend experience.", PresentsAsAnswer: true, Citation: fixtureCitation()},
		},
		{
			name:    "observation answer without citation",
			input:   interviewreport.StatementInput{Kind: interviewreport.KindObservation, Body: "Described backend experience.", PresentsAsAnswer: true},
			wantErr: interviewreport.ErrCitationRequired,
		},
		{
			name:  "gap without evidence",
			input: interviewreport.StatementInput{Kind: interviewreport.KindGap, Body: "Compensation was not answered."},
		},
		{
			name:    "gap with evidence",
			input:   interviewreport.StatementInput{Kind: interviewreport.KindGap, Body: "Compensation was not answered.", Citation: fixtureCitation()},
			wantErr: interviewreport.ErrGapShape,
		},
		{
			name:    "gap presented as answer",
			input:   interviewreport.StatementInput{Kind: interviewreport.KindGap, Body: "Compensation was not answered.", PresentsAsAnswer: true},
			wantErr: interviewreport.ErrGapShape,
		},
		{
			name:  "uncertainty with citation",
			input: interviewreport.StatementInput{Kind: interviewreport.KindUncertainty, Body: "Expectation was unclear.", Citation: fixtureCitation()},
		},
		{
			name:  "uncertainty without citation",
			input: interviewreport.StatementInput{Kind: interviewreport.KindUncertainty, Body: "Expectation was unclear."},
		},
		{
			name:    "uncertainty presented as answer",
			input:   interviewreport.StatementInput{Kind: interviewreport.KindUncertainty, Body: "Expectation was unclear.", PresentsAsAnswer: true, Citation: fixtureCitation()},
			wantErr: interviewreport.ErrInvalidStatement,
		},
		{
			name:    "fourth kind",
			input:   interviewreport.StatementInput{Kind: interviewreport.Kind("recommendation"), Body: "Strong profile."},
			wantErr: interviewreport.ErrInvalidStatement,
		},
		{
			name:    "empty body",
			input:   interviewreport.StatementInput{Kind: interviewreport.KindGap, Body: "   "},
			wantErr: interviewreport.ErrInvalidStatement,
		},
		{
			name:  "body at 2000 characters",
			input: interviewreport.StatementInput{Kind: interviewreport.KindGap, Body: strings.Repeat("é", 2000)},
		},
		{
			name:    "body over 2000 characters",
			input:   interviewreport.StatementInput{Kind: interviewreport.KindGap, Body: strings.Repeat("é", 2001)},
			wantErr: interviewreport.ErrInvalidStatement,
		},
		{
			name:    "citation revision zero",
			input:   interviewreport.StatementInput{Kind: interviewreport.KindUncertainty, Body: "Unclear.", Citation: &interviewreport.Citation{EntryID: "entry-1", Revision: 0}},
			wantErr: interviewreport.ErrInvalidCitation,
		},
		{
			name:    "citation without entry",
			input:   interviewreport.StatementInput{Kind: interviewreport.KindUncertainty, Body: "Unclear.", Citation: &interviewreport.Citation{Revision: 1}},
			wantErr: interviewreport.ErrInvalidCitation,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := interviewreport.NewStatement(tt.input)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.input.Kind, got.Kind())
			assert.Equal(t, tt.input.Body, got.Body())
			assert.Equal(t, tt.input.PresentsAsAnswer, got.PresentsAsAnswer())
			assert.Equal(t, tt.input.Citation, got.Citation())
		})
	}
}

// TestNewStatementVerdictWords covers verdict word matching on Unicode word boundaries.
func TestNewStatementVerdictWords(t *testing.T) {
	// verdictCase is one row in the verdict wording table-driven test.
	type verdictCase struct {
		name    string
		body    string
		wantErr error
	}
	cases := []verdictCase{
		{name: "hire", body: "We should hire this person.", wantErr: interviewreport.ErrVerdictWording},
		{name: "hiring upper case", body: "HIRING is advised.", wantErr: interviewreport.ErrVerdictWording},
		{name: "reject", body: "Reject.", wantErr: interviewreport.ErrVerdictWording},
		{name: "rejection", body: "A rejection is likely.", wantErr: interviewreport.ErrVerdictWording},
		{name: "rank", body: "Rank: first.", wantErr: interviewreport.ErrVerdictWording},
		{name: "ranking", body: "Top ranking candidate.", wantErr: interviewreport.ErrVerdictWording},
		{name: "after punctuation", body: "Summary:hire", wantErr: interviewreport.ErrVerdictWording},
		{name: "higher is not hire", body: "Expects a higher salary band."},
		{name: "shire is not hire", body: "Lives in the Shire."},
		{name: "frank is not rank", body: "Was frank about gaps."},
		{name: "non ASCII neighbours", body: "Une réponse naïve et claire."},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := interviewreport.NewStatement(interviewreport.StatementInput{Kind: interviewreport.KindUncertainty, Body: tt.body})
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

// TestStatementCitationIsDetached verifies callers cannot mutate a statement through its citation.
func TestStatementCitationIsDetached(t *testing.T) {
	in := interviewreport.StatementInput{Kind: interviewreport.KindObservation, Body: "Answered.", PresentsAsAnswer: true, Citation: fixtureCitation()}
	s, err := interviewreport.NewStatement(in)
	require.NoError(t, err)

	in.Citation.Revision = 99
	require.NotNil(t, s.Citation())
	s.Citation().Revision = 98

	assert.Equal(t, &interviewreport.Citation{EntryID: "entry-1", Revision: 2}, s.Citation())
}

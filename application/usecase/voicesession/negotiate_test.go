package voicesession_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/usecase/voicesession"
	"github.com/rfanazhari/amer-prototype/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const interviewerInstructions = "You are a simple interviewer in a feasibility prototype. Speak Indonesian. English technical terms from the candidate are allowed and should be understood. When the first response is requested, greet the candidate in Indonesian and ask one opening question before waiting for an answer. On each later turn, stay in this same conversation, use what the candidate already said, and refer to an earlier answer from this session. Do not score, grade, pass, or fail any answer. Do not select questions from a fixed HR list, and do not apply compensation or hiring policy."

// negotiateCase defines setup input and port-call expectations.
type negotiateCase struct {
	name           string
	offer          string
	exchangeErr    error
	want           string
	wantErr        error
	wantExchange   bool
	exchangedOffer string
}

// TestNegotiate checks offer trimming, instruction ownership, answers and port-failure mapping.
func TestNegotiate(t *testing.T) {
	portFailure := errors.New("provider failure")
	tests := []negotiateCase{
		{
			name:           "returns the provider answer",
			offer:          "v=0",
			want:           "v=0 answer",
			wantExchange:   true,
			exchangedOffer: "v=0",
		},
		{
			name:           "trims surrounding space before the port call",
			offer:          "  v=0 offer  ",
			want:           "answer",
			wantExchange:   true,
			exchangedOffer: "v=0 offer",
		},
		{
			name:    "rejects a blank offer",
			offer:   "",
			wantErr: voicesession.ErrInvalidOffer,
		},
		{
			name:    "rejects a whitespace offer",
			offer:   " \n\t ",
			wantErr: voicesession.ErrInvalidOffer,
		},
		{
			name:           "maps port failure",
			offer:          "v=0",
			exchangeErr:    portFailure,
			wantErr:        voicesession.ErrSessionUnavailable,
			wantExchange:   true,
			exchangedOffer: "v=0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			negotiator := mocks.NewNegotiator(t)
			if tt.wantExchange {
				negotiator.On("Exchange", mock.Anything, tt.exchangedOffer, interviewerInstructions).Return(tt.want, tt.exchangeErr)
			}
			got, err := (voicesession.Negotiate{Negotiator: negotiator}).Execute(context.Background(), tt.offer)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.NotErrorIs(t, err, portFailure)
				assert.Empty(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

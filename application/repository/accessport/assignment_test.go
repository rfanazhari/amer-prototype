package accessport_test

import (
	"testing"

	"github.com/rfanazhari/amer-prototype/application/repository/accessport"
	"github.com/stretchr/testify/assert"
)

const assignmentID = "5b1c0e22-7d4f-4a91-8c33-2b3c4d5e6f70"
const candidateID = "6c2d1f33-8e50-4b02-9d44-3c4d5e6f7081"
const otherCandidateID = "7d3e2044-9f61-4c13-ae55-4d5e6f708192"
const revisionID = "8e4f3155-a072-4d24-bf66-5e6f708192a3"

// assignmentFactsCase names one Interview-owned assignment fact combination and expected access predicates.
type assignmentFactsCase struct {
	name         string
	facts        accessport.AssignmentFacts
	candidateID  string
	wantInvite   bool
	wantExchange bool
	wantSession  bool
}

// TestAssignmentFactsGovernInvitationExchangeAndSessionAccess encodes T10 assignment collaboration without a fake assignment table.
func TestAssignmentFactsGovernInvitationExchangeAndSessionAccess(t *testing.T) {
	live := accessport.AssignmentFacts{
		AssignmentID: assignmentID,
		CandidateID:  candidateID,
		RevisionID:   revisionID,
	}
	tests := []assignmentFactsCase{
		{
			name:         "live assignment permits invitation exchange and session",
			facts:        live,
			candidateID:  candidateID,
			wantInvite:   true,
			wantExchange: true,
			wantSession:  true,
		},
		{
			name: "revoked assignment denies invitation exchange and session",
			facts: accessport.AssignmentFacts{
				AssignmentID:  assignmentID,
				CandidateID:   candidateID,
				RevisionID:    revisionID,
				AccessRevoked: true,
			},
			candidateID: candidateID,
		},
		{
			name: "completed assignment still permits access without implying execution restart",
			facts: accessport.AssignmentFacts{
				AssignmentID:      assignmentID,
				CandidateID:       candidateID,
				RevisionID:        revisionID,
				TerminalCompleted: true,
			},
			candidateID:  candidateID,
			wantInvite:   true,
			wantExchange: true,
			wantSession:  true,
		},
		{
			name:         "candidate mismatch denies invitation while facts remain assignment-scoped",
			facts:        live,
			candidateID:  otherCandidateID,
			wantExchange: true,
			wantSession:  true,
		},
		{
			name:        "missing assignment identity denies all access predicates",
			facts:       accessport.AssignmentFacts{CandidateID: candidateID, RevisionID: revisionID},
			candidateID: candidateID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantInvite, tt.facts.PermitsInvitation(tt.candidateID))
			assert.Equal(t, tt.wantExchange, tt.facts.PermitsExchange())
			assert.Equal(t, tt.wantSession, tt.facts.PermitsSessionAccess())
			assert.Equal(t, tt.facts.CandidateID == tt.candidateID && tt.candidateID != "", tt.facts.BoundToCandidate(tt.candidateID))
		})
	}
}

package integration_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/query/accessquery"
	"github.com/rfanazhari/amer-prototype/application/usecase/accessusecase"
	"github.com/rfanazhari/amer-prototype/application/usecase/agentbindingusecase"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/accesshandler"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/accessrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/agentbindingrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/agentconfigrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accessauthority"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accesscredentials"
	"github.com/rfanazhari/amer-prototype/test/testdata"
	"github.com/stretchr/testify/require"
)

// newCandidateAccessHTTP wires candidate operations over the release schema, which already owns the candidate and assignment tables.
func newCandidateAccessHTTP(t *testing.T) accessHTTP {
	t.Helper()
	graph := newAccessHTTP(t)
	store := accessrepository.Store{DB: graph.db}
	hasher := accesscredentials.Passwords{}
	tokens := accesscredentials.Tokens{}
	csrf := accesscredentials.CSRF{Secret: []byte(accessSecret)}
	sessions := accessauthority.Sessions{Store: store, Tokens: tokens, Clock: graph.clock}
	candidates := accessrepository.CandidateIdentities{Store: store}
	invitations := accessrepository.InvitationSecrets{Store: store}
	candidateSecrets := accessrepository.CandidateSessionSecrets{Store: store}
	candidateSessions := accessauthority.CandidateSessions{Store: candidateSecrets, Tokens: tokens, Clock: graph.clock}
	graph.authority = accessauthority.Authority{Sessions: sessions, Clock: graph.clock}
	graph.handler = accesshandler.Handler{
		Login:           accessusecase.Login{Accounts: store, Issuance: store, Hasher: hasher, Tokens: tokens, Clock: graph.clock},
		Current:         accessquery.CurrentSession{Current: sessions, CSRF: csrf, Clock: graph.clock},
		Provision:       accessusecase.ProvisionHR{Accounts: store, Current: sessions, Hasher: hasher, Clock: graph.clock},
		Logout:          accessusecase.Logout{Current: sessions, Sessions: store, Clock: graph.clock},
		CreateCandidate: accessusecase.CreateCandidate{Candidates: candidates, Current: sessions, Clock: graph.clock},
		CreateAssignment: agentbindingusecase.CreateAssignment{
			Current: sessions, Clock: graph.clock, Candidates: candidates,
			Instances: agentconfigrepository.Store{DB: graph.db}, Assignments: agentbindingrepository.Assignments{DB: graph.db},
		},
		Candidate:        accessquery.Candidate{Candidates: candidates, Current: sessions, Clock: graph.clock},
		ListCandidates:   accessquery.ListCandidates{Candidates: candidates, Current: sessions, Clock: graph.clock},
		IssueInvitation:  accessusecase.IssueInvitation{Assignments: store, Current: sessions, Tokens: tokens, Clock: graph.clock},
		Exchange:         accessusecase.ExchangeInvitation{Invitations: invitations, Candidates: candidates, Assignments: store, Tokens: tokens, Clock: graph.clock},
		CandidateSession: accessquery.CandidateSession{Current: candidateSessions, Candidates: candidates, Assignments: store, CSRF: csrf, Clock: graph.clock},
		CSRF:             csrf,
		Sessions:         sessions,
		Origin:           accessOrigin,
		SecureCookie:     false,
		Limiter:          &accesshandler.LoginLimiter{},
		ExchangeLimiter:  &accesshandler.LoginLimiter{},
	}
	return graph
}

// insertAssignment writes one release-schema assignment row, with its published instance and revision, for Access collaboration.
func insertAssignment(t *testing.T, graph accessHTTP, candidateID string, revoked, completed bool) string {
	t.Helper()
	configStore := agentconfigrepository.Store{DB: graph.db}
	account := insertACPAccount(t, graph.db)
	doc := persistDocument(t, configStore, uuid.NewString(), []string{"total"})
	instanceID := doc.Document.Snapshot().ID
	validation := insertOwnedValidation(t, configStore, instanceID, 1, true)
	revisionID := publishOwned(t, configStore, instanceID, validation, account.ID, uuid.NewString(), testdata.TokenHash)
	id := uuid.NewString()
	now := graph.clock.Now()
	var revokedAt, completedAt *time.Time
	if revoked {
		revokedAt = &now
	}
	if completed {
		completedAt = &now
	}
	require.NoError(t, graph.db.Exec(
		`INSERT INTO interview_assignments (id, candidate_id, instance_id, published_revision_id, access_revoked_at, completed_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, candidateID, instanceID, revisionID, revokedAt, completedAt, now, now,
	).Error)
	t.Cleanup(func() {
		require.NoError(t, graph.db.Exec(`DELETE FROM candidate_sessions WHERE assignment_id = ?`, id).Error)
		require.NoError(t, graph.db.Exec(`DELETE FROM interview_invitations WHERE assignment_id = ?`, id).Error)
		require.NoError(t, graph.db.Exec(`DELETE FROM interview_assignments WHERE id = ?`, id).Error)
	})
	return id
}

// revokeAssignment stamps access_revoked_at on the Interview-owned assignment row.
func revokeAssignment(t *testing.T, graph accessHTTP, assignmentID string) {
	t.Helper()
	now := graph.clock.Now()
	require.NoError(t, graph.db.Exec(`UPDATE interview_assignments SET access_revoked_at = ? WHERE id = ?`, now, assignmentID).Error)
}

// countExecutions reports the Interview execution rows pinned to one assignment, which Access must never create.
func countExecutions(t *testing.T, graph accessHTTP, assignmentID string) int64 {
	t.Helper()
	var n int64
	require.NoError(t, graph.db.Raw(`SELECT COUNT(*) FROM interview_executions WHERE assignment_id = ?`, assignmentID).Scan(&n).Error)
	return n
}

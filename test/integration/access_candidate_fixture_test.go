package integration_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/query/accessquery"
	"github.com/rfanazhari/amer-prototype/application/usecase/accessusecase"
	"github.com/rfanazhari/amer-prototype/delivery/http/handler/accesshandler"
	"github.com/rfanazhari/amer-prototype/infrastructure/repository/accessrepository"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accessauthority"
	"github.com/rfanazhari/amer-prototype/infrastructure/service/accesscredentials"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newCandidateAccessHTTP installs the Interview-shaped assignment fixture and wires candidate operations.
func newCandidateAccessHTTP(t *testing.T) accessHTTP {
	t.Helper()
	graph := newAccessHTTP(t)
	installInterviewAssignmentFixture(t, graph.db)
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
		Login:            accessusecase.Login{Accounts: store, Issuance: store, Hasher: hasher, Tokens: tokens, Clock: graph.clock},
		Current:          accessquery.CurrentSession{Current: sessions, CSRF: csrf, Clock: graph.clock},
		Provision:        accessusecase.ProvisionHR{Accounts: store, Current: sessions, Hasher: hasher, Clock: graph.clock},
		Logout:           accessusecase.Logout{Current: sessions, Sessions: store, Clock: graph.clock},
		CreateCandidate:  accessusecase.CreateCandidate{Candidates: candidates, Current: sessions, Clock: graph.clock},
		Candidate:        accessquery.Candidate{Candidates: candidates, Current: sessions, Clock: graph.clock},
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

// installInterviewAssignmentFixture creates candidate access tables plus a disposable Interview-shaped assignment table.
func installInterviewAssignmentFixture(t *testing.T, db *gorm.DB) {
	t.Helper()
	statements := []string{
		`CREATE TABLE IF NOT EXISTS candidates (
			id uuid PRIMARY KEY,
			name text NOT NULL,
			email text NOT NULL,
			created_by uuid NOT NULL,
			created_at timestamptz NOT NULL,
			updated_at timestamptz NOT NULL,
			CONSTRAINT candidates_created_by_fkey FOREIGN KEY (created_by) REFERENCES administrative_accounts(id) ON DELETE RESTRICT ON UPDATE RESTRICT,
			CONSTRAINT candidates_updated_at_check CHECK (updated_at >= created_at)
		)`,
		`CREATE INDEX IF NOT EXISTS "candidates_created_by_INDEX" ON candidates (created_by)`,
		`CREATE TABLE IF NOT EXISTS interview_assignments (
			id uuid PRIMARY KEY,
			candidate_id uuid NOT NULL,
			published_revision_id uuid NOT NULL,
			access_revoked_at timestamptz NULL,
			completed_at timestamptz NULL,
			CONSTRAINT interview_assignments_candidate_id_fkey FOREIGN KEY (candidate_id) REFERENCES candidates(id) ON DELETE RESTRICT ON UPDATE RESTRICT
		)`,
		`CREATE TABLE IF NOT EXISTS interview_invitations (
			id uuid PRIMARY KEY,
			assignment_id uuid NOT NULL,
			token_hash text NOT NULL,
			created_by uuid NOT NULL,
			created_at timestamptz NOT NULL,
			expires_at timestamptz NOT NULL,
			revoked_at timestamptz NULL,
			CONSTRAINT interview_invitations_assignment_id_id_key UNIQUE (assignment_id, id),
			CONSTRAINT interview_invitations_token_hash_check CHECK (token_hash ~ '^[0-9a-f]{64}$'),
			CONSTRAINT interview_invitations_expiry_check CHECK (expires_at > created_at),
			CONSTRAINT interview_invitations_revoked_at_check CHECK (revoked_at IS NULL OR revoked_at >= created_at),
			CONSTRAINT interview_invitations_assignment_id_fkey FOREIGN KEY (assignment_id) REFERENCES interview_assignments(id) ON DELETE RESTRICT ON UPDATE RESTRICT,
			CONSTRAINT interview_invitations_created_by_fkey FOREIGN KEY (created_by) REFERENCES administrative_accounts(id) ON DELETE RESTRICT ON UPDATE RESTRICT
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS "interview_invitations_token_hash_INDEX" ON interview_invitations (token_hash)`,
		`CREATE INDEX IF NOT EXISTS "interview_invitations_assignment_id_INDEX" ON interview_invitations (assignment_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS "interview_invitations_current_assignment_INDEX" ON interview_invitations (assignment_id) WHERE revoked_at IS NULL`,
		`CREATE TABLE IF NOT EXISTS candidate_sessions (
			id uuid PRIMARY KEY,
			assignment_id uuid NOT NULL,
			invitation_id uuid NOT NULL,
			token_hash text NOT NULL,
			created_at timestamptz NOT NULL,
			expires_at timestamptz NOT NULL,
			revoked_at timestamptz NULL,
			CONSTRAINT candidate_sessions_token_hash_check CHECK (token_hash ~ '^[0-9a-f]{64}$'),
			CONSTRAINT candidate_sessions_expiry_check CHECK (expires_at > created_at),
			CONSTRAINT candidate_sessions_revoked_at_check CHECK (revoked_at IS NULL OR revoked_at >= created_at),
			CONSTRAINT candidate_sessions_assignment_id_fkey FOREIGN KEY (assignment_id) REFERENCES interview_assignments(id) ON DELETE RESTRICT ON UPDATE RESTRICT,
			CONSTRAINT candidate_sessions_invitation_same_assignment_fkey FOREIGN KEY (assignment_id, invitation_id) REFERENCES interview_invitations (assignment_id, id) ON DELETE RESTRICT ON UPDATE RESTRICT
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS "candidate_sessions_token_hash_INDEX" ON candidate_sessions (token_hash)`,
		`CREATE INDEX IF NOT EXISTS "candidate_sessions_assignment_id_INDEX" ON candidate_sessions (assignment_id)`,
		`CREATE INDEX IF NOT EXISTS "candidate_sessions_expires_at_INDEX" ON candidate_sessions (expires_at)`,
	}
	for _, statement := range statements {
		require.NoError(t, db.Exec(statement).Error)
	}
}

// insertAssignment writes one Interview-shaped assignment row for Access collaboration.
func insertAssignment(t *testing.T, graph accessHTTP, candidateID string, revoked, completed bool) string {
	t.Helper()
	id := uuid.NewString()
	revision := uuid.NewString()
	var revokedAt, completedAt *time.Time
	now := graph.clock.Now()
	if revoked {
		revokedAt = &now
	}
	if completed {
		completedAt = &now
	}
	require.NoError(t, graph.db.Exec(
		`INSERT INTO interview_assignments (id, candidate_id, published_revision_id, access_revoked_at, completed_at) VALUES (?, ?, ?, ?, ?)`,
		id, candidateID, revision, revokedAt, completedAt,
	).Error)
	return id
}

// revokeAssignment stamps access_revoked_at on the Interview-owned assignment row.
func revokeAssignment(t *testing.T, graph accessHTTP, assignmentID string) {
	t.Helper()
	now := graph.clock.Now()
	require.NoError(t, graph.db.Exec(`UPDATE interview_assignments SET access_revoked_at = ? WHERE id = ?`, now, assignmentID).Error)
}

// countExecutions reports Interview execution tables that Access must not create.
func countExecutions(t *testing.T, graph accessHTTP) int64 {
	t.Helper()
	var n int64
	require.NoError(t, graph.db.Raw(`SELECT COUNT(*) FROM pg_tables WHERE schemaname = 'public' AND tablename IN ('interview_executions','consent_records','interview_runs')`).Scan(&n).Error)
	return n
}

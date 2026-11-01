-- Invitation and candidate-session access. Requires Interview-owned interview_assignments
-- (id, candidate_id, published_revision_id, access_revoked_at, completed_at) to exist first.
-- Access never creates interview_assignments.

CREATE TABLE interview_invitations (
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
);

CREATE UNIQUE INDEX "interview_invitations_token_hash_INDEX" ON interview_invitations (token_hash);
CREATE INDEX "interview_invitations_assignment_id_INDEX" ON interview_invitations (assignment_id);
CREATE UNIQUE INDEX "interview_invitations_current_assignment_INDEX" ON interview_invitations (assignment_id) WHERE revoked_at IS NULL;

CREATE TABLE candidate_sessions (
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
);

CREATE UNIQUE INDEX "candidate_sessions_token_hash_INDEX" ON candidate_sessions (token_hash);
CREATE INDEX "candidate_sessions_assignment_id_INDEX" ON candidate_sessions (assignment_id);
CREATE INDEX "candidate_sessions_expires_at_INDEX" ON candidate_sessions (expires_at);

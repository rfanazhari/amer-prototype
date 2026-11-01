-- Disposable databases only. Drop sessions before invitations. Never drop Interview-owned tables.

DROP TABLE IF EXISTS candidate_sessions;
DROP TABLE IF EXISTS interview_invitations;

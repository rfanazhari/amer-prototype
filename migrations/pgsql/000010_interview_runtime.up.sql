BEGIN;
-- Interview runtime progress storage. Apply after clean version 9. interview_executions keeps its version 7 through 9 shape.
-- Stage names are background, screening, compensation, and closing. attempt_id columns have no foreign key.

CREATE TABLE public.interview_progress (
    execution_id uuid PRIMARY KEY,
    follow_up_limit integer NOT NULL,
    current_stage text NOT NULL,
    stage_activated_at timestamptz NOT NULL,
    committed_version bigint NOT NULL,
    controller_id uuid NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT interview_progress_follow_up_limit_check CHECK (follow_up_limit = 2),
    CONSTRAINT interview_progress_current_stage_check CHECK (current_stage IN ('background', 'screening', 'compensation', 'closing')),
    CONSTRAINT interview_progress_committed_version_check CHECK (committed_version >= 1),
    CONSTRAINT interview_progress_updated_at_check CHECK (updated_at >= created_at),
    CONSTRAINT interview_progress_execution_id_fkey FOREIGN KEY (execution_id)
        REFERENCES public.interview_executions(id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE TABLE public.interview_coverage_items (
    id uuid PRIMARY KEY,
    execution_id uuid NOT NULL,
    question_id text NOT NULL,
    stage text NOT NULL,
    state text NOT NULL,
    omission_reason text NULL,
    clarification_count integer NOT NULL,
    answer_entry_id uuid NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT interview_coverage_items_stage_check CHECK (stage IN ('background', 'screening', 'compensation', 'closing')),
    CONSTRAINT interview_coverage_items_state_check CHECK (state IN ('pending', 'asked', 'answered', 'omitted')),
    CONSTRAINT interview_coverage_items_omission_reason_check CHECK (
        (state = 'omitted' AND omission_reason IN ('candidate_skip', 'no_answer_after_clarification', 'soft_budget_expired'))
        OR (state <> 'omitted' AND omission_reason IS NULL)
    ),
    CONSTRAINT interview_coverage_items_clarification_count_check CHECK (clarification_count >= 0 AND clarification_count <= 1),
    CONSTRAINT interview_coverage_items_execution_id_question_id_key UNIQUE (execution_id, question_id),
    CONSTRAINT interview_coverage_items_execution_id_fkey FOREIGN KEY (execution_id)
        REFERENCES public.interview_progress(execution_id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE TABLE public.interview_technical_losses (
    id uuid PRIMARY KEY,
    execution_id uuid NOT NULL,
    question_id text NOT NULL,
    session_id uuid NOT NULL,
    attempt_id uuid NOT NULL,
    resolved_at timestamptz NULL,
    created_at timestamptz NOT NULL,
    CONSTRAINT interview_technical_losses_execution_id_fkey FOREIGN KEY (execution_id)
        REFERENCES public.interview_progress(execution_id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE INDEX "interview_technical_losses_unresolved_INDEX" ON public.interview_technical_losses USING btree (execution_id) WHERE resolved_at IS NULL;

CREATE TABLE public.interview_voice_sessions (
    id uuid PRIMARY KEY,
    execution_id uuid NOT NULL,
    stage text NOT NULL,
    attempt_id uuid NOT NULL,
    provider_session_id text NULL,
    readiness_at timestamptz NULL,
    replaced_session_id uuid NULL,
    opened_at timestamptz NOT NULL,
    closed_at timestamptz NULL,
    CONSTRAINT interview_voice_sessions_stage_check CHECK (stage IN ('background', 'screening', 'compensation', 'closing')),
    CONSTRAINT interview_voice_sessions_execution_id_fkey FOREIGN KEY (execution_id)
        REFERENCES public.interview_progress(execution_id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT interview_voice_sessions_replaced_session_id_fkey FOREIGN KEY (replaced_session_id)
        REFERENCES public.interview_voice_sessions(id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE UNIQUE INDEX "interview_voice_sessions_open_execution_INDEX" ON public.interview_voice_sessions (execution_id) WHERE closed_at IS NULL;
CREATE UNIQUE INDEX "interview_voice_sessions_provider_session_INDEX" ON public.interview_voice_sessions (provider_session_id) WHERE provider_session_id IS NOT NULL;

CREATE TABLE public.interview_transcript_entries (
    id uuid PRIMARY KEY,
    execution_id uuid NOT NULL,
    session_id uuid NOT NULL,
    attempt_id uuid NOT NULL,
    speaker text NOT NULL,
    ordinal integer NOT NULL,
    revision integer NOT NULL,
    evidence_state text NOT NULL,
    body text NOT NULL,
    cites_entry_id uuid NULL,
    supersedes_entry_id uuid NULL,
    committed_at timestamptz NOT NULL,
    CONSTRAINT interview_transcript_entries_speaker_check CHECK (speaker IN ('candidate', 'model')),
    CONSTRAINT interview_transcript_entries_evidence_state_check CHECK (evidence_state IN ('fragment', 'final')),
    CONSTRAINT interview_transcript_entries_ordinal_check CHECK (ordinal >= 1),
    CONSTRAINT interview_transcript_entries_revision_check CHECK (revision >= 1),
    CONSTRAINT interview_transcript_entries_session_id_ordinal_revision_key UNIQUE (session_id, ordinal, revision),
    CONSTRAINT interview_transcript_entries_execution_id_fkey FOREIGN KEY (execution_id)
        REFERENCES public.interview_progress(execution_id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT interview_transcript_entries_session_id_fkey FOREIGN KEY (session_id)
        REFERENCES public.interview_voice_sessions(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT interview_transcript_entries_cites_entry_id_fkey FOREIGN KEY (cites_entry_id)
        REFERENCES public.interview_transcript_entries(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT interview_transcript_entries_supersedes_entry_id_fkey FOREIGN KEY (supersedes_entry_id)
        REFERENCES public.interview_transcript_entries(id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

ALTER TABLE public.interview_coverage_items
    ADD CONSTRAINT interview_coverage_items_answer_entry_id_fkey FOREIGN KEY (answer_entry_id)
        REFERENCES public.interview_transcript_entries(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

CREATE TABLE public.interview_checkpoints (
    id uuid PRIMARY KEY,
    execution_id uuid NOT NULL,
    session_id uuid NOT NULL,
    attempt_id uuid NOT NULL,
    through_ordinal integer NOT NULL,
    created_at timestamptz NOT NULL,
    CONSTRAINT interview_checkpoints_execution_id_fkey FOREIGN KEY (execution_id)
        REFERENCES public.interview_progress(execution_id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT interview_checkpoints_session_id_fkey FOREIGN KEY (session_id)
        REFERENCES public.interview_voice_sessions(id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE INDEX "interview_checkpoints_execution_id_created_at_INDEX" ON public.interview_checkpoints USING btree (execution_id, created_at);

CREATE TABLE public.interview_handoffs (
    id uuid PRIMARY KEY,
    execution_id uuid NOT NULL,
    source_stage text NOT NULL,
    target_stage text NULL,
    revision integer NOT NULL,
    summary text NULL,
    created_at timestamptz NOT NULL,
    CONSTRAINT interview_handoffs_source_stage_check CHECK (source_stage IN ('background', 'screening', 'compensation', 'closing')),
    CONSTRAINT interview_handoffs_target_stage_check CHECK (
        target_stage IS NULL OR target_stage IN ('background', 'screening', 'compensation', 'closing')
    ),
    CONSTRAINT interview_handoffs_revision_check CHECK (revision >= 1),
    CONSTRAINT interview_handoffs_execution_id_source_stage_key UNIQUE (execution_id, source_stage),
    CONSTRAINT interview_handoffs_execution_id_revision_key UNIQUE (execution_id, revision),
    CONSTRAINT interview_handoffs_execution_id_fkey FOREIGN KEY (execution_id)
        REFERENCES public.interview_progress(execution_id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE TABLE public.interview_session_commands (
    id uuid PRIMARY KEY,
    execution_id uuid NOT NULL,
    idempotency_key text NOT NULL,
    intent text NOT NULL,
    attempt_id uuid NOT NULL,
    created_at timestamptz NOT NULL,
    CONSTRAINT interview_session_commands_intent_check CHECK (intent IN ('activate', 'reconnect')),
    CONSTRAINT interview_session_commands_execution_id_idempotency_key_key UNIQUE (execution_id, idempotency_key),
    CONSTRAINT interview_session_commands_execution_id_fkey FOREIGN KEY (execution_id)
        REFERENCES public.interview_progress(execution_id) ON UPDATE RESTRICT ON DELETE RESTRICT
);
COMMIT;

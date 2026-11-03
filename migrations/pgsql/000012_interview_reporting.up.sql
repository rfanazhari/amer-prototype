BEGIN;
-- Interview reporting storage. Apply after clean version 11. Existing tables are not altered and nothing is seeded.
-- Statement bodies are the only free text. Evidence references have no foreign key: the evidence port owns finality.

CREATE TABLE public.interview_reports (
    id uuid PRIMARY KEY,
    execution_id uuid NOT NULL,
    assignment_id uuid NOT NULL,
    created_at timestamptz NOT NULL,
    CONSTRAINT interview_reports_execution_id_key UNIQUE (execution_id),
    CONSTRAINT interview_reports_assignment_id_key UNIQUE (assignment_id),
    CONSTRAINT interview_reports_execution_id_fkey FOREIGN KEY (execution_id)
        REFERENCES public.interview_executions(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT interview_reports_assignment_id_fkey FOREIGN KEY (assignment_id)
        REFERENCES public.interview_assignments(id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE TABLE public.interview_report_generations (
    id uuid PRIMARY KEY,
    report_id uuid NOT NULL,
    generation_number integer NOT NULL,
    state text NOT NULL,
    failure_reason text NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT interview_report_generations_generation_number_check CHECK (generation_number > 0),
    CONSTRAINT interview_report_generations_state_check CHECK (state IN ('pending', 'running', 'succeeded', 'failed')),
    CONSTRAINT interview_report_generations_failure_reason_check CHECK ((state = 'failed') = (failure_reason IS NOT NULL)),
    CONSTRAINT interview_report_generations_updated_at_check CHECK (updated_at >= created_at),
    CONSTRAINT interview_report_generations_report_id_generation_number_key UNIQUE (report_id, generation_number),
    CONSTRAINT interview_report_generations_report_id_fkey FOREIGN KEY (report_id)
        REFERENCES public.interview_reports(id) ON UPDATE RESTRICT ON DELETE RESTRICT
);
CREATE INDEX "interview_report_generations_report_id_INDEX" ON public.interview_report_generations USING btree (report_id);
CREATE UNIQUE INDEX "interview_report_generations_one_active_INDEX" ON public.interview_report_generations (report_id) WHERE state IN ('pending', 'running');

CREATE TABLE public.interview_report_statements (
    id uuid PRIMARY KEY,
    generation_id uuid NOT NULL,
    ordinal integer NOT NULL,
    kind text NOT NULL,
    body text NOT NULL,
    evidence_entry_id uuid NULL,
    evidence_revision integer NULL,
    presents_as_answer boolean NOT NULL,
    CONSTRAINT interview_report_statements_ordinal_check CHECK (ordinal >= 0),
    CONSTRAINT interview_report_statements_kind_check CHECK (kind IN ('observation', 'gap', 'uncertainty')),
    CONSTRAINT interview_report_statements_body_check CHECK (char_length(body) BETWEEN 1 AND 2000),
    CONSTRAINT interview_report_statements_evidence_revision_check CHECK (evidence_revision IS NULL OR evidence_revision > 0),
    CONSTRAINT interview_report_statements_evidence_pair_check CHECK ((evidence_entry_id IS NULL) = (evidence_revision IS NULL)),
    CONSTRAINT interview_report_statements_answer_check CHECK (
        NOT presents_as_answer OR (kind = 'observation' AND evidence_entry_id IS NOT NULL AND evidence_revision IS NOT NULL)
    ),
    CONSTRAINT interview_report_statements_gap_check CHECK (
        kind <> 'gap' OR (presents_as_answer = false AND evidence_entry_id IS NULL AND evidence_revision IS NULL)
    ),
    CONSTRAINT interview_report_statements_generation_id_ordinal_key UNIQUE (generation_id, ordinal),
    CONSTRAINT interview_report_statements_generation_id_fkey FOREIGN KEY (generation_id)
        REFERENCES public.interview_report_generations(id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE TABLE public.interview_report_commands (
    id uuid PRIMARY KEY,
    scope uuid NOT NULL,
    command_kind text NOT NULL,
    request_key text NOT NULL,
    intent_fingerprint text NOT NULL,
    generation_id uuid NOT NULL,
    created_at timestamptz NOT NULL,
    CONSTRAINT interview_report_commands_command_kind_check CHECK (command_kind = 'regenerate-interview-report/v1'),
    CONSTRAINT interview_report_commands_intent_fingerprint_check CHECK (intent_fingerprint ~ '^[0-9a-f]{64}$'),
    CONSTRAINT interview_report_commands_scope_command_kind_request_key_key UNIQUE (scope, command_kind, request_key),
    CONSTRAINT interview_report_commands_scope_fkey FOREIGN KEY (scope)
        REFERENCES public.administrative_accounts(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT interview_report_commands_generation_id_fkey FOREIGN KEY (generation_id)
        REFERENCES public.interview_report_generations(id) ON UPDATE RESTRICT ON DELETE RESTRICT
);
CREATE INDEX "interview_report_commands_generation_id_INDEX" ON public.interview_report_commands USING btree (generation_id);

CREATE TABLE public.interview_report_jobs (
    id uuid PRIMARY KEY,
    generation_id uuid NOT NULL,
    state text NOT NULL,
    attempt_count integer NOT NULL,
    fencing_token bigint NOT NULL,
    created_at timestamptz NOT NULL,
    wait_deadline_at timestamptz NOT NULL,
    window_started_at timestamptz NULL,
    deadline_at timestamptz NULL,
    eligible_at timestamptz NOT NULL,
    attempt_deadline_at timestamptz NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT interview_report_jobs_state_check CHECK (state IN ('pending', 'published', 'waiting', 'claimed', 'succeeded', 'failed', 'expired')),
    CONSTRAINT interview_report_jobs_attempt_count_check CHECK (attempt_count >= 0 AND attempt_count <= 2),
    CONSTRAINT interview_report_jobs_fencing_token_check CHECK (fencing_token > 0),
    CONSTRAINT interview_report_jobs_wait_deadline_at_check CHECK (wait_deadline_at = created_at + interval '30 minutes'),
    CONSTRAINT interview_report_jobs_window_check CHECK (
        (window_started_at IS NULL AND deadline_at IS NULL)
        OR (
            window_started_at IS NOT NULL AND deadline_at IS NOT NULL
            AND deadline_at > window_started_at
            AND deadline_at <= window_started_at + interval '5 minutes'
        )
    ),
    CONSTRAINT interview_report_jobs_unwindowed_states_check CHECK (
        state NOT IN ('pending', 'published', 'waiting') OR (window_started_at IS NULL AND deadline_at IS NULL)
    ),
    CONSTRAINT interview_report_jobs_updated_at_check CHECK (updated_at >= created_at),
    CONSTRAINT interview_report_jobs_generation_id_key UNIQUE (generation_id),
    CONSTRAINT interview_report_jobs_generation_id_fkey FOREIGN KEY (generation_id)
        REFERENCES public.interview_report_generations(id) ON UPDATE RESTRICT ON DELETE RESTRICT
);
CREATE INDEX "interview_report_jobs_eligible_INDEX" ON public.interview_report_jobs USING btree (eligible_at, id) WHERE state IN ('pending', 'published');

CREATE TABLE public.interview_report_attempts (
    id uuid PRIMARY KEY,
    job_id uuid NOT NULL,
    attempt_number integer NOT NULL,
    fencing_token bigint NOT NULL,
    started_at timestamptz NOT NULL,
    deadline_at timestamptz NOT NULL,
    finished_at timestamptz NULL,
    outcome text NULL,
    CONSTRAINT interview_report_attempts_attempt_number_check CHECK (attempt_number IN (1, 2)),
    CONSTRAINT interview_report_attempts_deadline_at_check CHECK (
        deadline_at > started_at AND deadline_at <= started_at + interval '60 seconds'
    ),
    CONSTRAINT interview_report_attempts_outcome_check CHECK (outcome IS NULL OR outcome IN ('succeeded', 'failed', 'expired', 'superseded')),
    CONSTRAINT interview_report_attempts_job_id_attempt_number_key UNIQUE (job_id, attempt_number),
    CONSTRAINT interview_report_attempts_job_id_fkey FOREIGN KEY (job_id)
        REFERENCES public.interview_report_jobs(id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE TABLE public.outbox_messages (
    id uuid PRIMARY KEY,
    job_kind text NOT NULL,
    job_id uuid NOT NULL,
    created_at timestamptz NOT NULL,
    publish_claimed_at timestamptz NULL,
    confirmed_at timestamptz NULL,
    CONSTRAINT outbox_messages_job_kind_check CHECK (job_kind = 'interview_report')
);
CREATE INDEX "outbox_messages_unconfirmed_INDEX" ON public.outbox_messages USING btree (created_at, id) WHERE confirmed_at IS NULL;
COMMIT;

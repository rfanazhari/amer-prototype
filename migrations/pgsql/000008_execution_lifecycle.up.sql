BEGIN;
-- Shared execution lifecycle. Apply after clean version 7. Pin columns of interview_executions stay as version 7 defines them.
-- Tokens (scope, request_key, operation_key, actor_ref, reason) are 1 to 64 characters from A-Z a-z 0-9 . _ : -

CREATE TABLE public.agent_executions (
    id uuid NOT NULL,
    instance_id uuid NOT NULL,
    published_revision_id uuid NOT NULL,
    state text NOT NULL,
    state_version bigint NOT NULL,
    committed_progress_ref uuid NULL,
    activated_at timestamptz NULL,
    terminal_at timestamptz NULL,
    terminal_reason text NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT agent_executions_state_check CHECK (state IN ('CREATED', 'RUNNING', 'COMPLETED', 'FAILED', 'CANCELLED')),
    CONSTRAINT agent_executions_state_version_check CHECK (state_version >= 1),
    CONSTRAINT agent_executions_updated_at_check CHECK (updated_at >= created_at),
    CONSTRAINT agent_executions_terminal_reason_check CHECK (terminal_reason IS NULL OR terminal_reason ~ '^[A-Za-z0-9._:-]{1,64}$'),
    CONSTRAINT agent_executions_state_shape_check CHECK (
        (state = 'CREATED'
            AND activated_at IS NULL AND terminal_at IS NULL AND committed_progress_ref IS NULL AND terminal_reason IS NULL)
        OR (state = 'RUNNING'
            AND activated_at IS NOT NULL AND activated_at >= created_at AND terminal_at IS NULL AND terminal_reason IS NULL)
        OR (state = 'COMPLETED'
            AND activated_at IS NOT NULL AND terminal_at IS NOT NULL AND committed_progress_ref IS NOT NULL
            AND terminal_at >= activated_at AND terminal_reason IS NULL)
        OR (state IN ('FAILED', 'CANCELLED')
            AND terminal_at IS NOT NULL AND terminal_at >= created_at
            AND (activated_at IS NULL OR activated_at <= terminal_at) AND terminal_reason IS NOT NULL)
    )
);

CREATE TABLE public.agent_execution_commands (
    id uuid NOT NULL,
    scope text NOT NULL,
    command_kind text NOT NULL,
    request_key text NOT NULL,
    intent_fingerprint char(64) NOT NULL,
    execution_id uuid NOT NULL,
    created_at timestamptz NOT NULL,
    CONSTRAINT agent_execution_commands_scope_check CHECK (scope ~ '^[A-Za-z0-9._:-]{1,64}$'),
    CONSTRAINT agent_execution_commands_command_kind_check CHECK (command_kind IN ('start', 'activate', 'complete', 'fail', 'cancel', 'attempt')),
    CONSTRAINT agent_execution_commands_request_key_check CHECK (request_key ~ '^[A-Za-z0-9._:-]{1,64}$'),
    CONSTRAINT agent_execution_commands_intent_fingerprint_check CHECK (intent_fingerprint ~ '^[0-9a-f]{64}$')
);

CREATE TABLE public.agent_execution_operations (
    execution_id uuid NOT NULL,
    operation_key text NOT NULL,
    state text NOT NULL,
    current_authority_generation bigint NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT agent_execution_operations_state_check CHECK (state IN ('PENDING', 'IN_PROGRESS', 'SUCCEEDED', 'FAILED', 'CANCELLED')),
    CONSTRAINT agent_execution_operations_operation_key_check CHECK (operation_key ~ '^[A-Za-z0-9._:-]{1,64}$'),
    CONSTRAINT agent_execution_operations_generation_check CHECK (current_authority_generation >= 1),
    CONSTRAINT agent_execution_operations_updated_at_check CHECK (updated_at >= created_at)
);

CREATE TABLE public.agent_execution_attempts (
    id uuid NOT NULL,
    execution_id uuid NOT NULL,
    operation_key text NOT NULL,
    attempt_number bigint NOT NULL,
    authority_generation bigint NOT NULL,
    outcome text NOT NULL,
    reason text NULL,
    deadline_at timestamptz NULL,
    created_at timestamptz NOT NULL,
    CONSTRAINT agent_execution_attempts_attempt_number_check CHECK (attempt_number >= 1),
    CONSTRAINT agent_execution_attempts_generation_check CHECK (authority_generation >= 1),
    CONSTRAINT agent_execution_attempts_outcome_check CHECK (outcome IN ('PENDING', 'IN_PROGRESS', 'SUCCEEDED', 'FAILED', 'CANCELLED')),
    CONSTRAINT agent_execution_attempts_reason_check CHECK (reason IS NULL OR reason ~ '^[A-Za-z0-9._:-]{1,64}$')
);

CREATE TABLE public.agent_execution_transitions (
    id uuid NOT NULL,
    execution_id uuid NOT NULL,
    from_state text NULL,
    to_state text NOT NULL,
    state_version bigint NOT NULL,
    actor_ref text NOT NULL,
    operation_key text NULL,
    reason text NULL,
    occurred_at timestamptz NOT NULL,
    CONSTRAINT agent_execution_transitions_from_state_check CHECK (from_state IS NULL OR from_state IN ('CREATED', 'RUNNING', 'COMPLETED', 'FAILED', 'CANCELLED')),
    CONSTRAINT agent_execution_transitions_to_state_check CHECK (to_state IN ('CREATED', 'RUNNING', 'COMPLETED', 'FAILED', 'CANCELLED')),
    CONSTRAINT agent_execution_transitions_state_version_check CHECK (state_version >= 1),
    CONSTRAINT agent_execution_transitions_actor_ref_check CHECK (actor_ref ~ '^[A-Za-z0-9._:-]{1,64}$'),
    CONSTRAINT agent_execution_transitions_operation_key_check CHECK (operation_key IS NULL OR operation_key ~ '^[A-Za-z0-9._:-]{1,64}$'),
    CONSTRAINT agent_execution_transitions_reason_check CHECK (reason IS NULL OR reason ~ '^[A-Za-z0-9._:-]{1,64}$')
);

ALTER TABLE ONLY public.agent_executions
    ADD CONSTRAINT agent_executions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.agent_execution_commands
    ADD CONSTRAINT agent_execution_commands_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.agent_execution_commands
    ADD CONSTRAINT agent_execution_commands_identity_key UNIQUE (scope, command_kind, request_key);

ALTER TABLE ONLY public.agent_execution_operations
    ADD CONSTRAINT agent_execution_operations_pkey PRIMARY KEY (execution_id, operation_key);

ALTER TABLE ONLY public.agent_execution_attempts
    ADD CONSTRAINT agent_execution_attempts_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.agent_execution_attempts
    ADD CONSTRAINT agent_execution_attempts_number_key UNIQUE (execution_id, operation_key, attempt_number);

ALTER TABLE ONLY public.agent_execution_attempts
    ADD CONSTRAINT agent_execution_attempts_generation_key UNIQUE (execution_id, operation_key, authority_generation);

ALTER TABLE ONLY public.agent_execution_transitions
    ADD CONSTRAINT agent_execution_transitions_pkey PRIMARY KEY (id);

CREATE INDEX "agent_executions_instance_id_INDEX" ON public.agent_executions USING btree (instance_id);

CREATE INDEX "agent_executions_published_revision_id_INDEX" ON public.agent_executions USING btree (published_revision_id);

CREATE INDEX "agent_execution_attempts_execution_id_INDEX" ON public.agent_execution_attempts USING btree (execution_id);

ALTER TABLE ONLY public.agent_executions
    ADD CONSTRAINT agent_executions_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.agent_instances(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE ONLY public.agent_executions
    ADD CONSTRAINT agent_executions_published_revision_id_fkey FOREIGN KEY (published_revision_id) REFERENCES public.agent_revisions(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE ONLY public.agent_execution_commands
    ADD CONSTRAINT agent_execution_commands_execution_id_fkey FOREIGN KEY (execution_id) REFERENCES public.agent_executions(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE ONLY public.agent_execution_operations
    ADD CONSTRAINT agent_execution_operations_execution_id_fkey FOREIGN KEY (execution_id) REFERENCES public.agent_executions(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE ONLY public.agent_execution_attempts
    ADD CONSTRAINT agent_execution_attempts_operation_fkey FOREIGN KEY (execution_id, operation_key) REFERENCES public.agent_execution_operations(execution_id, operation_key) ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE ONLY public.agent_execution_transitions
    ADD CONSTRAINT agent_execution_transitions_execution_id_fkey FOREIGN KEY (execution_id) REFERENCES public.agent_executions(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

-- Existing pins become accepted executions with the pin's own binding and time. No history is invented.
INSERT INTO public.agent_executions (id, instance_id, published_revision_id, state, state_version, created_at, updated_at)
SELECT id, instance_id, published_revision_id, 'CREATED', 1, created_at, created_at
FROM public.interview_executions;

ALTER TABLE ONLY public.interview_executions
    ADD CONSTRAINT interview_executions_id_fkey FOREIGN KEY (id) REFERENCES public.agent_executions(id) ON UPDATE RESTRICT ON DELETE RESTRICT;
COMMIT;

BEGIN;
-- Candidate interview consent and controller. Apply after clean version 8. interview_executions keeps its version 7 and 8 shape.
-- Identifiers (notice_version, controller_id) are 1 to 64 characters from A-Z a-z 0-9 . _ : - and start with a letter or digit.

CREATE TABLE public.interview_consents (
    id uuid PRIMARY KEY,
    assignment_id uuid NOT NULL,
    notice_version text NOT NULL,
    accepted_at timestamptz NOT NULL,
    candidate_session_id uuid NOT NULL,
    CONSTRAINT interview_consents_notice_version_check CHECK (notice_version ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$'),
    CONSTRAINT interview_consents_assignment_id_notice_version_key UNIQUE (assignment_id, notice_version),
    CONSTRAINT interview_consents_assignment_id_fkey FOREIGN KEY (assignment_id)
        REFERENCES public.interview_assignments(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT interview_consents_candidate_session_id_fkey FOREIGN KEY (candidate_session_id)
        REFERENCES public.candidate_sessions(id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE INDEX "interview_consents_assignment_id_INDEX" ON public.interview_consents USING btree (assignment_id);

CREATE TABLE public.interview_controllers (
    assignment_id uuid PRIMARY KEY,
    execution_id uuid NOT NULL,
    controller_id text NOT NULL,
    version bigint NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT interview_controllers_controller_id_check CHECK (controller_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$'),
    CONSTRAINT interview_controllers_version_check CHECK (version >= 1),
    CONSTRAINT interview_controllers_assignment_id_fkey FOREIGN KEY (assignment_id)
        REFERENCES public.interview_assignments(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT interview_controllers_execution_id_fkey FOREIGN KEY (execution_id)
        REFERENCES public.interview_executions(id) ON UPDATE RESTRICT ON DELETE RESTRICT
);
COMMIT;

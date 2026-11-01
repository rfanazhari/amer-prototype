-- Interview-owned assignment pins. Apply after candidates (4) and configuration (3).
-- Access never creates this table; invitation FKs in version 6 require it.

CREATE TABLE public.interview_assignments (
    id uuid PRIMARY KEY,
    candidate_id uuid NOT NULL,
    instance_id uuid NOT NULL,
    published_revision_id uuid NOT NULL,
    access_revoked_at timestamptz NULL,
    completed_at timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT interview_assignments_updated_at_check CHECK (updated_at >= created_at),
    CONSTRAINT interview_assignments_revoked_at_check CHECK (access_revoked_at IS NULL OR access_revoked_at >= created_at),
    CONSTRAINT interview_assignments_completed_at_check CHECK (completed_at IS NULL OR completed_at >= created_at)
);

CREATE INDEX "interview_assignments_candidate_id_INDEX" ON public.interview_assignments USING btree (candidate_id);
CREATE INDEX "interview_assignments_instance_id_INDEX" ON public.interview_assignments USING btree (instance_id);
CREATE INDEX "interview_assignments_published_revision_id_INDEX" ON public.interview_assignments USING btree (published_revision_id);

ALTER TABLE ONLY public.interview_assignments
    ADD CONSTRAINT interview_assignments_candidate_id_fkey FOREIGN KEY (candidate_id) REFERENCES public.candidates(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE ONLY public.interview_assignments
    ADD CONSTRAINT interview_assignments_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.agent_instances(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE ONLY public.interview_assignments
    ADD CONSTRAINT interview_assignments_published_revision_id_fkey FOREIGN KEY (published_revision_id) REFERENCES public.agent_revisions(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

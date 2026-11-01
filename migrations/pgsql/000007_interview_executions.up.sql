-- Interview-owned execution pins. One execution per assignment; restore via published_revision_id.

CREATE TABLE public.interview_executions (
    id uuid PRIMARY KEY,
    assignment_id uuid NOT NULL,
    instance_id uuid NOT NULL,
    published_revision_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT interview_executions_assignment_id_key UNIQUE (assignment_id)
);

CREATE INDEX "interview_executions_instance_id_INDEX" ON public.interview_executions USING btree (instance_id);
CREATE INDEX "interview_executions_published_revision_id_INDEX" ON public.interview_executions USING btree (published_revision_id);

ALTER TABLE ONLY public.interview_executions
    ADD CONSTRAINT interview_executions_assignment_id_fkey FOREIGN KEY (assignment_id) REFERENCES public.interview_assignments(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE ONLY public.interview_executions
    ADD CONSTRAINT interview_executions_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.agent_instances(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE ONLY public.interview_executions
    ADD CONSTRAINT interview_executions_published_revision_id_fkey FOREIGN KEY (published_revision_id) REFERENCES public.agent_revisions(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

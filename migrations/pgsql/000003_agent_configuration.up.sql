BEGIN;
-- five configuration tables. Apply after clean version 2. Do not AutoMigrate.
-- zero/null update mappings: write active_revision_id even when NULL; write eligible even when false;
-- write configuration and findings even when empty JSON ({}/[]). Never rely on ORM omission.

CREATE TABLE public.agent_instances (
    id uuid NOT NULL,
    name text NOT NULL,
    template_id text NOT NULL,
    template_version text NOT NULL,
    behavior_version text NOT NULL,
    current_draft_version bigint NOT NULL,
    active_revision_id uuid,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT agent_instances_name_check CHECK (((char_length(name) > 0) AND (char_length(name) <= 120))),
    CONSTRAINT agent_instances_current_draft_version_check CHECK ((current_draft_version > 0)),
    CONSTRAINT agent_instances_updated_at_check CHECK ((updated_at >= created_at))
);

CREATE TABLE public.agent_drafts (
    instance_id uuid NOT NULL,
    version bigint NOT NULL,
    configuration jsonb NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT agent_drafts_version_check CHECK ((version > 0)),
    CONSTRAINT agent_drafts_updated_at_check CHECK ((updated_at >= created_at))
);

CREATE TABLE public.agent_validations (
    id uuid NOT NULL,
    instance_id uuid NOT NULL,
    draft_version bigint NOT NULL,
    template_id text NOT NULL,
    template_version text NOT NULL,
    behavior_version text NOT NULL,
    eligible boolean NOT NULL,
    findings jsonb NOT NULL,
    created_at timestamp with time zone NOT NULL,
    CONSTRAINT agent_validations_draft_version_check CHECK ((draft_version > 0))
);

CREATE TABLE public.agent_revisions (
    id uuid NOT NULL,
    instance_id uuid NOT NULL,
    revision_number bigint NOT NULL,
    source_draft_version bigint NOT NULL,
    validation_id uuid NOT NULL,
    template_id text NOT NULL,
    template_version text NOT NULL,
    behavior_version text NOT NULL,
    configuration jsonb NOT NULL,
    created_at timestamp with time zone NOT NULL,
    CONSTRAINT agent_revisions_revision_number_check CHECK ((revision_number > 0)),
    CONSTRAINT agent_revisions_source_draft_version_check CHECK ((source_draft_version > 0))
);

CREATE TABLE public.agent_publication_commands (
    scope uuid NOT NULL,
    command_kind text NOT NULL,
    request_key text NOT NULL,
    intent_fingerprint text NOT NULL,
    resulting_revision_id uuid NOT NULL,
    created_at timestamp with time zone NOT NULL,
    CONSTRAINT agent_publication_commands_command_kind_check CHECK ((command_kind = 'publish-agent-configuration/v1'::text)),
    CONSTRAINT agent_publication_commands_request_key_check CHECK ((request_key ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'::text)),
    CONSTRAINT agent_publication_commands_intent_fingerprint_check CHECK ((intent_fingerprint ~ '^[0-9a-f]{64}$'::text))
);

ALTER TABLE ONLY public.agent_instances
    ADD CONSTRAINT agent_instances_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.agent_drafts
    ADD CONSTRAINT agent_drafts_pkey PRIMARY KEY (instance_id);

ALTER TABLE ONLY public.agent_drafts
    ADD CONSTRAINT agent_drafts_instance_id_version_key UNIQUE (instance_id, version);

ALTER TABLE ONLY public.agent_validations
    ADD CONSTRAINT agent_validations_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.agent_validations
    ADD CONSTRAINT agent_validations_instance_id_id_key UNIQUE (instance_id, id);

ALTER TABLE ONLY public.agent_revisions
    ADD CONSTRAINT agent_revisions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.agent_revisions
    ADD CONSTRAINT agent_revisions_instance_id_id_key UNIQUE (instance_id, id);

ALTER TABLE ONLY public.agent_revisions
    ADD CONSTRAINT agent_revisions_instance_id_revision_number_key UNIQUE (instance_id, revision_number);

ALTER TABLE ONLY public.agent_publication_commands
    ADD CONSTRAINT agent_publication_commands_pkey PRIMARY KEY (scope, command_kind, request_key);

ALTER TABLE ONLY public.agent_publication_commands
    ADD CONSTRAINT agent_publication_commands_scope_command_kind_request_key_key UNIQUE (scope, command_kind, request_key);

CREATE INDEX "agent_instances_active_revision_id_INDEX" ON public.agent_instances USING btree (active_revision_id);

CREATE INDEX "agent_validations_instance_id_INDEX" ON public.agent_validations USING btree (instance_id);

CREATE INDEX "agent_revisions_instance_id_INDEX" ON public.agent_revisions USING btree (instance_id);

CREATE INDEX "agent_publication_commands_scope_INDEX" ON public.agent_publication_commands USING btree (scope);

CREATE INDEX "agent_publication_commands_resulting_revision_id_INDEX" ON public.agent_publication_commands USING btree (resulting_revision_id);

ALTER TABLE ONLY public.agent_drafts
    ADD CONSTRAINT agent_drafts_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.agent_instances(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE ONLY public.agent_instances
    ADD CONSTRAINT agent_instances_current_draft_fkey FOREIGN KEY (id, current_draft_version) REFERENCES public.agent_drafts(instance_id, version) ON UPDATE RESTRICT ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED;

ALTER TABLE ONLY public.agent_validations
    ADD CONSTRAINT agent_validations_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.agent_instances(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE ONLY public.agent_validations
    ADD CONSTRAINT agent_validations_draft_fkey FOREIGN KEY (instance_id, draft_version) REFERENCES public.agent_drafts(instance_id, version) ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE ONLY public.agent_revisions
    ADD CONSTRAINT agent_revisions_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES public.agent_instances(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE ONLY public.agent_revisions
    ADD CONSTRAINT agent_revisions_draft_fkey FOREIGN KEY (instance_id, source_draft_version) REFERENCES public.agent_drafts(instance_id, version) ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE ONLY public.agent_revisions
    ADD CONSTRAINT agent_revisions_validation_fkey FOREIGN KEY (instance_id, validation_id) REFERENCES public.agent_validations(instance_id, id) ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE ONLY public.agent_instances
    ADD CONSTRAINT agent_instances_active_revision_fkey FOREIGN KEY (id, active_revision_id) REFERENCES public.agent_revisions(instance_id, id) ON UPDATE RESTRICT ON DELETE RESTRICT DEFERRABLE INITIALLY DEFERRED;

ALTER TABLE ONLY public.agent_publication_commands
    ADD CONSTRAINT agent_publication_commands_scope_fkey FOREIGN KEY (scope) REFERENCES public.administrative_accounts(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE ONLY public.agent_publication_commands
    ADD CONSTRAINT agent_publication_commands_resulting_revision_id_fkey FOREIGN KEY (resulting_revision_id) REFERENCES public.agent_revisions(id) ON UPDATE RESTRICT ON DELETE RESTRICT;
COMMIT;

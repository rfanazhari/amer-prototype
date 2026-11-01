--
-- PostgreSQL database dump
--


-- Dumped from database version 17.11 (Debian 17.11-1.pgdg13+2)
-- Dumped by pg_dump version 17.11 (Debian 17.11-1.pgdg13+2)




--
-- Name: administrative_accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.administrative_accounts (
    id uuid NOT NULL,
    name text NOT NULL,
    email text NOT NULL,
    password_hash text NOT NULL,
    role text NOT NULL,
    status text NOT NULL,
    created_by uuid,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT administrative_accounts_password_hash_check CHECK ((char_length(password_hash) > 0)),
    CONSTRAINT administrative_accounts_role_check CHECK ((role = ANY (ARRAY['ADMIN'::text, 'HR'::text]))),
    CONSTRAINT administrative_accounts_status_check CHECK ((status = ANY (ARRAY['ACTIVE'::text, 'DISABLED'::text])))
);


--
-- Name: administrative_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.administrative_sessions (
    id uuid NOT NULL,
    account_id uuid NOT NULL,
    token_hash text NOT NULL,
    created_at timestamp with time zone NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    revoked_at timestamp with time zone,
    CONSTRAINT administrative_sessions_expiry_check CHECK ((expires_at > created_at)),
    CONSTRAINT administrative_sessions_revoked_at_check CHECK (((revoked_at IS NULL) OR (revoked_at >= created_at))),
    CONSTRAINT administrative_sessions_token_hash_check CHECK ((token_hash ~ '^[0-9a-f]{64}$'::text))
);


--
-- Name: tasks; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tasks (
    id uuid NOT NULL,
    title text NOT NULL,
    status text NOT NULL,
    due_date timestamp with time zone,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    completed_at timestamp with time zone,
    CONSTRAINT task_status_check CHECK ((status = ANY (ARRAY['todo'::text, 'in_progress'::text, 'done'::text])))
);


--
-- Name: administrative_accounts administrative_accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.administrative_accounts
    ADD CONSTRAINT administrative_accounts_pkey PRIMARY KEY (id);


--
-- Name: administrative_sessions administrative_sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.administrative_sessions
    ADD CONSTRAINT administrative_sessions_pkey PRIMARY KEY (id);


--
-- Name: tasks tasks_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tasks
    ADD CONSTRAINT tasks_pkey PRIMARY KEY (id);


--
-- Name: administrative_accounts_created_by_INDEX; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX "administrative_accounts_created_by_INDEX" ON public.administrative_accounts USING btree (created_by);


--
-- Name: administrative_accounts_email_INDEX; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX "administrative_accounts_email_INDEX" ON public.administrative_accounts USING btree (email);


--
-- Name: administrative_sessions_account_id_INDEX; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX "administrative_sessions_account_id_INDEX" ON public.administrative_sessions USING btree (account_id);


--
-- Name: administrative_sessions_expires_at_INDEX; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX "administrative_sessions_expires_at_INDEX" ON public.administrative_sessions USING btree (expires_at);


--
-- Name: administrative_sessions_token_hash_INDEX; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX "administrative_sessions_token_hash_INDEX" ON public.administrative_sessions USING btree (token_hash);


--
-- Name: administrative_accounts administrative_accounts_created_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.administrative_accounts
    ADD CONSTRAINT administrative_accounts_created_by_fkey FOREIGN KEY (created_by) REFERENCES public.administrative_accounts(id) ON UPDATE RESTRICT ON DELETE RESTRICT;


--
-- Name: administrative_sessions administrative_sessions_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.administrative_sessions
    ADD CONSTRAINT administrative_sessions_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.administrative_accounts(id) ON UPDATE RESTRICT ON DELETE RESTRICT;


--
-- PostgreSQL database dump complete
--



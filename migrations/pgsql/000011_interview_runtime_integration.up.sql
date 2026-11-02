BEGIN;
-- Retain every version-10 command as legacy: original SDP cannot be reconstructed.
ALTER TABLE public.interview_session_commands
 ADD COLUMN state text NOT NULL DEFAULT 'legacy',
 ADD COLUMN session_id uuid NULL,
 ADD COLUMN answer_sdp text NULL,
 ADD COLUMN execution_version bigint NULL,
 ADD COLUMN updated_at timestamptz NULL;
UPDATE public.interview_session_commands SET updated_at = created_at;
ALTER TABLE public.interview_session_commands ALTER COLUMN updated_at SET NOT NULL;
ALTER TABLE public.interview_session_commands ALTER COLUMN state DROP DEFAULT;
ALTER TABLE public.interview_voice_sessions
 ADD CONSTRAINT interview_voice_sessions_command_owner_key UNIQUE (execution_id, id, attempt_id);
ALTER TABLE public.interview_session_commands
 ADD CONSTRAINT interview_session_commands_result_check CHECK (
  (state = 'legacy' AND session_id IS NULL AND answer_sdp IS NULL AND execution_version IS NULL)
  OR (state IN ('pending','failed') AND session_id IS NOT NULL AND answer_sdp IS NULL AND execution_version IS NOT NULL AND execution_version > 0)
  OR (state = 'succeeded' AND session_id IS NOT NULL AND answer_sdp IS NOT NULL AND length(btrim(answer_sdp)) > 0 AND execution_version IS NOT NULL AND execution_version > 0)
 ),
 ADD CONSTRAINT interview_session_commands_updated_at_check CHECK (updated_at >= created_at),
 ADD CONSTRAINT interview_session_commands_session_owner_fkey FOREIGN KEY (execution_id, session_id, attempt_id)
 REFERENCES public.interview_voice_sessions (execution_id, id, attempt_id) ON UPDATE RESTRICT ON DELETE RESTRICT;
CREATE INDEX "interview_session_commands_session_id_INDEX" ON public.interview_session_commands (session_id) WHERE session_id IS NOT NULL;
-- A generation identifies exactly one authority across every operation of the same execution.
ALTER TABLE public.agent_execution_attempts
 ADD CONSTRAINT agent_execution_attempts_execution_generation_key UNIQUE (execution_id, authority_generation);
COMMIT;

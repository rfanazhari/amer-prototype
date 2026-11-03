BEGIN;
ALTER TABLE public.agent_execution_attempts DROP CONSTRAINT agent_execution_attempts_execution_generation_key;
DROP INDEX public."interview_session_commands_session_id_INDEX";
ALTER TABLE public.interview_session_commands
 DROP CONSTRAINT interview_session_commands_session_owner_fkey,
 DROP CONSTRAINT interview_session_commands_result_check,
 DROP CONSTRAINT interview_session_commands_updated_at_check,
 DROP COLUMN state,
 DROP COLUMN session_id,
 DROP COLUMN answer_sdp,
 DROP COLUMN execution_version,
 DROP COLUMN updated_at;
ALTER TABLE public.interview_voice_sessions DROP CONSTRAINT interview_voice_sessions_command_owner_key;
COMMIT;

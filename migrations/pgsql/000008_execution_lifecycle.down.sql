BEGIN;
-- Disposable databases only. Populated lifecycle rows use forward recovery or a backup taken before version 8.
-- The version 7 pin table and its rows are kept.

ALTER TABLE public.interview_executions DROP CONSTRAINT IF EXISTS interview_executions_id_fkey;

DROP TABLE IF EXISTS public.agent_execution_transitions;
DROP TABLE IF EXISTS public.agent_execution_attempts;
DROP TABLE IF EXISTS public.agent_execution_operations;
DROP TABLE IF EXISTS public.agent_execution_commands;
DROP TABLE IF EXISTS public.agent_executions;
COMMIT;

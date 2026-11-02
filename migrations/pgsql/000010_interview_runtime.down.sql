BEGIN;
-- Disposable databases only. Populated progress rows use forward recovery or a backup taken before version 10.
-- The assignment, session, execution, consent and controller tables and their rows are kept.

DROP TABLE IF EXISTS public.interview_session_commands;
DROP TABLE IF EXISTS public.interview_handoffs;
DROP TABLE IF EXISTS public.interview_checkpoints;
DROP TABLE IF EXISTS public.interview_coverage_items;
DROP TABLE IF EXISTS public.interview_technical_losses;
DROP TABLE IF EXISTS public.interview_transcript_entries;
DROP TABLE IF EXISTS public.interview_voice_sessions;
DROP TABLE IF EXISTS public.interview_progress;
COMMIT;

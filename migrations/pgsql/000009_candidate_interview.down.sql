BEGIN;
-- Disposable databases only. Populated consent rows use forward recovery or a backup taken before version 9.
-- The assignment, session and execution tables and their rows are kept.

DROP TABLE IF EXISTS public.interview_controllers;
DROP TABLE IF EXISTS public.interview_consents;
COMMIT;

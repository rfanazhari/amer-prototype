BEGIN;
-- Disposable databases only. Populated report rows use forward recovery or a backup taken before version 12.
-- Only the seven reporting tables are dropped. Assignments, executions, accounts and runtime rows are kept.

DROP TABLE IF EXISTS public.outbox_messages;
DROP TABLE IF EXISTS public.interview_report_attempts;
DROP TABLE IF EXISTS public.interview_report_jobs;
DROP TABLE IF EXISTS public.interview_report_commands;
DROP TABLE IF EXISTS public.interview_report_statements;
DROP TABLE IF EXISTS public.interview_report_generations;
DROP TABLE IF EXISTS public.interview_reports;
COMMIT;

BEGIN;
-- Disposable reversal only. Retained windowed retry rows intentionally refuse
-- this stricter constraint; no job data or original budget is rewritten.
ALTER TABLE public.interview_report_jobs
    DROP CONSTRAINT interview_report_jobs_unwindowed_states_check;
ALTER TABLE public.interview_report_jobs
    ADD CONSTRAINT interview_report_jobs_unwindowed_states_check CHECK (
        state NOT IN ('pending', 'published', 'waiting') OR (window_started_at IS NULL AND deadline_at IS NULL)
    );
COMMIT;

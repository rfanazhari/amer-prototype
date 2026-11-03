BEGIN;
-- Retry jobs retain the window opened by their first evidence-backed claim.
-- Jobs with no attempt and jobs waiting for evidence still have no window.
ALTER TABLE public.interview_report_jobs
    DROP CONSTRAINT interview_report_jobs_unwindowed_states_check;
ALTER TABLE public.interview_report_jobs
    ADD CONSTRAINT interview_report_jobs_unwindowed_states_check CHECK (
        (state <> 'waiting' OR (window_started_at IS NULL AND deadline_at IS NULL))
        AND (attempt_count <> 0 OR (window_started_at IS NULL AND deadline_at IS NULL))
    );
COMMIT;

-- Disposable databases only. Populated or referenced publication rows use forward recovery.

DROP TABLE IF EXISTS public.agent_publication_commands;
DROP TABLE IF EXISTS public.agent_revisions;
DROP TABLE IF EXISTS public.agent_validations;
DROP TABLE IF EXISTS public.agent_drafts;
DROP TABLE IF EXISTS public.agent_instances;

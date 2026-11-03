# PostgreSQL migration operations

Release target: **clean version 12 only**. Tickets: CORE-IAM-018–020, CORE-ACP-004, CORE-EXL-003, CORE-CIN-004, CORE-IRT-004, CORE-IRP-003, and Interview assignment/execution, shared execution lifecycle, candidate interview consent/controller, and interview runtime progress SQL; policy: Identity Access D8.
The migration-aware application reads version/dirty metadata before binding HTTP or bootstrapping ADMIN. It never installs tables or metadata. Missing, empty, unreadable, dirty, older or newer metadata fails startup and closes acquired connections. Startup remains bounded to ten seconds; process shutdown to five seconds.

## Credentials and artifacts

Supply secrets through environment configuration, without echoing URLs or putting them in arguments/history:

- `MIGRATION_DATABASE_URL`: authorized PostgreSQL owner/DDL connection, used only by `cmd/migrate`.
- `DATABASE_URL`: application login with CONNECT, schema USAGE, CRUD on `tasks`, `administrative_accounts`, `administrative_sessions`, `agent_instances`, `agent_drafts`, `agent_validations`, `agent_revisions`, `agent_publication_commands`, `candidates`, `interview_assignments`, `interview_invitations`, `candidate_sessions`, `interview_executions`, `agent_executions`, `agent_execution_commands`, `agent_execution_operations`, `agent_execution_attempts`, `agent_execution_transitions`, `interview_consents`, `interview_controllers`, `interview_progress`, `interview_coverage_items`, `interview_technical_losses`, `interview_voice_sessions`, `interview_transcript_entries`, `interview_checkpoints`, `interview_handoffs`, `interview_session_commands`, `interview_reports`, `interview_report_generations`, `interview_report_statements`, `interview_report_commands`, `interview_report_jobs`, `interview_report_attempts`, `outbox_messages`, and SELECT on `schema_migrations`. It must not own these objects or inherit CREATE/owner privileges. No sequence grants are needed for UUID keys. The operator provisions this role; startup does not.
- `TEST_DATABASE_URL`: explicitly authorized disposable integration database.
- `TEST_MIGRATION_DATABASE_URL`: separately authorized disposable database; acceptance tests **drop its public schema**. Never point it at retained data.

The image includes `/migrate` and `/migrations/pgsql`; its working directory is `/`. The app entrypoint stays `/amer-prototype`. Versions 1–11 are immutable and checked against the [release SHA-256 manifest](database-migration-checksums.sha256), also enforced by the runner's embedded `checksums.json`. Version/dirty metadata does not establish artifact integrity.

Owner-approved exception (2026-11-03, `CORE-IRT-018` follow-up): correct only the dependency order in `000010_interview_runtime.down.sql`. Coverage citations must be dropped before transcript entries; checkpoint/transcript dependents must be dropped before voice sessions. The approved down digest changes from `049a54c3f2191d627a9b9778ec542e441710cd2725457b6799ba38c4464a7fab` to `8af2b71e7103d14326e475b885dc86c4c2ec558e124c3c20846c45e73f81702f` in both manifests. Every up artifact, the release version (12), and the guarded disposable-only down policy remain unchanged. This is an explicit historical rollback-artifact correction, not a schema upgrade, automatic repair or production-data recovery. Full rollback/re-up and Compose verification are recorded in [the follow-up evidence](../services/core-ai/features/interview-runtime/t18-rollback-verification.md).


## Fresh local database

From the repository root after supplying both connections and the administrative settings documented in README:

```sh
make migrate.up MIGRATION_VERSION=12
make migrate.status
# Start the application only after version=12 dirty=false.
```

For Compose, configure the local environment, then explicitly prepare the database:

```sh
docker compose up -d --wait database
make docker.migrate MIGRATION_VERSION=12
make docker.up
```

`make docker.up` performs a read-only status preflight and refuses an unprepared database with preparation instructions; it does not migrate. An unprepared database makes app startup fail with explicit migration instructions. The migration profile is a separately invoked job, never an app dependency hook. Normal Compose uses the local owner role for convenience; provision the DDL-denied `DATABASE_URL` separately for a target environment. Do not delete a normal development volume to work around drift.

## Populated cutover

1. Schedule maintenance; stop every pre-cutover binary, migrator and schema writer. The advisory lock coordinates migration-aware commands, not legacy AutoMigrate or arbitrary external DDL.
2. Take a restorable backup/snapshot using protected storage and environment-backed PostgreSQL credentials. Rehearse restore into a separate database; verify complete retained row equality without printing contacts, digests or hashes.
3. Inspect the target catalog against [the captured baseline](database-schema-baseline.md). Confirm tables, columns/defaults/nullability, checks, all named indexes and RESTRICT FKs; inspect existing metadata separately. Check release SQL checksums. The captured baseline is synthetic evidence, not an inspected deployment.
4. Choose exactly one preparation path:

   | Observed state | Command | Expected |
   | --- | --- | --- |
   | Empty owned schema | `make migrate.up MIGRATION_VERSION=12` | Clean 12 |
   | Exact Task-only legacy, absent metadata | `make migrate.baseline`, then `make migrate.up MIGRATION_VERSION=12` | Baseline 1, clean 12, Task history retained |
   | Exact three-table legacy, absent metadata | `make migrate.baseline`, then `make migrate.up MIGRATION_VERSION=12` | Baseline 2, clean 12, all existing rows retained |
   | Recognized clean 1–11 | `make migrate.up MIGRATION_VERSION=12` | Upgrade or no-op |
   | Drift, partial/foreign schema, dirty/unsupported metadata | Stop and investigate | No guessed stamping or automatic force |

5. Run `make migrate.status`; require `version=12 dirty=false`. Verify row equality, creator links, stored hash equality and original expiry/revocation. Baseline never updates business rows or seeds accounts. Empty configuration, candidate, assignment, execution, interview consent/controller and runtime progress tables are additive. Version 8 backfills one `CREATED` shared execution per existing `interview_executions` pin (same id, instance, revision and `created_at`; no commands, operations, attempts or transitions) before it adds `interview_executions_id_fkey`.
6. Start the migration-aware binary with the restricted runtime role. Verify health, Task create/start/complete/read, ADMIN login → HR provisioning/login → session/logout, and restart. Bootstrap must preserve existing ADMIN credentials.
7. Retain the backup, checksum manifest, catalog inspection and sanitized evidence. Release acceptance for candidates, ACP and Interview remains with those owners.

Migration operations allow 59 seconds plus one second of cleanup; lock waits are bounded to five seconds. Cancellation/failure exits nonzero. Confirm no legacy writer can resume before a successful clean-version check.

## Failure, restore and compatible binary rollback

Stop rollout on error. Inspect metadata, SQL transaction outcome and the actual catalog before any repair. There is no generic force action. Baseline SQL uses explicit transactions; a failed down records its **target** as dirty, so dirty metadata alone does not identify the actual schema. Do not merely clear a dirty flag.

Choose reviewed forward repair with explicit preconditions and verified catalog/data postconditions, or restore the verified backup into a separate target and redirect after validation. The disposable acceptance suite demonstrates a cancelled transactional down: proves the original catalog remains, explicitly restores the established version/clean metadata, then checks compatibility and no-op up. That repair is fixture-specific, not a universal production command.

For application rollback, retain the migrated database and start a previously verified migration-aware binary that explicitly supports clean version 12, or unmount configuration routes and start a binary that still accepts the previous clean version while retaining the new tables. The full-stack rehearsal records the exact compatible image, stops it, and later recreates it by immutable image ID against retained version 11 before Task/admin HTTP smoke. A pre-cutover AutoMigrate binary is excluded. Do not run down against retained publication data.

Destructive reversal is only for disposable databases, after stopping consumers:

```sh
make migrate.down MIGRATION_STEPS=1 MIGRATION_DISPOSABLE=yes
make migrate.up MIGRATION_VERSION=12
```

Down deletes affected tables/data; it is never a retained-data recovery mechanism.

## Repeatable evidence

`make test.docker` creates its own Compose project, PostgreSQL databases and random credentials, prepares SQL explicitly, runs real adoption/failure/restricted-role/runtime/admin/Task tests, then dumps a populated database and restores a clone. The clone drops metadata to represent legacy state, runs guarded baseline/up/status, compares all retained rows, and repeats Task/admin smoke. Cleanup removes only that project's containers/volumes. `make docker.verify` additionally verifies the packaged app and optional services.

See [T18–T20 evidence](../services/core-ai/features/identity-access/t18-t20-evidence.md). These operations authorize no non-disposable target cutover. A real release still requires its inspected catalog, protected verified backup, provisioned roles and approved maintenance window.

## Version 8: execution lifecycle (CORE-EXL-003)

`000008_execution_lifecycle` creates `agent_executions`, `agent_execution_commands`, `agent_execution_operations`, `agent_execution_attempts` and `agent_execution_transitions`, backfills the pins, then adds `interview_executions_id_fkey`. It inserts no seed rows. Stop writers and back up a populated database before applying version 8. A version-7 binary refuses a version-8 database, so apply version 8 before starting the new binary. Returning a populated database to version 7 is a restore, not `migrate.down`; the disposable down drops the foreign key and the five new tables and keeps `interview_executions`. Grant the runtime role CRUD on the five new tables and keep SELECT on `schema_migrations`; it must not receive CREATE.

## Version 9: candidate interview (CORE-CIN-004)

`000009_candidate_interview` creates `interview_consents` (one acceptance per assignment and notice version, with the accepting `candidate_sessions` row as provenance) and `interview_controllers` (one controlling tab per assignment, referencing `interview_executions`). It inserts no seed rows, stores no notice text, and adds no column to `interview_executions`. Stop writers and back up a populated database before applying version 9 with the command in the preparation table. A version-8 binary refuses a version-9 database, so apply version 9 before starting the new binary, and a version-9 binary refuses a version-8 database. Returning a populated database to version 8 is a restore, not `migrate.down`; the disposable down drops `interview_controllers` then `interview_consents` and keeps the assignment, session and execution tables. Grant the runtime role CRUD on the two new tables and keep SELECT on `schema_migrations`; it must not receive CREATE.

## Version 10: interview runtime (CORE-IRT-004)

`000010_interview_runtime` creates `interview_progress`, `interview_coverage_items`, `interview_technical_losses`, `interview_voice_sessions`, `interview_transcript_entries`, `interview_checkpoints`, `interview_handoffs`, and `interview_session_commands`. Progress references `interview_executions`; child tables reference progress. `attempt_id` columns are UUID facts without a foreign key to shared execution attempts. Progress also stores follow-up citations and reconnect counters, and coverage stores clarification and omission entry ids. It inserts no seed rows and adds no column to `interview_executions`. Stop writers and back up a populated database before applying version 10. A version-9 binary refuses a version-10 database, so apply version 10 before starting the new binary, and a version-10 binary refuses a version-9 database. Returning a populated database to version 9 is a restore, not `migrate.down`; the disposable down drops the eight new tables in child-to-parent order and keeps assignments, sessions, executions, consents and controllers. Grant the runtime role CRUD on the eight new tables and keep SELECT on `schema_migrations`; it must not receive CREATE.

## Version 11: Interview Runtime integration (CORE-IRT-018)

`000011_interview_runtime_integration` adds reservation state, original SDP, session ownership, original execution version and an update timestamp to session commands. Version-10 rows remain `legacy` with null replay fields and return `IRT_CONFLICT`; no original SDP is backfilled and no rows are deleted. Composite ownership prevents another execution/attempt's session being replayed. Execution-wide generation uniqueness is enforced; inspect any duplicate retained generations before upgrade rather than deleting authority records automatically. Versions 1–10 are unchanged and their checksums stay pinned.

Stop writers, take a restorable backup, apply `make migrate.up MIGRATION_VERSION=11`, check clean status, then start a release requiring version 11. Runtime does not migrate. New columns use existing table grants. Disposable down removes only the new constraints, index and columns, retaining pins, progress and original command columns; successful replay data in the removed columns cannot be recovered by down. Restore or reviewed forward repair is required for retained production data. Database proof uses separate disposable integration and migration databases.

## Version 12: interview reporting (CORE-IRP-003)

`000012_interview_reporting` adds seven tables: `interview_reports`, `interview_report_generations`, `interview_report_statements`, `interview_report_commands`, `interview_report_jobs`, `interview_report_attempts` and `outbox_messages`. Apply it only after clean version 11: `make migrate.up MIGRATION_VERSION=12`, then `make migrate.status` must show `version=12 dirty=false`. No existing table is altered, no row is backfilled and nothing is seeded. Foreign keys are RESTRICT. A partial unique index allows at most one pending or running generation per report. Statement checks refuse an answer without evidence and a gap with evidence. `outbox_messages.job_kind` admits only `interview_report` in this version.

The application role needs CRUD on the seven new tables in addition to the list above. Startup still performs no DDL. Disposable down drops only these seven tables in dependency order; it never touches assignments, executions, accounts or runtime rows. Populated environments do not run down: keep version 12 and unwire the report workers, or restore a backup taken before version 12. A binary that requires clean version 11 refuses to start on version 12, which is the existing startup guard.

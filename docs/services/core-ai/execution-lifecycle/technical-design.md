# Technical Design: Execution Lifecycle

Date: 2026-11-02. Mode A — new feature. The capability is planned. Nothing in this document claims that BR-CORE-050..BR-CORE-059 are already enforced.

Business authority is [Execution Lifecycle](../business/capabilities/execution-lifecycle.md), BR-CORE-050..BR-CORE-059. Platform sources, in the order the user set for lifecycle states:

- [Technical design v0.1, sections 3 and 5](../../../design/core-ai/technical-design.md) wins on states, binding, lock order, and the first terminal commit.
- [Scope freeze, sections 4 and 5](../../../design/core-ai/shared_ai_agent_platform_consolidated_architecture_mvp_scope_freeze_v0.1.md) freezes the five-state lifecycle and INV-01..INV-09.
- [Architecture design, sections 6 and 9](../../../design/core-ai/shared_ai_agent_platform_architecture_design_v0.1.md) supplies kernel responsibilities and initiation semantics. Its suggested `WAITING_INPUT` state is a proposal. This design does not use it.

Root `AGENTS.md` and Identity and Access D8 require reviewed SQL in `migrations/pgsql/`, GORM for CRUD, and no startup DDL. Those instructions override the platform technical design's section 5 sentence that still says to retain AutoMigrate. No root `CLAUDE.md` and no scoped instruction file for this capability were present.

## 1. Overview

Give every admitted agent run one stable identity, one current state, and one committed ending. A repeated start finds the same run. A finished run stays finished. Later configuration publication does not rewrite the run that already began. The same rules must be usable later by a Document Analyzer that has no interview stages and no invoice fields.

### Goals

- Commit execution identity and its configuration binding before any provider work starts.
- Move that execution only through `CREATED`, `RUNNING`, `COMPLETED`, `FAILED`, and `CANCELLED`.
- Make the same start resolve the original execution, and make a reused start identity with a different intent fail without replacing it.
- Fence attempts so a late result cannot overwrite newer committed progress.
- Let the first committed terminal outcome win a race, and leave that outcome unchanged.
- Bind an interview execution to the assignment's already pinned published revision.
- Cancel a nonterminal execution when assignment access is revoked or the candidate's authority to continue expires during the run.
- Keep HR stage identifiers and invoice fields out of the shared lifecycle.

### Non-Goals

Consent screens, notice text, candidate HTML, stage workflow, question coverage, handoff content, transcripts, voice or WebRTC, provider protocols, report wording, hire/reject decisions, RabbitMQ, artifact files, and a runnable Document Analyzer product. This tripack defines the kernel operations those siblings call. It does not mount them on HTTP. Monitoring UI (scope-freeze P-11) and the runnable second agent (P-12's product half) stay outside this slice. The kernel source check for P-12 is in scope; a working invoice extraction is not.

Activation retry counts, voice timeouts, and stage time budgets are Interview Runtime policy. This kernel stores a deadline when the caller supplies one. It does not invent a retry budget.

### Verified project context and impact

| Source inspected | Observed fact and design consequence |
|---|---|
| Root `AGENTS.md` | Four root layers. Application-owned ports. Operation interface plus plain injected struct. Detached DTOs. Trusted `application/service.Clock` at UTC millisecond precision. Transactional mutation. Startup performs no DDL. Manual SQL is the schema policy. Every authored Go declaration is documented. `test/architecture/documentation_test.go` enforces that rule. |
| `go.mod` | Module `github.com/rfanazhari/amer-prototype`. Go 1.26.10. GORM 1.25.12. `github.com/golang-migrate/migrate/v4` v4.19.0. |
| `infrastructure/database/schema/version.go` | `schema.ReleaseVersion` is **7**. `ownedPublicTables` and `RuntimeGrantTables` already list `interview_executions`. |
| `migrations/pgsql/` | Release files are `000001` through `000007` only. No `000008` file exists. The next version is **000008**. |
| `migrations/pgsql/000007_interview_executions.up.sql` | Pin columns, and only these columns: `id uuid` primary key, `assignment_id uuid NOT NULL` with `UNIQUE (assignment_id)`, `instance_id uuid NOT NULL`, `published_revision_id uuid NOT NULL`, `created_at timestamptz NOT NULL DEFAULT now()`. Foreign keys, all `ON UPDATE RESTRICT ON DELETE RESTRICT`, point at `interview_assignments(id)`, `agent_instances(id)`, and `agent_revisions(id)`. Indexes exist on `instance_id` and `published_revision_id`. There is no state, version, attempt, progress, actor, or idempotency column. |
| `migrations/pgsql/000007_interview_executions.down.sql` | Disposable `DROP TABLE`. Populated pins need forward recovery. |
| `migrations/pgsql/000005_interview_assignments.up.sql` | Assignment facts this lifecycle may read, and no others: `id`, `candidate_id`, `instance_id`, `published_revision_id`, nullable `access_revoked_at`, nullable `completed_at`, `created_at`, `updated_at`, plus the checks and foreign keys in that file. |
| `application/usecase/agentbindingusecase/start_execution.go` | `StartExecution.Execute` loads the assignment, checks that the pinned revision still exists, and inserts `ExecutionBinding{ExecutionID, AssignmentID, InstanceID, RevisionID}`. The caller supplies the execution id. There is no state, no idempotent return, no intent comparison, and no lock of the assignment before the insert. |
| `application/repository/agentbindingport/store.go`, `infrastructure/repository/agentbindingrepository/store.go` | `ExecutionBinding` and `ExecutionModel` carry only those four identifiers. `created_at` is left to the database default. Insert is a GORM `Create`. Find by execution id returns `ErrNotFound` or the pin. The model is separate from domain and API types. |
| `infrastructure/runtime` search | `StartExecution` is not registered as an HTTP route. |
| `test/integration/agent_configuration_binding_test.go` | `TestAgentConfigurationBindingConsumers` proves a pin copied from the assignment still points at the original revision after a later publication. The test creates its own short tables and does not use migration `000007`. It does not prove lifecycle state. |
| `application/usecase/agentbindingusecase/requirements.go` | `LoadRequirements` reads the pin and then document required fields. That is configuration consumption. The kernel must not call it. |
| `infrastructure/repository/taskrepository/task.go` | Suitable pattern for a row lock inside a transaction callback and explicit `Select` updates. Task status rules are not execution rules. |
| `infrastructure/repository/accessrepository/candidate_store.go` | Already locks an assignment with `SELECT id FROM interview_assignments WHERE id = ? FOR UPDATE` before reading `access_revoked_at` and `completed_at`. Reuse that lock shape. Access does not cancel an execution. The only production-adjacent writer of `access_revoked_at` found is the integration fixture helper `revokeAssignment`. |
| `infrastructure/repository/agentconfigrepository/store.go` | Publication moves `active_revision_id` under an instance lock. It does not update `interview_executions` or assignment pins. |
| `application/service/ports.go` | `Clock.Now` is the trusted time port. |
| `test/architecture/dependencies_test.go` | Domain may import only `errors`, `time`, `strings`, `unicode`, `unicode/utf8`, `fmt`, `sort`, and other `domain/` packages. Application must not import delivery, infrastructure, or GORM. |
| `pkg/errs/codes.go`, `translations/errors.en.toml`, `translations/errors.id.toml` | Public codes and English/Indonesian messages. No execution-lifecycle codes exist yet. |
| `Makefile`, `infrastructure/database/schema/compatible.go`, `scripts/verify_compose.py`, `README.md`, `docs/design/database-migrations.md` | Startup and local commands currently require clean version 7, including the hardcoded phrase in `compatible.go` and `version=7 dirty=false` in `Makefile`. Shipping `000008` must move every one of those floors together. |
| Backend Service Standard and Design Assessment | Read. Project instructions override the standard where they differ: repository ports live in `application/`, delivery is not unit-tested, and this slice adds no delivery package. |

Impact of the pin: it already stores one row per assignment and copies `published_revision_id`. That is necessary and not sufficient. It does not represent acceptance versus activation, a terminal outcome, attempt authority, a progress reference, or a conflicting retry. Designing the lifecycle means adding shared execution storage and operations in front of that pin, not describing the pin as the lifecycle.

Binding contracts are the BR rules, the five states, idempotent start, conflicting intent, attempt fencing, first terminal commit, frozen binding, assignment-before-execution lock order, and the absence of HR stage and invoice rules. Package and helper layout below is an internal choice and may be refined when behavior stays the same. Changing a state, an error's meaning, or a transaction boundary needs a new decision and a synchronized tripack.

## 2. API / Interface Contract Changes

No public HTTP route is added. `docs/api/openapi.yaml` stays unchanged. Candidate Interview will map these operations onto its own routes later. This slice does not publish those routes early.

Siblings call application operations. Each operation is a narrow interface with one plain struct, injected ports, and a detached `application/dto` result. Domain objects and GORM models do not cross the operation boundary.

| Operation | Caller | Result |
|---|---|---|
| `Start` | Candidate Interview, after its own consent and session checks | The assignment's one execution. First success commits `CREATED`. The same start returns that execution. A conflicting intent is rejected. |
| `Find` | Candidate Interview, Interview Runtime, Interview Reporting | Detached execution by id, including a terminal one. Does not create or change state. |
| `FindByAssignment` | Candidate Interview | The execution linked by the pin, or not found. Does not create. A page refresh uses this instead of `Start`. |
| `Cancel` | Candidate Interview, for a confirmed early stop | `CANCELLED` when the execution is not terminal. The same cancel returns the same outcome. |
| `Activate` | Interview Runtime | `CREATED` becomes `RUNNING` on the same execution, or a failed activation attempt leaves it `CREATED`. |
| `RecordAttempt` | Interview Runtime | Appends one attempt and moves current authority to it. Does not create an execution. |
| `CommitProgress` | Interview Runtime | Records the opaque progress reference while `RUNNING`. Does not by itself complete the execution. |
| `Complete` | Interview Runtime | `COMPLETED` only when the progress port says the required progress is already committed. |
| `Fail` | Interview Runtime | `FAILED` from `CREATED` or `RUNNING`. |
| `CancelNonterminal` | The owner that committed revocation or in-interview authority expiry | `CANCELLED` when the execution is not terminal. |
| `Admit` | A future non-interview agent, and tests of the shared kernel | `CREATED` bound to the revision the caller already resolved. Does not write `interview_executions` and does not read an assignment. |

Interview Reporting receives only `Find`. It does not receive `Start`, `Activate`, `Complete`, `Fail`, or `Cancel`.

`StartExecution` remains the unmounted pin writer described above. Runtime must not gain a reference to it. After version 8, a pin insert on the release schema also requires the shared execution row. The binding-preservation test that builds private tables may keep calling the pin writer. Release-schema proof belongs to this capability's PostgreSQL tests.

## 3. Data Model & Schema Changes

### Concepts and rules

**Execution** is the aggregate root. Its identity is a UUID minted by the kernel, stable across retries, reconnects, and new attempts. Product language "accepted" is the state `CREATED`: identity and binding are committed, and agent work has not started. `RUNNING` means activation succeeded and the execution is not terminal. `COMPLETED`, `FAILED`, and `CANCELLED` are terminal and never reopen. Disconnect is not a state and is not a cancellation.

**Binding** is the instance id plus the published revision id copied at admission. Equality is those two ids. Later publication is not a reason to change them.

**Attempt** is one try of one opaque operation key inside one execution. A newer attempt replaces the authority of an older one by generation. History rows stay. An operation key is an opaque token supplied by the caller. The kernel does not interpret it as a stage id or an invoice field.

**Progress reference** is an opaque id for progress that a sibling has already committed in its own store. The kernel does not store transcript text, coverage, or extracted invoice fields.

**Start identity** is the pair (scope, request key) plus a command kind. **Intent** is the fingerprint of scope, instance, published revision, and the caller's ordered input references. The fingerprint is computed from the binding being committed, never from a newer active revision.

Interview scope is the assignment id. One assignment has one execution because `interview_executions.assignment_id` is already unique and `Start` locks that assignment before writing. That uniqueness is not a column on the shared execution table. A future document run uses `Admit` with its own scope and may have more than one execution for one agent instance. Scope freeze section 4.2 says that business uniqueness is not a universal kernel rule. BR-CORE-050 still holds for interview assignments.

`assignment.completed_at` is an interview marker this kernel does not write. `Start` will not create an execution when that marker is already set. Completion of the execution is `agent_executions.state = COMPLETED`, not that assignment column.

| Material rule | Owner | Enforcement paths | Proof plan |
|---|---|---|---|
| BR-CORE-050 one execution per assignment | `Start` plus the existing unique `assignment_id` | Lock assignment, find the pin, insert pin and execution together | Two starts, including concurrent starts, leave one pin and one execution. Planned PostgreSQL test. Not claimed from the current pin insert, which returns a unique-violation error instead of the original row. |
| BR-CORE-051 accepted is not activated | Execution aggregate | `Start` and `Admit` commit `CREATED` only. `Activate` is a later call | After start, state is `CREATED`, `activated_at` is null, and no provider port is invoked. |
| BR-CORE-052 same start returns the original | Command row plus `Start`/`Admit` | Same scope, kind, key, and fingerprint loads the stored execution id | Lost-response retry returns the same id and does not insert a second row. |
| BR-CORE-053 different intent conflicts | Command fingerprint | Same scope, kind, and key with a different fingerprint | Reject before any replacement. The stored execution stays. |
| BR-CORE-054 legal states; terminal stays terminal | Execution aggregate | `Activate`, `Complete`, `Fail`, `Cancel`, `CommitProgress`, restoration | Allowed edges only. Each terminal source rejects `Activate`, `Complete`, `Fail`, a different `Cancel`, and progress writes. |
| BR-CORE-055 revocation or expiry cancels a nonterminal run | `CancelNonterminal` | Called under the assignment lock when access has ended | Revoked assignment with `RUNNING` becomes `CANCELLED`. Authority-expired call does the same. Already terminal state is unchanged. End-to-end HR button is blocked; see §9. |
| BR-CORE-056 completion needs committed progress | Execution aggregate and `ProgressCommitment` port | `Complete` | Port false leaves `RUNNING`. Port true from `RUNNING` commits `COMPLETED`. No provider-success argument exists. |
| BR-CORE-057 old attempt cannot overwrite newer progress | Attempt authority on the execution | `CommitProgress`, `Complete`, `Fail` | Lower generation or a deadline that is not strictly in the future writes nothing. |
| BR-CORE-058 later publication does not rewrite the binding | Binding value, insert-only columns | `Start` copies `interview_assignments.published_revision_id`. Updates omit both binding columns | Publish a newer revision, then read the execution and the pin. Both still hold the original revision. |
| BR-CORE-059 no agent-specific checklist | Execution aggregate and `Admit` | Kernel packages and a no-assignment execution | Source scan rejects stage, invoice, transcript, and salary identifiers in kernel packages. An `Admit` test completes an execution that has no `interview_executions` row. |
| INV-01 stable identity | Execution id | Start retry, new attempt, activate | Same id after each of those. |
| INV-02 no reopen | Execution aggregate | Terminal paths above | Same as BR-CORE-054. |
| INV-03 one committed state per version | `state_version` under the execution row lock | Every mutation | Stale expected version writes nothing. Two terminals: one commit, one conflict. |
| INV-04 no duplicate committed outcome | Command unique key | Start, activate, complete, fail, cancel | Same command returns the first outcome. |
| INV-05 no invented progress | Progress port | `Complete`, restoration | Load does not call the port. Complete does not synthesize a reference. |
| INV-06 fencing | Authority generation | Late attempt | Same as BR-CORE-057. |
| INV-07 terminal points at committed progress | `COMPLETED` check and column | `Complete`, schema check | `COMPLETED` with a null progress reference cannot be stored. |
| INV-08 traceable changes | Transition append | Every successful state change and attempt authority change | Row stores actor, optional operation key, from/to state, version, and time. |
| INV-09 frozen binding | Binding columns | Update statements | Same as BR-CORE-058. |

All proof rows are planned. The existing pin test is observed evidence only for "the copied revision survives a later publication" on private tables.

### Creation, restoration, and transition

Creation accepts a non-empty execution id, a binding whose instance and revision ids are non-empty, a positive start version of 1, and no progress, activation, or terminal facts. It always yields `CREATED`. It does not inspect provider availability.

Restoration loads a row that already satisfies the stored checks: known state, version at least 1, binding present, and the state/time/progress combination in the table below. It does not require the assignment to still be open, does not re-read the active revision, and does not call the progress port. A row that violates those stored rules is a technical failure, not a successful execution and not a caller input error. Pin rows that exist before version 8 are backfilled as `CREATED` with version 1, the pin's instance and revision, the pin's `created_at`, null activation, null progress, and null terminal facts. They get no invented transition history.

Transition authority is only the aggregate methods used by the operations in §2. Direct field assignment is not an allowed path. Time values that enter the aggregate are already UTC and truncated to milliseconds by the application. The aggregate copies time pointers when it exposes a snapshot.

### Storage layout

Shared tables are new. The pin's current columns stay as they are in `000007`. Version 8 adds one foreign key so a pin cannot exist without the shared execution that uses the same id.

| Table | Columns and constraints | Role |
|---|---|---|
| `agent_executions` | `id uuid` primary key. `instance_id uuid NOT NULL` and `published_revision_id uuid NOT NULL`, both foreign keys with `RESTRICT` to `agent_instances` and `agent_revisions`. `state text NOT NULL` limited to `CREATED`, `RUNNING`, `COMPLETED`, `FAILED`, `CANCELLED`. `state_version bigint NOT NULL` check `>= 1`. `committed_progress_ref uuid NULL` with no foreign key. `activated_at`, `terminal_at` nullable `timestamptz`. `terminal_reason text NULL`. `created_at`, `updated_at timestamptz NOT NULL`, check `updated_at >= created_at`. | Authoritative lifecycle. No assignment column. No stage column. No invoice column. |
| `agent_execution_commands` | `id uuid` primary key. `scope text NOT NULL`. `command_kind text NOT NULL` in `start`, `activate`, `complete`, `fail`, `cancel`, `attempt`. `request_key text NOT NULL`. `intent_fingerprint char(64) NOT NULL`. `execution_id uuid NOT NULL` foreign key `RESTRICT`. `created_at timestamptz NOT NULL`. Unique `(scope, command_kind, request_key)`. | Idempotent result pointer. |
| `agent_execution_operations` | Primary key `(execution_id, operation_key)`. `state text NOT NULL` in `PENDING`, `IN_PROGRESS`, `SUCCEEDED`, `FAILED`, `CANCELLED`. `current_authority_generation bigint NOT NULL` check `>= 1`. `created_at`, `updated_at`. Foreign key to `agent_executions`. | Current authority for one opaque operation. |
| `agent_execution_attempts` | `id uuid` primary key. `execution_id`, `operation_key`, `attempt_number bigint >= 1`, `authority_generation bigint >= 1`, `outcome` using the operation states, nullable `reason`, nullable `deadline_at`, `created_at`. Unique `(execution_id, operation_key, attempt_number)` and unique `(execution_id, operation_key, authority_generation)`. Composite foreign key to the operation. | Immutable history. |
| `agent_execution_transitions` | `id uuid` primary key. `execution_id` foreign key. `from_state text NULL`, `to_state text NOT NULL`, `state_version bigint NOT NULL`, `actor_ref text NOT NULL`, nullable `operation_key`, nullable `reason`, `occurred_at timestamptz NOT NULL`. | Append-only trace. Not an event log used to rebuild state. |
| `interview_executions` | Unchanged columns from `000007`. New foreign key: `id` references `agent_executions(id)` `RESTRICT`. | Interview admission pin. Same id as the shared execution. |

State checks on `agent_executions`:

- `CREATED`: `activated_at`, `terminal_at`, `committed_progress_ref`, and `terminal_reason` are all null.
- `RUNNING`: `activated_at` is not null and is at or after `created_at`. `terminal_at` and `terminal_reason` are null. A progress reference may already be present.
- `COMPLETED`: `activated_at` and `terminal_at` and `committed_progress_ref` are not null. `terminal_at` is at or after `activated_at`. `terminal_reason` is null.
- `FAILED` or `CANCELLED`: `terminal_at` is not null and at or after `created_at`. If `activated_at` is present it is not after `terminal_at`. `terminal_reason` is required. A progress reference may remain, so a failure can keep partial progress.

`terminal_reason`, `operation_key`, `request_key`, `actor_ref`, and each input reference are tokens of 1 to 64 characters from `A-Z`, `a-z`, `0-9`, `.`, `_`, `:`, and `-`. Scope uses the same token rule. At most 16 input references are part of one intent. These are storage bounds so the fingerprint is stable, not a product catalog of reasons. Callers choose tokens. The kernel does not publish an HR reason list.

Indexes: `agent_executions(instance_id)`, `agent_executions(published_revision_id)`, and `agent_execution_attempts(execution_id)`. The command unique key and the attempt unique keys are the other lookup paths. No soft-delete column is added.

### Adapter and transactions

GORM 1.25.12 maps separate models. Inserts use `Create`. Mutations use `Updates` with an explicit column map or `Select` list that includes zero and null values the change actually owns. The binding columns are absent from every update list. `Updates(struct)` omission must not decide whether a null progress reference or a null terminal time is written. There is no `DeletedAt`, no association autosave, and no preload of attempt history. A mutation loads the execution and, when an operation key is involved, that one operation row.

The assignment lock is the raw-SQL exception: `SELECT id FROM interview_assignments WHERE id = ? FOR UPDATE`, the statement `accessrepository` already uses. It is justified because it locks one row without mapping the assignment into an execution model. Execution row locks use GORM `clause.Locking{Strength: "UPDATE"}`, as `taskrepository` does. No other raw SQL is required. Conditional terminal updates still go through GORM `Where` plus `RowsAffected`.

Lock order for interview start, early stop, and access-ended cancellation: assignment row, then execution row, then command, operation, attempt, and transition writes. Activation, attempt, progress, complete, and fail lock the execution only. Those paths must not query `interview_assignments` after taking the execution lock. `Admit` has no assignment lock.

One database transaction holds the locks, the domain mutation, and the writes. Provider and network calls stay outside it. The progress port is a local read of a commitment made in an earlier transaction. The completion transaction reads that fact again under the execution lock.

First terminal commit: update `agent_executions` where `id` and `state_version` match and `state` is `CREATED` or `RUNNING`. Zero rows means the locked reread decides the result. The same command returns the stored terminal execution. A different terminal request returns a conflict and does not change the row.

Ambiguous commit returns an uncertain error and no new success id. The same command key reconciles on retry. An aborted transaction is not reused.

Application time is `Clock.Now().UTC().Truncate(time.Millisecond)` before it enters the domain. An attempt deadline rejects a result when trusted now is not strictly before `deadline_at`. The exact instant of the deadline is already too late.

### Migration artifacts

Proposed pair, unallocated until T3 adds it:

| Artifact | Forward | Reverse / recovery |
|---|---|---|
| `migrations/pgsql/000008_execution_lifecycle.up.sql` | Create the five shared tables, checks, indexes, and foreign keys. Backfill one `agent_executions` row per existing `interview_executions` row with the same `id`, `instance_id`, `published_revision_id`, state `CREATED`, `state_version` 1, `created_at` copied, `updated_at` equal to that `created_at`, and null lifecycle facts. Then add `interview_executions_id_fkey`. Do not change pin column definitions. Do not insert commands, operations, attempts, or synthetic transitions. | `migrations/pgsql/000008_execution_lifecycle.down.sql` drops the new foreign key and the five new tables. It does not drop `interview_executions`. Disposable databases only. Populated databases keep version 8 and recover forward, or restore a backup taken before the up. |

No data migration rewrites pin identifiers. A database already at clean version 7 upgrades by this one version. A fresh database applies 1 through 8.

T3 sets `schema.ReleaseVersion` to 8 and adds the five table names to `ownedPublicTables` and `RuntimeGrantTables`. It updates the embedded `infrastructure/database/schema/checksums.json`, `docs/design/database-migration-checksums.sha256`, the version-7 phrases in `infrastructure/database/schema/compatible.go`, `Makefile` `docker.up`, `scripts/verify_compose.py`, `README.md`, and `docs/design/database-migrations.md`. The runtime role gains CRUD on the five new tables and keeps SELECT on `schema_migrations`. It still must not receive CREATE. Startup continues to call `schema.RequireCompatible` only. No `AutoMigrate`.

Versions `000001` through `000007` stay immutable.

Rollback of the application after a populated upgrade is a binary that still accepts clean version 8, with these operations simply not called. The current version-7 binary refuses a version-8 database. Returning a populated database to version 7 is a restore, not `migrate.down`.

### Seed / reference data

No reference table and no seed rows. States and command kinds are check constraints plus code constants. Migrations insert no accounts, executions, or demo interviews. Integration tests create synthetic rows through the operations or explicit fixtures. Rerun of version 8 is a no-op once it is applied. There is no open dataset decision.

### Transaction scenarios

- **Start, assignment open, no execution.** Lock assignment. Copy `published_revision_id` and `instance_id` from that row. Reject if `access_revoked_at` or `completed_at` is set. Insert execution `CREATED`, command, initial transition, and the pin, same id, one commit.
- **Start, same key and same fingerprint.** Return the stored execution. No second pin. No state change. This includes a terminal execution when access is still open, so a retry can see the committed ending.
- **Start, same key and different fingerprint.** Conflict. No write.
- **Start, different key, same assignment, same binding fingerprint.** Return the original execution and store the new key pointing at it.
- **Start, different key, same assignment, fingerprint would bind something else.** Conflict. Do not replace the pin.
- **Start, assignment revoked, no execution.** `EXL_ASSIGNMENT_UNAVAILABLE`. No insert.
- **Start, assignment revoked, nonterminal execution.** Cancel that execution in the same transaction, then return `EXL_ASSIGNMENT_UNAVAILABLE`. Do not present a successful start.
- **Start, assignment revoked, already terminal.** Leave the terminal row. Return `EXL_ASSIGNMENT_UNAVAILABLE`.
- **Refresh.** `Find` or `FindByAssignment` only. No insert.
- **Activate success.** From `CREATED` to `RUNNING`, set `activated_at`, increment version, append attempt and transition.
- **Activate failure.** Execution stays `CREATED`. The attempt is `FAILED`. A later `Activate` may still succeed. `Fail` is how the runtime gives up.
- **Disconnect.** No operation. State stays where it was.
- **CommitProgress.** Only `RUNNING`, only the current attempt generation, only before the deadline. Stores the reference. State stays `RUNNING`.
- **Complete.** Only `RUNNING`, current generation, progress port true, reference stored, then `COMPLETED`.
- **Fail.** From `CREATED` or `RUNNING` to `FAILED`. Partial progress may remain.
- **Cancel / CancelNonterminal.** From `CREATED` or `RUNNING` to `CANCELLED`.
- **Invitation replacement.** Does not call these operations. Reissue must not cancel a run. That is BR-CORE-037, owned by access, and this kernel must not gain a hook inside invitation issue.

## 4. Component Breakdown

| Layer/component | Status | Justification |
|---|---|---|
| `domain/aggregate/execution` | New | Owns identity, binding, state, version, progress reference, and attempt authority. |
| `application/usecase/executionusecase` | New | Start, admit, activate, attempt, progress, complete, fail, and both cancels. |
| `application/query/executionquery` | New | Find and find-by-assignment. Reporting depends only on this read side. |
| `application/repository/executionport` | New | Transaction and read contracts. No GORM types. |
| `application/dto` | New execution file | Detached snapshots. |
| `application/service.Clock` | Reused as-is | Trusted time. |
| `infrastructure/repository/executionrepository` | New | GORM models, locks, explicit updates. |
| `infrastructure/repository/agentbindingrepository` | Reused as-is for the old pin tests | Not the lifecycle writer on the release schema. |
| `migrations/pgsql/000008_execution_lifecycle.*.sql` | New | Schema delivery. |
| `infrastructure/database/schema` | Modified | Release floor 8, grants, checksums. No DDL at startup. |
| `infrastructure/runtime` | Modified only to avoid mounting the pin writer or a new route | No HTTP handler and no provider worker. |
| `delivery/` | Reused as-is, no new package | Siblings own HTTP later. |
| `pkg/errs`, `translations/` | Modified | Kernel error codes and English/Indonesian messages. |
| `cmd/app/` | Reused as-is | No new signal or shutdown rule. Existing ten-second startup and five-second shutdown budgets stay. |

### Detail per component

**Execution aggregate.** Unexported fields. `NewExecution` validates a new `CREATED` execution and returns an error. `RestoreExecution` validates stored consistency and does not apply creation-time admission policy. Methods `Activate`, `RejectActivation`, `BeginAttempt`, `CommitProgress`, `Complete`, `Fail`, and `Cancel` return the next snapshot plus the attempt or transition facts the repository must append. `Complete` requires a progress reference the caller has already proven; the method does not accept a provider-success flag. `Snapshot` copies time pointers. A separate `Binding` value owns equality of instance and revision. A `ProgressRef` value owns a non-empty token. An attempt's current generation is the value the execution compares; historical attempts are results of the method, not a second aggregate.

**Operations.** `Start` depends on an assignment lock port and the execution store. It does not depend on consent storage, session clocks, voice, or `LoadRequirements`. The caller is responsible for consent and for "the session covers the interview" before `Start`. `Start` itself refuses a locked assignment whose `access_revoked_at` or `completed_at` is set, because those facts are on the assignment row and a revoked start must not admit a run. `CancelNonterminal` does not compute the two-hour session rule. The sibling passes `authority_expired` or `access_revoked` after its own check, and the operation locks the assignment and then the execution. `Admit` takes a caller-resolved binding and writes no pin.

**Progress port.** `ProgressCommitment` answers whether a given execution id and progress reference are already committed. The interview implementation of that port does not exist in this slice. Tests supply a fake. Interview Runtime replaces the fake later without a kernel change.

**Repository.** `WithAssignment` locks the assignment, then runs the callback. `Update` locks the execution by id and runs the callback. Both roll back when the callback returns an error. Insert of an interview execution writes `agent_executions` before `interview_executions`. Column lists for updates exclude `instance_id` and `published_revision_id`.

**Errors.** Domain sentinels stay inside the domain. The application maps them to `pkg/errs` codes. Translations gain the same codes. No HTTP mapper is added here.

Kernel packages for the source scan are `domain/aggregate/execution`, `application/usecase/executionusecase`, `application/query/executionquery`, `application/repository/executionport`, and `infrastructure/repository/executionrepository`. They must not name interview stages, invoice fields, transcripts, or salary.

### Material structure decisions and assessment

Rule and source → problem → alternatives → comparison → decision.

| Choice | Reason and material alternatives/tradeoffs | Authority and refinement boundary | Assessment outcome |
|---|---|---|---|
| States `CREATED`, `RUNNING`, `COMPLETED`, `FAILED`, `CANCELLED`. `CREATED` means accepted and not activated. | Technical design §3 and scope freeze §4.3 require this set. Architecture design §6's `CREATED → RUNNING ↔ WAITING_INPUT` would add a top-level wait state the later design rejected. Operation rows already represent pending work, so a sixth execution state would duplicate that fact and force every reader to learn it. | Binding. User instruction forbids `WAITING_INPUT`, `ACTIVATING`, and `RECOVERING` as top-level states. | Revised away from the architecture proposal. Retained from the later technical design. |
| Shared `agent_executions` plus the existing pin, same id, new foreign key. | Putting state columns only on `interview_executions` would keep `assignment_id NOT NULL` in the lifecycle record. BR-CORE-059 and scope freeze §4.2 require a document run to exist without an interview assignment. A second, fully duplicated state machine for documents would copy terminal and fencing rules. The pin remains the assignment uniqueness constraint already shipped in `000007`. | Binding behavior from BR-CORE-050 and BR-CORE-059. Table names are internal. The pin column list is an explicit compatibility constraint: do not invent or rename those columns. | Revised relative to "alter the pin in place". |
| Execution aggregate owns transitions. Attempt history is append-only under the same execution lock. | A domain service would have no natural state of its own. Stuffing transcript-sized history into the aggregate would load unrelated rows on every cancel. One operation row holds current authority, which is the invariant BR-CORE-057 names. | Internal. The lock and the "history is not updated" rule are binding. | Retained. |
| Interview `Start` is an application policy around a shared `Admit` write. | Domain code that branches on assignment versus invoice would violate architecture design §9 ("the core must not contain `if candidate link` or `if invoice upload`") and BR-CORE-059. Leaving one-per-assignment only in the database would make `Admit` accidentally interview-shaped if it required the pin. | Binding split: domain has no assignment field. `Start` may read the verified assignment columns. | Retained. |
| No HTTP in this slice. | The capability's actors are sibling capabilities, not a new public resource. Publishing routes now would freeze consent and runtime payloads this design does not own. | Explicit from the user scope. A later sibling may add routes without changing these operations. | Retained. |
| GORM for CRUD. Raw SQL only for the assignment id lock. | Task and configuration already use GORM locks and explicit updates. The assignment lock statement is already raw SQL in access. A second ORM lock style for that one statement adds a mapping the execution model must not own. Full raw SQL for ordinary inserts would discard the project's CRUD default without a query that needs it. | Project GORM default. The lock statement is the justified exception. | Retained. |
| Manual `000008`. No AutoMigrate. | `ReleaseVersion` is 7 and the highest file is `000007`. Platform technical design §5 still mentions AutoMigrate; root `AGENTS.md`, Identity and Access D8, and the user instruction replace that sentence for this service. | Explicit required structure. | Retained. |
| Kernel does not write `interview_assignments.completed_at`. | That column is an interview marker. Writing it inside the shared complete path would couple every future agent to an interview table. `Start` still honors it when it is already set, using the column that exists today. | Binding for the kernel. A later interview coordinator may set the marker in its own transaction. | Retained. Gap is recorded in §9. It does not block kernel tasks. |
| Activation retry budget stays with Interview Runtime. | Scope freeze §4.4 says a failed activation is an attempt outcome and bounded retries then activate or fail. It does not choose the bound. Inventing a number here would be a product rule. The kernel already allows another `Activate` from `CREATED` and an explicit `Fail`. | Deferred to Interview Runtime. No kernel task waits on the number. | Deferred, non-blocking. |
| End-to-end revocation producer. | BR-CORE-055 needs a caller when access ends. No production writer of `access_revoked_at` exists beyond the test helper. The operation and its fixture proof do not need that writer. The HR-facing path does. | Blocks only T10. | Deferred. |

Domain assessment applies. The core language is execution, accepted (`CREATED`), running, terminal outcome, activation, and attempt. "Assignment" stays an interview admission word and is not a field on the shared execution. The consistency boundary is one execution row plus the operation authority being changed. Cross-boundary work is the assignment lock before that row, and the progress port read before completion.

Ordinary `Start` path: validate the key, lock the assignment, copy the pin, insert `CREATED`, commit, return the snapshot. Invalid key: error before the lock, no insert. The alternative of folding activation into `Start` would mark a run `RUNNING` when no provider exists, which breaks BR-CORE-051. Callers would then need a compensating transition the state model does not allow.

Ordinary `Complete` path: lock the execution, reject a stale generation, read the progress port, commit `COMPLETED`. Port false: leave `RUNNING` and write nothing. The alternative of completing on a boolean from the voice provider would let provider success finish the run, which breaks BR-CORE-056.

## 5. Flow Sequence

### Interview start

1. Candidate Interview checks consent and session coverage. This kernel does not.
2. `Start` checks the request key, actor, and input-reference shape. A bad shape returns `INVALID_INPUT` and takes no lock.
3. Open a transaction. Lock the assignment by id.
4. If the assignment row is missing, return `EXL_NOT_FOUND`.
5. If `access_revoked_at` is set, cancel a nonterminal execution when one exists, commit that cancellation, and return `EXL_ASSIGNMENT_UNAVAILABLE`. If it is already terminal, leave it and return the same error.
6. If `completed_at` is set and no execution exists, return `EXL_ASSIGNMENT_UNAVAILABLE` without inserting.
7. Load an existing command for scope, kind `start`, and key. The same fingerprint returns that execution. A different fingerprint returns `EXL_IDEMPOTENCY_CONFLICT`.
8. If a pin already exists, compare its stored binding with the fingerprint built from the assignment row. Match: return it and record a new key when the key is new. Differ: conflict.
9. Otherwise mint the execution id, insert `agent_executions` as `CREATED`, the command, the transition, and the pin. Commit. A unique violation on the pin or the command rolls back and reconciles by reading the winner. Ambiguous commit returns `EXL_UNCERTAIN_COMMIT`.

### Activation and completion

1. Interview Runtime calls `Activate` with its own request key and an opaque operation key.
2. Lock the execution. Replay of the same activate command returns the stored result. From `CREATED`, success sets `RUNNING`. A failed attempt leaves `CREATED`.
3. Runtime later calls `CommitProgress` only after its own progress transaction has committed.
4. Runtime calls `Complete`. The kernel locks the execution, checks version and generation and deadline, re-reads the progress port, and commits `COMPLETED` only when that read succeeds.

### Cancellation race

1. One transaction locks the assignment, then the execution, and writes `CANCELLED` if the row is still nonterminal.
2. A completion transaction locks only the execution. It waits if cancellation holds that lock.
3. The first commit that changes a nonterminal row to a terminal state wins. The second command sees the terminal row and either returns it, when the command is the same, or returns `EXL_TERMINAL_CONFLICT`.

## 6. Request / Response

There is no HTTP body. The operation inputs and detached outputs are the contract.

`Start` input: assignment id, request key, actor ref, zero or more input reference tokens. It does not accept a revision id, an execution id, a provider result, or a consent flag. Output: execution id, assignment id, instance id, revision id, state `CREATED` or a later state when the original execution has already moved, `state_version`, nullable activated time, nullable terminal time, nullable progress reference. Times are RFC3339 UTC with millisecond precision when a sibling later renders them. Null times stay null.

`Admit` input: scope, request key, actor ref, instance id, revision id, input references. Output: the same execution view without an assignment id.

`Activate`, `RecordAttempt`, `CommitProgress`, `Complete`, `Fail`, and `Cancel` input: execution id, request key, actor ref, expected `state_version`, operation key, and attempt generation when the action is fenced. `Complete` has no provider-success field. `CommitProgress` and `Complete` carry the progress reference token. `Fail` and `Cancel` carry a reason token.

`Find` output is the execution view. Absence is `EXL_NOT_FOUND`.

### Path contracts

| Path | Accepted inputs and normalization | Errors and failure timing | Side effects and compatibility | Proof plan |
|---|---|---|---|---|
| Create via `Start` or `Admit` | Trim is not applied. Tokens must already match the alphabet. Fingerprint uses the binding copied from the assignment (`Start`) or the caller-supplied revision (`Admit`), sorted unique input refs, length-prefixed so values cannot collide. | Shape errors before any lock. Not found, unavailable, and conflict after the assignment lock and before insert. Unique races roll back the loser. Uncertain commit returns no success id. | Inserts execution, command, and transition. `Start` also inserts the pin. No provider call, no report job, no `completed_at` write, no active-revision write. | Valid start, bad key, revoked assignment, duplicate key, conflicting key, concurrent starts. |
| Restore | Rows that match the checks, including backfilled `CREATED` pins. Unknown state or `COMPLETED` without progress is technical. | Failure at load. No write. | Does not reopen a terminal row and does not hide a corrupt row as `CREATED`. | Round-trip each state. Backfill fixture. Corrupt fixture. |
| Transition | Source states in §3. Same command replays. Expected version must match. Generation must be current. Deadline must be strictly in the future when set. | `EXL_INVALID_TRANSITION`, `EXL_STALE_VERSION`, `EXL_STALE_ATTEMPT`, `EXL_PROGRESS_REQUIRED`, `EXL_TERMINAL_CONFLICT`. Failed checks write nothing. | One version increment, one transition append, attempt insert only when a new attempt is requested. Binding columns unchanged. | Table of legal and illegal edges, stale generation, exact deadline, first terminal commit. |

### Error catalog

Add these codes to `pkg/errs` and both translation files. Order for `Start` is top to bottom; the first match wins.

| Code | When |
|---|---|
| `INVALID_INPUT` | Key, actor, scope, or input-reference shape is illegal. No lock yet. |
| `EXL_NOT_FOUND` | Assignment or execution is absent. |
| `EXL_ASSIGNMENT_UNAVAILABLE` | Revoked or already completed assignment, after any required cancellation commit. |
| `EXL_IDEMPOTENCY_CONFLICT` | Same start identity, different intent. |
| `EXL_INVALID_TRANSITION` | Source state does not allow the action, including a reopen. |
| `EXL_STALE_VERSION` | Expected `state_version` does not match. |
| `EXL_STALE_ATTEMPT` | Generation is older than current authority, or the deadline is not strictly ahead. |
| `EXL_PROGRESS_REQUIRED` | Completion was asked for and the progress port said the progress is not committed. |
| `EXL_TERMINAL_CONFLICT` | A different terminal outcome is already committed. |
| `EXL_DEPENDENCY_UNAVAILABLE` | Pinned revision or progress store cannot be read. |
| `EXL_UNCERTAIN_COMMIT` | The commit result is unknown. Retry the same key. |
| Existing technical mapping | Unexpected database errors stay technical and are not shown as input errors. |

## 7. Error Handling

Domain methods return sentinel errors. Usecase code maps them after the transaction ends, except when the transaction itself committed a cancellation and must still return `EXL_ASSIGNMENT_UNAVAILABLE`. Repository not-found becomes `EXL_NOT_FOUND`. A foreign-key failure on a missing revision becomes `EXL_DEPENDENCY_UNAVAILABLE` after rollback. Unique violations on the pin or the command are reconciled inside a new transaction; they are not returned raw to the sibling. The task event queue is not involved. No completion event is published by this kernel.

## 8. Constraints Checklist

These are obligations for implementation. Checked items are recorded in this design. None is observed runtime evidence.

- [x] States are only `CREATED`, `RUNNING`, `COMPLETED`, `FAILED`, `CANCELLED`.
- [x] `CREATED` is accepted and not activated.
- [x] Idempotent start, conflicting intent, attempt fencing, and first terminal commit are specified.
- [x] Interview binding copies the assignment's published revision.
- [x] Lock order is assignment before execution, and execution paths do not lock the assignment afterward.
- [x] Next migration is `000008`, verified against `ReleaseVersion` 7 and `migrations/pgsql`.
- [x] Startup performs no DDL. AutoMigrate is not used.
- [x] GORM for CRUD. Raw SQL only for the assignment lock.
- [x] Four root layers. Domain does not import infrastructure.
- [x] Kernel stores no HR stage id and no invoice field.
- [x] Pin columns are only those in `000007`, plus the new foreign key.
- [x] Sibling calls are kernel operations. Consent, stages, transcript, voice, and report content are out of scope.
- [x] Every new Go declaration is required to satisfy `test/architecture/documentation_test.go`.
- [ ] PostgreSQL proof of locking, rollback, and idempotency still has to be produced by T8.
- [ ] A production revocation or expiry caller still has to exist before BR-CORE-055 is demonstrated outside a fixture.

## 9. Risks & Open Questions

| Item | Effect | Readiness |
|---|---|---|
| Q1. No production command writes `access_revoked_at` or detects authority expiry during a run. | T1–T9 can implement and prove `CancelNonterminal` with a fixture. T10 cannot show an HR revocation or a live session expiry cancelling a run. | Ready when the owning sibling has a real command that commits the access change and calls `CancelNonterminal` under the assignment lock. Owner: Identity and Access together with Candidate Interview. |
| Q2. `interview_assignments.completed_at` is not set when the execution completes. | BR-CORE-039 is met for `Start` because a second execution is not created while the pin exists. A reader that looks only at `completed_at` will not see completion. | Not a blocker. Revisit when the interview completion coordinator is designed. That coordinator may set the marker. This kernel still must not. |
| Q3. Activation retry budget and voice deadlines are unset. | Interview Runtime chooses them when it calls `Fail` or sets `deadline_at`. | Not a blocker for this kernel. |
| Q4. Platform retention (content versus metadata) is not implemented. | Execution rows are kept. This slice stores no transcript bytes. | Out of scope until a retention decision names purge order. No task deletes rows. |
| Q5. P-12's runnable Document Analyzer does not exist. | T9 can show the kernel source and an `Admit` execution with no pin. T11 cannot show a real invoice run on this kernel. | Ready when a Document Analyzer capability is authorized to call `Admit`. |
| Risk. Version-7 binaries will not start after the upgrade. | Operators must run `000008` before the new binary, and must not expect the old binary to tolerate version 8. | T3 documents the floor change in the runbook. |
| Risk. `StartExecution` against a version-8 database fails the new foreign key. | The function is unmounted. Private-table pin tests still pass and still do not prove the release schema. | T8 covers the release schema. T9 checks runtime does not reference `StartExecution`. |
| Risk. Two terminal transactions deadlock if a new path locks the execution and then the assignment. | The repository must not offer that method. | T8 includes a lock-order test and review of the SQL order. |

No independent implementation task is waiting on Q2, Q3, or Q4.

## 10. Revision History

| Version | Date | Change | Reason |
|---|---|---|---|
| v1 | 2026-11-02 | Initial design | Mode A plan for BR-CORE-050..BR-CORE-059. The existing pin is recorded as impact, not as an implementation of the lifecycle. |

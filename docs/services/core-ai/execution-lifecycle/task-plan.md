# Task Plan: Execution Lifecycle

Planning date: 2026-11-02. Mode A. No task is implemented. Do not copy Done status from Identity and Access or Agent Configuration. The execution pin and `StartExecution` are existing code and are not evidence that this plan is underway.

## Ticket Identity

Capability group: **EXL — Execution Lifecycle**. Ticket prefix: `CORE-EXL`. These are repository planning IDs. No external tracker tickets are created by this plan.

| Task | Ticket |
|---|---|
| T1 | `CORE-EXL-001` |
| T2 | `CORE-EXL-002` |
| T3 | `CORE-EXL-003` |
| T4 | `CORE-EXL-004` |
| T5 | `CORE-EXL-005` |
| T6 | `CORE-EXL-006` |
| T7 | `CORE-EXL-007` |
| T8 | `CORE-EXL-008` |
| T9 | `CORE-EXL-009` |
| T10 | `CORE-EXL-010` |
| T11 | `CORE-EXL-011` |

IDs stay stable. Later revisions add T12 as `CORE-EXL-012` and do not renumber these.

## Legend

- Complexity: S (<2h), M (2–4h), L (>4h). Engineers still validate the estimate.
- Status: `[ ]` Todo, `[~]` In Progress, `[x]` Done, `~~obsolete~~` superseded work.
- `ref` points at [technical-design.md](technical-design.md).
- A blocked task is not executable until its readiness condition is true and the design agrees.
- At closure, update the design, this plan, and the strategy to match what was actually verified. Record commands and skips. A skipped database suite is not a pass.

Companion facts: [Identity and Access](../identity-access/technical-design.md) owns sessions and does not cancel executions. [Agent Configuration](../agent-configuration-publication/technical-design.md) owns publication. BR-CORE-013 is satisfied here by never updating the copied revision. Decision log: [decision-log.md](decision-log.md).

## Tasks

### Sequential foundation

- [x] **T1. Model execution identity, binding, and transitions** — Domain — M
  - Ticket: `CORE-EXL-001` — depends: none — ref: design §3, §4, §6 path contracts; decision-log D2, D3, D4, D5.
  - DoD: `domain/aggregate/execution` exposes public behavior for a new `CREATED` execution, restoration of a stored `CREATED` row, the legal edges among the five states, and rejection of `WAITING_INPUT`, `ACTIVATING`, and `RECOVERING` as states. Terminal executions reject activate, progress, complete, fail, and a different cancel. `Complete` fails when no progress reference is supplied. Binding equality ignores any newer revision. A lower attempt generation and a deadline equal to trusted now are rejected. Snapshots copy time pointers. The aggregate has no assignment, stage, invoice, transcript, or salary field. Domain imports stay inside the architecture allowlist. Every new declaration is commented. `make test` and `make test.architecture` pass. No migration and no HTTP.

- [x] **T2. Define sibling-facing operations and ports** — Application — M
  - Ticket: `CORE-EXL-002` — depends: none — ref: design §2, §4, §6, §7.
  - DoD: operation interfaces exist for `Start`, `Admit`, `Activate`, `RecordAttempt`, `CommitProgress`, `Complete`, `Fail`, `Cancel`, and `CancelNonterminal`, plus query interfaces for `Find` and `FindByAssignment`. Reporting's allowed dependency is the query package only. Ports express the assignment-first transaction, the execution lock, progress commitment, and classified errors without GORM or provider types. `.mockery.yml` registers the interfaces and `make mocks` regenerates `test/mocks/` with generator-owned comments. `make test.architecture` passes. No implementation of the rules beyond the signatures is claimed.

T1 and T2 may proceed in parallel. They are not a request to run two agents.

### Schema and persistence

- [x] **T3. Add migration 000008 and raise the clean schema floor to 8** — Infrastructure/operations — L
  - Ticket: `CORE-EXL-003` — depends: T1 — ref: design §3 migration artifacts and seed assessment; decision-log D1, D6.
  - DoD: `migrations/pgsql/000008_execution_lifecycle.up.sql` and `.down.sql` match design §3. Pin files `000007` are not edited. The up creates the five shared tables, backfills `CREATED` from existing pins, then adds `interview_executions_id_fkey`. The down drops that foreign key and the five new tables and does not drop `interview_executions`. No seed rows. `schema.ReleaseVersion` becomes 8. `ownedPublicTables`, `RuntimeGrantTables`, `infrastructure/database/schema/checksums.json`, and `docs/design/database-migration-checksums.sha256` include the new files and tables. The version-7 wording in `compatible.go`, `Makefile` `docker.up`, `scripts/verify_compose.py`, `README.md`, and `docs/design/database-migrations.md` now requires clean version 8. Startup still has no DDL and no `AutoMigrate`. A disposable database proves fresh up to 8, upgrade from a version-7 database that already contains a pin row, backfill shape, rerun no-op, and disposable down/up. `make test.architecture` passes. Record whether `TEST_MIGRATION_DATABASE_URL` was set; a skip is not a pass.

- [x] **T4. Implement the GORM execution store** — Infrastructure — L
  - Ticket: `CORE-EXL-004` — depends: T1, T2, T3 — ref: design §3 adapter and transactions, §5.
  - DoD: models are separate from the aggregate and the DTO. Inserts and updates use GORM. Updates list columns explicitly and omit `instance_id` and `published_revision_id`. Null progress and terminal times follow the map, not struct omission. There is no soft delete and no association preload. `WithAssignment` runs the verified assignment lock statement before any execution lock. `Update` locks the execution and does not lock the assignment. Interview insert writes `agent_executions` before the pin. sqlmock covers that order and a zero-row terminal update. PostgreSQL, through `make test.integration` or `make test.docker`, covers rollback of a failed callback, the unique assignment pin, and a backfilled row round-trip. Declaration comments exist. `make test` and `make lint` pass for the packages touched, and the database command's result is recorded. If the database command is skipped, say so and do not call the lock proof done.

### Application behavior

- [x] **T5. Implement interview start, shared admit, and reads** — Application — L
  - Ticket: `CORE-EXL-005` — depends: T1, T2 — ref: design §5 interview start, §6, §7; decision-log D1, D2, D5.
  - DoD: mocked tests show one `CREATED` execution, no provider call, revision copied from the assignment rather than the active pointer, same key returns the same id, a different fingerprint returns `EXL_IDEMPOTENCY_CONFLICT` without a second row, a second key for the same assignment returns the original, a revoked assignment inserts nothing, a missing assignment is `EXL_NOT_FOUND`, and a malformed key is `INVALID_INPUT` before the lock. `Find` and `FindByAssignment` do not insert. `Admit` commits `CREATED` without an assignment port. Ambiguous commit returns `EXL_UNCERTAIN_COMMIT` and no new id. Error codes and both translation files contain the catalog from design §6. `make test` passes. Mocks are not cited as SQL proof.

- [x] **T6. Implement activation, fencing, and terminal commits** — Application — L
  - Ticket: `CORE-EXL-006` — depends: T1, T2, T5 — ref: design §3 transitions, §5 activation and cancellation race, §6; decision-log D3, D4.
  - DoD: mocked tests show `Activate` moves the same execution to `RUNNING`, a failed activation attempt leaves `CREATED`, and no test uses disconnect as a cancel. `CommitProgress` while `RUNNING` does not complete. `Complete` commits only when the progress port returns true and then stores that reference. There is no provider-success parameter. Stale version and stale generation write nothing. A deadline equal to now is stale. The same terminal command returns the first outcome. A different terminal command returns `EXL_TERMINAL_CONFLICT`. `Fail` from `CREATED` and from `RUNNING` is allowed once. Repeated complete after success does not change the binding. `make test` passes.

- [x] **T7. Implement cancellation when access has ended** — Application — M
  - Ticket: `CORE-EXL-007` — depends: T1, T2, T6 — ref: design §3 transaction scenarios, §5; decision-log D7.
  - DoD: `Cancel` from a nonterminal execution becomes `CANCELLED` and a second identical cancel returns it. `CancelNonterminal` under the assignment lock cancels `CREATED` and `RUNNING` for `access_revoked` and `authority_expired`, and leaves an already terminal execution unchanged. `Start` against a revoked assignment with a running execution cancels it and still returns `EXL_ASSIGNMENT_UNAVAILABLE`. The operation does not read session expiry itself and does not call invitation issue. A test or source check shows `application/usecase/accessusecase/issue_invitation.go` does not call these operations. `make test` passes. This task does not add an HR revocation route.

### Proof

- [x] **T8. Prove locking, rollback, and idempotency on PostgreSQL** — Integration — L
  - Ticket: `CORE-EXL-008` — depends: T3, T4, T5, T6, T7 — ref: design §3 proof column, §5, §8; tactical-strategy §5.
  - DoD: on an authorized disposable database, `make test.integration` or `make test.docker` shows concurrent starts leave one execution, a lost-response retry returns it, a conflicting intent does not replace it, a callback failure rolls back both the shared row and the pin, completion versus cancellation leaves one terminal state, a later active revision does not change the stored binding, a version-7 pin upgrades to `CREATED`, and `Admit` can complete with no pin row. Record the exact command, pass or fail, and any skip. `go test` without the database variable is not this DoD. `make test` also passes.

- [x] **T9. Pass architecture, contract, and lint gates** — Architecture/verification — M
  - Ticket: `CORE-EXL-009` — depends: T4, T6, T7 — ref: design §4, §8; decision-log D6.
  - DoD: `make test`, `make test.all`, `make test.contract`, `make test.architecture`, and `make lint` pass. OpenAPI has no new execution route. The documentation test passes for every new declaration, including unexported ones and interface methods. A source check fails the build if the kernel packages named in design §4 contain stage, invoice, transcript, or salary identifiers. `infrastructure/runtime` does not reference `StartExecution` and does not register a lifecycle route. State the commands and that T8, if not yet run, is still required before calling the database rules proved. T9 does not close T10 or T11.

### Blocked / Deferred

- [ ] **T10. Prove the HR revocation command cancels a nonterminal execution** — Application integration — M
  - Ticket: `CORE-EXL-010` — depends: T7, T8, and Identity and Access T22 — ref: design §9 Q1; decision-log D7, D8.
  - Blocked by: `CORE-IAM-022` is not Done. No production command writes `access_revoked_at`. The only observed writer is `revokeAssignment` in the access integration fixture. In-interview session expiry stays Identity and Access Q7 and is not this task.
  - Ready when: `CORE-IAM-022` is Done and `CancelNonterminal` still accepts assignment id, actor, and reason `access_revoked` without reading the revocation marker.
  - DoD: an end-to-end case on PostgreSQL calls the real `POST /admin/interview-assignments/{id}/revocation` command and shows a nonterminal execution becoming `CANCELLED`, a repeat not reopening it, and invitation reissue still not cancelling it. `make test.integration` or `make test.docker` covers that case. Fixture-only proof remains T7/T8 and does not close this task. This task does not add the route, the marker write, or an `authority_expired` detector.

- [ ] **T11. Run a non-interview agent on this kernel** — Later capability — L
  - Ticket: `CORE-EXL-011` — depends: T8, T9 — ref: design §9 Q5; decision-log D6.
  - Blocked by: no Document Analyzer capability is in this tripack. Scope freeze P-12's runnable half is not authorized here.
  - Ready when: a Document Analyzer design names its scope, revision selection, and progress port, and calls `Admit` without adding stage or invoice columns to the kernel.
  - DoD: that later capability's own tests show one admitted execution reaching a terminal state on `agent_executions` with no interview pin and no kernel diff that adds invoice fields. This task's closure is not a substitute for those tests. The `Admit` fixture in T8 stays the evidence this plan can claim.

## Dependency Graph

```text
T1 → T3 → T4 → T8 → T10
T1 → T5 → T6 → T7 → T8
T2 → T4
T2 → T5
T6 → T9
T7 → T9
T4 → T9
T8 → T11
T9 → T11
Identity and Access T22 (`CORE-IAM-022`) → T10
```

T1 is parallel with T2. T5 is parallel with T3 and T4 once T1 and T2 are done. T9 is parallel with T8 once T4, T6, and T7 are done. T10 stays blocked until `CORE-IAM-022` is Done. T11 stays blocked on its own readiness condition.

## Revision History

| Version | Date | Change | Reason |
|---|---|---|---|
| v1 | 2026-11-02 | Initial plan, all tasks Todo | Mode A. No historical Done status copied. |
| v1.1 | 2026-11-02 | T10 waits on Identity and Access T22. In-interview expiry stays out of T10 | Owner approved the HR revocation command as `CORE-IAM-022`. Documents only |

# Task Plan: Agent Configuration and Publication

Planning baseline: 2026-11-02, Mode C. T1–T14 are implemented. Q6 is closed. Startup accepts clean version 7. No new SQL was added for assignment create or the instance collection.

## Ticket Identity

Capability group: **ACP — Agent Configuration and Publication**. Ticket prefix: `CORE-ACP` (`CORE` identifies the service group). These are repository planning IDs; no external tracker tickets have been created.

Each task has a stable ticket ID in the form `CORE-ACP-NNN`, mapped to its existing local task ID: `T1` → `CORE-ACP-001`, through `T11` → `CORE-ACP-011`. T12 is `CORE-ACP-012`, T13 is `CORE-ACP-013`, and T14 is `CORE-ACP-014`. Keep both identities on revisions, including obsolete tasks; allocate new numbers without reusing earlier IDs. Other capability groups use their own distinct prefix under `CORE`.

## Legend

- IDs T1–T14 are stable; never renumber them on revision.
- Complexity: S (<2h), M (2–4h), L (>4h); rough estimates requiring engineer validation.
- Status: `[ ]` Todo, `[~]` In Progress, `[x]` Done, `~~obsolete~~` superseded work.
- `ref` identifies sections of [technical-design.md](technical-design.md); DoD is required observable evidence.
- A decision-blocked task is not executable until its stated readiness condition is met. Ordinary dependency blocks also apply.
- At closure synchronize affected design/strategy/status. Material changes to contracts or transactions need a resolved decision before dependent tasks; local structural refinements may proceed with preserved-contract evidence.

Companion dependency: [Identity and Access task plan](../identity-access/task-plan.md) and [decision log](../identity-access/decision-log.md). CORE-IAM-007/009 supply the authority adapter and [q2-access-evidence.md](q2-access-evidence.md). That evidence does not mark ACP T8/T9 complete. Identity and Access D8 governs schema installation. D9 does not supply the assignment or execution consumers required by T11.

## Access and schema revision impact (2026-11-01)

Documents-only Mode C revision: preserve T1/T2/T3 Done and all task IDs. No configuration code exists to repair. T6 mocks HR allow, ADMIN deny, candidate deny, and current-authority replay through the existing port and operation names. T8/T9 prove the same behavior on real configuration routes, including HR access to another HR's company configuration. ADMIN account management never implies configuration authority. Publication scope remains the stable account UUID.

T4 registered `migrations/pgsql/000003_agent_configuration.{up,down}.sql`, shifted unpublished candidate/assignment SQL to versions 4–5, and raised the startup floor to clean version 3. That paragraph is the 2026-11-01 baseline. The release stream is now versions 1–7, including `000005_interview_assignments`. There is no `000005_assignment_access` file. Q4, Q5, and Q6 are resolved.

## Tasks

### Independent foundation work

- [x] **T2. Define application transaction, authorization and reference boundaries** — Application — M
  - Ticket ID: `CORE-ACP-002`
  - depends: none — ref: design §2–5, §9 Q2.
  - Evidence: application contracts and six Mockery-generated mocks; `TestCommitUncertainError` RED `39d7c05` then GREEN. See [T2 boundary assessment](t2-boundaries.md).
  - DoD: purpose-specific read/mutation/replay contracts describe rollback, ambiguous commit, trusted principal, and immutable reference behavior without ORM handles; authority cannot be supplied as a client role; interfaces are documented and generated mocks registered. Exact Q1-dependent DTO fields remain deferred, not guessed. This does not implement or declare real admin authentication complete.

- [x] **T3. Implement instance version and ownership mechanics** — Domain — M
  - Ticket ID: `CORE-ACP-003`
  - depends: none — ref: design §3–4, §6.
  - Evidence: public domain version/ownership/restoration tests; RED `2c5d335` then GREEN; architecture/declaration checks pass. See [T3 domain assessment](t3-domain.md).
  - DoD: public tests prove version changes, stale replacement rejection, validation version/ownership matching, active selection protection, immutable revision snapshots, separate restoration, and copied mutable values. Use minimal test configuration fixtures; production template field validation is T5. Architecture/declaration checks pass; no JSON/hashing or outward imports in domain.

T2 and T3 are independent work streams, not instructions to spawn agents. Changes to their interface shape must be synchronized before integration.

### Contract decisions before dependent implementation

- [x] **T1. Resolve template, publication, and public error contracts** — Product/API/domain design — M
  - Ticket ID: `CORE-ACP-001`
  - Evidence: user approved sections A–E on 2026-11-01; see [approval record](t1-contract-proposal.md), [accepted contracts in design §6](technical-design.md#6-request--response), and [decision D5](decision-log.md#d5-t1-template-publication-and-public-error-contracts-are-approved). Document checks pass; no behavioral RED/GREEN commands or SHAs apply to this document-only task.
  - depends: none — ref: design §2–3, §6–7, §9 Q1/Q3.
  - Decision gate: Q1/Q3 resolved by the user approval of all A–E choices, including limits, normalization, warning policy, identity and error ordering. Q2/Q4/Q5 remain independent gates.
  - DoD satisfied: design §6 records accepted input/output schemas, valid/incomplete/invalid examples for both templates, warning behavior, same/different-intent examples, and error ordering. Source-approved constraints are preserved; new limits are backed by explicit A–E approval. Technical design, tactical strategy and task plan are synchronized. No dependent task is implemented here.

### Sequential implementation after contracts are resolved

- [x] **T4. Specify and register the configuration SQL version** — Infrastructure/database schema — M
  - Ticket ID: `CORE-ACP-004`
  - depends: T1, T3 — ref: design §3–4; Identity and Access D8.
  - Evidence: sqlmock compatibility/option tests and artifact/upgrade coverage; RED `d7fd755` then GREEN. Isolated `TestAgentConfigurationVersionUpgrade` skips without `TEST_MIGRATION_DATABASE_URL`.
  - DoD: five configuration tables, ownership and uniqueness constraints, the account-scope foreign key, and explicit zero/null update mappings are reviewable in `migrations/pgsql/000003_agent_configuration.{up,down}.sql`. Unpublished `000003_candidates` became `000004_candidates` and `000004_assignment_access` became `000005_assignment_access`, with Identity and Access design and task references synchronized. `schema.RequireCompatible`, the checksum manifest, runtime grants, and the migration operations document accept clean version 3. Startup still performs no DDL and architecture checks still reject `AutoMigrate`. An isolated database test covers fresh migrate-to-3 and upgrade from version 2 with existing Task and administrative rows. No publication adapter behavior is assumed from the SQL file.

- [x] **T5. Implement both concrete catalogs and deterministic validators** — Domain/application reference resolution — L
  - Ticket ID: `CORE-ACP-005`
  - depends: T1, T3 — ref: design §3–4, §6.
  - Evidence: catalog lookup, HR/Document field-bound construction, deterministic findings, warning eligibility, and Mockery `AgentReferenceReader` exact-version tests; RED `7653be6` then GREEN. sqlmock is not applicable because T5 authors no SQL.
  - DoD: both versioned templates discoverable; full supported field/boundary matrix tested; editable incomplete drafts yield correct findings; warning policy tested; exact reference versions preserved; no model/provider calls. BR-CORE-001/002/006/007/014 local proof captured.

- [x] **T6. Implement configuration commands and detached queries** — Application — L
  - Ticket ID: `CORE-ACP-006`
  - depends: T1, T2, T3, T5 — ref: design §2–7.
  - decision gate: Q1/Q3 resolved; the real authority adapter is not required for mocked application tests.
  - Evidence: Mockery-generated authority, repository, reference-reader, and delivery-operation mocks; HR allow/ADMIN deny/candidate deny; exact-version validation race; replay-before-eligibility; mismatched intent; no result before commit; ambiguous-commit reconciliation; old-command replay after later publication; command scope is the account UUID. RED `63069e9` then GREEN. sqlmock is not applicable because T6 authors no SQL.
  - DoD: create, replace, validate, publish, and read operations call `AgentConfigurationAuthority` with the existing operation names. Exact-version validation recording rejects an edit race; replay precedes current-draft eligibility; mismatched intent conflicts; trusted UTC time and detached outputs are used. Generated mock tests cover HR allow, ADMIN deny, candidate deny, reference failures, no result before commit, ambiguous-commit reconciliation, and old-command replay after later publication. Scope stored for a command is the account UUID. No assumption that mocks establish SQL or cookie/CSRF correctness.

- [x] **T7. Implement PostgreSQL publication transactions and repository proof** — Infrastructure — L
  - Ticket ID: `CORE-ACP-007`
  - depends: T1, T2, T3, T4, T5, T6 — ref: design §3, §5–7.
  - decision gate: Q1/Q3 resolved.
  - Evidence: sqlmock adapter tests plus version-3 PostgreSQL repository proof; RED `2c92d8e` then GREEN. Isolated `TestAgentConfigurationPublicationRepository` skips without `TEST_DATABASE_URL`.
  - DoD: real PostgreSQL tests apply the version-3 migration and cover exact-version replacement/validation/publication races, all write/commit rollback points, duplicate keys with same/different intent, cross-instance constraint checks, replay after new active revision, corrupted restoration, and null/zero mapping. Unique-constraint loser resolves in a fresh transaction. Restart retains revisions. Immutable read-by-ID supports future binding; do not claim full BR-CORE-013 consumer proof. Use isolated Docker or an explicitly authorized integration database. Do not call AutoMigrate.

T4 and T5 may proceed independently once their shared prerequisites are complete. T6 can use generated repository mocks while T4 is prepared; T7 integrates the finalized application contract.

### Sequential exposure after persistence

- [x] **T8. Add secured HTTP endpoints and synchronized API documentation** — Delivery/runtime/contracts — L
  - Ticket ID: `CORE-ACP-008`
  - depends: T1, T2, T5, T6, T7 — ref: design §2, §4, §6–7.
  - DoD: all seven endpoint directions implemented using operation interfaces and the existing `accessauthority.Authority`. Dispatch `/admin/agent-templates` and `/admin/agent-instances` before the access `/admin/` catch-all. Reuse the administrative cookie and CSRF verifier, and map failures to §6.5 ACP codes. Strict input, confidentiality, safe EN/ID errors, correct new/replayed response statuses, and exact precedence are tested. Update OpenAPI, endpoint documentation, and changelog together. Candidate, ADMIN, expired, and revoked sessions fail as specified. Existing access, Task, and voice routes and lifecycle budgets stay intact. No permissive authorizer.

- [x] **T9. Verify the complete backend configuration flow and regressions** — Integration/verification — L
  - Ticket ID: `CORE-ACP-009`
  - depends: T4, T5, T6, T7, T8 — ref: design §3 proof matrix, §8–9.
  - Q1/Q3 are resolved. Configuration-route proof comes from T8; no separate access decision remains.
  - Evidence: `TestAgentConfigurationHTTPAcceptance` creates every fixture through ordinary HTTP create/replace/validate/publish/read. RED does not apply: T4–T8 already implement the flow, and the suite passed on first execution against isolated PostgreSQL. sqlmock and new Mockery mocks do not apply; this task adds no SQL and no interface. Per-rule evidence is below. UI and T11 remain open; this is not a platform MVP-complete claim.
  - Commands, from the repository root: focused `go test ./test/integration -count=1 -run TestAgentConfigurationHTTPAcceptance -v` passed on isolated Postgres (13 subtests). `make test` passed. `make test.all` passed; integration and smoke database cases skipped because `TEST_DATABASE_URL` was unset. `make test.contract` passed. `make test.architecture` passed. `make lint` passed. `make test.security` passed with 0 called vulnerabilities and 1 uncalled required-module finding. `make test.docker` passed, included the new suite, and reported no skips. Existing Task, voice, and access suites in that run passed.
  - Rule evidence: BR-CORE-001 both templates plus unknown version `400`; BR-CORE-002 incomplete document findings and a valid field subset; BR-CORE-003 HR allow, cross-HR read, ADMIN `403` with no insert, candidate/expired/revoked `401`, missing CSRF `403`; BR-CORE-004 stale expected draft `409` (concurrent writers remain the passing T7 repository cases); BR-CORE-005 later edit keeps the active revision and published content; BR-CORE-006 exact eligible and ineligible assessments; BR-CORE-007 blocking publication `422` and `CURRENCY_NOT_REQUIRED` still publishes; BR-CORE-008 stale validation `409`, foreign validation `404`; BR-CORE-009 stored revision text stays unchanged; BR-CORE-010 fault points remain the passing T7 repository cases rerun by `make test.docker`; BR-CORE-011 same key returns the original revision after a newer publication and the active pointer stays on the newest; BR-CORE-012 different intent `409`, and command `scope` equals the account UUID; BR-CORE-013 old revision remains readable, while assignment/execution consumers stay T11; BR-CORE-014 publication uses the local reference resolver and creates no execution.
  - DoD: each template completes create/replace/validate/publish/read through real HTTP/PostgreSQL; stale publication and replay-after-new-publication negative paths pass; HR, ADMIN, and candidate-cookie cases pass; fixture seeding uses ordinary operations. Run `make test`, `make test.all`, `make test.contract`, `make test.architecture`, `make lint`, `make test.security`, and `make test.docker` with the new suites included. Record failures/skips, command outputs and per-rule evidence. Existing Task, voice, and access tests remain passing. Clearly retain UI and T11 acceptance limitations; no unqualified platform MVP-complete claim.

### Blocked / deferred

- [x] **T10. Establish historical template deployment compatibility** — Platform/domain/infrastructure — M
  - Ticket ID: `CORE-ACP-010`
  - depends: T1, T3, T4, T5 — ref: design §3 restoration, §6, §9 Q4.
  - Evidence: domain `RestorationSupported`, sqlmock `TestRequireAgentConfigurationDeploymentCompatible`, adapter historical/sparse restoration and unsupported-behavior rejection; architecture contract test; RED `9e6762c` then GREEN. See [t10-deployment.md](t10-deployment.md) and [t10-contract.md](t10-contract.md).
  - DoD: supported historical revisions restore without current-creation-policy reinterpretation; unsupported deployments are prevented or rejected explicitly under approved policy; rollback rehearsal preserves readable data. Must pass before multi-version rollout; not a blocker for isolated first-version development.

- [x] **T11. Prove binding preservation with actual assignment/execution consumers** — Cross-capability integration — M
  - Ticket ID: `CORE-ACP-011`
  - depends: T9 — ref: design §3 BR-CORE-013, §5, §9 Q5.
  - Evidence: Mockery `PinAssignment`/`StartExecution`/`LoadRequirements` tests, sqlmock `TestBindingPersistenceSQL`, architecture contract, and PostgreSQL `TestAgentConfigurationBindingConsumers`. RED `3e64890` then GREEN. Q5 consumers are thin Interview-shaped pin adapters, not the full execution capability.
  - DoD: assignment pinned before new publication keeps its original revision at start; execution started before publication retains its original requirements; intentional new binding selects the appropriate active revision. Record real integration evidence. Do not implement the entire execution capability inside this task or count a fake consumer as full acceptance. T11 does not mount a public create route.

### Assignment create after Q6

- [x] **T12. Approve the interview-assignment create contract** — Product/API — S
  - Ticket ID: `CORE-ACP-012`
  - depends: T11 — ref: design §11, §9 Q6; [decision D10](decision-log.md#d10-publication-does-not-create-the-interview-assignment-hr-must-pin).
  - Evidence: design §11 approved body/success/errors; Q6 resolved; D10 active for the contract. Documents only. No behavioral RED/GREEN SHAs apply to this document-only task.
  - DoD: §11 states one approved body, success payload, and error mapping. The invitation path still uses the returned assignment id. Access still does not create the row. A later publication still does not move an existing pin. No route, OpenAPI edit, or SQL change is made here.

- [x] **T13. Mount HR assignment create on the existing pin** — Application/delivery/runtime — M
  - Ticket ID: `CORE-ACP-013`
  - depends: T11, T12 — ref: design §11.
  - Evidence: Mockery application and HTTP tests, sqlmock `TestCreateAssignmentPersistenceSQL`, architecture contract, PostgreSQL `TestInterviewAssignmentCreateHTTP`. RED `4127338` then GREEN.
  - DoD: `POST /admin/interview-assignments` follows that contract, the standard envelope, Origin, the HR cookie, and CSRF. The server mints the assignment id. The implementation reads the current active revision through the existing pin path unless T12 explicitly allows a caller-selected revision. `data.id` is the id required by `POST /admin/interview-assignments/{id}/invitations`. ADMIN and a candidate cookie are denied before insert. An unpublished instance is rejected with the code T12 selected. A later publication leaves `published_revision_id` unchanged. The command does not start consent, execution, a provider session, voice binding, or a Document Analyzer job. No migration is added, and `schema.ReleaseVersion` stays 7. Update OpenAPI, the endpoint document, and `docs/api/CHANGELOG.md` together. Prove the HTTP mapping with a mocked operation, the pin selection with mocked ports, and the foreign keys plus the retained pin with PostgreSQL. Run `make test`, `make test.contract`, `make test.architecture`, and `make lint`. Existing invitation, configuration, Task, and voice routes stay intact.

### Instance collection

- [x] **T14. List every agent instance for the current HR** — Application/delivery/infrastructure — S
  - Ticket ID: `CORE-ACP-014`
  - depends: T8 — ref: design §2; [decision D11](decision-log.md#d11-hr-lists-every-agent-instance-without-creator-ownership).
  - Evidence: application `TestListInstancesReturnsEveryFilteredInstance`, HTTP `TestListAgentInstancesHTTP`, and sqlmock `TestListInstancesSQL`. The list does not introduce creator ownership and does not narrow BR-CORE-023. No phase commit: the implementation request forbade commits. OpenAPI, `docs/api/agent-configuration.md`, and `docs/api/CHANGELOG.md` stay with the contract update.
  - DoD: `GET /admin/agent-instances` returns every instance for the current HR. Authorize once with `AgentConfigurationRead` and an empty resource. Each item is the existing filtered view, including `salaryCeilingConfigured` and never the salary reference. Empty `data` is `[]`. Order is `created_at DESC, id DESC`. No search, pagination, `created_by` column, or migration. ADMIN is 403 `ACP_FORBIDDEN`. Unauthenticated uses the existing ACP 401. POST create is unchanged. Other methods on the exact path stay 405.

## Dependency Graph

```text
Q1 + Q3 decisions (resolved) → T1 (Done)
T2 (Done; authority port reused)
T3 (Done; domain version mechanics)
T1 + T3 + unpublished migration versions 3–4 → T4 (SQL version 3)
T1 + T3 → T5 (templates/validators)
T1 + T2 + T3 + T5 → T6 (application)
T1 + T2 + T3 + T4 + T5 + T6 → T7 (PostgreSQL adapter)
T1 + T2 + T5 + T6 + T7 + existing authority adapter → T8 (HTTP)
T4 + T5 + T6 + T7 + T8 → T9 (backend acceptance)
T1 + T3 + T4 + T5 + Q4 → T10 (multi-version deployment gate)
T9 + Q5 consumers [and T10 for multiple behavior versions] → T11 (Done)
T11 + Q6 → T12 (approved create contract; documents only)
T12 → T13 (HTTP create on the existing pin; no new SQL)
T8 → T14 (instance collection; no new SQL; does not narrow BR-CORE-023)
```

## Revision History

| Version | Date | Change |
|---|---|---|
| v1 | 2026-11-01 | Initial remaining-work plan; all tasks Todo and decision dependencies explicit |
| v1.1 | 2026-11-01 | Add capability-group ticket prefix CORE-ACP and stable ticket IDs for T1–T11; task statuses and dependencies unchanged |
| v1.2 | 2026-11-01 | T1 In Progress: link reviewable Q1/Q3 contract candidate; no owner acceptance or dependent-task readiness claimed |
| v1.3 | 2026-11-01 | Record user approval of A–E, resolve Q1/Q3 and mark T1 Done; remaining tasks retain their statuses/dependencies |
| v1.4 | 2026-11-01 | Mode C: preserve T1/T2/T3 Done; require HR allow/ADMIN deny and shared company scope in T6/T8/T9; Q2 remains an implementation gate |
| v1.5 | 2026-11-01 | Link CORE-IAM-009 administrative evidence to Q2/T8/T9 without changing stable task IDs or status |
| v1.6 | 2026-11-01 | Replace T4 AutoMigrate with SQL version 3 and remove the absent-session block from T8; T4–T11 stay Todo |
| v1.7 | 2026-11-01 | Mark T4 Done for configuration SQL version 3, unpublished candidate/assignment reallocation, and clean-version-3 startup |
| v1.8 | 2026-11-01 | Mark T5 Done for versioned catalogs, typed drafts, deterministic validators, and exact reference resolution |
| v1.9 | 2026-11-01 | Mark T6 Done for authorized configuration commands, detached queries, and Mockery-generated application tests |
| v1.11 | 2026-11-01 | Mark T10 Done; resolve Q4 with retain-and-restrict deployment and restoration fixtures |
| v1.12 | 2026-11-01 | Mark T11 Done; resolve Q5 with assignment/execution revision-pin consumers |
| v1.13 | 2026-11-02 | Add decision-blocked T12 and dependent T13 for the public assignment create; record release version 7. Design §11 and D10. Q6 stays open, so T13 is not executable |
| v1.14 | 2026-11-02 | Mark T12 and T13 Done; Q6 closed with the approved create contract and mounted route. No new SQL; startup floor remains version 7 |
| v1.15 | 2026-11-02 | Add T14 `CORE-ACP-014` for `GET /admin/agent-instances`. No creator column and no narrowing of BR-CORE-023 |

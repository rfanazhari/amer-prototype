# Technical Design: Agent Configuration and Publication

## 1. Overview

Date: 2026-11-01. Mode C — align this plan with the implemented Identity and Access administrative adapter and the owner-approved service-wide SQL migrations (Identity and Access D8/D9). Status: T1–T3 remain Done. The administrative access prerequisite for Q2 is satisfied; Q4 and Q5 remain open. Configuration schema, validators, operations, PostgreSQL proof, and HTTP exposure are not implemented. This revision changes documents only.

T1 (`CORE-ACP-001`) is Done. The user approved sections A–E on 2026-11-01:
“approve. - Setujui bagian A–E sebagai kontrak T1”. Section 6 now contains
the accepted contract; Q1/Q3 are resolved. See [approval record](t1-contract-proposal.md)
and [decision log](decision-log.md). T2/T3 are implemented as recorded in task-plan and their assessment notes. This revision changes documents only; no runtime verification is claimed.

Business authority is [Agent Configuration and Publication](../business/capabilities/agent-configuration-publication.md), BR-CORE-001 through BR-CORE-014. The [platform technical design](../../../design/core-ai/technical-design.md), especially sections 3–5 and 9–10, supplies accepted configuration, consistency, and access decisions. The [scope freeze](../../../design/core-ai/shared_ai_agent_platform_consolidated_architecture_mvp_scope_freeze_v0.1.md) retains final P-01–P-05 acceptance requirements. This feature tripack refines that baseline; it does not replace the platform design or describe the entire platform.

### Goals

- Support both predefined, versioned templates and reusable instances with editable drafts.
- Validate a specific draft and publish an immutable revision without stale edits or duplicate publication effects.
- Expose configuration operations to authorized HR actors through the existing REST conventions.
- Preserve published references for future assignment/execution consumers.

### Non-Goals

Execution initiation, lifecycle, candidate access/consent, interview workflow, invoice extraction, reporting, brokers, and artifact storage are separate capabilities. Full admin UI remains a later platform slice. This slice consumes the existing administrative session and authority adapter; it does not build a second authentication system. No changes to Task business behavior, new service process, universal configuration DSL, or template behavior-migration engine are planned. Configuration tables join the existing SQL migration stream.

### Verified context and authority

| Source inspected | Finding / governing constraint |
|---|---|
| Root `AGENTS.md`, Architecture and implementation / Verification / Local resources | Four root layers, application-owned ports, operation interfaces, detached DTOs, trusted Clock, declaration documentation, and HTTP tests. Schema policy is manual versioned SQL (Identity and Access D8). New configuration tables must not use AutoMigrate. |
| `go.mod` | Module `github.com/rfanazhari/amer-prototype`; Go 1.26.10; GORM 1.25.12, PostgreSQL driver 1.5.11; testify and standard `net/http` patterns. |
| `domain/aggregate/task.go`, `domain/valueobject/title.go` | Encapsulated aggregate state, validated value objects, separate restoration and copied snapshots. Task title limits are not agent naming requirements. |
| `application/usecase/taskusecase/create.go`, `application/repository/task.go` | Operation interface + injected plain struct; repository transaction callback; detached result after persistence. |
| `infrastructure/repository/taskrepository/task.go` | Separate ORM model, row-lock transaction, explicit update fields, technical classification of corrupt persisted state. One-row Task storage cannot be reused as the publication repository. |
| `infrastructure/runtime/resources.go`, `infrastructure/database/schema/compatible.go`, `docs/design/database-migrations.md` | Startup calls `schema.RequireCompatible` and accepts only one clean version 2. It performs no DDL. `test/architecture/schema_test.go` rejects executable `AutoMigrate` calls. Versions `000001`–`000002` are checksum-pinned. |
| `delivery/http/handler/taskhandler/handler.go`, `docs/api/openapi.yaml`, `docs/api/task.md` | Bounded strict JSON, standard envelope, EN/ID errors; helpers are currently package-private and Task-oriented. |
| `application/service/agentconfiguration.go`, `application/service/policy.go`, `infrastructure/service/accessauthority/authority.go` | `AdminPrincipal.SubjectID` and `AgentConfigurationAuthority` exist. The real adapter allows current HR, denies ADMIN, and rejects unbound, expired, revoked, and disabled sessions. Production HTTP composition does not yet construct this adapter for configuration routes. |
| `infrastructure/runtime/voice.go`, `infrastructure/runtime/access.go`, `delivery/http/handler/accesshandler` | Every `/admin/` request currently goes to the access handler. That handler owns the administrative cookie, origin check, and `X-CSRF-Token` verification, and maps its own access error catalog. | 
| `migrations/pgsql/000003_candidates.*.sql`, `000004_assignment_access.up.sql` | Candidate identity SQL can follow version 2. Assignment-access SQL references `interview_assignments`, which this repository does not migrate. Those versions are packaged and are not in `infrastructure/database/schema/checksums.json`. |
| `Makefile`, `.mockery.yml`, `test/architecture/` | Generated mocks, race checks, contract checks, import allowlist, per-declaration documentation enforcement. |

The skill's Backend Service Standard and Design Assessment were read. Project instructions and Identity and Access D8 override the bundled golang-migrate default only by fixing the already chosen service stream: GORM remains the CRUD adapter, and schema changes are reviewed SQL applied by `cmd/migrate` before startup. Test delivery through mocked operations, application-owned repository contracts, and encapsulated aggregates remain in force. Schema assertions below are planned additions. No live database was inspected for this revision.

Binding behavior includes BR rules, authorization, version matching, immutable revision meaning, idempotency, and stable bindings. Package names and helper decomposition below are assessed internal choices. Changes to consistency boundaries, normalization, public errors, or retention require a resolved contract decision and synchronized documents before dependent work.

Companion planning: [Identity and Access capability](../business/capabilities/identity-access.md), BR-CORE-020..039, its [decision log](../identity-access/decision-log.md), and its [assessed design](../identity-access/technical-design.md). Its five access tables stay separate from the five configuration tables. D1–D7 and CORE-IAM-007/009 supply the administrative principal, HR-only authorization, cookies, and CSRF. D8 supplies the schema policy. D9 approves candidate contact and exchange contracts; it does not create assignment or execution consumers, so Q5 stays open. Identity and Access Q2 means candidate collaboration and is not this document's Q2.

### Access revision baseline and impact (2026-11-01)

The owner approved separate ADMIN and HR roles, one company with shared HR configuration, and invitation-scoped candidates. ADMIN manages HR accounts and has no configuration permission. The `/admin` route prefix and `AdminPrincipal` type describe the administrative access channel, not a requirement to hold the ADMIN role. Every configuration operation requires current HR authority; a valid ADMIN session resolves a principal but authorization returns 403. Candidate sessions cannot resolve an administrative principal and return 401 under §6.5. An HR may access another HR's configuration within the same company. Existing salary confidentiality remains enforced for HR too.

| Coverage at revision | Difference assessment | Reconciliation |
|---|---|---|
| T2 authority and transaction ports present in `application/service/agentconfiguration.go` and `application/repository/agentconfigport/`; T3 types/tests present in `domain/aggregate/agentconfig/` | Implemented foundation; valid structural refinements recorded in T2/T3 assessment notes | Preserve T2/T3 Done and existing signatures; no code repair identified |
| `accessauthority.Authority` | Implemented administrative adapter; conforms to the T2 port. Production composition does not mount configuration routes | Reuse the adapter in T8. Do not add a second authorizer or a permissive stub |
| Prior combined HR/admin wording | Binding authorization change resolved by owner approval; see decision D6 | BR-CORE-003 already states HR-only authority. T6/T8/T9 still prove it on configuration operations |
| Planned AutoMigrate installation | Contract violation against Identity and Access D8 and the current startup check | T4 uses the next applicable SQL version. No ACP code repair exists because T4–T11 are not implemented |
| T4–T11 configuration behavior | Not yet implemented | Preserve Todo. Q2 access prerequisite is satisfied; configuration-route proof remains T8/T9 |

Publication identity remains `AdminPrincipal.SubjectID`, the administrative account UUID, including across new sessions. Command scope stores that UUID and references `administrative_accounts(id)`. Role separation does not change approved payloads, fingerprints, or error precedence. No existing database was inspected. No implemented configuration behavior violates the refined access contract.

### Identity and access alignment (2026-11-01)

Identity and Access D8 requires reviewed SQL applied before startup. The current binary starts only on clean version 2, and `000004_assignment_access.up.sql` cannot run until Interview creates `interview_assignments`. Appending configuration tables as version 5 would make them unreachable. Versions 1–2 stay immutable. Versions 3–4 are not checksum-pinned and are outside the current release target, so T4 inserts `000003_agent_configuration` and shifts the unpublished candidate and assignment-access files forward. Identity and Access technical-design §3.1 already allows that reallocation before those versions are applied. If a database has already applied version 3 or higher, T4 stops; applied files are not renamed.

D9 does not change configuration payloads, salary filtering, or publication identity. Candidate cookies still fail principal resolution with 401 `ACP_UNAUTHENTICATED`. Q5 remains blocked on real assignment and execution consumers.

## 2. API / Interface Contract Changes

The platform design already selects these endpoint directions. The accepted payloads and error ordering are in §6; they are not yet present in the current OpenAPI.

| Method and path | Operation | Planned result |
|---|---|---|
| GET `/admin/agent-templates` | List supported template versions and configurable fields | 200 catalog, with HR Interviewer and Document Analyzer |
| POST `/admin/agent-instances` | Create an instance for a specific template version | 201 instance with draft and nullable active revision |
| GET `/admin/agent-instances/{id}` | Read authorized draft and active revision identity | 200 detached view |
| PUT `/admin/agent-instances/{id}/draft` | Full replacement using expected version | 200 new draft version; stale replacement 409 |
| POST `/admin/agent-instances/{id}/validations` | Validate exact draft version | Persisted validation findings; 201 result even when findings block publication |
| POST `/admin/agent-instances/{id}/publications` | Publish exact draft/validation using request identity | 201 new revision; 200 same logical publication replay |
| GET `/admin/agent-instances/{id}/revisions/{revision_id}` | Read a specific immutable revision in authorized scope | 200 revision with confidentiality filtering |

Use one narrow operation interface per command/query consumed by delivery, in `application/usecase/agentconfigusecase` and `application/query/agentconfigquery`. Implement with plain structs injecting repository, authorization, reference-readiness, and Clock ports. DTOs live under `application/dto`; never return live domain or ORM objects.

At implementation, update `docs/api/openapi.yaml`, add `docs/api/agent-configuration.md`, and update `docs/api/CHANGELOG.md` together. Existing Task, voice, and access contracts remain compatible. Dispatch these paths before the current `/admin/` access catch-all. Candidate sessions cannot authorize these routes. Reuse `accessauthority.Authority`, the administrative cookie, and the existing CSRF verifier. Map configuration failures to the ACP catalog in §6.5, not to access error codes. Do not expose routes with a placeholder permissive authorizer.

## 3. Data Model & Schema Changes

### Concepts and consistency

`AgentInstance` is the aggregate root for draft replacement and active-revision selection. Its mutable draft is an owned entity identified by instance and version. `PublishedRevision` is an immutable owned entity, queried separately so publication does not load all historical revisions. `ValidationResult` is an immutable assessment record tied to instance, draft version, template version, and validator/behavior interpretation. Its findings are values, not workflow states.

Concrete HR and Document configuration values own their supported field rules. Drafts may be incomplete or semantically invalid so users can correct them; malformed transport and unsupported structural fields are rejected at the transport boundary. Validation converts assessable configuration into findings; publication requires eligibility. Do not use a constructor that makes invalid-but-editable drafts impossible. No speculative `BaseAgent` or arbitrary map-based policy engine is needed.

`TemplateVersionRef`, `DraftVersion`, and immutable configuration snapshots are value concepts. Version identity and equality are owned in the domain; JSON serialization and command fingerprints are application/infrastructure concerns because the domain import allowlist excludes JSON and hashing. Defensively copy configuration slices, reference values, findings, and time pointers on input/output. Naming limits and canonicalization are defined in §6.1, not inherited from Task.Title.

### Proposed additive storage layout

Names below are selected internal design choices. T1 bounds and payload contracts are settled in §6; T4 must derive exact DDL, indexes and DTO-to-JSONB mappings before adapter implementation.

| Table | Planned stored facts | Integrity requirement |
|---|---|---|
| `agent_instances` | UUID identity, template/version, current draft version, nullable active revision ID, created/updated UTC times | One authoritative instance; active revision belongs to that instance |
| `agent_drafts` | Instance ID, version, versioned concrete configuration JSONB, updated time | One current draft per instance; positive version; full replacement |
| `agent_validations` | UUID, instance ID, draft/template/behavior versions, findings JSONB, eligibility, created time | Immutable assessment tied to exact configuration interpretation |
| `agent_revisions` | UUID, instance ID, revision number, source draft/validation, template/behavior version, frozen configuration JSONB, created time | Unique `(instance_id, revision_number)`; insert-only behavior; validation and references belong to same instance |
| `agent_publication_commands` | Scope = administrative account UUID, command kind, request key, intent fingerprint, resulting revision ID | Unique `(scope, command_kind, request_key)`; `scope` references `administrative_accounts(id)` with `ON DELETE RESTRICT`; durable reference to original outcome |

Use primary/foreign keys and uniqueness constraints. Same-instance ownership needs composite references or equivalent checked constraints, not only an arbitrary revision UUID FK. Active revision may be null before first publication. Keep revision/validation references retained together; no purge or soft-delete policy is introduced here. Do not copy confidential CV or salary content into generic command records or responses.

Template definitions are versioned developer-provided data with a concrete two-template catalog, not editable database schemas. Stored references must remain resolvable; replacing an implementation under the same version is forbidden. Deployment compatibility is Q4. No provider readiness network call is part of validation or publication.

### Transactions and persistence mapping

- Create instance and initial draft together. Initial draft content/version and non-idempotent creation follow §6.1 and §6.3.
- Draft replacement locks instance, then current draft; compare expected version before replacement. Explicitly update zero, empty, false, and null fields. Never use ORM omission semantics as replacement semantics.
- Validation reads one detached draft and stable referenced versions. Compute deterministic findings without provider I/O. Persist the result only against the same expected current draft; a concurrent edit returns conflict. External reference reads happen outside the transaction and must resolve immutable versions; mutable readiness is not a publication promise.
- Publish uses an application-owned transaction callback/commit port. Lock instance before draft/validation/revision rows. In the transaction, check the existing command result before current-draft eligibility: a replay must remain successful after later edits/publications. Verify fingerprint, or validate the current exact draft/assessment and allocate a revision. Insert revision, update active pointer, and store command result atomically.
- The unique command constraint arbitrates concurrent requests across processes. A losing insert rolls back all provisional changes, then resolves the committed matching result; never continue using an aborted database transaction. Cross-instance reuse of the same scoped key is a conflict.
- A reported ambiguous commit does not authorize a new identity. Reconcile using the same command key. If the database is unavailable, return a safe uncertain technical outcome; a later retry can resolve it. Do not claim rollback or issue a second publication without evidence.
- The adapter uses GORM 1.25.12, with explicit field selections and separate mapping types. Use bounded direct queries for current draft, validation, or a single revision, avoiding implicit association writes and N+1 history loading. Do not use `Updates(struct)` omission for zero, empty, false, or null replacement. Constraint DDL belongs in the versioned SQL file, not in adapter startup code.

Forward schema policy: add `migrations/pgsql/000003_agent_configuration.up.sql` and its down file for the five tables, primary keys, same-instance composite references, uniqueness, and the account-scope foreign key. Shift the current unpublished `000003_candidates` pair to `000004_candidates` and `000004_assignment_access` to `000005_assignment_access` only after T4 confirms that `checksums.json` still pins versions 1–2 and the release target is still clean version 2. Raise `schema.RequireCompatible`, the checksum manifest, runtime grants, and `docs/design/database-migrations.md` from clean version 2 to clean version 3 in that same change. Startup still performs no DDL. GORM models map rows only.

Verify a fresh migrate to version 3 and an upgrade from a version-2 database that already contains Task and administrative rows. No Task or account backfill is planned. Rollback unmounts configuration routes and starts a binary that still accepts the previous clean version while retaining the new tables. Do not use an AutoMigrate binary as rollback, and do not run down against retained publication data. Physical deletion remains a separate retention decision. The runtime role gains CRUD on the five configuration tables and keeps SELECT on `schema_migrations`; it still must not receive CREATE.

### Rule ownership and proof plan

Every row is planned verification, not passing evidence.

| Material rule | Owner | Enforcement paths | Proof plan |
|---|---|---|---|
| BR-CORE-001 | Versioned template catalog + instance creation | Catalog query, instance creation, restoration reference check | Both templates selectable; unknown version rejected; reference survives round trip |
| BR-CORE-002 | Concrete configuration validation | Full replacement decoding, domain assessment, publication eligibility | Unsupported fields rejected; supported but incomplete drafts yield errors; valid subsets accepted |
| BR-CORE-003 | Existing `AgentConfigurationAuthority` plus configuration operations | Every command/query, including replay/read; delivery maps session and CSRF failures before disclosure | HR allowed, including another HR's company configuration; ADMIN 403; candidate and revoked or expired session 401; no writes after denial |
| BR-CORE-004 | AgentInstance replacement + repository serialization | Draft replacement | Two writers on version 4 produce one version 5 and one conflict |
| BR-CORE-005 | AgentInstance | Draft replacement | Active reference and published content unchanged after edits |
| BR-CORE-006 | Concrete validator + application reference resolution | Validation and recording exact result | Deterministic findings, exact versions, no provider calls; concurrent edit rejected |
| BR-CORE-007 | Validation findings and publication eligibility | Validation result, publish | Correctable safe reason; no revision on blocking errors |
| BR-CORE-008 | AgentInstance publication predicate | New publication transaction | Stale/foreign validation and wrong template version cannot publish |
| BR-CORE-009 | Immutable PublishedRevision + insert-only adapter surface | Publish, read, restoration | Detached snapshots cannot mutate history; later drafts leave old revision unchanged |
| BR-CORE-010 | Publication transaction | New publication | Fault at each write and commit leaves no partial active/revision/command outcome |
| BR-CORE-011 | Publication command lookup + transaction | Duplicate/ambiguous retry | Same ID returns same revision, including after a newer active publication |
| BR-CORE-012 | Command identity/fingerprint contract | New request and replay | Same key with different intent conflicts; concurrent duplicates create one revision |
| BR-CORE-013 | Immutable revision query; future admission consumers own binding selection | Read-by-revision now; assignment/execution integration later | Old revision remains retrievable; later consumer integration must prove pinned references, not claimed complete here |
| BR-CORE-014 | Validation/application boundary | Validation/publication; future execution admission | Provider outage cannot invalidate local configuration; no activation side effects; live revocation remains enforced |

## 4. Component Breakdown

| Layer/Component | Status | Justification |
|---|---|---|
| `domain/aggregate`, `domain/valueobject`, `domain/enum` | New agent types; Task reused unchanged | Own configuration versions, draft mutations, validation findings, publication rules |
| `application/usecase/agentconfigusecase`, `query/agentconfigquery`, `dto` | New | Coordinate authority, exact-version operations, and detached views |
| `application/repository`, `application/service` | Authority port reused; configuration repository ports new | `AgentConfigurationAuthority` and operation names in `policy.go` stay. Add configuration transaction and read ports |
| `infrastructure/repository/agentconfigrepository` | New | Separate GORM models and atomic multi-record mapping; no schema DDL in the adapter |
| `delivery/http/handler/agentconfighandler` | New | Strict transport and ACP error mapping through operation interfaces; reuse access cookie and CSRF verification |
| `infrastructure/runtime`, `infrastructure/database/schema` | Modified | Wire the existing authority adapter, dispatch configuration routes, and raise the clean-version floor with the new SQL version |
| `migrations/pgsql`, `docs/design/database-migrations.md` | Modified | Insert version 3 and shift unpublished versions 3–4 forward; document the new release target |
| `pkg/errs`, `translations/errors.*.toml` | Modified | Feature error categories and safe EN/ID messages |
| `.mockery.yml`, `test/mocks`, contract/integration tests | Modified/new | Generated dependency mocks and real-resource proof |

Authorization is enforced by use cases and queries, not only middleware, so non-HTTP callers cannot bypass it. The authenticated principal must come from a trusted adapter, never a JSON role or arbitrary header. Reference readiness resolves immutable reference identities before deterministic domain assessment; raw file/provider I/O stays outside the domain and transaction.

Existing envelope helpers are private. Prefer a small feature-local formatting implementation with parity tests initially, rather than importing Task handler internals or performing unrelated HTTP framework cleanup. Extraction into a shared transport helper is an allowed internal refinement only with Task/voice contract regression evidence.

### Material structure decisions and assessment

| Choice | Reason and material alternatives/tradeoffs | Authority and refinement boundary | Assessment outcome |
|---|---|---|---|
| Instance aggregate with owned draft and immutable revisions | Handler-only guards leave seed/non-HTTP paths unsafe; a whole-platform aggregate would lock unrelated executions. Instance-level mutation protects the exact affected invariant. | BR-004–010; changing atomicity is material | Retained: snapshot copying and public mutation tests planned |
| Configuration values distinguish editable from publishable | Always-valid constructors would prevent correction of incomplete drafts; raw maps would spread field rules across callers. Typed drafts plus deterministic validation support both paths. | BR-002/006/008; concrete field contract Q1 | Retained structure; validation policy approved in §6.1–6.2, implementation in T5 |
| Narrow transaction port rather than Task repository reuse | Task.Update proves a pattern, not multi-record publication behavior. Adapter must commit revision, pointer, and command result together. | Platform design §5; no GORM in application | Revised for new capability; rollback/concurrency proof required |
| Concrete HR and Document validators | Two real variants need distinct rules; a generic schema/plugin runtime adds interpretation and test burden without current requirements. | Platform design §4 | Retained; new variants require explicit scope |
| Application interfaces at I/O and delivery boundaries | Mocked authority/store faults and actual PostgreSQL adapter must preserve the same contract. Interfaces around pure value operations would add no isolation. | AGENTS.md architecture | Retained; mock success is insufficient persistence proof |
| Explicit restoration path | Reapplying current creation policy could make a historical published revision unreadable. Loader checks stored shape and identity, not today's publication eligibility. | BR-009/013; platform design §3 | Retained; deployment compatibility Q4 remains deferred |
| Versioned SQL after clean version 2 | Startup already refuses every schema except clean version 2 and rejects AutoMigrate. Version 5 cannot be reached while version 4 references a missing Interview table. Inserting version 3 keeps HR configuration installable. | Identity and Access D8; checksum-pinned versions 1–2; unpublished versions 3–4 | Revised: T4 inserts `000003_agent_configuration` and shifts unpublished candidate/assignment files forward. Stop if version 3 or higher was already applied |
| Existing authority adapter | The port, HR allow, ADMIN deny, and current-session checks already exist. A second authorizer or a permissive HTTP stub would duplicate or bypass BR-003. | T2 port; Identity and Access D1/D6/D7; CORE-IAM-009 evidence | Retained adapter. T8 wires it and maps ACP errors. Configuration routes remain unmounted until T8 |

Assessment gate: all fourteen rules have an owner, enforcement path, and proof plan. Q1/Q3 are resolved by T1. The administrative half of Q2 is satisfied by the existing adapter and CORE-IAM-009 evidence. Q4, Q5, and configuration implementation remain. T4 is executable only while versions 3–4 are still unpublished; that confirmation is part of its definition of done.

## 5. Flow Sequence

1. Authorized user reads both template versions and creates an instance with an editable draft.
2. User replaces the complete draft against its current version. Application authorizes; domain checks version; repository commits the replacement.
3. User requests validation against the new version. Resolve stable references, run the template's deterministic checks, and record findings for the exact version. Return errors as findings, not a provider request.
4. User requests publication with expected draft version, validation ID, and request key.
5. Authorize before returning any stored result. Parse/compare the normalized intent under §6.4; inside serialized publication, resolve matching command first. A replay returns the original revision and cannot update the active pointer.
6. For a new command, verify current draft, validation ownership/version, and eligibility. Commit revision, active selection, and command result together.
7. Return a detached, confidentiality-filtered revision after commit. Logs include safe identifiers/outcome only, never configuration bodies or confidential salary data.
8. New bindings may select the active revision; consumers holding an explicit revision continue to read it. This feature does not create those consumers or start model work.

## 6. Request / Response

Accepted by the user on 2026-11-01 for `CORE-ACP-001` (original sections A–E).
These contracts preserve BR-CORE-001–014. T1 adds explicit product choices to
the earlier sources, including the hard 300–600 second publication range and
30-second question/follow-up capacity rules; the source interview target remains
5–10 minutes. These are approved design requirements, not runtime proof.

### 6.1 Template and field schemas (approved A)

Use immutable identities `hr-interviewer` / `document-analyzer`, template
version `1`, behavior version `1`. Catalog is developer-owned; clients cannot
choose behavior independently. Instance template/version cannot be replaced.
API versions are strings; draft/revision numbers are integers in
`1..9007199254740991` (JSON-safe). Overflow rejects mutation as conflict.
Resource IDs are UUIDs. Stable references are `{ "id": UUID, "version": "1" }`;
reference versions are positive decimal strings without leading zeros, at most
10 digits. Reference kinds derive from the field, never from caller authority.

#### Common transport and draft policy

- Create body requires exactly `templateId`, `templateVersion`, `name`,
  `configuration`. Initial draft is version 1, with no active revision.
- `name`: trim outer Unicode whitespace; 1–120 Unicode code points; reject
  control characters. Store on the instance, outside behavioral configuration.
  No rename operation in this slice. Names need not be unique.
- Configuration must be an object matching the selected template; `{}` is a
  supported incomplete draft. Full replacement requires every top-level
  configuration field, including explicit nulls/empty arrays. Creation may omit
  fields and uses the empty defaults below. Neither path auto-publishes.
- Reject duplicate JSON keys, unknown fields, wrong JSON types, unrecognized
  enum values, malformed references and values exceeding storage bounds as 400.
  Reject null except where explicitly nullable. A missing required publication
  value within storage bounds remains editable and yields validation findings.
- Normalize line endings to LF and trim outer whitespace in text. No Unicode
  composition, case folding, or internal whitespace collapse. Empty text is
  editable; no hidden default behavioral content. Reject control characters
  other than LF and tab. Counts below use Unicode code points after normalization.
- Keep question order. Treat competency topics and required invoice fields as
  sets: reject duplicates after normalization; store/return in catalog order for
  invoice fields, lexical order for topics. Do not silently deduplicate.
- All arrays are non-null; empty defaults are `[]`. Nullable defaults are null.

#### HR configuration

| Field | Editable shape/storage bound | Publication requirement |
|---|---|---|
| `jobDescriptionRef` | null or stable reference to immutable synthetic JD | Non-null, accessible, locally ready JD for demo Backend Engineer role |
| `mandatoryQuestions` | 0–30 objects; each exactly `id`, `stage`, `text`; id matches `[a-z][a-z0-9_-]{0,63}`, unique across array; text ≤2000 code points | At least one nonempty question in background, screening and compensation stages |
| `competencyTopics` | 0–20 strings, each ≤120 code points | At least one nonempty topic |
| `stageSeconds` | null or object with exactly `background`, `screening`, `compensation`, `closing`; each integer 0–900 | All positive; total 300–600 seconds; hard range approved in T1; earlier source defined only a target |
| `compensationPolicy` | null or exactly `discussionMode`, `salaryCeilingRef`; mode `expectations_only`; reference nullable, immutable server-only compensation policy | Non-null mode/policy; accessible locally ready confidential ceiling reference |

Question stage enum: `background`, `screening`, `compensation`, `closing`.
Use a deterministic capacity rule: each non-closing mandatory question
requires 30 seconds in its stage budget; a closing question also requires 30
seconds. Exceeding capacity is an error, not a model feasibility judgment.
At least 30 seconds spare in screening is a publication requirement for an
adaptive follow-up. These business limits were approved in T1; the earlier source did not specify them.
Candidate CV, candidate identity, invitation schedule, provider credentials and
arbitrary prompts are excluded from reusable configuration; assignment supplies
candidate-specific inputs later. Compensation reference contents never appear in
catalog, general configuration responses, findings, command records or logs.
Admin response returns `compensationPolicy.discussionMode` and
`salaryCeilingConfigured` (boolean), never the ceiling reference or value.
The confidential reference is write-only; full replacement requires resubmission
through an authorized policy selector, without masked-value placeholders.

#### Document configuration

Exactly one field: `requiredFields`, array of 0–5 distinct enum values in catalog
order `invoice_number`, `vendor`, `date`, `total`, `currency`. Empty draft default
is `[]`; publication requires at least one field. Selecting a subset controls
required validation, not a new extraction schema. Unsupported fields are 400,
not persisted as arbitrary schemas. No document bytes or input artifact belongs
in a reusable instance draft.

#### Examples (accepted expectations)

Valid HR configuration, assuming the referenced versions are available:

```json
{"jobDescriptionRef":{"id":"11111111-1111-4111-8111-111111111111","version":"1"},"mandatoryQuestions":[{"id":"background","stage":"background","text":"Describe your backend experience."},{"id":"screening","stage":"screening","text":"How do you protect a concurrent update?"},{"id":"compensation","stage":"compensation","text":"What are your compensation expectations?"}],"competencyTopics":["concurrency"],"stageSeconds":{"background":60,"screening":240,"compensation":90,"closing":30},"compensationPolicy":{"discussionMode":"expectations_only","salaryCeilingRef":{"id":"22222222-2222-4222-8222-222222222222","version":"1"}}}
```

Incomplete HR: create with `{}` → editable draft with defaults; validation
returns required-value errors. Invalid HR: change the valid example's screening
budget to 0 → replacement is stored, validation returns `STAGE_BUDGET_INVALID`;
publication returns 422. Adding an unsupported `systemPrompt` → 400 before save.

Valid Document: `{"requiredFields":["invoice_number","total","currency"]}`.
Incomplete Document: `{"requiredFields":[]}` → editable, validation returns
`REQUIRED_FIELDS_EMPTY`, publication 422. Invalid Document:
`{"requiredFields":["customer_credit_score"]}` → 400; duplicate `total` also
400. Missing versus explicitly empty is equivalent only during creation;
replacement omitting `requiredFields` is 400.

### 6.2 Validation and warnings (approved B; Q3 resolved)

Use **no acknowledgment gate** in version 1. All error findings block
publication; warnings are informational and never become errors implicitly.
Only warning: Document selects `total` without `currency` →
`CURRENCY_NOT_REQUIRED` at `/requiredFields`. No HR warning kinds in v1.
Example `{"requiredFields":["total"]}` is eligible with that warning.

Findings contain exactly `severity` (`error` or `warning`), stable `code`, JSON
Pointer `path`, and localized safe `message`. Do not echo rejected values.
Sort by path, severity (error first), then code. Eligibility is `errors == 0`;
warnings are retained in the exact immutable validation result. No timestamp or
provider availability enters deterministic assessment. Technical reference
failure returns 503 and records no assessment; missing/ineligible immutable
reference produces a safe `REFERENCE_NOT_READY` finding without its contents.

Error finding codes: `REQUIRED_VALUE`, `QUESTION_TEXT_REQUIRED`,
`QUESTION_STAGE_REQUIRED`, `COMPETENCY_REQUIRED`, `STAGE_BUDGET_INVALID`,
`QUESTION_BUDGET_EXCEEDED`, `FOLLOWUP_BUDGET_REQUIRED`, `REFERENCE_NOT_READY`,
`REQUIRED_FIELDS_EMPTY`. Codes must have EN/ID message catalog entries in T5/T8.

### 6.3 Operation and result schemas (approved C)

Paths and status directions remain §2. Success wraps the data
below in the existing envelope; timestamps are RFC3339 UTC milliseconds.
All response members listed are required, including explicit nullable members.

| Operation | Exact request | Response `data` |
|---|---|---|
| Catalog | No body | `templates`: array of `templateId`, `templateVersion`, `behaviorVersion`, `displayName`, `configurationSchema` (typed constraints/defaults above), `warningPolicy` (`informational`) |
| Create | Common create body above | Instance view below; 201 |
| Read instance | No body | Instance view; 200 |
| Replace | `expectedDraftVersion`, complete `configuration` | Instance view with incremented draft version; 200 |
| Validate | `expectedDraftVersion` | `validationId`, `instanceId`, `draftVersion`, `templateId`, `templateVersion`, `behaviorVersion`, `eligible`, `findings`, `createdAt`; 201 even with errors |
| Publish | `expectedDraftVersion`, `validationId`; required `Idempotency-Key` header | Revision view below; 201 new, 200 replay |
| Read revision | No body | Revision view; 200 |

Instance view: `id`, `name`, `templateId`, `templateVersion`, `behaviorVersion`,
`draftVersion`, filtered `configuration`, nullable `activeRevisionId`,
`createdAt`, `updatedAt`. Revision view: `revisionId`, `instanceId`,
`revisionNumber`, `sourceDraftVersion`, `validationId`, `templateId`,
`templateVersion`, `behaviorVersion`, filtered `configuration`, `publishedAt`.
No actor role, active pointer, eligibility, behavior version or timestamps are
accepted as writable input. Compensation filtering applies to both views.
Creation has no idempotency key in v1: retry may create another instance; clients
must retain returned ID. Seed create/find identity remains a separate explicit
automation concern; do not assume names provide deduplication.

### 6.4 Publication identity, intent and retries (approved D)

Scope = authenticated administrative subject ID; command kind =
`publish-agent-configuration/v1`. Scope is adapter-derived, never request data.
Key syntax `[A-Za-z0-9][A-Za-z0-9._:-]{0,127}`, case-sensitive, one header value,
no trimming; duplicate headers rejected. No expiry/purge in this slice. Different
subjects have distinct scopes; current authorization is required on every retry.

Fingerprint = lowercase hex SHA-256 of UTF-8 compact JSON array
`["publish-agent-configuration/v1",instanceUUID,expectedDraftVersion,validationUUID]`.
UUIDs normalize to lowercase; version is a decimal JSON integer. Order and
serialization are fixed. Exclude configuration content, name, active pointer,
timestamps, actor roles and provider state. Different validation ID means
different intent, even for the same draft. No client-supplied fingerprint.

Example: key `publish-invoice-007` + instance A + draft 3 + validation V3 creates
revision R1. Identical retry after draft 4 and revision R2 exist → return R1, 200;
R2 stays active. Same scoped key + draft 4 or validation V4 or instance B → 409.
Uppercase/lowercase spellings of the same UUID are same intent. Different keys
represent distinct publication commands, even for the same draft/validation.
Known rollback retains no command result; retry with the same key can attempt
again. Ambiguous commit reconciles by the same identity in a fresh read; if still
unavailable, 503 `ACP_PUBLICATION_UNCERTAIN` asks for the same-key retry without
asserting rollback. Never create success before commit, reactivate on replay,
or continue in an aborted transaction.

### 6.5 Public errors and precedence (approved E)

Use the existing standard error envelope and existing `INVALID_INPUT`,
`NOT_FOUND`, `INTERNAL_ERROR` codes where applicable. New codes below are
approved contracts for future `pkg/errs` and EN/ID translation changes, not implemented by T1.

| Priority | Condition | HTTP / code |
|---|---|---|
| 1 | Missing, invalid, expired or revoked admin session | 401 `ACP_UNAUTHENTICATED` |
| 2 | Insufficient collection/operation authority or invalid CSRF | 403 `ACP_FORBIDDEN` |
| 3 | Malformed route/body/key, unsupported schema/field/template/version | 400 `INVALID_INPUT` |
| 4 | Absent or outside-authority instance/revision/validation/reference on direct lookup | 404 `NOT_FOUND` |
| 5 | Existing scoped publication key with different intent | 409 `ACP_IDEMPOTENCY_CONFLICT` |
| 6 | Existing key and same intent | Success replay; skip all current-draft eligibility checks |
| 7 | Stale expected draft, including a concurrent edit during validation | 409 `ACP_DRAFT_CONFLICT` |
| 8 | Owned validation has wrong draft/template/behavior version | 409 `ACP_VALIDATION_CONFLICT` |
| 9 | Applicable validation contains blocking errors | 422 `ACP_PUBLICATION_BLOCKED` |
| — | Reference/store unavailable before known acceptance | 503 `ACP_DEPENDENCY_UNAVAILABLE` |
| — | Commit uncertain and reconciliation unavailable | 503 `ACP_PUBLICATION_UNCERTAIN` |
| — | Corrupt storage or unexpected technical error, including wrapped domain error | 500 `INTERNAL_ERROR` |

Order applies to checks actually reached. Technical failures terminate the
current stage and must be classified before nested domain errors; the table
does not require probing later stages after failure. After authorization of the
target, look up a publication command before current validation lookup; this
keeps a committed replay independent of current draft or validation eligibility.
Out-of-scope instance returns 404 even for a matching command. Foreign validation
returns 404, not a disclosure of its owner. Stale draft wins over absent validation
for a new command. A mismatched key wins over stale draft; corruption during
command restoration returns 500 instead of an apparent replay/conflict.
Method/route/shutdown behavior retains the existing service contract.

Safe messages: conflicts describe stale input or reused identity without values;
422 asks the client to inspect its authorized validation result; 503 uncertain
instructs retry with the same key. No database causes, salary references/values,
configuration bodies or cross-scope resource identities are returned or logged.

### Creation, restoration, and transition contracts

| Path | Accepted inputs/states and normalization | Errors/order and failure timing | Side effects and compatibility | Proof plan |
|---|---|---|---|---|
| Create | Supported version, editable typed draft; fields/defaults §6.1 | Authenticate/authorize before data disclosure; reject unsupported template before persistence | Instance + draft only; no publication | Invalid template, unauthorized caller, rollback |
| Restore | Stored template/configuration schema version, IDs, versions, owned references, historical timestamps | Inconsistent/unknown stored representation is technical failure; never client validation error; no repair-on-read | No current provider check, mutation, or revalidation against latest policy | Corrupt rows, round trip, older supported interpretation |
| Replace | Current expected version, whole configuration | Stale conflict before overwrite; normalization §6.1 | Increment draft only; active revision unchanged | Concurrent replacement; zero/null/empty replacement; snapshot isolation |
| Validate | Exact current draft, immutable reference versions | Concurrent edit conflicts before result publication; findings distinct from technical reference failure | Persist exact result; no model session | Determinism, edit race, missing versus unavailable reference |
| New publish | Exact current draft and applicable validation | Authorization; command resolution; ownership/version; eligibility; exact precedence §6.5 | Atomic revision + active pointer + command | Stale/foreign validation, write/commit fault |
| Replay | Same scoped request identity and same normalized intent | Authorization first; fingerprint mismatch conflicts; do not demand that original draft is still current | Original result only; no pointer change | Replay after newer publication or draft edit |
| Read | Authorized owned instance/revision | Out-of-scope resources hidden; corruption technical | Detached filtered result | Candidate denial; cross-instance references; alias mutation |

## 7. Error Handling

Keep domain rule errors separate from repository corruption and infrastructure failures. Adapter wraps restoration failures as technical; delivery must map technical classification before inspecting a nested domain error, following the existing Task safety pattern.

Planned categories: malformed/unsupported input 400; missing/invalid session 401; forbidden action 403; absent/out-of-scope resource 404; draft/validation/idempotency conflict 409; unavailable dependency before durable acceptance 503; unexpected/corrupt persisted state 500. Validation findings are normal 201 result data; a publication with blocking findings returns 422 `ACP_PUBLICATION_BLOCKED`. Exact code and overlap precedence follow §6.5.

Implement the approved §6.5 codes and EN/ID messages in the dependent tasks; T1 does not modify the runtime catalog. No SQL errors, tokens, cookies, configuration bodies, or sensitive field contents in client errors/default logs. Ambiguous commit results must direct safe same-identity retry without asserting that publication did not happen.

## 8. Constraints Checklist

Design constraints below remain implementation verification gates:

- [ ] BR-CORE-001–014 covered at the ownership paths above; partial external-consumer evidence clearly labeled.
- [ ] Four root layers and allowed imports preserved; all authored declarations documented; gofmt and generated mocks used.
- [ ] Deterministic validation; no model call, execution creation, broker dispatch, or volatile Task queue use.
- [ ] Atomic publication, same-intent replay, conflict detection, and stable detached revisions proven against PostgreSQL.
- [ ] Admin authentication/authorization and safe responses precede HTTP exposure.
- [ ] Explicit SQL migration to the new clean configuration version; startup performs no DDL, keeps the existing startup and shutdown budgets, and still refuses a dirty or unexpected version. Existing Task and administrative data survive the upgrade.
- [ ] OpenAPI, endpoint document, changelog, public codes, and translations synchronized during implementation.

## 9. Risks & Open Questions

| ID | Decision / risk and owner | Affected work | Ready when |
|---|---|---|---|
| Q1 | **Resolved 2026-11-01:** user approved A–E for T1; exact schemas, defaults, normalization, validation/publication statuses, intent and error precedence are in §6 | T1 Done; contract prerequisite met for T4–T9 | Approval recorded in decision-log; implementation and other dependencies remain pending |
| Q2 | **Access prerequisite satisfied 2026-11-01:** `accessauthority.Authority` resolves `administrative_accounts.id`, allows current HR, denies ADMIN, and rejects candidate, expired, revoked, and disabled sessions. Evidence: [q2-access-evidence.md](q2-access-evidence.md) and CORE-IAM-009. Configuration routes are still absent | T8/T9 prove secured configuration exposure with that adapter. T2–T7 do not wait on another access decision | Ready for T8 once T4–T7 are done. No permissive stub. ACP error codes stay §6.5 |
| Q3 | **Resolved 2026-11-01:** informational warnings without acknowledgment; only v1 warning is `CURRENCY_NOT_REQUIRED`; §6.2 | T1 Done; implement/prove in T5–T9 | Approval recorded in decision-log; warnings retained and errors block publication |
| Q4 | Platform/template owners: supported historical behavior versions and deployment compatibility | T10 multi-version deployment gate | Version support/restriction policy and restoration fixtures approved; first-version publication work may proceed independently |
| Q5 | Future assignment/execution capability owners: end-to-end pinned binding proof needs absent consumers. Identity and Access D9 does not supply those consumers | T11 external integration evidence | Real consumers bind immutable revision IDs and pass publication-during-run/assignment tests |

No live database was inspected and no runtime tests were run for this planning task. Exact SQL constraints must be exercised on the project's selected versions, not assumed from a generic ORM recipe. Sensitive HR configuration also needs response-filtering tests even for authorized views; confidentiality is not satisfied by route protection alone.

## 10. Revision History

| Version | Date | Change | Reason |
|---|---|---|---|
| v1 | 2026-11-01 | Initial capability-scoped assessed design | Derive implementation planning from BR-CORE-001–014 and verified repository patterns |
| v1.1 | 2026-11-01 | Link T1 contract candidate without changing accepted contracts | Enable concrete Q1/Q3 owner review before dependent implementation |
| v1.2 | 2026-11-01 | Incorporate user-approved A–E contracts into §6; resolve Q1/Q3 | Complete T1 with synchronized planning and approval evidence |
| v1.3 | 2026-11-01 | Mode C: require HR authority, deny ADMIN configuration access, preserve T2/T3 foundations and Q2 exposure gate | Owner-approved ADMIN/HR separation and shared single-company scope |

| v1.4 | 2026-11-01 | Link companion access capability/tripack and CORE-IAM-009 evidence handoff | Resolve planning ownership for Q2 while preserving its implementation gate |
| v1.5 | 2026-11-01 | Consume the existing HR authority adapter; replace AutoMigrate with SQL version 3 ahead of unpublished candidate/assignment versions; keep Q4/Q5 open | Identity and Access D8/D9 and the version-2 startup contract; configuration routes still unproven |

Business-driven consequences are recorded in [decision-log.md](decision-log.md). Execution order and verification are in [tactical-strategy.md](tactical-strategy.md) and [task-plan.md](task-plan.md).

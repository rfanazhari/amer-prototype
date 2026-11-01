# Shared AI Agent Execution Platform — Technical Design v0.1

Date: 2026-11-01. Status: **Accepted design consolidation; implementation readiness conditional**.
Related feasibility ticket: CORE-000. This is a design record, not an implementation-complete claim.

## 1. Authority, scope and reading guide

The [consolidated architecture and scope freeze](shared_ai_agent_platform_consolidated_architecture_mvp_scope_freeze_v0.1.md) remains the Source of Truth for architecture invariants and MVP acceptance. This document records the owner's subsequent explicit approvals from the technical-design discussion. It does not silently revise that source.

**Accepted** means explicitly approved; **Proposed** means not yet approved; **Open** requires a decision/evidence; **Deferred** is outside current delivery depth. Sections 2–12 describe Accepted decisions unless explicitly qualified. Tables describe planned implementation, not existing database tables or endpoints.

Current development prioritizes backend and candidate interview UI. HR/admin UI is deferred to a later delivery slice; all final MVP platform journeys and P/H/D criteria remain mandatory. Use synthetic data, seeded through normal validation/publication operations. Do not bypass configuration binding with fixtures.

Explicit amendments: ADR-15 logical plane separation and ADR-17 modular monolith are now ratified. RabbitMQ replaces the initially discussed polling-only dispatch for Document/report jobs. Local disk replaces the initially proposed database artifact bytes. GPT-Live is the approved voice provider choice, conditional on feasibility; this clarifies/revises the earlier OpenAI Realtime provider wording without weakening execution semantics.

No microservices, workflow DSL, event sourcing, distributed scheduler, generic plugins, multi-agent orchestration, or automatic hiring decisions are required.

## 2. Architecture overview and Go boundaries

One Go module and one backend process use the existing four root layers. PostgreSQL stores authoritative state; RabbitMQ delivers asynchronous work; local persistent disk stores immutable artifacts. Browser WebRTC carries voice; backend sideband carries provider events/control. REST and polling connect the candidate UI to application state.

| Boundary | Owns | Must not own |
| --- | --- | --- |
| Control Plane | Templates, drafts, validation, publication, active revision, monitoring reads | Execution progression |
| Domain application/admission | Access, consent, eligibility, assignment uniqueness | Kernel transition commits |
| Kernel | Execution identity, legal lifecycle, binding integrity, attempts/fencing, committed references | HR stage or invoice field logic |
| HR/Document coordinators | Concrete mode-specific sequencing and recovery | Eligibility policy or provider protocol |
| Workflows | Coverage, handoff, field validation, completion prerequisites | Generic lifecycle persistence |
| Drivers | Provider/session interaction and classified observations | Business completion or authoritative state |
| HR reporting | Independent report generations and evidence-linked output | Changing terminal interview |

| Layer | Implementation placement |
| --- | --- |
| `domain/` | Configuration, execution, HR and Document invariants; no I/O |
| `application/usecase/`, `query/` | Commands, reads, admission, coordinators, workflow orchestration |
| `application/repository/`, `service/`, `dto/` | Transaction/provider ports, trusted clock, detached DTOs |
| `delivery/http/` | Strict transport validation, error mapping, candidate HTML |
| `infrastructure/` | PostgreSQL, GPT-Live, extraction/report provider, broker and artifact adapters |
| `infrastructure/runtime/` | Composition and resource lifecycle |
| `cmd/app/` | Configuration, signals, shutdown enforcement |

Dependencies point inward: delivery → application → domain; adapters implement application contracts. Only runtime may import delivery from infrastructure. Kernel spans domain invariants and application commit operations; it is not a fifth root layer.

Use interfaces at delivery operations, persistence transactions, clock and provider boundaries. Keep pure rules concrete. HR and Document have separate coordination and provider ports, sharing integrity operations. Do not introduce `BaseAgent`, universal driver execution, or generic CRUD repositories. Exact feature package names and signatures remain Open; layer placement is binding.

### Existing PoC assessment and design rationale

| Verified source | Reuse/adjustment | Rationale and limitation |
| --- | --- | --- |
| `application/usecase/voicesession/negotiate.go` | Retain SDP normalization; replace static interviewer policy during integration | Existing instructions do not implement mandatory HR questions |
| `infrastructure/service/realtimecall/adapter.go` | Adapt session identity, event/control and readiness | Existing adapter returns SDP only and translates defaults to GPT-Live |
| `delivery/http/voicepage/page.html` | Reuse microphone, playback, peer connection and ICE mechanics | Add execution/controller context, bounded recovery and stale-response handling |
| `infrastructure/repository/taskrepository/task.go` | Reuse transactional mutation pattern | Multi-record execution consistency needs dedicated contracts |
| `infrastructure/runtime/` | Extend composition and bounded shutdown | Existing HTTP/Task queue lifecycle does not supervise long-lived voice work |
| `test/livevoice/` | Isolated transport instrumentation | Event counts cannot prove transcript finality or durable recovery |

The chosen structure retains useful transport/persistence patterns while assigning new invariants to their owners. A universal coordinator would hide mode differences; independent per-agent lifecycle implementations would duplicate correctness logic. Local storage is sufficient for one-process synthetic demo but requires persistent mounts and coordinated backup. RabbitMQ adds delivery complexity, accepted for Document/report background work. Task's existing volatile queue retains its original scope.

## 3. Domain model and state transitions

| Model | Identity, ownership and invariants |
| --- | --- |
| Template version | Developer-defined supported configuration and behavior interpretation |
| Instance | Reusable configured agent; owns current draft and active revision selection |
| Draft/validation | Mutable versioned draft; validation tied to exact draft/template version |
| Published revision | Immutable configuration; later publication only affects new bindings |
| HR assignment | Pins published revision and stable inputs; at most one execution |
| Execution | Stable run identity, lifecycle/version, frozen binding, deadlines and committed references |
| Operation/attempt | Logical bounded work and its separate attempts; current authority generation |
| Model session | Local/provider identity and connectivity; subordinate to attempt authority |
| Domain progress/checkpoint | Committed HR coverage/evidence/handoff or Document progress/output |
| Report generation | Separate lifecycle and output revision over finalized interview evidence |

Document need not create an artificial realtime session. Types above do not imply one aggregate/table per concept.

Allowed transitions: `CREATED → RUNNING/FAILED/CANCELLED`; `RUNNING → COMPLETED/FAILED/CANCELLED`. Terminal states never reopen. Reconnect and operation retry remain within the same execution. Provider success alone cannot complete execution.

Operations use PENDING, IN_PROGRESS, SUCCEEDED, FAILED, CANCELLED. Attempts retain outcome, reason, deadline and authority. Retry creates another attempt; it never erases history. Track bounded work rather than every token/audio fragment.

Every progress commit checks nonterminal state, expected committed version, current attempt authority and workflow prerequisites. Row locking serializes writes; version detects stale intent; fencing rejects superseded work. The first terminal commit wins cancellation/completion races. Cleanup diagnostics may follow terminal state but cannot mutate domain progress or terminal output binding.

Creation validates current inputs/policy. Restoration validates stored identity/state/reference consistency without reapplying mutable creation-time policy. Exact restoration error contracts remain Open; corrupt records must not become successful outcomes or user-input errors.

## 4. Configuration and publishing

Use concrete HR and Document configuration types, not a DSL. Validation is deterministic and makes no model call: structural, semantic, stable-reference and locally checkable readiness checks; errors block publication, warnings do not necessarily block it. Provider availability remains an activation concern.

Full draft replacement increments version. Validation records the exact draft and template version. Publish checks expected version and applicable validation, atomically creates immutable revision, updates active pointer, and stores idempotent result. Later edits invalidate the earlier validation for new publication.

Execution binds published revision, template/behavior version, stable input versions and effective behavioral limits. Candidate UI has no arbitrary per-run behavioral override. HR assignment pins revision at preparation; Document resolves active revision on new initiation. Duplicate initiation/reconnect always returns the original binding.

Freeze mandatory questions, stage policy, extraction fields and behavioral limits. Access revocation and safety prohibitions remain live. Credentials remain deployment settings. Confidential salary ceiling stays backend-only, outside candidate responses and model context. Versioned master data must not change active execution behavior.

Seed follows create/find instance → prepare draft → validate → publish → prepare assignment. Reruns deduplicate identical work; changed fixtures create new versions. Old behavior versions need continued support or deployment restrictions while referenced executions remain active; exact deployment policy is Open.

## 5. Persistence and consistency

PostgreSQL/GORM is the authoritative store. Use relational identity/status/version/deadline/reference fields and versioned JSONB for bounded configuration/domain payloads. Keep ORM models separate from domain and DTOs. Explicitly update zero/null values; avoid accidental association persistence and N+1 reads. No soft-delete policy has been selected.

Retain AutoMigrate for additive development/demo schema, per explicit approval. Destructive changes/backfills require a new schema-policy decision; versioned migration tooling is Deferred until required.

### Record relationships and constraints

Configuration instance → draft/validation/revisions; assignment → pinned revision/input references/execution; execution → operations → attempts; attempts → model sessions; execution → domain progress/checkpoints/output revisions; completed interview → report generations; jobs → outbox records.

| Concern | Required constraint |
| --- | --- |
| Commands | Unique scope + command kind + idempotency key; normalized intent fingerprint |
| Draft/revisions | One current draft per instance; unique revision number within instance |
| Assignment | At most one execution; replacement uses a new assignment |
| Operation/attempt | Unique logical operation key within owner; unique attempt number within operation |
| Session | Unique local identity; unique provider identity when known |
| Checkpoint/output | Unique revision number within owning result/execution |
| Report | At most one active generation per interview |
| Outbox | Unique message identity independent of job status |
| Transcript | Scoped provider identity uniqueness after event semantics are verified — Open |

Use foreign keys, required-field constraints and valid-status checks. Exact DDL, indexes and partial uniqueness predicates remain Open; this is a logical persistence contract.

### Atomic transaction contracts

| Operation | Commit together |
| --- | --- |
| Publish | Exact draft check, revision, active pointer, command result |
| Initiate | Deduplication, admission/binding, CREATED, initial operation and applicable outbox |
| Claim | Eligible operation, new attempt, generation, deadline and state |
| Checkpoint | Finalized evidence/progress, checkpoint, execution pointer/version |
| Handoff | Required handoff, authorized advancement, next activation authorization/work |
| Complete | Required output/progress binding, relevant operation outcome, terminal state and authority revocation |
| Cancel | Terminal cancellation and authority revocation |
| Interview completion | Finalized progress + COMPLETED + report generation/job/outbox |
| Report completion | Report output and generation outcome only |

Implement narrow transaction callbacks or commit requests owned by application. Never expose GORM handles or run network I/O within transactions.

Lock order: instance before draft/revision; assignment before execution for HR admission/revocation; execution before operation/attempt/domain children; report generation before its operations. Execution mutation paths must not acquire assignment after execution. Publication and Document initiation serialize active revision selection. Revocation and initiation serialize on assignment.

Idempotency stores resource/result references, not arbitrary HTTP responses. Same key and same intent resolves original result; different intent conflicts. Authorization precedes result disclosure. Do not recompute fingerprint using a newly active revision. Ambiguous database commit is reconciled using the same command identity.

### Local artifacts and retention

Server-generated paths under a private persistent directory; no public static serving or arbitrary client paths. Write temporary file, verify checksum, finalize immutable file on the same filesystem, then commit available metadata. Atomic rename needs appropriate flush for crash durability. Failed metadata commit may leave an orphan. Missing/corrupt file fails explicitly.

Execution-owned content retention: seven days after terminal. Minimal content-free metadata: 30 days. Orphans: 24-hour grace. Shared master/published data and artifacts with active references remain retained. Cleanup covers files, transcript/evidence/results, candidate-specific snapshots, reports, transport results and sensitive diagnostics. Repeatable cleanup records actual deletion outcome; reads explicitly report expired content. Idempotency retention must remain compatible with retained resources; exact purge ordering remains Open.

## 6. Orchestration, RabbitMQ and runtime

Document: commit CREATED/binding/job/outbox → publish → claim attempt and RUNNING → provider extraction → workflow validation → guarded result/terminal commit → ACK.

Report: interview completion atomically creates report job/outbox → publish → independent report attempts/output. Report failure never changes interview outcome.

Use durable queues, persistent messages, publisher confirms and manual ACK. Publisher claims bounded batches and records confirmed publication; lost confirmation can cause duplicates. Consumers read authoritative job state, claim eligible work, and ACK only after durable handling. Finished/terminal jobs are ACKed without provider work. A duplicate while another attempt is active is ACKed; reconciler must recover expired/stuck work. Database unavailability must not be ACKed as successful handling. Malformed messages are isolated without endless requeue; exact dead-letter topology is Open.

Provider retry budget differs from delivery retry. Persist eligible retry time in PostgreSQL; reconciler dispatches through outbox. No delayed-message plugin is required. Reconcile unpublished outbox, published-but-unclaimed jobs and expired attempts on startup and periodically. Never infer job completion from publisher confirmation.

HTTP, publisher, consumer, session work and reconciler have explicit runtime ownership. Background work uses runtime-owned context and bounded operation deadlines. Stop admission/intake, drain within the approved process budget, cancel local work, and recover from durable records. Shutdown does not automatically cancel every execution. Existing five-second process budget remains until explicitly revised.

## 7. HR integration

Invitation → scoped access → notice/consent → microphone readiness → create/find execution → initial activation. Reload never creates a new run. Final candidate screen does not wait for report.

GPT-Live is the approved target. Reuse browser WebRTC. Backend creates session, retains provider identity, attaches sideband and owns application controls. Driver exposes create, attach, observe, supply context, control and close capabilities; it never commits lifecycle. Normalize only observations needed by workflow and retain provenance.

Activation requires bound provider identity, backend control/configuration readiness, and browser media readiness for the same attempt. Browser readiness is not proof that a question was heard. Provider event mapping remains Open. Sideband loss triggers bounded recovery; do not assume replay of missing events. Best-effort media termination and fencing have separate outcomes.

| Stage | Target | Purpose |
| --- | --- | --- |
| Opening/experience | 2 minutes | Context and relevant experience |
| Role exploration | 4 minutes | Mandatory role questions and adaptive follow-ups |
| Expectations/closing | 2 minutes | Availability, expectations, optional configured compensation questions |

Ordered stages, no dynamic branching. Seed at most six mandatory questions; at most two follow-ups per stage. Representative demo must show at least one relevant adaptive follow-up. Coverage distinguishes pending/asked/answered/omitted with finalized evidence. Answered means response recorded, not competence proven.

Permitted omissions: candidate explicitly skips; no answer after one clarification/repetition; soft stage budget expires. Record reasons and gaps. Technical evidence loss cannot masquerade as an omission to authorize progression.

Model proposes transition → workflow validates coverage/time/policy → finalize evidence → commit structured handoff and advancement → activate new session. Handoff includes source/checkpoint, coverage/omissions, provenance-backed carryover and outstanding gaps. Narrative summary is optional; required structured evidence is not. Failed target activation retries the committed target stage.

Checkpoint at meaningful in-stage progress, not only stage end. Partial transcript is not evidence. Finalized transcript carries speaker/session/attempt provenance; correction uses revisions. Recovery uses only committed context and may require repeating uncommitted speech. No hidden model-state restoration.

Completion requires final coverage, permitted omissions and required finalized progress. UI enters Finalizing within the same hard deadline; missing required final evidence causes FAILED with partial committed progress. Early stop is cancellation. Report includes evidence-linked observations, gaps and uncertainty, never hire/reject or ranking. Validate reference ownership and semantic grounding; reference existence alone is insufficient.

## 8. Document integration

One immutable PNG/JPEG invoice, at most 5 MiB and 4096 pixels per side; verify content/type/dimensions before initiation. PDF/multipage and credit notes are Deferred. Provider image support remains a feasibility gate.

Extract invoice_number, vendor, date, total, currency; user chooses required subset. Number/vendor must be nonempty; date must be valid and unambiguous, normalized YYYY-MM-DD; total is exact decimal grand total without line-item recomputation; negative total is invalid. Currency supports IDR/USD. Rp maps to IDR; ambiguous dollar symbol is not guessed.

Every field carries value, valid/missing/invalid, reason and evidence when available. Missing/ambiguous values stay explicit. Valid means extraction rules pass, not invoice authenticity. Evidence references bind artifact version; unsupported location evidence is not fabricated.

All required fields valid permits COMPLETED, even with optional gaps. Required-field failure, malformed output or retryable provider failure share a two-attempt total budget. Retry uses same input/binding with structured feedback; final result comes from one attempt without merging. Exhaustion produces FAILED with a permitted committed diagnostic result. Missing artifact/checksum mismatch fails without model retry.

Documents are untrusted input. Extraction gets no tools to change workflow/state or access unrelated files. Render output as text. Model selection/invocation schema remains Open.

## 9. API and candidate frontend contracts

Planned endpoints below are Accepted contract direction; exact payload schemas, response examples, public error codes and OpenAPI remain Open. Preserve existing envelope, strict fields and EN/ID safe messages. Update OpenAPI, endpoint documentation and changelog together when implemented.

### Candidate

| Method/path | Contract |
| --- | --- |
| POST `/candidate/access/exchange` | Invitation token → scoped cookie and safe assignment context |
| GET `/candidate/interview` | Notice, consent, current execution, available actions |
| POST `/candidate/consents` | Notice version and explicit acceptance → consent identity/time |
| POST `/candidate/executions` | Consent identity + idempotency → create/find bound execution |
| GET `/candidate/executions/{id}` | Scoped state, progress, session condition, actions and version |
| POST `/candidate/executions/{id}/session-attempts` | SDP, join/reconnect intent, expected version, controller/activation authorization |
| POST `/candidate/executions/{id}/session-attempts/{attempt_id}/ready` | Current-attempt browser readiness only |
| POST `/candidate/executions/{id}/takeover` | Explicit controller transfer with expected version and authority revocation |
| POST `/candidate/executions/{id}/cancel` | Explicit early stop, expected version → committed outcome |

Candidate scope comes from session. Tab controller_id in sessionStorage is not a credential. One active controller; other tabs read status. Takeover fences old work. Activation retry with the same key returns the same attempt/result; an unusable old peer connection requires explicit replacement. Ambiguous provider creation remains pending/uncertain within deadline, not automatic duplicate creation.

After committed handoff, polling returns next_action=activate_session with opaque authorization bound to stage advancement/current controller. Browser prepares SDP automatically; it does not choose stage authority.

REST commands/reads and light polling; no browser/backend application WebSocket required. UI states: access error, pre-interview/consent, connecting, active, stage transition, interrupted/reconnect, finalizing, completed, failed/cancelled. Partial transcript display is optional. Stop confirms early termination and stops local capture; backend remains authoritative. Ignore superseded asynchronous responses.

### Admin/configuration, artifacts and results

| Method/path | Contract |
| --- | --- |
| POST `/admin/sessions`; GET `/admin/session`; DELETE `/admin/sessions/current` | Provisioned account login/read/logout |
| GET `/admin/agent-templates` | Predefined templates/versions |
| POST `/admin/agent-instances`; GET `/admin/agent-instances/{id}` | Create/read instance |
| PUT `/admin/agent-instances/{id}/draft` | Full replacement with expected draft version |
| POST `/admin/agent-instances/{id}/validations` | Exact-version validation result |
| POST `/admin/agent-instances/{id}/publications` | Exact draft + validation identity + idempotency |
| GET `/admin/agent-instances/{id}/revisions/{revision_id}` | Immutable revision, appropriate confidentiality filtering |
| POST `/admin/interview-assignments`; GET `/admin/interview-assignments/{id}` | Explicit pinned revision and stable inputs |
| POST `/admin/interview-assignments/{id}/invitations` | Expiring token shown once; reissue revokes old invitation |
| POST `/admin/interview-assignments/{id}/revoke` | Revoke access/sessions and cancel nonterminal interview |
| GET `/admin/interview-assignments/{id}/report` | Separate report lifecycle/output |
| POST `/admin/interview-assignments/{id}/report-generations` | Explicit idempotent regeneration; completed interview, available evidence, no active generation |
| POST `/admin/artifacts` | Multipart file/purpose; idempotency fingerprints content/purpose; no path exposure |
| POST `/admin/document-executions` | Instance + available invoice artifact + idempotency |
| GET `/admin/executions/{id}`; GET `/admin/executions/{id}/operations` | State, attempts, safe outcomes and output availability |
| GET `/admin/document-executions/{id}/result` | Successful/diagnostic result, output revision, fields and bound required set |
| POST `/admin/executions/{id}/cancel` | Expected version and reason |

Creation returns 201, deduplicated creation 200, accepted asynchronous command without new resource creation 202. Errors: 400 malformed, 401 invalid session, 403 forbidden action, 404 outside authorized resource scope, 409 version/idempotency/controller/state conflict, 410 expired invitation/content where applicable, 413 size, 415 media type, 429 rate limit, 503 dependency unavailable before durable acceptance. Validation failures are explicit results; publication rejects errors. Exact precedence is Open where categories overlap.

## 10. Security, observability and operational limits

Companion access planning: [capability BR-CORE-020..039](../../services/core-ai/business/capabilities/identity-access.md) and [Identity and Access tripack](../../services/core-ai/identity-access/technical-design.md). Five access tables are approved as a target; their existence and access guarantees remain unverified until implementation.

Secure HttpOnly cookies, appropriate SameSite, CSRF protection, HTTPS for remote demo. Hash invitation tokens; clean token from address bar after exchange; avoid referrer/third-party leakage. Check expiry/revocation on every action. Candidate access cannot read HR report or internal configuration. The owner-approved access refinement separates ADMIN (HR account management) from HR (shared company configuration and interview operations); ADMIN has no automatic configuration/report authority. Bootstrap the first ADMIN from environment credentials, store password hashes, and provide no public signup. The target is one company, with no organization/workspace tables. Candidates have stable identities and assignment-scoped invitation access, without permanent login accounts; the same candidate may receive multiple assignments. Exchanging a valid invitation creates an access session, not an execution. Reissue invitation does not itself cancel an active interview; assignment revocation does.

Consent records notice version/time before initiation. Require sufficient candidate-session validity for the execution deadline; otherwise reauthenticate through a valid invitation. Revocation/expiry during interview follows committed cancellation. Consent withdrawal data handling remains Open.

| Limit | Accepted initial default |
| --- | --- |
| Invitation | 24 hours |
| Candidate/admin session | 2 hours absolute |
| HR initial activation | 2 attempts; 30 seconds each; 90-second total window |
| HR reconnect | 120 seconds from backend detection; 2 replacement attempts per episode; 2 episodes |
| HR duration | 5–10 minute target; 15-minute hard deadline from RUNNING including recovery/finalization |
| Document | 2 total attempts; 60 seconds each; 5-second retry delay; 3 minutes from initiation including queue |
| Report | 2 total attempts; 60 seconds each; 5-second retry delay; 5 minutes from job creation including queue |

Deadlines survive restart; exhaustion fails honestly. Live measurements may require explicit revision of defaults before demo. Next-stage activation budget details and finalization reserve remain Open.

Structured diagnostics correlate request, execution, operation, attempt, local session, job/outbox, version/generation, classification and timing. Record admission/deduplication, claims/expiry, provider outcomes, committed checkpoints/handoffs, stale rejection, terminal outcomes and reports. Exclude credentials, tokens, cookies, SDP, transcript/document bodies and salary policy from default logs.

Minimum metrics: active executions, pending/expired operations, oldest outbox age, job failures/redeliveries, provider latency, recovery and terminal outcomes. No additional distributed tracing stack is required.

## 11. Failure mode and recovery matrix

| Failure | Required behavior | Verification |
| --- | --- | --- |
| Draft changes after validation | Old publication rejected | Concurrent publish/update test |
| Publish during execution | Existing binding retained | Integration read/execute test |
| Duplicate initiation/message | Same logical result; no automatic extra attempt | DB/broker concurrency test |
| Commit outcome ambiguous | Reconcile same command identity | Fault injection |
| Creation committed before dispatch | Outbox/reconciler resumes bounded work | Process restart |
| Broker unavailable at interview completion | Interview completed; report job durable | Broker outage |
| Publish confirmed, marking fails | Duplicate delivery handled safely | Publisher failure injection |
| Worker dies after commit before ACK | Redelivery observes finished job | Broker integration |
| Provider success before commit failure | No successful committed state claimed | Repository rollback test |
| Provider setup response lost | Same attempt; bounded uncertain outcome | Live/controlled provider test |
| Attempt expires and late result arrives | Fence rejects progress | Concurrent integration test |
| Cancellation races completion | First terminal commit wins | PostgreSQL concurrency |
| Required handoff absent | No next-stage activation | Workflow and live ordering |
| Target activation fails after advancement | Retry committed target stage | HR integration |
| Sideband disconnect | Stop unsafe progress; bounded checkpoint recovery | Live interruption |
| Transcript missing at finalization | FAILED with partial committed evidence | Finalization fault injection |
| Optional summary fails | Use complete structured handoff | Workflow test |
| Artifact absent/changed | Explicit failure, no substitute input | File integrity test |
| Report fails | Interview remains COMPLETED | Application/integration |
| Cleanup provider fails | Terminal outcome retained; cleanup uncertainty recorded | Driver failure injection |
| Shutdown | Stop intake, bounded drain/cancel; durable recovery | Runtime integration |

## 12. Technical decision register

All following entries are **Accepted by explicit owner approval in this conversation**; IDs are local to this consolidation and do not renumber source ADRs.

| ID | Decision | Rationale/impact |
| --- | --- | --- |
| TD-01 | Ratify ADR-15/17; one module/four layers | Preserve repository, simple deployment |
| TD-02 | Concrete HR/Document coordination with shared Kernel | Reuse integrity without workflow engine |
| TD-03 | Backend + candidate UI now; admin UI later | Preserve all final platform acceptance |
| TD-04 | Seed via normal validation/publication | Stable master/configuration identities |
| TD-05 | Separate execution, operations/attempts, sessions, domain progress | Avoid combined state explosion |
| TD-06 | Version checks, fencing, first terminal commit wins | Protect concurrent/stale work |
| TD-07 | Exact publication; HR pinned assignment, Document active resolution | Stable behavior and intentional new runs |
| TD-08 | PostgreSQL/GORM, JSONB payloads, short locked transactions | Atomic multi-record integrity |
| TD-09 | RabbitMQ + outbox for Document/report; same-process workers | Explicit amendment to polling-only dispatch |
| TD-10 | Private immutable local disk artifacts | Explicit amendment to proposed DB bytes |
| TD-11 | GPT-Live + backend sideband | Explicit provider clarification; feasibility conditional |
| TD-12 | Readiness requires provider identity, backend and browser readiness | Avoid SDP-only activation |
| TD-13 | Ordered HR stages, evidence coverage, checkpoint recovery | Bounded showcase with honest gaps |
| TD-14 | Backend-only salary ceiling; consent; no hiring verdict | Confidentiality and safe domain output |
| TD-15 | Document required subset, two attempts, diagnostic results | Honest completion without fabricated values |
| TD-16 | REST/polling, scoped cookies, single tab controller/takeover | Simple candidate UI with authority |
| TD-17 | Accepted initial limits in section 10 | Bounded work, subject to measured validation |
| TD-18 | Retention and explicit content expiry | Synthetic-data lifecycle with stable references |
| TD-19 | Operation-specific contracts and lock ordering | Avoid unrestricted transaction abstraction |
| TD-20 | All P/H/D criteria and real-resource evidence mandatory | Design approval is not acceptance pass |
| TD-21 | Separate ADMIN and HR; one company; invitation-based candidate access | Owner approved ADMIN account management, shared HR configuration, repeat assignments per candidate, and access sessions distinct from executions; refines the previous combined admin/HR role |

Deferred: microservices, workflow DSL, generic plugins, distributed scheduling, event sourcing, enterprise RBAC, arbitrary schemas, batch/multipage documents, raw audio retention, full admin UI in current slice. No new recommendation is silently promoted to Accepted by this document.

## 13. Verification and acceptance traceability

Source criterion text in scope freeze section 7 is authoritative. The following are planned evidence mappings, all **not yet established for the platform**.

| Criteria | Evidence required |
| --- | --- |
| P-01, P-02 | Both templates/instances configurable by non-developer through final platform journey |
| P-03, P-04 | Invalid/exact-version publication and immutability tests |
| P-05 | Publication while execution runs leaves binding unchanged |
| P-06, P-07 | Common authorized admission and duplicate initiation tests |
| P-08, P-10 | Legal/terminal transitions, cancellation race, atomic required output |
| P-09, P-11 | Traceable attempts/errors and authoritative monitoring view |
| P-12 | Dependency review plus runnable Document using shared Kernel |
| H-01, H-04 | Real voice via reused WebRTC; distinct per-stage session identities |
| H-02, H-03 | Coverage/omissions, relevant follow-up, invalid transition rejection |
| H-05, H-06 | Handoff ordering, backend normalized transcript and checkpoint references |
| H-07, H-09 | Evidence-linked coverage/report, missing evidence and finalization tests |
| H-08 | Deliberate disconnect; same execution; committed checkpoint; stale/duplicate protection |
| H-10 | Report failure leaves interview complete |
| H-11, H-12 | No hiring decision and adversarial salary confidentiality |
| D-01, D-02 | Required subset publication drives one invoice execution |
| D-03, D-04, D-05 | Five fields, valid/missing/invalid, ambiguous data remains explicit |
| D-06, D-07, D-08 | Atomic output, controlled retry/exhaustion, shared tracking/lifecycle |

Domain tests cover public invariants; application tests mock ports; HTTP tests mock operations. Real PostgreSQL proves locks/rollback/atomicity; real RabbitMQ proves dispatch/redelivery; browser tests prove asynchronous UI; provider spike proves actual session/event capabilities. Retain scenario, fixture, expected result, evidence ID, pass/fail/not-run and limitations per criterion.

Use repository Makefile commands: `make test`, `make test.all`, `make test.contract`, `make test.architecture`, `make lint`, `make test.security`; isolated `make test.docker`/`make docker.verify` for database evidence. Only explicitly authorized TEST_DATABASE_URL may be used. Skipped database tests are not successful database verification.

CORE-000 transport harness checkpoints: RED `afacc52`, GREEN `6857f27`. Observed local checks passed: focused race tests, domain/application, all-package regression, OpenAPI contract, architecture and lint. Security reported zero reachable vulnerabilities, with dependency findings outside called paths. Manual test is opt-in and database suites skipped without authorized URL. These results establish harness/regression behavior only. No live S1–S4 pass was recorded in this discussion. See [spike guide](../../../test/livevoice/README.md).

## 14. Implementation dependency map and readiness checklist

```text
Provider feasibility ───────────────────────────→ HR driver/session integration
Configuration/publication + admission → shared execution/transaction foundation
Shared foundation → outbox/broker/reconciliation → Document proof
Shared foundation + HR driver → coverage/checkpoints/handoff → reconnect/finalization
Interview finalization + broker → independent HR report
Candidate access/contracts + HR integration → candidate browser flow
All slices + final admin journey → all P/H/D acceptance evidence → MVP complete
```

Security/observability and failure tests accompany every slice. Do not postpone voice feasibility until after broad UI investment. This dependency map is not an implementation task plan or authorization to build the full platform.

- [x] Architecture ownership and explicit amendments recorded.
- [x] Domain lifecycle, binding, attempt authority and completion rules accepted.
- [x] Persistence transactions, dispatch and artifact strategy accepted.
- [x] HR/Document flow and API direction accepted.
- [x] Operational defaults, access and retention decisions recorded.
- [x] Reuse map and implementation evidence distinguished from planned work.
- [ ] Live S1–S4 feasibility evidence collected and reviewed.
- [ ] Exact event identities/finality, early attachment and interruption semantics verified.
- [ ] Document/report model selection and provider contracts finalized.
- [ ] Exact Go signatures, DDL/indexes and complete OpenAPI schemas/error precedence finalized.
- [ ] Queue topology, prefetch/concurrency, reconcile intervals and claim budgets finalized.
- [ ] Next-stage activation bounds, finalization reserve and provider latency validated.
- [ ] CV format beyond structured synthetic seed, consent-withdrawal deletion behavior and purge ordering resolved.
- [ ] Versioned behavior deployment compatibility policy finalized.
- [ ] All P-01–P-12, H-01–H-12 and D-01–D-08 evidence passes.

**Readiness verdict:** sufficient to preserve approved design and prepare independent engineering slices; not an unqualified implementation-ready or MVP-complete declaration. Provider-dependent work remains gated. Open details require explicit resolution before encoding their behavior; routine internal refinements may preserve approved contracts without reopening architecture.

## 15. Revision history and sources

| Version/date | Change |
| --- | --- |
| v0.1 / 2026-11-01 | Consolidated approved technical-design discussion at owner's request; preserved unresolved feasibility/contracts |
| v0.2 / 2026-11-01 | Record owner-approved ADMIN/HR separation, single-company shared HR scope, and stable candidates with assignment-scoped invitation sessions; supersede combined-role wording in section 10 without claiming implementation |

Sources: scope freeze linked in section 1; owner approvals in this conversation; repository AGENTS.md, README.md, go.mod, Makefile, Task design and inspected voice/runtime/persistence sources. Provider research references: [WebRTC](https://developers.openai.com/api/docs/guides/voice-webrtc), [server controls](https://developers.openai.com/api/docs/guides/voice-server-controls). Documentation support is not account-specific or live feasibility evidence.

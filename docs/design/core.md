# Core AI — Project Definition

Status: definition record, derived from `docs/design/core-ai/`
Date: 2026-11-03
Audience: anyone who needs to know what this project builds, which concepts it relies on, and what it is meant to prove, before reading the business or technical design.

This document defines **what is being built, the concepts used, the core details, and the goals**. It does not restate the full technical design and it does not say what is delivered. Delivery status lives in [`docs/services/baseline.md`](../services/baseline.md).

## 1. Sources and precedence

| Source (`docs/design/core-ai/`) | Role |
| --- | --- |
| `shared_ai_agent_platform_consolidated_architecture_mvp_scope_freeze_v0.1.md` | **Source of truth** for architecture invariants, MVP scope and acceptance criteria. |
| `technical-design.md` | Accepted technical decisions that follow the scope freeze (TD-01 to TD-21). It does not silently revise the scope freeze. |
| `shared_ai_agent_platform_architecture_design_v0.1.md` | Original architecture decisions (ADR-01 to ADR-19). Background to the scope freeze. |
| `core_ai_agent_runtime_v0.1.md` | Early direction for a provider-agnostic runtime. Draft direction only, not a specification. |
| `acp-requirement-gap.md` | Gap analysis for Agent Configuration and Publication, mostly for the Document Analyzer. |

Where documents disagree, the order above applies. One known later change: the technical design retains AutoMigrate for development schema, but the repository has since adopted manual versioned SQL migrations (see [section 8](#8-decisions-that-changed-since-the-design)).

## 2. What is being built

**A template-based, no-code Shared AI Agent Execution Platform**, built as one Go modular monolith.

A non-developer picks a predefined **Agent Template**, configures an **Agent Instance** through constrained fields, **validates and publishes** an immutable revision, **starts an execution**, **monitors** its authoritative status, and **inspects** the result. The platform proves this with two different agents that share the same execution mechanics:

| Agent | Role in the project | Depth |
| --- | --- | --- |
| **HR Interviewer** | The showcase. A live, staged, voice interview with a candidate, followed by a separate evidence-linked HR report. | Full product slice for one role (synthetic Backend Engineer). |
| **Document Analyzer** | The architecture proof. Extracts configurable fields from a single invoice (invoice number, vendor, date, total, currency). | Thin runnable slice. It must prove reuse; it is not a second full product. |

The repository also contains **Task Tracking** (`amer-prototype`), a small reference feature that predates the platform and sets the code conventions. It is not part of the platform scope.

### 2.1 What the platform is not

- Not a workflow engine or workflow DSL.
- Not a general multi-agent system.
- Not an automated hiring system. The HR report never hires, rejects, or ranks a candidate.
- Not a user-authored agent builder, and not an arbitrary-schema document platform.
- Not a microservice system. Logical boundaries are not deployment boundaries.

## 3. Goals

### 3.1 Product goal (MVP success statement)

A non-developer can configure, validate, publish, start, monitor, and inspect the outcomes of **two predefined agents** without any code change, and both agents use the **same authoritative execution mechanics** while keeping their own domain workflows.

### 3.2 Goals the design is organised around

| # | Goal | What it means in practice |
| --- | --- | --- |
| G1 | Prove runtime reuse | Two unlike agents (live voice, background extraction) run on one shared execution foundation with no HR or invoice logic inside it. |
| G2 | Configuration without code | Templates and constrained drafts, validation, publication, immutable revisions. |
| G3 | Honest outcomes | A run is `COMPLETED` only when its required output is stored. Gaps, omissions, and failures are recorded, never invented. |
| G4 | Stable behavior | A later edit or publish never changes a run that already began. |
| G5 | Safe repeat | The same request finds the same execution; reload and retry never create a second run. |
| G6 | Bounded recovery | One deliberate HR reconnect from a committed checkpoint; one Document retry; stale attempts cannot overwrite newer progress. |
| G7 | Safe candidate interview | Scoped access, consent before start, confidential salary ceiling never reaches the candidate or the model, no automatic hiring decision. |
| G8 | Simple, risk-first delivery | Reuse the existing proof of concept, deliver in vertical slices, prove voice feasibility early, use synthetic data only. |

### 3.3 Acceptance gate

All Must Have criteria are mandatory for an unqualified "MVP complete" claim (decision G2 in the scope freeze):

| Group | IDs | Covers |
| --- | --- | --- |
| Shared platform | P-01 to P-12 | Templates, configuration, validation, immutable publish, binding, admission, idempotent initiation, lifecycle, traceability, no completion without output, status visibility, kernel free of agent logic. |
| HR Interviewer | H-01 to H-12 | Live voice, mandatory questions, stage authorization, separate session per stage, handoff before next stage, canonical transcript, evidence links, one reconnect, completion only with final record, separate report lifecycle, no hiring decision, salary ceiling never disclosed. |
| Document Analyzer | D-01 to D-08 | Field selection, configuration-driven run, structured result, valid/missing/invalid distinction, no fabrication, result before completion, bounded retry, shared kernel reuse. |

Design approval is not an acceptance pass. These are targets, not claims that the implementation passes them.

### 3.4 Non-goals for the current phase

Microservices, workflow DSL, event sourcing, distributed scheduler, generic plugins, multi-agent orchestration, enterprise RBAC, arbitrary document schemas, batch or multipage documents, raw audio retention, long-term agent memory, and the full HR/admin UI. Current development covers the backend and the candidate interview UI.

## 4. Concepts

### 4.1 Configuration chain

```
Template  →  Instance  →  Draft  →  Validation  →  Published Revision  →  Execution
(developer)  (HR)        (mutable)  (exact draft)   (immutable)             (bound to one revision)
```

| Concept | Meaning |
| --- | --- |
| **Template** | Developer-defined, versioned blueprint of what may be configured. Two exist: HR Interviewer and Document Analyzer. |
| **Instance** | A reusable configured agent. It is not a deployment, process, or provider session. |
| **Draft** | Mutable working settings. Replacing a draft increments its version and invalidates earlier validation for publishing. |
| **Validation** | Deterministic check of one exact draft and template version. No model call. Errors block publication. |
| **Published revision** | Immutable configuration. Publishing is idempotent and sets the active revision. |
| **Active revision** | Default for new Document executions. It never migrates an existing execution. |
| **Assignment** (HR) | Pins one candidate to one published revision of an HR Interviewer instance. At most one execution. |
| **Execution** | An admitted run with a stable identity, bound to one revision and stable inputs. |

### 4.2 Two planes and a minimal kernel

| Part | Owns | Must not own |
| --- | --- | --- |
| **Control Plane** (logical) | Templates, drafts, validation, publication, active revision, monitoring reads | Execution progression |
| **Application / admission** | Access, consent, eligibility, business uniqueness | Lifecycle commits |
| **Minimal Execution Kernel** | Execution identity, legal lifecycle, binding integrity, attempts and fencing, committed progress references | HR stages, invoice fields, model reasoning |
| **Agent workflows** (HR, Document) | Coverage, handoff, field validation, completion prerequisites | Generic lifecycle persistence |
| **Drivers** | Provider and session interaction, classified observations | Business completion, authoritative state |
| **HR reporting** | Independent report generations | Changing a finished interview |

Ownership rule: any layer may *request* a state change, but only the authoritative owner may *commit* it.

### 4.3 Execution lifecycle

```
CREATED ──activation──► RUNNING ──required output committed──► COMPLETED
   │                      │
   ├──unrecoverable───────┴──────────────────────────────────► FAILED
   └──authorized cancellation────────────────────────────────► CANCELLED
```

- Terminal states never reopen.
- `WAITING_INPUT`, `RECOVERING` and provider connectivity are not top-level states. They are conditions of operations or sessions.
- Creation and activation are logically separate. An accepted start means the execution is committed, not that provider work has begun.
- Provider success alone never completes an execution.

### 4.4 Operations, attempts and fencing

- An **operation** is bounded logical work (PENDING, IN_PROGRESS, SUCCEEDED, FAILED, CANCELLED). Each retry is a new **attempt**; history is never erased.
- Every progress commit checks: execution is not terminal, expected version matches, the attempt is the current authority, and workflow prerequisites hold.
- Row locking serializes writes, versions detect stale intent, and fencing rejects superseded work. The first terminal commit wins a cancel-versus-complete race.

### 4.5 Core invariants (frozen)

| ID | Invariant |
| --- | --- |
| INV-01 | Execution identity is stable across reconnects and retries. A new provider session is not a new execution. |
| INV-02 | Only valid transitions; a terminal state cannot reopen. |
| INV-03 | One authoritative committed state per execution version. |
| INV-04 | Repeating a logical command cannot duplicate a committed outcome. |
| INV-05 | Recovery uses only verifiable committed progress. |
| INV-06 | Superseded sessions and attempts are fenced. |
| INV-07 | A terminal outcome refers to committed required output. |
| INV-08 | State changes are traceable to an operation and actor. |
| INV-09 | An execution keeps its effective configuration binding. |

### 4.6 HR interview concepts

| Concept | Rule |
| --- | --- |
| **Stage** | Ordered stages with no dynamic branching. Each active stage has its own voice session. |
| **Hybrid transition** | The model *proposes* a stage change, the workflow *validates* it, the system *commits* it. |
| **Handoff** | The source stage's required evidence is committed before the next stage activates. A missing optional narrative summary may fall back to structured evidence; required evidence is never invented. |
| **Coverage** | Mandatory questions are pending, asked, answered or omitted. "Answered" means a response was recorded, not that competence was proven. |
| **Permitted omission** | Candidate skips; no answer after one clarification; soft stage budget expires. Technical loss of audio or evidence is not an omission. |
| **Canonical transcript** | The backend transcript is authoritative. It is not automatically verified evidence, and a partial transcript is not evidence. |
| **Checkpoint recovery** | Resume from committed progress with a fresh session. No hidden model state is restored. |
| **Report** | A separate, downstream, evidence-linked account of observations, gaps and uncertainty. It does not hire, reject, or rank. Its failure does not change a completed interview. |
| **Salary ceiling** | Backend-only. Never in candidate responses or model context. |

### 4.7 Document Analyzer concepts

One immutable PNG or JPEG invoice per execution (at most 5 MiB and 4096 pixels per side). The configuration chooses the required subset of five supported fields. Fields are `valid`, `missing`, or `invalid`; ambiguous values stay explicit. Two attempts at most, with the same input and binding. Exhaustion produces `FAILED` with a permitted diagnostic result.

### 4.8 Access concepts

| Concept | Meaning |
| --- | --- |
| **ADMIN** | Provisions HR accounts. Has no HR business authority. |
| **HR** | Shared company configuration and interview operations. One company, no organization tables. |
| **Candidate** | A stable identity without a permanent account; may have several assignments. |
| **Invitation** | Time-limited credential for exactly one assignment. Hashed at rest. |
| **Candidate session** | Assignment-scoped access established by exchanging an invitation. Distinct from an execution. |

### 4.9 Provider-agnostic direction

The early runtime direction asks for a core that is not coupled to one model provider. The accepted design keeps that intent through a driver boundary: drivers create, attach, observe, supply context, control and close sessions but never commit lifecycle. GPT-Live is the approved voice provider, conditional on feasibility.

## 5. Core details

### 5.1 Architecture

One Go module, one backend process, the existing four root layers, with dependencies pointing inward.

| Layer | Placement |
| --- | --- |
| `domain/` | Invariants and lifecycle rules for configuration, execution, HR and Document. No I/O. |
| `application/` | Commands in `usecase/`, reads in `query/`, DTOs, repository and service ports, trusted clock. |
| `delivery/` | Strict HTTP input validation, response format, error mapping, candidate HTML. |
| `infrastructure/` | PostgreSQL, GPT-Live, extraction and report provider, broker and artifact adapters. `infrastructure/runtime/` is the composition root. |
| `cmd/app/` | Configuration, logging, signals, process shutdown. |

The kernel spans domain invariants and application commit operations. It is not a fifth root layer. There is no `BaseAgent`, no universal driver loop, and no generic CRUD repository.

### 5.2 Technology

| Concern | Choice |
| --- | --- |
| Language and shape | Go modular monolith |
| State | PostgreSQL through GORM; relational identity, status, version and deadline fields plus versioned JSONB for bounded payloads |
| Async work | RabbitMQ with an outbox for Document and report jobs. Durable queues, persistent messages, publisher confirms, manual ACK. Runs in the same process |
| Artifacts | Private immutable files on local persistent disk |
| Voice | Browser WebRTC for media; backend sideband for provider events and control |
| Candidate UI | REST, polling, scoped cookies, single controlling tab with takeover |
| Schema | Manual versioned SQL migrations (see [section 8](#8-decisions-that-changed-since-the-design)) |

### 5.3 Consistency and messaging

- Short locked transactions. Completion or admission writes the job and its outbox row atomically.
- Delivery retry differs from provider retry budget. Eligible retry time is stored in PostgreSQL and a reconciler recovers unpublished outbox rows, unclaimed jobs and expired attempts on startup and periodically.
- Publisher confirmation never implies job completion. A database error is never acknowledged as success.
- Shutdown stops admission, drains within the process budget (five seconds), cancels local work, and recovers from durable records. Shutdown does not cancel every execution.

### 5.4 Accepted operating limits

| Limit | Default |
| --- | --- |
| Invitation | 24 hours |
| Candidate and admin session | 2 hours, absolute |
| HR initial activation | 2 attempts × 30 s within 90 s |
| HR reconnect | 120 s from detection; 2 replacement attempts per episode; 2 episodes |
| HR interview | 5–10 minute target; 15-minute hard deadline from RUNNING, including recovery and finalization |
| Document | 2 attempts × 60 s, 5 s retry delay, 3 minutes from initiation including queue |
| Report | 2 attempts × 60 s, 5 s retry delay, 5 minutes from job creation including queue |

Deadlines survive restart and exhaustion fails honestly. Defaults may be revised after live measurement.

### 5.5 Security and observability

- HttpOnly cookies, SameSite, CSRF protection, HTTPS for remote demo; expiry and revocation are checked on every action.
- Invitation token is cleaned from the address bar after exchange; the first ADMIN comes from environment credentials; there is no public signup.
- Consent records the notice version and time before initiation. The candidate session must cover the whole execution deadline.
- Logs exclude credentials, tokens, cookies, SDP, transcript or document bodies, and salary policy.
- Logs correlate request, execution, operation, attempt, session and job. Minimum metrics: active executions, pending or expired operations, oldest outbox age, job failures and redeliveries, provider latency, recovery and terminal outcomes.

### 5.6 Reused from the proof of concept

The platform reuses the transactional mutation pattern, SDP normalization, browser microphone/playback/peer mechanics, and bounded shutdown. It replaces the static interviewer policy, extends the call adapter with session identity and readiness, and adds execution and controller context. Event counts from the earlier live probe cannot prove transcript finality or durable recovery.

## 6. Capability map

Each platform capability has a business document under `docs/services/core-ai/business/capabilities/`.

| Capability | Rules | Purpose |
| --- | --- | --- |
| Agent Configuration and Publication | BR-CORE-001 to 014 | Configure, validate, publish; assignment pins a revision. |
| Identity and Access | BR-CORE-020 to 039 | ADMIN, HR, candidate identity, invitations, sessions. |
| Candidate Interview | BR-CORE-040 to 047 | Notice, consent, enter one interview, takeover, cancel, final screen. |
| Execution Lifecycle | BR-CORE-050 to 059 | One stable run per assignment, honest ending, fencing. |
| Interview Runtime | BR-CORE-060 to 071 | Staged voice interview, coverage, handoff, recovery, completion. |
| Interview Reporting | BR-CORE-080 to 086 | Separate evidence-linked HR report. |

Document Analyzer execution has no capability document yet; its design is in the platform technical design (§8) and the ACP gap analysis.

## 7. Reference journeys

**HR Interviewer.** HR configures, validates and publishes an HR Interviewer instance. HR records a candidate, creates an assignment (pinning the revision), and issues an invitation. The candidate exchanges the invitation, reviews the notice and consents, and starts. The execution is committed and activated. The interview asks mandatory questions and at least one relevant follow-up, moves stage by stage only after a committed handoff, survives one reconnect from a checkpoint, and completes only with the required record. A report is generated separately and shown only to HR.

**Document Analyzer.** A user configures required invoice fields, publishes, and starts extraction for one invoice. The run extracts and validates fields, may retry once, commits a structured result, and completes honestly, with missing or invalid fields shown rather than invented.

## 8. Decisions that changed since the design

| Topic | Design text | Current direction |
| --- | --- | --- |
| Schema management | `technical-design.md` §5 retains AutoMigrate for additive development schema and defers migration tooling. | Manual versioned SQL migrations are the approved policy (Identity and Access decision D8). Startup requires a clean migration version and never creates or repairs schema. |
| Provider wording | Scope freeze says OpenAI Realtime. | GPT-Live is the approved provider (TD-11), conditional on feasibility. |
| Dispatch | Original design allowed polling-only dispatch. | RabbitMQ with an outbox (TD-09). |
| Artifact storage | Original proposal stored bytes in the database. | Private local disk (TD-10). |
| Roles | Earlier design combined admin and HR. | Separate ADMIN and HR roles (TD-21). |

## 9. Open items recorded in the design

- Provider event finality map (blocks transcript ingestion, coverage from speech, and reconnect proof).
- Next-stage activation budget and finalization reserve (a budget of 2 attempts × 30 s within 90 s per stage was decided on 2026-11-03).
- Dead-letter topology, report model contract, and the finalized-evidence read model.
- Consent withdrawal data handling, and retention for invitations and candidate sessions.
- Document Analyzer capability design: revision selection at admission, result and progress storage, model and queue details.
- Deployment policy for old behavior versions while referenced executions remain active.
- Exact feature package names and signatures (layer placement is binding).

## 10. Where to go next

| Need | Document |
| --- | --- |
| What is delivered, in progress, not done | [`docs/services/baseline.md`](../services/baseline.md) |
| Business rules per capability | `docs/services/core-ai/business/capabilities/` |
| Full technical design and decision register | [`core-ai/technical-design.md`](core-ai/technical-design.md) |
| Architecture invariants and acceptance criteria | [`core-ai/shared_ai_agent_platform_consolidated_architecture_mvp_scope_freeze_v0.1.md`](core-ai/shared_ai_agent_platform_consolidated_architecture_mvp_scope_freeze_v0.1.md) |

## 11. Change history

Newest first. Add a row for every update, and change the date at the top of the document.

| Version | Date | Author | Change |
| --- | --- | --- | --- |
| 1.0 | 2026-11-03 | rfanazhari | Initial project definition derived from `docs/design/core-ai/`: what is built, concepts, core details, goals, decisions changed since the design. |

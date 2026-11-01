# Shared AI Agent Execution Platform — Architecture Design v0.1

**Status:** Architecture design consolidation (pre-implementation)  
**Date:** 2026-10-31  
**Scope:** Shared AI Agent Execution Platform, Minimal Execution Kernel, HR Interviewer reference agent, Document Analyzer reference agent  
**Next phase:** Architecture consolidation, MVP scope freeze, then technical design

> This document consolidates decisions and proposals discussed in the architecture-design conversation. It is **not** an implementation specification. Items labeled **Accepted** are user-selected decisions; items labeled **Proposed** are recommendations not yet explicitly confirmed. The earlier HR Interviewer MVP draft is background context, not an override of the platform architecture agreed later.

## 1. Executive Summary

Build a **template-based, no-code Shared AI Agent Execution Platform** in which non-developers select predefined templates, configure agent instances, publish validated revisions, request executions, and monitor results. The platform's **Core AI Agent Runtime** must remain reusable across interaction styles and domains.

**Initial reference agent:** HR Interviewer, a 5–10-minute autonomous voice interviewer using mandatory HR questions and adaptive follow-ups. **Second architectural reference agent:** Document Analyzer, a task-based agent that extracts and validates structured invoice information. The second agent is a **reusability test case**, not necessarily a commitment to ship a complete second product in the MVP.

**Architecture principle:** The **Control Plane** governs *what may run and with what configuration*; the **Execution Plane** governs *execution identity, state, progress, recovery, and outcomes*. The **Minimal Execution Kernel** protects execution integrity, not business-specific reasoning.

## 2. Goals and Non-Goals

### Goals

- Enable non-developers to configure and execute agents using constrained, predefined templates.
- Share execution lifecycle, state consistency, operation tracking, observability, and recovery semantics across agents.
- Preserve agent-specific workflows and completion contracts outside the Core Kernel.
- Make published configuration revisions stable and executions reproducible at the behavioral configuration level.
- Keep authorization, security policy, sensitive information, and human review explicit.
- Reuse the existing Go + HTML + WebRTC + OpenAI realtime voice PoC where practical.

### Non-goals for v0.1

- Arbitrary user-authored agent code, unrestricted prompts, or unrestricted tool permissions.
- Universal workflow DSL, visual workflow editor, multi-agent orchestration, distributed scheduler, or general event-sourcing platform.
- Mandatory microservices, independent deployment per Agent Instance, or speculative abstraction layers.
- Autonomous hiring/rejection decisions.
- Full implementation-level design of Go packages/interfaces, APIs, schemas, database tables, or infrastructure.

## 3. Architecture Decisions Register

| ID | Decision | Status | Notes |
|---|---|---|---|
| ADR-01 | One shared execution model with mode-specific execution strategies | Accepted | Avoid universal blocking loop and duplicated runners |
| ADR-02 | Hybrid stage transitions | Accepted | Model proposes; workflow validates; runtime commits |
| ADR-03 | Realtime session per interview stage | Accepted | Multiple session attempts possible during recovery |
| ADR-04 | Hybrid cross-stage carryover | Accepted | Structured evidence/coverage plus optional narrative summary |
| ADR-05 | Eager handoff before next stage activation | Accepted | Finalize and commit source-stage handoff first |
| ADR-06 | Conditional fallback for missing narrative summary | Accepted | Structured required evidence must remain valid |
| ADR-07 | Resume from committed checkpoint | Accepted | Do not attempt to restore hidden model state |
| ADR-08 | Backend canonical transcript authority | Accepted | Provider events normalized; transcript is not automatically evidence |
| ADR-09 | Post-execution HR report generation | Accepted | Interview completion independent of report lifecycle |
| ADR-10 | Separate Execution and Post-Processing | Accepted | Primary task output can still belong inside an execution |
| ADR-11 | Minimal Execution Kernel | Accepted | Kernel handles execution integrity, not HR semantics |
| ADR-12 | Shared Agent Execution Platform | Accepted | Beyond a reusable library |
| ADR-13 | Draft → Validate → Publish | Accepted | Configuration publishing boundary |
| ADR-14 | Unified Execution Request Model | Accepted | Common request boundary, diverse trigger sources |
| ADR-15 | Logical Control Plane / Execution Plane separation | Proposed | Can coexist in one backend |
| ADR-16 | Immutable published revisions and execution configuration snapshots | Proposed | Strongly recommended; details need confirmation |
| ADR-17 | Modular monolith for MVP | Proposed | Single backend, explicit logical boundaries |
| ADR-18 | Idempotent initiation, domain-specific business uniqueness | Proposed | Distinguish retries from intentional re-runs |
| ADR-19 | Decoupled creation and activation | **Open** | Recommended but not selected by user |

**Interpretation:** Accepted architectural directions do not automatically mean every supporting mechanism or detailed invariant has been separately approved.

## 4. Domain and Platform Concepts

### Agent Template

Platform-provided blueprint defining supported capabilities, execution behavior, configurable fields, validation rules, and policy constraints. Template versions may evolve independently.

### Agent Instance

A logical, reusable configured agent created from a template. It is **not** a process, deployment, realtime session, or individual execution. A single instance may have an editable draft and a separately active published revision.

### Published Revision

A validated, immutable revision of an Agent Instance configuration, suitable for binding to new executions. Editing the draft does not mutate already published revisions.

### Agent Execution

One bounded attempt to achieve an agent-defined outcome. Execution identity and lifecycle remain stable even if model sessions or operation attempts change. Domain-specific completion criteria are defined by the agent workflow/contract.

### Operation

A traceable unit of work attempted during an execution, such as model invocation, tool use, session creation, or output validation. An operation failure need not mean the entire execution failed.

### Model Session

A temporary external resource used by a driver; subordinate to an execution. HR Interviewer uses a new realtime model session per stage and may create replacement attempts after disconnection.

### Trigger / Execution Request / Execution

- **Trigger:** An event or action that initiates intent (UI, candidate link, API, future scheduler).
- **Execution Request:** An authorized, validated intent to run a specific agent with specific inputs.
- **Execution:** The authoritative runtime resource created or identified after admission.

A trigger is not authorization; an accepted request is not proof that the model workload is already running.

## 5. Logical Platform Boundaries

```text
[User UI]  [Candidate Link]  [API]  [Future Scheduler]
     \          |             |          /
       ------ Trigger / Application Boundaries ------
                           |
                 CONTROL PLANE / ADMISSION
          Templates · Drafts · Validation · Publishing
          Access · Eligibility · Configuration Resolution
          Execution Request Admission · Monitoring Views
                           |
                Authorized Execution Request
                           |
                     EXECUTION PLANE
              Execution Coordinator / Drivers
                 Minimal Execution Kernel
          Lifecycle · Consistency · Operations · Events
               Context Refs · Recovery Primitives
                           |
              Agent-Specific Workflows/Capabilities
            HR Voice                 Document Task
            Model/Tools              Model/Validation
                           |
             Shared Persistence / External Providers
```

This is **logical separation**, not a mandate for separate services, databases, or deployments.

### Ownership

| Concern | Primary owner |
|---|---|
| Template catalog, definition, versioning | Control Plane / Template developer |
| Agent Instance draft, validation, publishing, active revision | Control Plane |
| Request authentication, domain eligibility, admission | Application / Admission boundary |
| Authoritative execution identity and lifecycle | Execution Plane / Kernel |
| Generic operation tracking and committed progress references | Execution Plane / Kernel |
| Workflow transitions, coverage, completion criteria | Agent-specific workflow |
| Model session and transport specifics | Execution driver / capability |
| Context interpretation and prompt assembly | Agent workflow / context capability |
| Business artifacts and downstream report lifecycle | Domain application / post-processing |
| UI monitoring | Control Plane reads Execution Plane states/events |

Control Plane requests cancellation or initiation; it does **not** directly mutate authoritative execution state. Execution Plane does **not** mutate an Agent Instance's published configuration.

## 6. Minimal Execution Kernel

### Six minimum logical responsibilities

1. **Execution lifecycle:** Stable identity; create, run, wait where applicable, complete, fail, cancel.
2. **State consistency:** Authorized transitions, committed state revisions, concurrency safety.
3. **Operation tracking:** Operation identity, attempts, outcomes, errors, correlation.
4. **Context references:** References to committed, versioned context/progress; not interpretation of domain content.
5. **Execution events:** Traceable meaningful state/operation changes; events are diagnostics, **not** necessarily event sourcing.
6. **Recovery primitives:** Checkpoint boundaries, safe resume, attempt/session fencing where needed.

The kernel should **not** require stage IDs, transcript semantics, mandatory-question coverage, provider-specific realtime behavior, invoice extraction fields, report generation, or a universal workflow DSL.

### Proposed execution invariants

| ID | Invariant | Consequence |
|---|---|---|
| INV-01 | Execution identity remains stable across retries/reconnections | Session replacement does not create a new execution |
| INV-02 | State transitions are valid; terminal state cannot reopen | Prevents inconsistent lifecycle |
| INV-03 | One authoritative committed state per execution revision | Concurrent updates must be reconciled |
| INV-04 | Repeated logical commands do not duplicate committed outcomes | Idempotency at platform boundaries |
| INV-05 | Recovery resumes only from verifiable committed progress | No invented completion |
| INV-06 | Superseded session/operation attempts cannot overwrite newer progress | Fencing against stale events |
| INV-07 | Terminal outcome references the appropriate output revision | Agent-specific completion contract governs validity |
| INV-08 | State changes are traceable to operations/actors | Observability and audit support |

**Caveat:** Idempotent commands do not imply exactly-once external side effects. External tools need their own deduplication/reconciliation when relevant. The exact strength of output and audit invariants depends on agent contracts and policy.

### Execution model

- **Execution** is the root boundary, not conversation or provider session.
- **Agent Definition/Workflow** decides what completion means.
- **Execution Coordinator/Driver** connects agent behavior to shared kernel semantics; it need not be one universal component.
- **Conceptual phases:** Initialize → Prepare → Execute → Evaluate → Finalize. Realtime flows may be event-driven, not strictly linear.
- **Possible lifecycle:** `CREATED → RUNNING ↔ WAITING_INPUT → COMPLETED | FAILED | CANCELLED`. Treat this as a proposal, not a finalized state machine; do not add every temporary operational condition as an execution status.
- **Four conceptual contracts:** Input, Progress, Completion, Output. Defined according to agent requirements; runtime enforces shared mechanics.

## 7. Agent Configuration Lifecycle

### Draft → Validate → Publish

1. User creates an Agent Instance from an approved Template.
2. User edits a **mutable working draft**.
3. Platform validates the **specific draft revision**.
4. Publish checks that validation still applies, then creates a **new immutable published revision**.
5. Active revision reference determines the default configuration for **new** executions.
6. Existing executions remain bound to their original configuration references.

**Validation is an operation**, not necessarily a permanent Agent Instance state. Draft and published revision can coexist. An Agent Instance need not cycle through `EDITING` and `REPUBLISHING` states.

### Validation levels

| Level | Purpose | Example |
|---|---|---|
| Structural/configuration | Required fields, types, bounds | Interview duration in supported range |
| Semantic | Internal coherence, domain feasibility | Mandatory questions fit stage/time budget |
| Execution readiness | Required capabilities and permissions | Realtime capability allowed for this template |

Validation outcomes: **errors** block publishing; **warnings** communicate risk; **valid** means constraints pass. Prefer deterministic checks for deterministic requirements.

### Proposed publishing invariants

- Published revisions are immutable.
- Publish references the exact validated draft revision; editing invalidates stale validation.
- A single active published revision is the default for new executions.
- Publish retries must not create duplicate revisions.
- Existing executions are not silently migrated.
- Template version and Agent Instance revision are distinct dimensions.
- Template upgrade is explicit and compatibility-checked, not an automatic background mutation.
- Rollback, if added, selects a prior compatible revision rather than deleting history.

## 8. Effective Execution Configuration Snapshot

At initiation, execution is bound to a **stable effective configuration** assembled from:

- Template definition/version and supported workflow behavior.
- Agent Instance published revision and effective user settings.
- Applicable platform policies and capability permissions.
- Execution-specific input references and their relevant versions.

**Frozen behavioral configuration** protects reproducibility (questions, workflow settings, limits). **Live safety enforcement** can still revoke access, block tools, or apply emergency restrictions; immutable snapshots must not freeze unsafe privileges indefinitely.

Do not unnecessarily duplicate sensitive CVs, salary bands, or documents; immutable/versioned references and access boundaries may be preferable to physical copies.

### Validation at execution time

- Trust published structural/semantic validation rather than repeating all work.
- Recheck dynamic concerns: current authorization, invitation validity, resource availability, execution-specific input, and live safety restrictions.
- Bind configuration revision and execution identity consistently so retries cannot create an execution with ambiguous configuration.

## 9. Unified Execution Request Model

### Initiation stages (logical ordering)

1. **Receive intent** from UI, candidate link, API, or future trigger.
2. **Authenticate/resolve actor** and delegated authority.
3. **Authorize** access to the Agent Instance and relevant input resources.
4. **Resolve published configuration** and validate execution-specific inputs.
5. **Admit** under eligibility, duplicate detection, limits, and availability rules.
6. **Create or identify execution** with stable identity, configuration binding, request identity, and provenance.

These steps do not prescribe synchronous vs asynchronous infrastructure.

### Request semantics

A unified request must conceptually identify:

- **Target:** Agent Instance and allowed published revision.
- **Initiator/authority:** Actor, principal/delegation, resource ownership context.
- **Input:** Domain-specific input references.
- **Request identity:** Stable idempotency/deduplication identity.
- **Constraints:** Applicable limits and policies resolved from trusted configuration, not arbitrarily caller-controlled.
- **Provenance:** Trigger source, correlation, and audit context.

**Request accepted** means an execution is committed/identified; it does not mean a provider session has started. Admission rejection need not create an execution.

### Trigger ownership

| Trigger | Owner | Core receives |
|---|---|---|
| Run button | Platform application | Authorized execution request |
| Candidate link | HR application | Authorized execution request |
| API request | API/application boundary | Authorized execution request |
| Future scheduler | Scheduling capability | Authorized execution request |
| Future webhook | Integration capability | Authorized execution request |

The Core must not contain `if candidate link` or `if invoice upload` domain branches.

### Authorization and admission

- Distinguish **actor** (requesting identity), **principal** (authorization authority), and **resource owner**.
- Authorize initiation **and** relevant runtime tool/data operations; initiation is not perpetual permission.
- Candidate interview access grants only narrowly scoped rights; it does not expose internal salary ceiling, other candidates, or administrative configuration.
- Admission checks include availability, eligibility, duplicate requests, concurrency/capacity, and resource constraints.
- Realtime and task-based workloads may have different admission policies (immediate availability vs optional async handling).

### Idempotency vs business uniqueness

- Retrying the **same logical request** should resolve to the same execution.
- A domain may additionally enforce **one active execution per interview assignment**.
- Processing the same invoice twice intentionally may be valid; therefore **one input = one execution** must not be a universal rule.
- Candidate page load or reconnect must not automatically create another interview; use access verification then explicit start/join/recover behavior.
- Commit execution creation, request identity, and configuration binding consistently.

### Failure semantics

| Failure | Expected outcome |
|---|---|
| Timeout before commit | Safe retry using same request identity |
| Timeout after commit | Retry discovers existing execution |
| Duplicate candidate start | Existing authorized execution reused, subject to domain rules |
| Draft changes mid-initiation | Execution still binds one published revision |
| Authorization fails | Reject without creating execution |
| Provider fails after creation | Record operation failure; recover/fail under policy |
| Caller disconnects | Do not automatically cancel an accepted execution |
| Invitation expires | Reject new initiation under invitation policy |

## 10. Reference Agent A — HR Interviewer

### Product behavior

- HR configures interview schedule/eligibility, candidate CV, JD, mandatory questions, competency topics, stage policy, time budget, and permitted compensation discussion.
- Candidate participates through realtime voice, with notice/consent and device readiness as relevant.
- Target interview length **5–10 minutes** (earlier baseline: approximately eight minutes).
- Hybrid interaction: HR defines mandatory coverage; model adapts wording and follow-up questions; workflow controls progression.
- Stages: introduction/background/behavioral, role-specific screening, compensation expectations, closing. The precise stage structure and time allocation are template policy, not kernel concepts.
- No autonomous hiring verdict; HR receives evidence-backed report after interview.

### Stage transition and session model

1. Model proposes a stage transition.
2. HR workflow validates coverage, stage goals, time budget, and policy.
3. Runtime commits authorized transition.
4. Source stage finalizes transcript/coverage and commits handoff snapshot **before** activating the next stage.
5. New realtime session is created for the next stage.

### Context carryover

- Required: Structured evidence, question coverage, relevant finalized transcript references, and verified state.
- Optional: Narrative summary for conversational continuity.
- If optional summary fails, proceed using valid structured evidence.
- If mandatory context/policy/evidence is missing, block transition or follow explicit failure policy; never invent facts.
- Distinguish candidate claims from verified facts and trusted instructions from untrusted CV/transcript content.
- Keep confidential salary ceiling and other sensitive rules server-side.

### Transcript, evidence, recovery

- Backend maintains canonical transcript from normalized, ordered, deduplicated provider events.
- Canonical transcript is **not** proof that speech recognition is perfect or that a competency was demonstrated.
- Finalized segments/revisions have provenance; evidence references canonical transcript revisions.
- Corrections require revalidation of affected evidence.
- Checkpoints refer to committed transcript/progress boundaries, preferably completed turns.
- On disconnection, resume from checkpoint with a new model session; do not restore hidden model state.
- Fence stale session attempts so late events cannot overwrite current progress.

### Completion and reporting

- Interview execution completes after the interview and required transcript/progress finalization.
- HR report generation is **separate downstream post-processing**, with its own lifecycle, retry, immutable input references, and evidence-backed findings.
- Report failure does **not** turn an already completed interview into a failed interview.
- Report includes uncertainties and coverage gaps rather than unsupported competence or hiring judgments.

## 11. Reference Agent B — Document Analyzer

**Purpose:** Demonstrate kernel reusability with a task-based, non-realtime workload.

- **Input:** Invoice document reference and configured extraction requirements.
- **Goal:** Extract invoice number, vendor, date, total, currency; validate required fields.
- **Execution style:** Discrete model calls and validation operations, without conversational stages or realtime voice.
- **Progress:** Completed operations/checkpoints rather than interview turns.
- **Output:** Structured extraction result, field-level evidence references, and validation status.
- **Completion:** The task's own completion contract is satisfied; the primary extraction artifact belongs **inside** execution, unlike downstream HR reporting.
- **Retries:** Bounded extraction/validation attempts; incomplete fields are explicit, not invented.

### Reusability acceptance criteria

| Shared property | HR Interviewer | Document Analyzer |
|---|---|---|
| Stable execution identity | Yes | Yes |
| Common lifecycle/state invariants | Yes | Yes |
| Idempotent initiation | Candidate assignment/request | Document submission request |
| Configuration snapshot | Interview policy | Extraction policy |
| Progress references | Completed turns | Completed operations |
| Recovery | New session from checkpoint | Resume/retry unfinished work |
| Domain completion | Interview finalized | Extraction validated |
| Domain-specific states required in kernel | No | No |

Passing these tests is **evidence of initial reusability**, not proof of support for every future agent type.

## 12. MVP Delivery Boundary

### Must-have shared platform capabilities

- Template catalog with at least an HR Interviewer template.
- Agent Instance creation and constrained draft configuration.
- Validation and immutable publish revision.
- Unified execution request boundary with authorization and idempotency.
- Effective execution configuration binding.
- Minimal kernel lifecycle, authoritative state, operation outcomes, traceability.
- Basic execution monitoring and terminal output references.

### HR-specific MVP capabilities

- Candidate invitation/eligibility and start/join/recovery behavior.
- Realtime voice driver, hybrid mandatory/adaptive interview.
- Authorized stage transitions and session-per-stage handoff.
- Canonical transcript, evidence/coverage, committed checkpoint and basic recovery.
- Separate evidence-backed HR report generation.

### Conditional / defer unless validated

- Full Document Analyzer implementation: useful as architectural spike/test, not automatically part of shipped MVP.
- General-purpose tool registry, sophisticated queue, distributed scheduling, full workflow builder, approval chains, collaborative draft editing, automated template migration, comprehensive rollback UX.
- Dedicated deployments per agent, microservices, event sourcing, multi-agent planning.

**Suggested deployment shape:** Modular monolith in Go with explicit logical boundaries. Avoid turning conceptual ownership divisions into unnecessary infrastructure.

## 13. Risks, Failure Modes, and Mitigations

| Risk | Warning sign | Mitigation |
|---|---|---|
| Premature generic framework | Many extension points with no second use case | Validate with task-based Document Analyzer |
| HR semantics leak into kernel | Mandatory `stage_id`, `question_coverage` | Keep workflow progress agent-owned |
| Kernel too thin | Agents duplicate concurrency/recovery logic | Share execution integrity primitives |
| Mutable config affects running execution | Questions change mid-interview | Immutable revision binding |
| Authorization confused with trigger | Link possession grants broad permissions | Explicit actor, principal, eligibility, scoped access |
| Duplicate execution on retries | Two interviews for same assignment | Request idempotency + domain uniqueness |
| Provider session disconnect | Lost progress or duplicate questions | Committed checkpoints, attempt fencing |
| Summary/transcript mistaken for evidence | Unsupported competence assessment | Evidence provenance and validation |
| Report failure changes interview outcome | Completed interview marked failed | Separate downstream lifecycle |
| Analysis paralysis | Design decisions continue without MVP freeze | Consolidate architecture, freeze scope, then technical design |

## 14. Open Questions and Decisions to Freeze

1. **Execution creation vs activation:** Immediate start or logically decoupled start? Decoupled start was recommended, **not yet selected**.
2. **Confirm proposed platform invariants:** Immutable published revisions, active revision semantics, snapshot boundaries, live safety override, and idempotent publish/initiation.
3. **Define the minimum second-agent proof:** Conceptual design test, small runnable task-agent spike, or shipped feature?
4. **Finalize v0.1 lifecycle states:** `WAITING_INPUT` and activation/recovery handling without overloading execution states.
5. **Finalize MVP acceptance criteria:** Which end-to-end flows must be demo-ready, and which recovery cases are essential vs stretch?
6. **Define data retention/privacy policy:** Candidate consent, transcript access, sensitive CV handling, and deletion requirements.
7. **Confirm operational limits:** Realtime concurrency, provider failure fallback, timeout/deadline behavior, and cost budget.

## 15. Recommended Next Phase

**Architecture Consolidation & MVP Scope Freeze**, before technical design:

1. Review the Accepted vs Proposed decisions and resolve only those that block MVP delivery.
2. Freeze platform/kernel/HR workflow boundaries and reference-agent acceptance tests.
3. Write concrete MVP user journeys and demo success criteria.
4. Identify a minimal vertical slice: create agent → configure draft → validate/publish → authorized candidate start → realtime interview → finalized transcript → downstream report.
5. Only then proceed to Go component boundaries, API contracts, persistence, provider integrations, and implementation sequencing.

---

**Document note:** This is an architecture-design record assembled from the current conversation. It supersedes earlier **conceptual** assumptions where the discussion evolved (notably moving from an HR-only MVP framing to a shared-platform architectural vision), but it does not imply that every proposed capability must be implemented in v0.1.
